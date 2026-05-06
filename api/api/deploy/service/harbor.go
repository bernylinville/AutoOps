package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	ccdao "dodevops-api/api/configcenter/dao"
	ccmodel "dodevops-api/api/configcenter/model"
	logpkg "dodevops-api/pkg/log"
)

const (
	defaultHarborPollInterval = 15 * time.Second
	defaultHarborScanTimeout  = 30 * time.Minute
	harborAPIVersionPath      = "/api/v2.0"
	harborVulnAcceptHeader    = "application/vnd.security.vulnerability.report; version=1.1, application/vnd.scanner.adapter.vuln.report.harbor+json; version=1.0"
)

type IHarborAdapter interface {
	IsConfigured() bool
	GetArtifact(ctx context.Context, serverID uint, project, repository, tag string) (*HarborArtifact, error)
	TriggerScan(ctx context.Context, serverID uint, project, repository, digest string) error
	GetScanOverview(ctx context.Context, serverID uint, project, repository, digest string) (*HarborScanOverview, error)
	GetVulnerabilities(ctx context.Context, serverID uint, project, repository, digest string) ([]HarborVulnerability, error)
	PollScanUntilComplete(ctx context.Context, serverID uint, project, repository, digest string, timeout time.Duration) (*HarborScanOverview, error)
	EvaluateScanPolicy(overview *HarborScanOverview, policy *ScanPolicy) (*ScanEvaluationResult, error)
}

type HarborArtifact struct {
	Digest       string
	Tags         []string
	Size         int64
	PushTime     time.Time
	ScanOverview *HarborScanOverview
}

type HarborScanOverview struct {
	ScanStatus      string
	Severity        string
	CompletePercent int
	Summary         map[string]int
}

type HarborVulnerability struct {
	ID          string
	Severity    string
	Package     string
	Version     string
	FixVersion  string
	Description string
	Links       []string
}

type ScanPolicy struct {
	MaxCritical int
	MaxHigh     int
	MaxMedium   int
}

type ScanEvaluationResult struct {
	Passed   bool
	Critical int
	High     int
	Medium   int
	Low      int
	Message  string
}

type harborAccountStore interface {
	GetByID(id uint) (*ccmodel.AccountAuth, error)
}

type HarborAdapterOptions struct {
	AccountDao         harborAccountStore
	HTTPClient         *http.Client
	AllowInsecureHTTP  bool
	PollInterval       time.Duration
	DefaultScanTimeout time.Duration
}

type HarborAdapter struct {
	accountDao        harborAccountStore
	httpClient        *http.Client
	allowInsecureHTTP bool
	pollInterval      time.Duration
	defaultTimeout    time.Duration
}

type harborServerConfig struct {
	APIBaseURL string
	Username   string
	Password   string
	Alias      string
}

func NewHarborAdapter(opts HarborAdapterOptions) IHarborAdapter {
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultHarborPollInterval
	}

	defaultTimeout := opts.DefaultScanTimeout
	if defaultTimeout <= 0 {
		defaultTimeout = defaultHarborScanTimeout
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	accountDao := opts.AccountDao
	if accountDao == nil {
		accountDao = ccdao.NewAccountAuthDao()
	}

	return &HarborAdapter{
		accountDao:        accountDao,
		httpClient:        httpClient,
		allowInsecureHTTP: opts.AllowInsecureHTTP,
		pollInterval:      pollInterval,
		defaultTimeout:    defaultTimeout,
	}
}

func (h *HarborAdapter) IsConfigured() bool {
	return h != nil && h.accountDao != nil && h.httpClient != nil
}

func (h *HarborAdapter) GetArtifact(ctx context.Context, serverID uint, project, repository, tag string) (*HarborArtifact, error) {
	if !h.IsConfigured() {
		return nil, fmt.Errorf("Harbor 适配器未初始化")
	}

	var raw struct {
		Digest   string    `json:"digest"`
		Size     int64     `json:"size"`
		PushTime time.Time `json:"push_time"`
		Tags     []struct {
			Name string `json:"name"`
		} `json:"tags"`
		ScanOverview map[string]json.RawMessage `json:"scan_overview"`
	}

	endpoint := h.artifactEndpoint(project, repository, tag) + "?with_scan_overview=true&with_tag=true"
	if err := h.doJSON(ctx, serverID, http.MethodGet, endpoint, nil, &raw); err != nil {
		return nil, err
	}

	artifact := &HarborArtifact{
		Digest:   raw.Digest,
		Size:     raw.Size,
		PushTime: raw.PushTime,
	}
	for _, tagInfo := range raw.Tags {
		if name := strings.TrimSpace(tagInfo.Name); name != "" {
			artifact.Tags = append(artifact.Tags, name)
		}
	}

	if len(raw.ScanOverview) > 0 {
		artifact.ScanOverview = parseHarborScanOverview(raw.ScanOverview)
	}

	return artifact, nil
}

func (h *HarborAdapter) TriggerScan(ctx context.Context, serverID uint, project, repository, digest string) error {
	if !h.IsConfigured() {
		return fmt.Errorf("Harbor 适配器未初始化")
	}
	endpoint := h.artifactEndpoint(project, repository, digest) + "/scan"
	statusCode, _, err := h.doRequest(ctx, serverID, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	if statusCode != http.StatusAccepted && statusCode != http.StatusOK && statusCode != http.StatusCreated && statusCode != http.StatusNoContent {
		return fmt.Errorf("触发 Harbor 扫描失败: unexpected status %d", statusCode)
	}
	logpkg.Log().Infof("triggered Harbor scan: server=%d project=%s repository=%s digest=%s", serverID, project, repository, digest)
	return nil
}

func (h *HarborAdapter) GetScanOverview(ctx context.Context, serverID uint, project, repository, digest string) (*HarborScanOverview, error) {
	artifact, err := h.GetArtifact(ctx, serverID, project, repository, digest)
	if err != nil {
		return nil, err
	}
	if artifact == nil {
		return nil, nil
	}
	return artifact.ScanOverview, nil
}

func (h *HarborAdapter) GetVulnerabilities(ctx context.Context, serverID uint, project, repository, digest string) ([]HarborVulnerability, error) {
	if !h.IsConfigured() {
		return nil, fmt.Errorf("Harbor 适配器未初始化")
	}

	endpoint := h.artifactEndpoint(project, repository, digest) + "/additions/vulnerabilities"
	_, body, err := h.doRequest(ctx, serverID, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("解析 Harbor 漏洞报告失败: %w", err)
	}

	items := extractVulnerabilityItems(payload)
	vulnerabilities := make([]HarborVulnerability, 0, len(items))
	for _, item := range items {
		vulnerabilities = append(vulnerabilities, HarborVulnerability{
			ID:          firstNonEmptyString(stringValue(item["id"]), stringValue(item["vulnerabilityID"]), stringValue(item["cve_id"])),
			Severity:    normalizeSeverity(stringValue(item["severity"])),
			Package:     extractPackageName(item),
			Version:     firstNonEmptyString(stringValue(item["version"]), stringValue(item["package_version"]), nestedString(item["package"], "version")),
			FixVersion:  firstNonEmptyString(stringValue(item["fix_version"]), stringValue(item["fixed_version"]), nestedString(item["package"], "fix_version")),
			Description: stringValue(item["description"]),
			Links:       extractLinks(item["links"]),
		})
	}

	return vulnerabilities, nil
}

func (h *HarborAdapter) PollScanUntilComplete(ctx context.Context, serverID uint, project, repository, digest string, timeout time.Duration) (*HarborScanOverview, error) {
	if timeout <= 0 {
		timeout = h.defaultTimeout
	}
	if timeout <= 0 {
		timeout = defaultHarborScanTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	overview, err := h.GetScanOverview(ctx, serverID, project, repository, digest)
	if err != nil {
		return nil, err
	}
	if shouldTriggerHarborScan(overview) {
		if err := h.TriggerScan(ctx, serverID, project, repository, digest); err != nil {
			return nil, err
		}
	}

	ticker := time.NewTicker(h.pollInterval)
	defer ticker.Stop()

	for {
		overview, err = h.GetScanOverview(ctx, serverID, project, repository, digest)
		if err != nil {
			return nil, err
		}

		status := strings.ToLower(strings.TrimSpace(firstNonEmptyString(overvScanStatus(overview), "pending")))
		switch status {
		case "success", "finished", "complete", "completed":
			if overview != nil && overview.CompletePercent == 0 {
				overview.CompletePercent = 100
			}
			return overview, nil
		case "error", "failed", "stopped", "unsupported":
			return overview, fmt.Errorf("Harbor 扫描失败，状态=%s", firstNonEmptyString(overvScanStatus(overview), status))
		}

		select {
		case <-ctx.Done():
			return overview, fmt.Errorf("轮询 Harbor 扫描超时: %w", ctx.Err())
		case <-ticker.C:
			logpkg.Log().Infof("polling Harbor scan: server=%d project=%s repository=%s digest=%s status=%s", serverID, project, repository, digest, firstNonEmptyString(overvScanStatus(overview), "unknown"))
		}
	}
}

func (h *HarborAdapter) EvaluateScanPolicy(overview *HarborScanOverview, policy *ScanPolicy) (*ScanEvaluationResult, error) {
	if overview == nil {
		return nil, fmt.Errorf("Harbor 扫描概览为空")
	}

	result := &ScanEvaluationResult{
		Passed:   true,
		Critical: severityCount(overview.Summary, "Critical"),
		High:     severityCount(overview.Summary, "High"),
		Medium:   severityCount(overview.Summary, "Medium"),
		Low:      severityCount(overview.Summary, "Low"),
	}

	status := strings.ToLower(strings.TrimSpace(overview.ScanStatus))
	if status != "success" && status != "finished" && status != "complete" && status != "completed" {
		result.Passed = false
		result.Message = fmt.Sprintf("扫描未完成，当前状态=%s", firstNonEmptyString(overview.ScanStatus, "unknown"))
		return result, nil
	}

	if policy == nil {
		result.Message = fmt.Sprintf("未配置安全策略，扫描完成：Critical=%d High=%d Medium=%d Low=%d", result.Critical, result.High, result.Medium, result.Low)
		return result, nil
	}

	var reasons []string
	if result.Critical > policy.MaxCritical {
		result.Passed = false
		reasons = append(reasons, fmt.Sprintf("Critical 漏洞 %d 超过阈值 %d", result.Critical, policy.MaxCritical))
	}
	if result.High > policy.MaxHigh {
		result.Passed = false
		reasons = append(reasons, fmt.Sprintf("High 漏洞 %d 超过阈值 %d", result.High, policy.MaxHigh))
	}
	if policy.MaxMedium > 0 && result.Medium > policy.MaxMedium {
		result.Passed = false
		reasons = append(reasons, fmt.Sprintf("Medium 漏洞 %d 超过阈值 %d", result.Medium, policy.MaxMedium))
	}

	if len(reasons) == 0 {
		result.Message = fmt.Sprintf("扫描通过：Critical=%d/%d High=%d/%d", result.Critical, policy.MaxCritical, result.High, policy.MaxHigh)
		if policy.MaxMedium > 0 {
			result.Message += fmt.Sprintf(" Medium=%d/%d", result.Medium, policy.MaxMedium)
		}
		return result, nil
	}

	result.Message = strings.Join(reasons, "；")
	return result, nil
}

func (h *HarborAdapter) doJSON(ctx context.Context, serverID uint, method, endpoint string, body io.Reader, target interface{}) error {
	_, respBody, err := h.doRequest(ctx, serverID, method, endpoint, body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(respBody, target); err != nil {
		return fmt.Errorf("解析 Harbor 响应失败: %w", err)
	}
	return nil
}

func (h *HarborAdapter) doRequest(ctx context.Context, serverID uint, method, endpoint string, body io.Reader) (int, []byte, error) {
	serverCfg, err := h.getServerConfig(serverID)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, serverCfg.APIBaseURL+endpoint, body)
	if err != nil {
		return 0, nil, fmt.Errorf("创建 Harbor 请求失败: %w", err)
	}
	req.SetBasicAuth(serverCfg.Username, serverCfg.Password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Accept-Vulnerabilities", harborVulnAcceptHeader)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		logpkg.Log().Errorf("Harbor request failed: method=%s endpoint=%s err=%v", method, endpoint, err)
		return 0, nil, fmt.Errorf("Harbor 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return resp.StatusCode, nil, fmt.Errorf("读取 Harbor 响应失败: %w", readErr)
	}

	if resp.StatusCode >= 400 {
		message := extractHarborError(respBody)
		if message == "" {
			message = strings.TrimSpace(string(respBody))
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return resp.StatusCode, respBody, fmt.Errorf("Harbor API %s %s 返回 %d: %s", method, endpoint, resp.StatusCode, message)
	}

	return resp.StatusCode, respBody, nil
}

func (h *HarborAdapter) getServerConfig(serverID uint) (*harborServerConfig, error) {
	account, err := h.accountDao.GetByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("读取 Harbor 凭据失败: %w", err)
	}
	if account == nil {
		return nil, fmt.Errorf("Harbor 凭据不存在: %d", serverID)
	}

	password, err := account.DecryptPassword()
	if err != nil {
		return nil, fmt.Errorf("解密 Harbor 凭据失败: %w", err)
	}

	apiBaseURL, err := h.normalizeHarborBaseURL(account.Host, account.Port)
	if err != nil {
		return nil, err
	}

	return &harborServerConfig{
		APIBaseURL: apiBaseURL,
		Username:   account.Name,
		Password:   password,
		Alias:      account.Alias,
	}, nil
}

func (h *HarborAdapter) normalizeHarborBaseURL(host string, port int) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("Harbor Host 为空")
	}

	if !strings.Contains(host, "://") {
		scheme := "https"
		if h.allowInsecureHTTP {
			scheme = "http"
		}
		host = scheme + "://" + host
	}

	parsed, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("解析 Harbor Host 失败: %w", err)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("Harbor Host 无效: %s", host)
	}
	if parsed.Port() == "" && port > 0 {
		parsed.Host = parsed.Hostname() + ":" + strconv.Itoa(port)
	}
	if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
		parsed.Host = parsed.Hostname()
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(parsed.Path, harborAPIVersionPath) {
		parsed.Path += harborAPIVersionPath
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

func (h *HarborAdapter) artifactEndpoint(project, repository, reference string) string {
	return "/projects/" + url.PathEscape(strings.TrimSpace(project)) +
		"/repositories/" + url.PathEscape(strings.TrimSpace(repository)) +
		"/artifacts/" + url.PathEscape(strings.TrimSpace(reference))
}

func parseHarborScanOverview(raw map[string]json.RawMessage) *HarborScanOverview {
	for _, item := range raw {
		var payload map[string]interface{}
		if err := json.Unmarshal(item, &payload); err != nil {
			continue
		}
		overview := &HarborScanOverview{
			ScanStatus:      firstNonEmptyString(stringValue(payload["scan_status"]), stringValue(payload["status"])),
			Severity:        normalizeSeverity(firstNonEmptyString(stringValue(payload["severity"]), nestedString(payload["summary"], "severity"))),
			CompletePercent: intValue(payload["complete_percent"]),
			Summary:         extractSeveritySummary(payload["summary"]),
		}
		if overview.ScanStatus == "" && len(overview.Summary) == 0 && overview.Severity == "" {
			continue
		}
		if overview.CompletePercent == 0 && strings.EqualFold(overview.ScanStatus, "success") {
			overview.CompletePercent = 100
		}
		return overview
	}
	return nil
}

func extractSeveritySummary(raw interface{}) map[string]int {
	summary := map[string]int{}
	data, ok := raw.(map[string]interface{})
	if !ok {
		return summary
	}

	collectSeverityMap(summary, data)
	if nested, ok := data["summary"].(map[string]interface{}); ok {
		collectSeverityMap(summary, nested)
	}
	return summary
}

func collectSeverityMap(target map[string]int, values map[string]interface{}) {
	for key, value := range values {
		severityKey := canonicalSeverityKey(key)
		if severityKey == "" {
			continue
		}
		target[severityKey] = intValue(value)
	}
}

func extractVulnerabilityItems(payload interface{}) []map[string]interface{} {
	if list, ok := payload.([]interface{}); ok {
		return interfaceSliceToMaps(list)
	}
	root, ok := payload.(map[string]interface{})
	if !ok {
		return nil
	}
	if list, ok := root["vulnerabilities"].([]interface{}); ok {
		return interfaceSliceToMaps(list)
	}
	if report, ok := root["report"].(map[string]interface{}); ok {
		if list, ok := report["vulnerabilities"].([]interface{}); ok {
			return interfaceSliceToMaps(list)
		}
	}
	return nil
}

func interfaceSliceToMaps(items []interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, m)
		}
	}
	return result
}

func extractPackageName(item map[string]interface{}) string {
	if pkg := stringValue(item["package"]); pkg != "" {
		return pkg
	}
	if pkg := stringValue(item["package_name"]); pkg != "" {
		return pkg
	}
	if pkg := stringValue(item["packageName"]); pkg != "" {
		return pkg
	}
	return nestedString(item["package"], "name")
}

func extractLinks(raw interface{}) []string {
	var links []string
	switch value := raw.(type) {
	case []interface{}:
		for _, item := range value {
			switch link := item.(type) {
			case string:
				if strings.TrimSpace(link) != "" {
					links = append(links, strings.TrimSpace(link))
				}
			case map[string]interface{}:
				if href := firstNonEmptyString(stringValue(link["href"]), stringValue(link["url"])); href != "" {
					links = append(links, href)
				}
			}
		}
	case string:
		if strings.TrimSpace(value) != "" {
			links = append(links, strings.TrimSpace(value))
		}
	}
	return links
}

func extractHarborError(body []byte) string {
	var single map[string]interface{}
	if err := json.Unmarshal(body, &single); err == nil {
		return firstNonEmptyString(stringValue(single["message"]), stringValue(single["error"]))
	}

	var many []map[string]interface{}
	if err := json.Unmarshal(body, &many); err == nil {
		messages := make([]string, 0, len(many))
		for _, item := range many {
			message := firstNonEmptyString(stringValue(item["message"]), stringValue(item["error"]))
			if message != "" {
				messages = append(messages, message)
			}
		}
		return strings.Join(messages, "; ")
	}

	return ""
}

func shouldTriggerHarborScan(overview *HarborScanOverview) bool {
	if overview == nil {
		return true
	}
	status := strings.ToLower(strings.TrimSpace(overview.ScanStatus))
	return status == "" || status == "none" || status == "unknown" || status == "not_scanned" || status == "not scanned"
}

func severityCount(summary map[string]int, severity string) int {
	if len(summary) == 0 {
		return 0
	}
	return summary[canonicalSeverityKey(severity)]
}

func canonicalSeverityKey(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return "Critical"
	case "high":
		return "High"
	case "medium", "moderate":
		return "Medium"
	case "low":
		return "Low"
	case "none", "negligible", "unknown":
		return "None"
	default:
		return ""
	}
}

func normalizeSeverity(value string) string {
	if normalized := canonicalSeverityKey(value); normalized != "" {
		return normalized
	}
	return strings.TrimSpace(value)
}

func stringValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func nestedString(value interface{}, key string) string {
	if m, ok := value.(map[string]interface{}); ok {
		return stringValue(m[key])
	}
	return ""
}

func intValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(v))
		return i
	default:
		return 0
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func overvScanStatus(overview *HarborScanOverview) string {
	if overview == nil {
		return ""
	}
	return overview.ScanStatus
}

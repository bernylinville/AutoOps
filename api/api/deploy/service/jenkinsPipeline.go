package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	appmodel "dodevops-api/api/app/model"
	appservice "dodevops-api/api/app/service"
	ccdao "dodevops-api/api/configcenter/dao"
	ccmodel "dodevops-api/api/configcenter/model"
)

const (
	defaultJenkinsPipelinePollInterval = 10 * time.Second
	defaultJenkinsPipelineTimeout      = time.Hour
)

type IJenkinsPipelineAdapter interface {
	TriggerBuild(ctx context.Context, serverID uint, jobName string, params map[string]string) (queueID int, err error)
	GetBuildNumberFromQueue(ctx context.Context, serverID uint, queueID int) (buildNumber int, err error)
	GetBuildDetail(ctx context.Context, serverID uint, jobName string, buildNumber int) (*JenkinsBuildDetail, error)
	PollBuildUntilComplete(ctx context.Context, serverID uint, jobName string, buildNumber int, timeout time.Duration) (*JenkinsBuildDetail, error)
	ExtractImageTagFromBuildLog(ctx context.Context, serverID uint, jobName string, buildNumber int) (string, error)
}

type JenkinsBuildDetail struct {
	Number     int               `json:"number"`
	URL        string            `json:"url"`
	Result     string            `json:"result"`
	Building   bool              `json:"building"`
	Duration   int64             `json:"duration"`
	Timestamp  int64             `json:"timestamp"`
	Parameters map[string]string `json:"parameters"`
	ConsoleLog string            `json:"consoleLog,omitempty"`
}

type jenkinsAccountStore interface {
	GetByID(id uint) (*ccmodel.AccountAuth, error)
}

type JenkinsPipelineAdapter struct {
	accountDao       jenkinsAccountStore
	newClient        func(host string, port int, username, password string) *appservice.JenkinsClient
	pollInterval     time.Duration
	defaultTimeout   time.Duration
	imageTagPatterns []*regexp.Regexp
}

func NewJenkinsPipelineAdapter() IJenkinsPipelineAdapter {
	return &JenkinsPipelineAdapter{
		accountDao:       ccdao.NewAccountAuthDao(),
		newClient:        appservice.NewJenkinsClient,
		pollInterval:     defaultJenkinsPipelinePollInterval,
		defaultTimeout:   defaultJenkinsPipelineTimeout,
		imageTagPatterns: compileJenkinsImageTagPatterns(),
	}
}

func (a *JenkinsPipelineAdapter) TriggerBuild(ctx context.Context, serverID uint, jobName string, params map[string]string) (int, error) {
	client, err := a.getJenkinsClient(serverID)
	if err != nil {
		return 0, fmt.Errorf("获取 Jenkins 客户端失败: %v", err)
	}

	endpoint, err := buildJenkinsJobEndpoint(jobName)
	if err != nil {
		return 0, err
	}
	if len(params) > 0 {
		values := url.Values{}
		for key, value := range params {
			values.Set(key, value)
		}
		endpoint += "/buildWithParameters"
		if encoded := values.Encode(); encoded != "" {
			endpoint += "?" + encoded
		}
	} else {
		endpoint += "/build"
	}

	resp, err := a.doRequest(ctx, client, http.MethodPost, endpoint, nil, http.StatusCreated, http.StatusOK)
	if err != nil {
		return 0, fmt.Errorf("触发 Jenkins Pipeline 构建失败: %v", err)
	}
	defer resp.Body.Close()

	queueID, err := parseJenkinsQueueID(resp.Header.Get("Location"))
	if err != nil {
		return 0, fmt.Errorf("解析 Jenkins 队列 ID 失败: %v", err)
	}

	log.Printf("jenkins pipeline trigger accepted: server=%d job=%s queueID=%d", serverID, jobName, queueID)
	return queueID, nil
}

func (a *JenkinsPipelineAdapter) GetBuildNumberFromQueue(ctx context.Context, serverID uint, queueID int) (int, error) {
	if queueID <= 0 {
		return 0, fmt.Errorf("queueID 非法: %d", queueID)
	}

	client, err := a.getJenkinsClient(serverID)
	if err != nil {
		return 0, fmt.Errorf("获取 Jenkins 客户端失败: %v", err)
	}

	pollCtx, cancel := context.WithTimeout(ctx, a.effectiveTimeout(0))
	defer cancel()

	ticker := time.NewTicker(a.effectivePollInterval())
	defer ticker.Stop()

	for {
		buildNumber, done, err := a.getBuildNumberFromQueueOnce(pollCtx, client, queueID)
		if err != nil {
			if isContextError(err) || pollCtx.Err() != nil {
				return 0, fmt.Errorf("轮询 Jenkins 队列超时: %w", pollCtx.Err())
			}
			return 0, err
		}
		if done {
			log.Printf("jenkins queue resolved: server=%d queueID=%d buildNumber=%d", serverID, queueID, buildNumber)
			return buildNumber, nil
		}

		select {
		case <-pollCtx.Done():
			return 0, fmt.Errorf("轮询 Jenkins 队列超时: %w", pollCtx.Err())
		case <-ticker.C:
		}
	}
}

func (a *JenkinsPipelineAdapter) GetBuildDetail(ctx context.Context, serverID uint, jobName string, buildNumber int) (*JenkinsBuildDetail, error) {
	client, err := a.getJenkinsClient(serverID)
	if err != nil {
		return nil, fmt.Errorf("获取 Jenkins 客户端失败: %v", err)
	}
	return a.getBuildDetailWithClient(ctx, client, jobName, buildNumber)
}

func (a *JenkinsPipelineAdapter) PollBuildUntilComplete(ctx context.Context, serverID uint, jobName string, buildNumber int, timeout time.Duration) (*JenkinsBuildDetail, error) {
	if buildNumber <= 0 {
		return nil, fmt.Errorf("buildNumber 非法: %d", buildNumber)
	}

	pollTimeout := a.effectiveTimeout(timeout)
	pollCtx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()

	var lastDetail *JenkinsBuildDetail
	for {
		detail, err := a.GetBuildDetail(pollCtx, serverID, jobName, buildNumber)
		if err != nil {
			if isContextError(err) || pollCtx.Err() != nil {
				return nil, fmt.Errorf("轮询 Jenkins 构建超时: %w", pollCtx.Err())
			}
			return nil, err
		}
		lastDetail = detail
		if isJenkinsBuildComplete(detail) {
			log.Printf("jenkins build completed: server=%d job=%s buildNumber=%d result=%s", serverID, jobName, buildNumber, detail.Result)
			return detail, nil
		}

		select {
		case <-pollCtx.Done():
			return nil, fmt.Errorf("轮询 Jenkins 构建超时: %w", pollCtx.Err())
		case <-time.After(a.effectivePollInterval()):
			if lastDetail != nil {
				log.Printf("jenkins build still running: server=%d job=%s buildNumber=%d", serverID, jobName, buildNumber)
			}
		}
	}
}

func (a *JenkinsPipelineAdapter) ExtractImageTagFromBuildLog(ctx context.Context, serverID uint, jobName string, buildNumber int) (string, error) {
	client, err := a.getJenkinsClient(serverID)
	if err != nil {
		return "", fmt.Errorf("获取 Jenkins 客户端失败: %v", err)
	}

	consoleLog, err := a.getBuildConsoleLog(ctx, client, jobName, buildNumber)
	if err != nil {
		return "", err
	}

	for _, pattern := range a.imageTagPatterns {
		matches := pattern.FindStringSubmatch(consoleLog)
		if len(matches) < 2 {
			continue
		}
		value := strings.TrimSpace(matches[1])
		value = strings.Trim(value, `"'`)
		if value == "" {
			continue
		}
		log.Printf("jenkins build image tag extracted: server=%d job=%s buildNumber=%d value=%s", serverID, jobName, buildNumber, value)
		return value, nil
	}

	return "", fmt.Errorf("未在 Jenkins 构建日志中提取到镜像标签")
}

func (a *JenkinsPipelineAdapter) getJenkinsClient(serverID uint) (*appservice.JenkinsClient, error) {
	account, err := a.accountDao.GetByID(serverID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, fmt.Errorf("Jenkins 服务器不存在")
	}
	if account.Type != appmodel.JenkinsAccountType {
		return nil, fmt.Errorf("账号类型不是 Jenkins")
	}

	password, err := account.DecryptPassword()
	if err != nil {
		return nil, fmt.Errorf("解密 Jenkins 凭据失败: %v", err)
	}

	return a.newClient(account.Host, account.Port, account.Name, password), nil
}

func (a *JenkinsPipelineAdapter) getBuildDetailWithClient(ctx context.Context, client *appservice.JenkinsClient, jobName string, buildNumber int) (*JenkinsBuildDetail, error) {
	if buildNumber <= 0 {
		return nil, fmt.Errorf("buildNumber 非法: %d", buildNumber)
	}

	jobEndpoint, err := buildJenkinsJobEndpoint(jobName)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/%d/api/json", jobEndpoint, buildNumber)

	resp, err := a.doRequest(ctx, client, http.MethodGet, endpoint, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("获取 Jenkins 构建详情失败: %v", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Number    int    `json:"number"`
		URL       string `json:"url"`
		Result    string `json:"result"`
		Building  bool   `json:"building"`
		Duration  int64  `json:"duration"`
		Timestamp int64  `json:"timestamp"`
		Actions   []struct {
			Parameters []struct {
				Name  string      `json:"name"`
				Value interface{} `json:"value"`
			} `json:"parameters"`
		} `json:"actions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("解析 Jenkins 构建详情失败: %v", err)
	}

	detail := &JenkinsBuildDetail{
		Number:     payload.Number,
		URL:        payload.URL,
		Result:     payload.Result,
		Building:   payload.Building,
		Duration:   payload.Duration,
		Timestamp:  payload.Timestamp,
		Parameters: map[string]string{},
	}
	for _, action := range payload.Actions {
		for _, parameter := range action.Parameters {
			name := strings.TrimSpace(parameter.Name)
			if name == "" {
				continue
			}
			detail.Parameters[name] = fmt.Sprint(parameter.Value)
		}
	}

	return detail, nil
}

func (a *JenkinsPipelineAdapter) getBuildNumberFromQueueOnce(ctx context.Context, client *appservice.JenkinsClient, queueID int) (int, bool, error) {
	endpoint := fmt.Sprintf("/queue/item/%d/api/json", queueID)
	resp, err := a.doRequest(ctx, client, http.MethodGet, endpoint, nil, http.StatusOK)
	if err != nil {
		return 0, false, fmt.Errorf("获取 Jenkins 队列详情失败: %v", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Cancelled  bool `json:"cancelled"`
		Executable *struct {
			Number int `json:"number"`
		} `json:"executable"`
		Why string `json:"why"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, false, fmt.Errorf("解析 Jenkins 队列详情失败: %v", err)
	}

	if payload.Cancelled {
		return 0, false, fmt.Errorf("Jenkins 队列任务已取消")
	}
	if payload.Executable != nil && payload.Executable.Number > 0 {
		return payload.Executable.Number, true, nil
	}
	if payload.Why != "" {
		log.Printf("jenkins queue waiting: queueID=%d why=%s", queueID, payload.Why)
	}

	return 0, false, nil
}

func (a *JenkinsPipelineAdapter) getBuildConsoleLog(ctx context.Context, client *appservice.JenkinsClient, jobName string, buildNumber int) (string, error) {
	jobEndpoint, err := buildJenkinsJobEndpoint(jobName)
	if err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("%s/%d/consoleText", jobEndpoint, buildNumber)
	resp, err := a.doRequest(ctx, client, http.MethodGet, endpoint, nil, http.StatusOK)
	if err != nil {
		return "", fmt.Errorf("获取 Jenkins 构建日志失败: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Jenkins 构建日志失败: %v", err)
	}
	return string(data), nil
}

func (a *JenkinsPipelineAdapter) doRequest(ctx context.Context, client *appservice.JenkinsClient, method, endpoint string, body io.Reader, expectedStatus ...int) (*http.Response, error) {
	requestURL := client.BaseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(client.Username, client.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if !containsHTTPStatus(expectedStatus, resp.StatusCode) {
		defer resp.Body.Close()
		payload, _ := io.ReadAll(resp.Body)
		message := strings.TrimSpace(string(payload))
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("Jenkins 响应错误: %d %s", resp.StatusCode, message)
	}

	return resp, nil
}

func (a *JenkinsPipelineAdapter) effectivePollInterval() time.Duration {
	if a.pollInterval <= 0 {
		return defaultJenkinsPipelinePollInterval
	}
	return a.pollInterval
}

func (a *JenkinsPipelineAdapter) effectiveTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	if a.defaultTimeout <= 0 {
		return defaultJenkinsPipelineTimeout
	}
	return a.defaultTimeout
}

func buildJenkinsJobEndpoint(jobName string) (string, error) {
	segments := strings.Split(strings.TrimSpace(jobName), "/")
	parts := make([]string, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		parts = append(parts, url.PathEscape(segment))
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("jobName 不能为空")
	}
	return "/job/" + strings.Join(parts, "/job/"), nil
}

func parseJenkinsQueueID(location string) (int, error) {
	location = strings.TrimSpace(location)
	if location == "" {
		return 0, fmt.Errorf("响应头缺少 Location")
	}

	parsed, err := url.Parse(location)
	if err == nil && parsed.Path != "" {
		location = parsed.Path
	}

	match := regexp.MustCompile(`/queue/item/(\d+)/?`).FindStringSubmatch(location)
	if len(match) != 2 {
		return 0, fmt.Errorf("无法从 Location 中解析队列 ID: %s", location)
	}

	queueID, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, err
	}
	return queueID, nil
}

func containsHTTPStatus(expected []int, actual int) bool {
	for _, status := range expected {
		if status == actual {
			return true
		}
	}
	return false
}

func isJenkinsBuildComplete(detail *JenkinsBuildDetail) bool {
	if detail == nil {
		return false
	}
	return !detail.Building && strings.TrimSpace(detail.Result) != ""
}

func isContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

func compileJenkinsImageTagPatterns() []*regexp.Regexp {
	return []*regexp.Regexp{
		regexp.MustCompile(`(?im)\bIMAGE_TAG\s*[:=]\s*["']?([^\s"']+)`),
		regexp.MustCompile(`(?im)(?:镜像标签|image tag)\s*[:=]\s*["']?([^\s"']+)`),
		regexp.MustCompile(`(?im)\b(?:docker|podman)\s+(?:push|pull)\s+([^\s]+:[^\s]+)`),
		regexp.MustCompile(`(?im)\b(?:pushed image|image pushed|镜像地址)\s*[:=]\s*["']?([^\s"']+:[^\s"']+)`),
	}
}

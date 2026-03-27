package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/pkg/log"
)

// VMService VictoriaMetrics 查询服务
type VMService struct {
	httpClient *http.Client
}

var vmServiceInstance *VMService
var vmOnce sync.Once

// GetVMService 获取 VM 服务单例
func GetVMService() *VMService {
	vmOnce.Do(func() {
		vmServiceInstance = &VMService{
			httpClient: &http.Client{Timeout: 30 * time.Second},
		}
	})
	return vmServiceInstance
}

// VMQueryResult VictoriaMetrics 查询结果
type VMQueryResult struct {
	Status string     `json:"status"`
	Data   VMDataBody `json:"data"`
}

// VMDataBody 查询数据体
type VMDataBody struct {
	ResultType string           `json:"resultType"`
	Result     []VMResultSeries `json:"result"`
}

// VMResultSeries 单条时间序列
type VMResultSeries struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`  // instant query
	Values [][]interface{}   `json:"values,omitempty"` // range query
}

// HostMetricsSnapshot 主机实时监控快照
type HostMetricsSnapshot struct {
	Ident        string  `json:"ident"`
	CPUUsage     float64 `json:"cpuUsage"`
	MemoryUsage  float64 `json:"memoryUsage"`
	DiskUsage    float64 `json:"diskUsage"`
	Load1        float64 `json:"load1"`
	Load5        float64 `json:"load5"`
	Load15       float64 `json:"load15"`
	Uptime       float64 `json:"uptime"`
	CPUCores     float64 `json:"cpuCores"`
	MemTotal     float64 `json:"memTotal"`
	MemAvailable float64 `json:"memAvailable"`
}

// ClusterOverview 集群监控总览
type ClusterOverview struct {
	HostCount     int     `json:"hostCount"`
	AvgCPUUsage   float64 `json:"avgCpuUsage"`
	AvgMemUsage   float64 `json:"avgMemUsage"`
	MaxCPUUsage   float64 `json:"maxCpuUsage"`
	MaxMemUsage   float64 `json:"maxMemUsage"`
	MaxDiskUsage  float64 `json:"maxDiskUsage"`
	TopCPUHosts   []HostMetricValue `json:"topCpuHosts"`
	TopMemHosts   []HostMetricValue `json:"topMemHosts"`
	TopDiskHosts  []HostMetricValue `json:"topDiskHosts"`
}

// HostMetricValue 主机+指标值
type HostMetricValue struct {
	Ident string  `json:"ident"`
	Value float64 `json:"value"`
}

// getDatasourceURL 获取数据源 URL
func (s *VMService) getDatasourceURL(datasourceID uint) (string, error) {
	if datasourceID == 0 {
		// 尝试获取第一个可用数据源
		dsList, err := dao.GetN9EDataSources()
		if err != nil || len(dsList) == 0 {
			return "", fmt.Errorf("没有可用的数据源")
		}
		return strings.TrimRight(dsList[0].URL, "/"), nil
	}
	ds, err := dao.GetN9EDataSourceByID(datasourceID)
	if err != nil {
		return "", fmt.Errorf("数据源 %d 不存在: %w", datasourceID, err)
	}
	if ds.URL == "" {
		return "", fmt.Errorf("数据源 %d 未配置 URL", datasourceID)
	}
	return strings.TrimRight(ds.URL, "/"), nil
}

// QueryInstant 即时 PromQL 查询
func (s *VMService) QueryInstant(ctx context.Context, dsURL, query string) (*VMQueryResult, error) {
	u, err := url.Parse(dsURL + "/api/v1/query")
	if err != nil {
		return nil, fmt.Errorf("解析 URL 失败: %w", err)
	}
	params := url.Values{}
	params.Set("query", query)
	u.RawQuery = params.Encode()
	return s.doQuery(ctx, u.String())
}

// QueryRange 范围 PromQL 查询
func (s *VMService) QueryRange(ctx context.Context, dsURL, query, start, end, step string) (*VMQueryResult, error) {
	u, err := url.Parse(dsURL + "/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("解析 URL 失败: %w", err)
	}
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", start)
	params.Set("end", end)
	params.Set("step", step)
	u.RawQuery = params.Encode()
	return s.doQuery(ctx, u.String())
}

// doQuery 执行 HTTP 查询
func (s *VMService) doQuery(ctx context.Context, url string) (*VMQueryResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Log().Warnf("[VM] HTTP %d for URL: %s, body: %s", resp.StatusCode, url, string(body))
		return nil, fmt.Errorf("查询返回 HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result VMQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("查询状态不成功: %s", string(body))
	}

	return &result, nil
}

// extractInstantValue 从即时查询结果中提取第一个值
func extractInstantValue(result *VMQueryResult) float64 {
	if result == nil || len(result.Data.Result) == 0 {
		return 0
	}
	series := result.Data.Result[0]
	if len(series.Value) >= 2 {
		if valStr, ok := series.Value[1].(string); ok {
			var val float64
			fmt.Sscanf(valStr, "%f", &val)
			return val
		}
	}
	return 0
}

// extractInstantValues 从即时查询结果中提取所有 ident → value 映射
func extractInstantValues(result *VMQueryResult) map[string]float64 {
	m := make(map[string]float64)
	if result == nil {
		return m
	}
	for _, series := range result.Data.Result {
		// 优先使用 ident 标签，回退到 instance、__name__ 等
		ident := series.Metric["ident"]
		if ident == "" {
			ident = series.Metric["instance"]
		}
		if ident == "" {
			// 尝试从所有标签中找到第一个看起来像主机名的值
			for k, v := range series.Metric {
				if k != "__name__" && k != "cpu" && k != "path" && k != "job" {
					ident = v
					break
				}
			}
		}
		if ident == "" {
			continue
		}
		if len(series.Value) >= 2 {
			if valStr, ok := series.Value[1].(string); ok {
				var val float64
				fmt.Sscanf(valStr, "%f", &val)
				m[ident] = val
			}
		}
	}
	return m
}

// GetHostMetrics 获取指定主机的实时监控数据
func (s *VMService) GetHostMetrics(ctx context.Context, datasourceID uint, ident string) (*HostMetricsSnapshot, error) {
	dsURL, err := s.getDatasourceURL(datasourceID)
	if err != nil {
		return nil, err
	}

	snapshot := &HostMetricsSnapshot{Ident: ident}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Categraf 标准指标名（与 inspection-tool 一致）
	queries := map[string]string{
		"cpu":          fmt.Sprintf(`cpu_usage_active{cpu="cpu-total",ident="%s"}`, ident),
		"mem":          fmt.Sprintf(`100 - mem_available_percent{ident="%s"}`, ident),
		"disk":         fmt.Sprintf(`max(disk_used_percent{ident="%s"})`, ident),
		"load1":        fmt.Sprintf(`system_load1{ident="%s"}`, ident),
		"load5":        fmt.Sprintf(`system_load5{ident="%s"}`, ident),
		"load15":       fmt.Sprintf(`system_load15{ident="%s"}`, ident),
		"uptime":       fmt.Sprintf(`system_uptime{ident="%s"}`, ident),
		"cpuCores":     fmt.Sprintf(`system_n_cpus{ident="%s"}`, ident),
		"memTotal":     fmt.Sprintf(`mem_total{ident="%s"}`, ident),
		"memAvailable": fmt.Sprintf(`mem_available{ident="%s"}`, ident),
	}

	for name, query := range queries {
		wg.Add(1)
		go func(n, q string) {
			defer wg.Done()
			result, err := s.QueryInstant(ctx, dsURL, q)
			if err != nil {
				log.Log().Warnf("[VM] query %s failed: %v", n, err)
				return
			}
			val := extractInstantValue(result)
			mu.Lock()
			defer mu.Unlock()
			switch n {
			case "cpu":
				snapshot.CPUUsage = val
			case "mem":
				snapshot.MemoryUsage = val
			case "disk":
				snapshot.DiskUsage = val
			case "load1":
				snapshot.Load1 = val
			case "load5":
				snapshot.Load5 = val
			case "load15":
				snapshot.Load15 = val
			case "uptime":
				snapshot.Uptime = val
			case "cpuCores":
				snapshot.CPUCores = val
			case "memTotal":
				snapshot.MemTotal = val
			case "memAvailable":
				snapshot.MemAvailable = val
			}
		}(name, query)
	}
	wg.Wait()

	return snapshot, nil
}

// GetHostMetricsHistory 获取主机历史监控数据
func (s *VMService) GetHostMetricsHistory(ctx context.Context, datasourceID uint, ident, start, end, step string) (map[string]*VMQueryResult, error) {
	dsURL, err := s.getDatasourceURL(datasourceID)
	if err != nil {
		return nil, err
	}

	metrics := map[string]string{
		"cpu":    fmt.Sprintf(`cpu_usage_active{cpu="cpu-total",ident="%s"}`, ident),
		"memory": fmt.Sprintf(`100 - mem_available_percent{ident="%s"}`, ident),
		"disk":   fmt.Sprintf(`max(disk_used_percent{ident="%s"})`, ident),
		"load1":  fmt.Sprintf(`system_load1{ident="%s"}`, ident),
	}

	results := make(map[string]*VMQueryResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, query := range metrics {
		wg.Add(1)
		go func(n, q string) {
			defer wg.Done()
			result, err := s.QueryRange(ctx, dsURL, q, start, end, step)
			if err != nil {
				log.Log().Warnf("[VM] range query %s failed: %v", n, err)
				return
			}
			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(name, query)
	}
	wg.Wait()

	return results, nil
}

// GetClusterOverview 获取集群监控总览
func (s *VMService) GetClusterOverview(ctx context.Context, datasourceID uint) (*ClusterOverview, error) {
	dsURL, err := s.getDatasourceURL(datasourceID)
	if err != nil {
		return nil, err
	}

	overview := &ClusterOverview{}
	var wg sync.WaitGroup
	var mu sync.Mutex

	// 查询各聚合指标
	type aggQuery struct {
		name  string
		query string
	}
	aggQueries := []aggQuery{
		{"avgCPU", `avg(cpu_usage_active{cpu="cpu-total"})`},
		{"avgMem", `avg(100 - mem_available_percent)`},
		{"maxCPU", `max(cpu_usage_active{cpu="cpu-total"})`},
		{"maxMem", `max(100 - mem_available_percent)`},
		{"maxDisk", `max(disk_used_percent)`},
		{"hostCount", `count(cpu_usage_active{cpu="cpu-total"})`},
	}

	for _, aq := range aggQueries {
		wg.Add(1)
		go func(name, query string) {
			defer wg.Done()
			result, err := s.QueryInstant(ctx, dsURL, query)
			if err != nil {
				log.Log().Warnf("[VM] cluster query %s failed: %v", name, err)
				return
			}
			val := extractInstantValue(result)
			mu.Lock()
			defer mu.Unlock()
			switch name {
			case "avgCPU":
				overview.AvgCPUUsage = val
			case "avgMem":
				overview.AvgMemUsage = val
			case "maxCPU":
				overview.MaxCPUUsage = val
			case "maxMem":
				overview.MaxMemUsage = val
			case "maxDisk":
				overview.MaxDiskUsage = val
			case "hostCount":
				overview.HostCount = int(val)
			}
		}(aq.name, aq.query)
	}

	// TOP 5 hosts
	topQueries := []struct {
		name  string
		query string
	}{
		{"topCPU", `topk(5, cpu_usage_active{cpu="cpu-total"})`},
		{"topMem", `topk(5, mem_used_percent)`},
		{"topDisk", `topk(5, max by (ident) (disk_used_percent))`},
	}

	for _, tq := range topQueries {
		wg.Add(1)
		go func(name, query string) {
			defer wg.Done()
			result, err := s.QueryInstant(ctx, dsURL, query)
			if err != nil {
				log.Log().Warnf("[VM] top query %s failed: %v", name, err)
				return
			}
			values := extractInstantValues(result)
			var hosts []HostMetricValue
			for ident, val := range values {
				hosts = append(hosts, HostMetricValue{Ident: ident, Value: val})
			}
			mu.Lock()
			defer mu.Unlock()
			switch name {
			case "topCPU":
				overview.TopCPUHosts = hosts
			case "topMem":
				overview.TopMemHosts = hosts
			case "topDisk":
				overview.TopDiskHosts = hosts
			}
		}(tq.name, tq.query)
	}

	wg.Wait()
	return overview, nil
}

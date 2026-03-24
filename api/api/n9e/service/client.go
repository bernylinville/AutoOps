package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dodevops-api/api/n9e/model"
	"dodevops-api/pkg/log"

	"github.com/go-resty/resty/v2"
)

// N9EClient N9E API 客户端
type N9EClient struct {
	httpClient *resty.Client
}

// NewN9EClient 创建 N9E 客户端实例
func NewN9EClient(endpoint, token string, timeout int) *N9EClient {
	if timeout <= 0 {
		timeout = 30
	}

	httpClient := resty.New().
		SetBaseURL(strings.TrimRight(endpoint, "/")).
		SetTimeout(time.Duration(timeout) * time.Second).
		SetHeader("X-User-Token", token).
		SetHeader("Content-Type", "application/json").
		SetRetryCount(3).
		SetRetryWaitTime(time.Second).
		SetRetryMaxWaitTime(8 * time.Second).
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return true
			}
			return resp != nil && resp.StatusCode() >= http.StatusInternalServerError
		})

	return &N9EClient{
		httpClient: httpClient,
	}
}

// GetBusiGroups 获取 N9E 业务组列表
func (c *N9EClient) GetBusiGroups(ctx context.Context) ([]model.BusiGroupData, error) {
	data, err := getData[[]model.BusiGroupData](ctx, c, "/api/n9e/busi-groups", nil)
	if err != nil {
		return nil, err
	}

	log.Log().Infof("[N9E] fetched %d business groups", len(data))
	return data, nil
}

// GetTargets 获取 N9E 全部主机目标列表（分页遍历）
func (c *N9EClient) GetTargets(ctx context.Context) ([]model.TargetData, error) {
	const pageSize = 5000
	var allTargets []model.TargetData

	for page := 1; ; page++ {
		data, err := getData[model.TargetListData](ctx, c, "/api/n9e/targets", map[string]string{
			"limit": fmt.Sprintf("%d", pageSize),
			"p":     fmt.Sprintf("%d", page),
		})
		if err != nil {
			return nil, err
		}

		allTargets = append(allTargets, data.List...)
		log.Log().Infof("[N9E] fetched page %d: %d targets (total: %d, accumulated: %d)",
			page, len(data.List), data.Total, len(allTargets))

		// 已获取全部数据
		if len(allTargets) >= int(data.Total) || len(data.List) == 0 {
			break
		}
	}

	return allTargets, nil
}

// GetDatasourceBriefs 获取 N9E 数据源列表
func (c *N9EClient) GetDatasourceBriefs(ctx context.Context) ([]model.DatasourceData, error) {
	paths := []string{
		"/api/n9e/datasource/brief",
		"/api/n9e/datasource/briefs",
	}

	var lastErr error
	for _, path := range paths {
		data, err := getData[[]model.DatasourceData](ctx, c, path, nil)
		if err == nil {
			log.Log().Infof("[N9E] fetched %d datasource briefs via %s", len(data), path)
			return data, nil
		}
		lastErr = err
		log.Log().Warnf("[N9E] failed to fetch datasource briefs via %s: %v", path, err)
	}

	return nil, fmt.Errorf("fetch N9E datasource briefs: %w", lastErr)
}

// TestConnection 测试 N9E 连接
func (c *N9EClient) TestConnection(ctx context.Context) error {
	_, err := c.GetBusiGroups(ctx)
	if err != nil {
		return fmt.Errorf("test N9E connection: %w", err)
	}
	return nil
}

// QueryPromQL 透传 PromQL 查询到 VictoriaMetrics
func (c *N9EClient) QueryPromQL(ctx context.Context, datasourceURL, query, start, end, step string) (json.RawMessage, error) {
	client := resty.New().
		SetBaseURL(strings.TrimRight(datasourceURL, "/")).
		SetTimeout(30 * time.Second)

	var result json.RawMessage
	resp, err := client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"query": query,
			"start": start,
			"end":   end,
			"step":  step,
		}).
		SetResult(&result).
		Get("/api/v1/query_range")

	if err != nil {
		return nil, fmt.Errorf("PromQL query failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("PromQL query returned status %d: %s", resp.StatusCode(), string(resp.Body()))
	}

	return resp.Body(), nil
}

// getData 通用 N9E API 数据获取方法
func getData[T any](ctx context.Context, client *N9EClient, path string, queryParams map[string]string) (T, error) {
	var (
		result model.N9EResponse[T]
		zero   T
	)

	request := client.httpClient.R().
		SetContext(ctx).
		SetResult(&result)
	if len(queryParams) > 0 {
		request.SetQueryParams(queryParams)
	}

	resp, err := request.Get(path)
	if err != nil {
		log.Log().Errorf("[N9E] request %s failed: %v", path, err)
		return zero, fmt.Errorf("request N9E %s: %w", path, err)
	}

	if resp.StatusCode() != http.StatusOK {
		log.Log().Errorf("[N9E] %s returned status %d: %s", path, resp.StatusCode(), string(resp.Body()))
		return zero, fmt.Errorf("N9E %s returned status %d", path, resp.StatusCode())
	}

	if result.Err != "" {
		log.Log().Errorf("[N9E] %s API error: %s", path, result.Err)
		return zero, fmt.Errorf("N9E %s error: %s", path, result.Err)
	}

	return result.Dat, nil
}

// ExtractTargetMetadata 从 N9E target 扩展信息中提取主机元数据
func ExtractTargetMetadata(target model.TargetData) (hostname, cpuModel, memoryTotal, kernelVersion string) {
	if strings.TrimSpace(target.ExtendInfo) != "" {
		var info model.TargetExtendInfo
		if err := json.Unmarshal([]byte(target.ExtendInfo), &info); err == nil {
			hostname = strings.TrimSpace(info.Platform.Hostname)
			cpuModel = strings.TrimSpace(info.CPU.ModelName)
			memoryTotal = strings.TrimSpace(info.Memory.Total)
			kernelVersion = strings.TrimSpace(info.Platform.KernelRelease)
		}
	}

	if hostname == "" {
		hostname = strings.TrimSpace(target.Hostname)
	}
	if hostname == "" {
		hostname = strings.TrimSpace(target.Ident)
	}

	return hostname, cpuModel, memoryTotal, kernelVersion
}

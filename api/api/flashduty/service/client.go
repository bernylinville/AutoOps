package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"dodevops-api/common/config"
	"dodevops-api/pkg/log"
)

// Client FlashDuty HTTP 客户端
type Client struct {
	httpClient *http.Client
	baseURL    string
	appKey     string
}

var clientInstance *Client
var clientOnce sync.Once

// GetClient 获取 FlashDuty 客户端单例
func GetClient() *Client {
	clientOnce.Do(func() {
		cfg := config.Config.FlashDuty
		baseURL := cfg.BaseURL
		if baseURL == "" {
			baseURL = "https://api.flashcat.cloud"
		}
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 30
		}
		clientInstance = &Client{
			httpClient: &http.Client{Timeout: time.Duration(timeout) * time.Second},
			baseURL:    baseURL,
			appKey:     cfg.AppKey,
		}
		if cfg.AppKey != "" {
			log.Log().Info("[FlashDuty] 客户端初始化成功")
		} else {
			log.Log().Warn("[FlashDuty] app_key 未配置，FlashDuty 功能不可用")
		}
	})
	return clientInstance
}

// IsConfigured 检查 FlashDuty 是否已配置
func (c *Client) IsConfigured() bool {
	return c.appKey != ""
}

// Post 发送 POST 请求到 FlashDuty API
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	if !c.IsConfigured() {
		return fmt.Errorf("FlashDuty 未配置 app_key")
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s%s?app_key=%s", c.baseURL, path, c.appKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("[FlashDuty] 请求失败 %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("[FlashDuty] 读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Log().Warnf("[FlashDuty] HTTP %d for %s: %s", resp.StatusCode, path, string(respBody))
		return fmt.Errorf("FlashDuty 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("[FlashDuty] 解析响应失败: %w", err)
		}
	}

	return nil
}

// TestConnection 测试 FlashDuty 连接
func (c *Client) TestConnection(ctx context.Context) error {
	if !c.IsConfigured() {
		return fmt.Errorf("FlashDuty 未配置 app_key")
	}

	// 使用告警列表接口测试，只取1条
	now := time.Now().Unix()
	body := map[string]interface{}{
		"p":          1,
		"limit":      1,
		"start_time": now - 3600,
		"end_time":   now,
	}

	var resp map[string]interface{}
	err := c.Post(ctx, "/alert/list", body, &resp)
	if err != nil {
		return err
	}

	if errObj, ok := resp["error"]; ok && errObj != nil {
		errMap, _ := errObj.(map[string]interface{})
		if code, ok := errMap["code"]; ok && code != nil {
			return fmt.Errorf("FlashDuty 认证失败: %v - %v", code, errMap["message"])
		}
	}

	return nil
}

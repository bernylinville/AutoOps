package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"dodevops-api/common/config"
	"dodevops-api/pkg/log"
)

// StandardEvent 对应 FlashDuty 的标准事件结构
type StandardEvent struct {
	EventStatus string            `json:"event_status"` // "Warning"|"Critical"|"Info"|"Resolved"
	AlertKey    string            `json:"alert_key"`
	Description string            `json:"description"`
	Title       string            `json:"title"`
	Client      string            `json:"client"`
	ClientUrl   string            `json:"client_url"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// PushStandardEvent 推送标准告警事件到 FlashDuty
func PushStandardEvent(title, description, severity string) error {
	client := GetClient()
	if !client.IsConfigured() {
		log.Log().Warn("[FlashDuty Event] App key未配置，跳过推送事件")
		return nil
	}

	cfg := config.Config.FlashDuty
	if cfg.IntegrationKey == "" {
		log.Log().Warn("[FlashDuty Event] Integration Key 未配置，跳过推送事件")
		return nil
	}

	event := StandardEvent{
		EventStatus: severity,
		AlertKey:    fmt.Sprintf("autoops-job-failed-%d", time.Now().Unix()),
		Title:       title,
		Description: description,
		Client:      "AutoOps",
	}

	eventURL := fmt.Sprintf("%s/event/push/alert/standard?integration_key=%s", cfg.BaseURL, cfg.IntegrationKey)
	
	jsonBody, _ := json.Marshal(event)
	req, err := http.NewRequest("POST", eventURL, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Errorf("[FlashDuty Event] 推送请求失败: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Log().Errorf("[FlashDuty Event] 推送返回错误 HTTP %d", resp.StatusCode)
		return fmt.Errorf("FlashDuty event error %d", resp.StatusCode)
	}

	log.Log().Infof("[FlashDuty Event] 推送成功: %s", title)
	return nil
}

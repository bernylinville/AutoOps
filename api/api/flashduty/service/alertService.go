package service

import (
	"context"
	"time"

	"dodevops-api/api/flashduty/model"
	"dodevops-api/pkg/log"
)

// AlertService FlashDuty 告警服务
type AlertService struct {
	client *Client
}

// NewAlertService 创建告警服务
func NewAlertService() *AlertService {
	return &AlertService{client: GetClient()}
}

// GetActiveAlerts 获取活跃告警列表
func (s *AlertService) GetActiveAlerts(ctx context.Context, limit int) (*model.AlertListResponse, error) {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now().Unix()
	isActive := true
	req := model.AlertListRequest{
		P:        1,
		Limit:    limit,
		IsActive: &isActive,
		StartTime: now - 86400*30, // 最近 30 天
		EndTime:   now,
	}

	var resp model.AlertListResponse
	if err := s.client.Post(ctx, "/alert/list", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetActiveAlerts", resp.Error)
	}
	return &resp, nil
}

// GetAlertsByHost 获取指定主机的告警（通过 labels 中的 ident 字段）
func (s *AlertService) GetAlertsByHost(ctx context.Context, ident string, limit int) (*model.AlertListResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	now := time.Now().Unix()
	req := model.AlertListRequest{
		P:     1,
		Limit: limit,
		Query: ident, // 全文搜索主机 IP
		StartTime: now - 86400*30, // 最近 30 天（API 限制 31 天）
		EndTime:   now,
	}

	var resp model.AlertListResponse
	if err := s.client.Post(ctx, "/alert/list", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetAlertsByHost", resp.Error)
	}
	return &resp, nil
}

// GetAlertSummary 获取告警概况统计
func (s *AlertService) GetAlertSummary(ctx context.Context) (*model.DashboardAlertSummary, error) {
	resp, err := s.GetActiveAlerts(ctx, 100)
	if err != nil {
		return nil, err
	}

	summary := &model.DashboardAlertSummary{
		ActiveAlerts: resp.Data.Total,
	}

	for _, alert := range resp.Data.Items {
		switch alert.AlertSeverity {
		case "Critical":
			summary.CriticalCount++
		case "Warning":
			summary.WarningCount++
		case "Info":
			summary.InfoCount++
		}
	}

	return summary, nil
}

func logAndReturnError(method string, err *model.ErrorInfo) error {
	log.Log().Warnf("[FlashDuty] %s error: %s - %s", method, err.Code, err.Message)
	return &flashDutyError{Code: err.Code, Msg: err.Message}
}

type flashDutyError struct {
	Code string
	Msg  string
}

func (e *flashDutyError) Error() string {
	return e.Code + ": " + e.Msg
}

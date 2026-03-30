package service

import (
	"context"
	"time"

	"dodevops-api/api/flashduty/model"
)

// InsightService FlashDuty 分析看板服务
type InsightService struct {
	client *Client
}

// NewInsightService 创建分析看板服务
func NewInsightService() *InsightService {
	return &InsightService{client: GetClient()}
}

// GetAccountMetrics 获取账户级别指标 (MTTA/MTTR/降噪率)
func (s *InsightService) GetAccountMetrics(ctx context.Context, days int) (*model.SREMetrics, error) {
	if days <= 0 {
		days = 7
	}
	now := time.Now().Unix()
	req := model.InsightRequest{
		StartTime: now - int64(days*86400),
		EndTime:   now,
		TimeZone:  "Asia/Shanghai",
		Query:     "",
		Labels:    map[string]string{},
		Fields:    map[string]string{},
	}

	var resp model.InsightResponse
	if err := s.client.Post(ctx, "/insight/account", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetAccountMetrics", resp.Error)
	}

	metrics := &model.SREMetrics{}
	if len(resp.Data.Items) > 0 {
		item := resp.Data.Items[0]
		metrics.MTTA = item.MeanSecondsToAck
		metrics.MTTR = item.MeanSecondsToClose
		metrics.NoiseReductionPct = item.NoiseReductionPct
		metrics.AckPct = item.AcknowledgementPct
		metrics.TotalIncidents = item.TotalIncidentCnt
		metrics.TotalAlerts = item.TotalAlertCnt
		metrics.TotalAlertEvents = item.TotalAlertEventCnt
	}

	return metrics, nil
}

// GetTrendData 获取趋势数据 (按天聚合)
func (s *InsightService) GetTrendData(ctx context.Context, days int) (*model.SREMetrics, error) {
	if days <= 0 {
		days = 7
	}
	now := time.Now().Unix()
	req := model.InsightRequest{
		StartTime:     now - int64(days*86400),
		EndTime:       now,
		TimeZone:      "Asia/Shanghai",
		Query:         "",
		Labels:        map[string]string{},
		Fields:        map[string]string{},
		AggregateUnit: "day",
	}

	var resp model.InsightResponse
	if err := s.client.Post(ctx, "/insight/account", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetTrendData", resp.Error)
	}

	metrics := &model.SREMetrics{}
	for _, item := range resp.Data.Items {
		ts := int64(0)
		if item.Ts != nil {
			ts = *item.Ts
		}
		metrics.TrendData = append(metrics.TrendData, model.TrendDataPoint{
			Timestamp:     ts,
			IncidentCount: item.TotalIncidentCnt,
			AlertCount:    item.TotalAlertCnt,
			MTTA:          item.MeanSecondsToAck,
			MTTR:          item.MeanSecondsToClose,
		})
		metrics.TotalIncidents += item.TotalIncidentCnt
		metrics.TotalAlerts += item.TotalAlertCnt
		metrics.TotalAlertEvents += item.TotalAlertEventCnt
	}

	return metrics, nil
}

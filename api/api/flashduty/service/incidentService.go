package service

import (
	"context"
	"time"

	"dodevops-api/api/flashduty/model"
)

// IncidentService FlashDuty 故障服务
type IncidentService struct {
	client *Client
}

// NewIncidentService 创建故障服务
func NewIncidentService() *IncidentService {
	return &IncidentService{client: GetClient()}
}

// GetActiveIncidents 获取活跃故障列表
func (s *IncidentService) GetActiveIncidents(ctx context.Context, limit int) (*model.IncidentListResponse, error) {
	if limit <= 0 {
		limit = 100
	}
	now := time.Now().Unix()
	req := model.IncidentListRequest{
		P:          1,
		Limit:      limit,
		Progresses: []string{"Triggered", "Processing"},
		StartTime:  now - 86400*30,
		EndTime:    now,
	}

	var resp model.IncidentListResponse
	if err := s.client.Post(ctx, "/incident/list", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetActiveIncidents", resp.Error)
	}
	return &resp, nil
}

// GetIncidentSummary 获取故障统计
func (s *IncidentService) GetIncidentSummary(ctx context.Context) (triggered, processing int, err error) {
	resp, err := s.GetActiveIncidents(ctx, 100)
	if err != nil {
		return 0, 0, err
	}

	for _, inc := range resp.Data.Items {
		switch inc.Progress {
		case "Triggered":
			triggered++
		case "Processing":
			processing++
		}
	}
	return triggered, processing, nil
}

// ClaimIncident 认领故障
func (s *IncidentService) ClaimIncident(ctx context.Context, incidentID string) error {
	req := model.IncidentActionRequest{
		IncidentID: incidentID,
	}
	var resp model.Response
	if err := s.client.Post(ctx, "/incident/claim", req, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return logAndReturnError("ClaimIncident", resp.Error)
	}
	return nil
}

// CloseIncident 关闭故障
func (s *IncidentService) CloseIncident(ctx context.Context, incidentID, desc string) error {
	req := model.IncidentActionRequest{
		IncidentID: incidentID,
		Desc:       desc,
	}
	var resp model.Response
	if err := s.client.Post(ctx, "/incident/close", req, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return logAndReturnError("CloseIncident", resp.Error)
	}
	return nil
}

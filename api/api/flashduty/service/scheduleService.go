package service

import (
	"context"
	"time"

	"dodevops-api/api/flashduty/model"
)

// ScheduleService FlashDuty 值班服务
type ScheduleService struct {
	client *Client
}

// NewScheduleService 创建值班服务
func NewScheduleService() *ScheduleService {
	return &ScheduleService{client: GetClient()}
}

// GetSchedules 获取值班表列表
func (s *ScheduleService) GetSchedules(ctx context.Context, teamIDs []int) (*model.ScheduleListResponse, error) {
	if len(teamIDs) == 0 {
		// 不传 team_ids 时，API 要求该字段必填，传空数组
		teamIDs = []int{}
	}
	req := model.ScheduleListRequest{
		P:       1,
		Limit:   50,
		TeamIDs: teamIDs,
	}

	var resp model.ScheduleListResponse
	if err := s.client.Post(ctx, "/schedule/list", req, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, logAndReturnError("GetSchedules", resp.Error)
	}
	return &resp, nil
}

// GetTodayOnCall 获取今日值班人
func (s *ScheduleService) GetTodayOnCall(ctx context.Context, teamIDs []int) ([]model.OnCallInfo, error) {
	// 1. 获取值班表列表
	schedResp, err := s.GetSchedules(ctx, teamIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Unix()
	todayEnd := todayStart + 86400

	var onCallList []model.OnCallInfo

	// 2. 遍历值班表查预览
	for _, sched := range schedResp.Data.Items {
		if !sched.Enabled {
			continue
		}

		previewReq := model.SchedulePreviewRequest{
			ScheduleID: sched.ScheduleID,
			Start:      todayStart,
			End:        todayEnd,
		}
		var previewResp model.SchedulePreviewResponse
		if err := s.client.Post(ctx, "/schedule/preview", previewReq, &previewResp); err != nil {
			continue // 跳过失败的值班表
		}
		if previewResp.Error != nil {
			continue
		}

		// 找出当前时间正在值班的人
		nowTs := now.Unix()
		for _, shift := range previewResp.Data.Shifts {
			if shift.Start <= nowTs && shift.End > nowTs {
				onCallList = append(onCallList, model.OnCallInfo{
					ScheduleName: sched.ScheduleName,
					PersonName:   shift.PersonName,
					Email:        shift.Email,
					Phone:        shift.Phone,
					ShiftStart:   shift.Start,
					ShiftEnd:     shift.End,
				})
			}
		}
	}

	return onCallList, nil
}

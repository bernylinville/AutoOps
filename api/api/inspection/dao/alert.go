// Package dao provides data access objects for inspection.
package dao

import (
	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// AlertDAO 巡检告警明细数据访问
type AlertDAO struct {
	db *gorm.DB
}

// NewAlertDAO creates an AlertDAO.
func NewAlertDAO(db *gorm.DB) *AlertDAO {
	return &AlertDAO{db: db}
}

// BatchCreate 批量创建告警明细
func (d *AlertDAO) BatchCreate(alerts []*model.InspectionAlert) error {
	if len(alerts) == 0 {
		return nil
	}
	return d.db.CreateInBatches(alerts, 100).Error
}

// RecentAlertDTO is a lightweight DTO for the overview page.
type RecentAlertDTO struct {
	Hostname          string `json:"hostname"`
	MetricName        string `json:"metric"`
	MetricDisplayName string `json:"metricDisplayName"`
	Level             string `json:"level"`
	Message           string `json:"message"`
}

// GetRecentAlerts returns the most recent critical and warning alerts across all runs.
func (d *AlertDAO) GetRecentAlerts(limit int) ([]*RecentAlertDTO, error) {
	if limit <= 0 {
		limit = 10
	}
	var alerts []*RecentAlertDTO
	err := d.db.Model(&model.InspectionAlert{}).
		Select("hostname, metric_name, metric_display_name, level, message").
		Where("level IN ?", []string{string(model.AlertLevelWarning), string(model.AlertLevelCritical)}).
		Order("id DESC").
		Limit(limit).
		Find(&alerts).Error
	return alerts, err
}

// ListByRunID 按运行记录 ID 查询告警明细（支持筛选和分页）
func (d *AlertDAO) ListByRunID(runID uint, q *model.AlertListQuery) ([]*model.InspectionAlert, int64, error) {
	var alerts []*model.InspectionAlert
	var total int64

	query := d.db.Model(&model.InspectionAlert{}).Where("run_id = ?", runID)
	if q.Level != "" {
		query = query.Where("level = ?", q.Level)
	}
	if q.Hostname != "" {
		query = query.Where("hostname ILIKE ?", "%"+q.Hostname+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := q.Page
	pageSize := q.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if err := query.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

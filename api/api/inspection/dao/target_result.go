// Package dao provides data access objects for inspection.
package dao

import (
	"encoding/json"
	"fmt"
	"strings"

	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// TargetResultDAO 巡检主机结果数据访问
type TargetResultDAO struct {
	db *gorm.DB
}

// NewTargetResultDAO creates a TargetResultDAO.
func NewTargetResultDAO(db *gorm.DB) *TargetResultDAO {
	return &TargetResultDAO{db: db}
}

// BatchCreate 批量创建主机结果
func (d *TargetResultDAO) BatchCreate(results []*model.InspectionTargetResult) error {
	if len(results) == 0 {
		return nil
	}
	return d.db.CreateInBatches(results, 100).Error
}

// ListByRunID 按运行记录 ID 查询主机结果列表（分页）
func (d *TargetResultDAO) ListByRunID(runID uint, page, pageSize int) ([]*model.InspectionTargetResult, int64, error) {
	var results []*model.InspectionTargetResult
	var total int64

	query := d.db.Model(&model.InspectionTargetResult{}).Where("run_id = ?", runID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if err := query.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, err
	}
	d.enrichResults(results)
	return results, total, nil
}

// ListByRunIDWithQuery 按运行记录 ID 查询主机结果列表（支持筛选）
func (d *TargetResultDAO) ListByRunIDWithQuery(runID uint, q *model.ResultListQuery) ([]*model.InspectionTargetResult, int64, error) {
	var results []*model.InspectionTargetResult
	var total int64

	query := d.db.Model(&model.InspectionTargetResult{}).Where("run_id = ?", runID)
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
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

	if err := query.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&results).Error; err != nil {
		return nil, 0, err
	}
	d.enrichResults(results)
	return results, total, nil
}

func (d *TargetResultDAO) enrichResults(results []*model.InspectionTargetResult) {
	if len(results) == 0 {
		return
	}

	ids := make([]uint, 0, len(results))
	for _, result := range results {
		if result == nil {
			continue
		}
		ids = append(ids, result.ID)
		result.MetricSummary = BuildMetricSummary(result.Metrics)
	}
	if len(ids) == 0 {
		return
	}

	type alertCountRow struct {
		TargetResultID uint
		Count          int
	}
	var rows []alertCountRow
	if err := d.db.Model(&model.InspectionAlert{}).
		Select("target_result_id, COUNT(*) AS count").
		Where("target_result_id IN ?", ids).
		Group("target_result_id").
		Scan(&rows).Error; err != nil {
		return
	}

	counts := make(map[uint]int, len(rows))
	for _, row := range rows {
		counts[row.TargetResultID] = row.Count
	}
	for _, result := range results {
		if result == nil {
			continue
		}
		result.AlertCount = counts[result.ID]
	}
}

// BuildMetricSummary returns a compact display string for the host result list.
func BuildMetricSummary(raw model.JSONRaw) string {
	if len(raw) == 0 {
		return ""
	}

	var metrics map[string]struct {
		FormattedValue string  `json:"formatted_value"`
		RawValue       float64 `json:"raw_value"`
		IsNA           bool    `json:"is_na"`
	}
	if err := json.Unmarshal(raw, &metrics); err != nil {
		return ""
	}

	parts := make([]string, 0, 4)
	appendMetric := func(label, key string) {
		metric, ok := metrics[key]
		if !ok || metric.IsNA {
			return
		}
		value := strings.TrimSpace(metric.FormattedValue)
		if value == "" {
			value = fmt.Sprintf("%.2f", metric.RawValue)
		}
		parts = append(parts, label+value)
	}

	appendMetric("CPU ", "cpu_usage")
	appendMetric("内存 ", "memory_usage")
	appendMetric("磁盘 ", "disk_usage_max")
	appendMetric("负载 ", "load_per_core")

	return strings.Join(parts, " / ")
}

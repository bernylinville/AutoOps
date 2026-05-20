// Package dao provides data access objects for inspection.
package dao

import (
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
	return results, total, nil
}

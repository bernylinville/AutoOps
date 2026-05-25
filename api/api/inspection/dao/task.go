// Package dao provides data access objects for inspection.
package dao

import (
	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// TaskDAO 巡检任务数据访问
type TaskDAO struct {
	db *gorm.DB
}

// NewTaskDAO creates a TaskDAO.
func NewTaskDAO(db *gorm.DB) *TaskDAO {
	return &TaskDAO{db: db}
}

// List 查询任务列表（支持分页和关键词筛选）
func (d *TaskDAO) List(q *model.TaskListQuery) ([]*model.InspectionTask, int64, error) {
	var tasks []*model.InspectionTask
	var total int64

	db := d.db.Model(&model.InspectionTask{})
	if q.Enabled != nil {
		db = db.Where("enabled = ?", *q.Enabled)
	}
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		db = db.Where("name ILIKE ? OR n9e_group_name ILIKE ?", kw, kw)
	}

	if err := db.Count(&total).Error; err != nil {
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

	if err := db.Order("id ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetByID 按 ID 查询任务
func (d *TaskDAO) GetByID(id uint) (*model.InspectionTask, error) {
	var task model.InspectionTask
	if err := d.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByN9EGroupID 按 N9E 业务组 ID 查询任务
func (d *TaskDAO) GetByN9EGroupID(groupID int64) (*model.InspectionTask, error) {
	var task model.InspectionTask
	if err := d.db.Where("n9e_group_id = ?", groupID).First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// ExistsByN9EGroupID 检查指定业务组的任务是否存在
func (d *TaskDAO) ExistsByN9EGroupID(groupID int64) bool {
	var count int64
	d.db.Model(&model.InspectionTask{}).Where("n9e_group_id = ?", groupID).Count(&count)
	return count > 0
}

// Create 创建任务
func (d *TaskDAO) Create(task *model.InspectionTask) error {
	return d.db.Create(task).Error
}

// CreateTaskWithEnabled 创建任务并强制写入 Enabled 字段（解决 GORM 零值忽略问题）
// GORM 在 struct 中 bool=false 时会跳过该字段并使用数据库默认值(true)，
// 通过 Select 显式指定字段可强制写入 false。
func (d *TaskDAO) CreateTaskWithEnabled(task *model.InspectionTask) error {
	return d.db.Select("N9EGroupID", "N9EGroupName", "Name", "Enabled", "Cron", "CreateTime", "UpdateTime").Create(task).Error
}

// Update 更新任务
func (d *TaskDAO) Update(task *model.InspectionTask) error {
	return d.db.Save(task).Error
}

// ListAllEnabled 查询所有启用的任务
func (d *TaskDAO) ListAllEnabled() ([]*model.InspectionTask, error) {
	var tasks []*model.InspectionTask
	if err := d.db.Where("enabled = ?", true).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// ListAll 查询所有任务（用于 N9E 同步对比）
func (d *TaskDAO) ListAll() ([]*model.InspectionTask, error) {
	var tasks []*model.InspectionTask
	if err := d.db.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// UpdateField 更新单个字段
func (d *TaskDAO) UpdateField(id uint, field string, value interface{}) error {
	return d.db.Model(&model.InspectionTask{}).Where("id = ?", id).Update(field, value).Error
}

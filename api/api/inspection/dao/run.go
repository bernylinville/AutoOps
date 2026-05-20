package dao

import (
	"time"

	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// RunDAO 巡检运行记录数据访问
type RunDAO struct {
	db *gorm.DB
}

// NewRunDAO creates a RunDAO.
func NewRunDAO(db *gorm.DB) *RunDAO {
	return &RunDAO{db: db}
}

// Create 创建运行记录
func (d *RunDAO) Create(run *model.InspectionRun) error {
	return d.db.Create(run).Error
}

// Update 更新运行记录字段
func (d *RunDAO) Update(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.InspectionRun{}).Where("id = ?", id).Updates(updates).Error
}

// GetByID 按 ID 查询
func (d *RunDAO) GetByID(id uint) (*model.InspectionRun, error) {
	var run model.InspectionRun
	if err := d.db.First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// ListByTaskID 按任务 ID 查询运行记录
func (d *RunDAO) ListByTaskID(taskID uint, page, pageSize int) ([]*model.InspectionRun, int64, error) {
	var runs []*model.InspectionRun
	var total int64

	db := d.db.Model(&model.InspectionRun{}).Where("task_id = ?", taskID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	if err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&runs).Error; err != nil {
		return nil, 0, err
	}
	return runs, total, nil
}

// List 查询运行记录列表（支持多条件筛选和分页）
func (d *RunDAO) List(q *model.RunListQuery) ([]*model.InspectionRun, int64, error) {
	var runs []*model.InspectionRun
	var total int64

	query := d.db.Model(&model.InspectionRun{})

	if q.TaskID > 0 {
		query = query.Where("task_id = ?", q.TaskID)
	}
	if q.N9EGroupID > 0 {
		query = query.Where("task_id IN (SELECT id FROM inspection_task WHERE n9e_group_id = ?)", q.N9EGroupID)
	}
	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}
	if q.TriggerType != "" {
		query = query.Where("trigger_type = ?", q.TriggerType)
	}
	if q.DateFrom != "" {
		query = query.Where("run_date >= ?", q.DateFrom)
	}
	if q.DateTo != "" {
		query = query.Where("run_date <= ?", q.DateTo)
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

	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&runs).Error; err != nil {
		return nil, 0, err
	}

	return runs, total, nil
}

// HasActiveRunByTaskID 检查指定任务是否有未完成的运行记录（pending 或 running），防止重入。
func (d *RunDAO) HasActiveRunByTaskID(taskID uint) (bool, error) {
	var count int64
	err := d.db.Model(&model.InspectionRun{}).
		Where("task_id = ? AND status IN ?", taskID, []string{model.RunStatusPending, model.RunStatusRunning}).
		Count(&count).Error
	return count > 0, err
}

// TodayStats 今日巡检统计
type TodayStats struct {
	TotalRuns     int `json:"totalRuns"`
	TotalHosts    int `json:"totalHosts"`
	NormalHosts   int `json:"normalHosts"`
	WarningHosts  int `json:"warningHosts"`
	CriticalHosts int `json:"criticalHosts"`
	FailedHosts   int `json:"failedHosts"`
	TotalAlerts   int `json:"totalAlerts"`
}

// GetTodayStats 获取今日汇总统计
func (d *RunDAO) GetTodayStats() (*TodayStats, error) {
	var stats TodayStats
	err := d.db.Model(&model.InspectionRun{}).
		Select(`
			COUNT(*) as total_runs,
			COALESCE(SUM(total_hosts), 0) as total_hosts,
			COALESCE(SUM(normal_hosts), 0) as normal_hosts,
			COALESCE(SUM(warning_hosts), 0) as warning_hosts,
			COALESCE(SUM(critical_hosts), 0) as critical_hosts,
			COALESCE(SUM(failed_hosts), 0) as failed_hosts,
			COALESCE(SUM(total_alerts), 0) as total_alerts
		`).
		Where("run_date = ?", time.Now().Format("2006-01-02")).
		Scan(&stats).Error
	return &stats, err
}

package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"dodevops-api/common/util"

	"gorm.io/gorm"
)

// InspectionRun 单次巡检运行记录
type InspectionRun struct {
	ID             uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	TaskID         uint       `gorm:"column:task_id;index:idx_run_task;NOT NULL;comment:关联 inspection_task.id" json:"taskId"`
	TriggerType    string     `gorm:"column:trigger_type;type:varchar(10);default:'cron';comment:cron/manual" json:"triggerType"`
	TriggeredBy    *uint      `gorm:"column:triggered_by;comment:手动触发用户 ID" json:"triggeredBy,omitempty"`
	Status         string     `gorm:"column:status;type:varchar(20);default:'pending';comment:pending/running/success/partial/failed" json:"status"`
	StartedAt      *time.Time `gorm:"column:started_at" json:"startedAt"`
	EndedAt        *time.Time `gorm:"column:ended_at" json:"endedAt"`
	DurationMs     int64      `gorm:"column:duration_ms;comment:执行耗时 ms" json:"durationMs"`
	TotalHosts     int        `gorm:"column:total_hosts;default:0" json:"totalHosts"`
	NormalHosts    int        `gorm:"column:normal_hosts;default:0" json:"normalHosts"`
	WarningHosts   int        `gorm:"column:warning_hosts;default:0" json:"warningHosts"`
	CriticalHosts  int        `gorm:"column:critical_hosts;default:0" json:"criticalHosts"`
	FailedHosts    int        `gorm:"column:failed_hosts;default:0" json:"failedHosts"`
	TotalAlerts    int        `gorm:"column:total_alerts;default:0" json:"totalAlerts"`
	ConfigSnapshot JSONRaw    `gorm:"column:config_snapshot;type:jsonb;comment:运行时配置快照" json:"configSnapshot,omitempty"`
	ErrorMessage   string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	RunDate        string     `gorm:"column:run_date;type:date;comment:运行日期(用于唯一约束)" json:"-"`
	CreatedAt      util.HTime `gorm:"column:created_at;NOT NULL" json:"createdAt"`
	N9EGroupName   string     `gorm:"-" json:"n9eGroupName,omitempty"`
	TaskName       string     `gorm:"-" json:"taskName,omitempty"`
}

func (InspectionRun) TableName() string { return "inspection_run" }

func (r *InspectionRun) BeforeCreate(tx *gorm.DB) error {
	if r.RunDate == "" {
		r.RunDate = time.Now().Format("2006-01-02")
	}
	return nil
}

// Run status constants.
const (
	RunStatusPending  = "pending"
	RunStatusRunning  = "running"
	RunStatusSuccess  = "success"
	RunStatusPartial  = "partial"
	RunStatusFailed   = "failed"
	TriggerTypeCron   = "cron"
	TriggerTypeManual = "manual"
)

// JSONRaw is a json.RawMessage that implements sql.Scanner and driver.Valuer.
type JSONRaw json.RawMessage

func (j *JSONRaw) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	*j = make(JSONRaw, len(bytes))
	copy(*j, bytes)
	return nil
}

func (j JSONRaw) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

// RunListQuery 运行历史查询参数
type RunListQuery struct {
	TaskID      uint   `form:"taskId"`
	N9EGroupID  int64  `form:"n9eGroupId"`
	Status      string `form:"status"`
	TriggerType string `form:"triggerType"`
	DateFrom    string `form:"dateFrom"`
	DateTo      string `form:"dateTo"`
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
}

package model

import "dodevops-api/common/util"

// InspectionAlert 巡检异常明细
type InspectionAlert struct {
	ID                uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	RunID             uint       `gorm:"column:run_id;index:idx_alert_run;NOT NULL;constraint:OnDelete:CASCADE;comment:关联 inspection_run.id" json:"runId"`
	TargetResultID    uint       `gorm:"column:target_result_id;index:idx_alert_result;comment:关联 inspection_target_result.id" json:"targetResultId"`
	Hostname          string     `gorm:"column:hostname;type:varchar(300);NOT NULL" json:"hostname"`
	MetricName        string     `gorm:"column:metric_name;type:varchar(100);NOT NULL" json:"metricName"`
	MetricDisplayName string     `gorm:"column:metric_display_name;type:varchar(200)" json:"metricDisplayName"`
	CurrentValue      float64    `gorm:"column:current_value" json:"currentValue"`
	WarningThreshold  float64    `gorm:"column:warning_threshold" json:"warningThreshold"`
	CriticalThreshold float64    `gorm:"column:critical_threshold" json:"criticalThreshold"`
	Level             string     `gorm:"column:level;type:varchar(20);NOT NULL;comment:warning/critical" json:"level"`
	Message           string     `gorm:"column:message;type:text" json:"message"`
	Labels            JSONRaw    `gorm:"column:labels;type:jsonb" json:"labels,omitempty"`
	CreatedAt         util.HTime `gorm:"column:created_at;NOT NULL" json:"createdAt"`
}

func (InspectionAlert) TableName() string { return "inspection_alert" }

// AlertListQuery 异常明细查询参数
type AlertListQuery struct {
	Level    string `form:"level"`
	Hostname string `form:"hostname"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

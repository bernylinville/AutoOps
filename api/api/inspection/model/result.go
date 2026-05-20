package model

import "dodevops-api/common/util"

// InspectionTargetResult 主机级巡检结果
type InspectionTargetResult struct {
	ID          uint        `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	RunID       uint        `gorm:"column:run_id;index:idx_result_run;NOT NULL;constraint:OnDelete:CASCADE;comment:关联 inspection_run.id" json:"runId"`
	Hostname    string      `gorm:"column:hostname;type:varchar(300);NOT NULL" json:"hostname"`
	Ident       string      `gorm:"column:ident;type:varchar(300);comment:N9E ident" json:"ident"`
	IP          string      `gorm:"column:ip;type:varchar(100)" json:"ip"`
	OS          string      `gorm:"column:os;type:varchar(100)" json:"os"`
	Status      string      `gorm:"column:status;type:varchar(20);default:'normal';comment:normal/warning/critical/failed" json:"status"`
	Error       string      `gorm:"column:error;type:text" json:"error,omitempty"`
	Metrics     JSONRaw     `gorm:"column:metrics;type:jsonb;comment:结构化指标数据" json:"metrics,omitempty"`
	BootTime    string      `gorm:"column:boot_time;type:varchar(50)" json:"bootTime,omitempty"`
	CollectedAt *util.HTime `gorm:"column:collected_at" json:"collectedAt"`
	CreatedAt   util.HTime  `gorm:"column:created_at;NOT NULL" json:"createdAt"`
}

func (InspectionTargetResult) TableName() string { return "inspection_target_result" }

// ResultListQuery 主机结果查询参数
type ResultListQuery struct {
	Status   string `form:"status"`
	Hostname string `form:"hostname"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

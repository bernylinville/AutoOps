package model

import (
	"dodevops-api/common/util"
)

// InspectionReportArtifact 巡检报告文件
type InspectionReportArtifact struct {
	ID           uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	RunID        uint       `gorm:"column:run_id;uniqueIndex:idx_report_run;NOT NULL;constraint:OnDelete:CASCADE;comment:关联 inspection_run.id" json:"runId"`
	FilePath     string     `gorm:"column:file_path;type:varchar(500);NOT NULL" json:"filePath"`
	FileSize     int64      `gorm:"column:file_size;default:0" json:"fileSize"`
	Format       string     `gorm:"column:format;type:varchar(20);default:'excel'" json:"format"`
	Status       string     `gorm:"column:status;type:varchar(20);default:'pending';comment:pending/success/failed" json:"status"`
	ErrorMessage string     `gorm:"column:error_message;type:text" json:"errorMessage,omitempty"`
	CreatedAt    util.HTime `gorm:"column:created_at;NOT NULL" json:"createdAt"`
	ExpiresAt    util.HTime `gorm:"column:expires_at;index:idx_report_expires;comment:过期时间，created_at + 30 天" json:"expiresAt"`
}

func (InspectionReportArtifact) TableName() string { return "inspection_report_artifact" }

const (
	ReportStatusPending = "pending"
	ReportStatusSuccess = "success"
	ReportStatusFailed  = "failed"
)

package model

import (
	"time"

	"dodevops-api/common/util"
)

// InspectionNotification 巡检通知记录
type InspectionNotification struct {
	ID        uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	RunID     uint       `gorm:"column:run_id;index:idx_notify_run;NOT NULL;constraint:OnDelete:CASCADE;comment:关联 inspection_run.id" json:"runId"`
	Channel   string     `gorm:"column:channel;type:varchar(20);default:'dingtalk'" json:"channel"`
	Payload   string     `gorm:"column:payload;type:text;comment:通知内容摘要" json:"payload"`
	Status    string     `gorm:"column:status;type:varchar(20);default:'pending';comment:pending/sent/failed/skipped" json:"status"`
	ErrorMsg  string     `gorm:"column:error_msg;type:text" json:"errorMsg,omitempty"`
	SentAt    *time.Time `gorm:"column:sent_at" json:"sentAt,omitempty"`
	CreatedAt util.HTime `gorm:"column:created_at;NOT NULL" json:"createdAt"`
}

func (InspectionNotification) TableName() string { return "inspection_notification" }

const (
	NotifyStatusPending = "pending"
	NotifyStatusSent    = "sent"
	NotifyStatusFailed  = "failed"
	NotifyStatusSkipped = "skipped"
)

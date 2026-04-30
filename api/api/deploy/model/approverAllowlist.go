package model

import "time"

// AgentApproverAllowlist Agent 审批人白名单
type AgentApproverAllowlist struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ApproverAdminID uint      `gorm:"not null;index:idx_approver_app_env,unique" json:"approverAdminId"`
	ApplicationCode string    `gorm:"size:128;not null;index:idx_approver_app_env,unique" json:"applicationCode"`
	Env             string    `gorm:"size:64;not null;index:idx_approver_app_env,unique" json:"env"`
	CreatedBy       string    `gorm:"size:64" json:"createdBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (AgentApproverAllowlist) TableName() string {
	return "agent_approver_allowlist"
}

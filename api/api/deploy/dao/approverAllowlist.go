package dao

import (
	"dodevops-api/api/deploy/model"

	"gorm.io/gorm"
)

// IsApproverAllowed 检查审批人是否在 (app, env) 白名单中
func IsApproverAllowed(db *gorm.DB, adminID uint, appCode, env string) (bool, error) {
	var count int64
	err := db.Model(&model.AgentApproverAllowlist{}).
		Where("approver_admin_id = ? AND application_code = ? AND env = ?", adminID, appCode, env).
		Count(&count).Error
	return count > 0, err
}

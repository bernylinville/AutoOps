package dao

import (
	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// NotificationDAO 巡检通知记录数据访问
type NotificationDAO struct {
	db *gorm.DB
}

// NewNotificationDAO creates a NotificationDAO.
func NewNotificationDAO(db *gorm.DB) *NotificationDAO {
	return &NotificationDAO{db: db}
}

// Create 创建通知记录
func (d *NotificationDAO) Create(notification *model.InspectionNotification) error {
	return d.db.Create(notification).Error
}

// Update 更新通知记录字段
func (d *NotificationDAO) Update(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.InspectionNotification{}).Where("id = ?", id).Updates(updates).Error
}

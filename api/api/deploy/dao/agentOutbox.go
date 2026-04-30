package dao

import (
	"dodevops-api/api/deploy/model"

	"gorm.io/gorm"
)

// WriteOutboxEvent inserts a new AgentOutboxEvent row.
func WriteOutboxEvent(db *gorm.DB, event *model.AgentOutboxEvent) error {
	return db.Create(event).Error
}

// ListOutboxEvents returns events for the given requestNo where ID > sinceID, ordered by ID ASC.
func ListOutboxEvents(db *gorm.DB, requestNo string, sinceID uint64) ([]model.AgentOutboxEvent, error) {
	var events []model.AgentOutboxEvent
	err := db.Where("request_no = ? AND id > ?", requestNo, sinceID).
		Order("id ASC").
		Find(&events).Error
	return events, err
}

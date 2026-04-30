package model

import "time"

// AgentOutboxEvent records deploy state-change events for reliable Hermes delivery.
type AgentOutboxEvent struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestNo   string     `gorm:"size:64;not null;index"   json:"requestNo"`
	EventType   string     `gorm:"size:32;not null"         json:"eventType"`
	Status      string     `gorm:"size:16;not null;default:'pending'" json:"status"`
	Payload     string     `gorm:"type:text;not null"       json:"payload"`
	DeliveredAt *time.Time `json:"deliveredAt,omitempty"`
	CreatedAt   time.Time  `gorm:"index"                    json:"createdAt"`
}

func (AgentOutboxEvent) TableName() string { return "agent_outbox_event" }

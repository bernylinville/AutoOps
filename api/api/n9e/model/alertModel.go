package model

import (
	"dodevops-api/common/util"
	"time"
)

// AlertRule 告警规则
type AlertRule struct {
	ID             uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name           string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:'规则名称'" json:"name" binding:"required"`
	Severity       string     `gorm:"column:severity;type:varchar(20);default:'warning';comment:'严重级别: critical/warning/info'" json:"severity"`
	Source         string     `gorm:"column:source;type:varchar(50);default:'n9e';comment:'告警来源: n9e/prometheus/custom'" json:"source"`
	MatchLabels    string     `gorm:"column:match_labels;type:text;comment:'匹配标签 JSON'" json:"matchLabels"`
	NotifyChannels string     `gorm:"column:notify_channels;type:text;comment:'通知渠道 JSON [\"wechat\",\"dingtalk\"]'" json:"notifyChannels"`
	NotifyTarget   string     `gorm:"column:notify_target;type:text;comment:'通知目标 JSON'" json:"notifyTarget"`
	Enabled        bool       `gorm:"column:enabled;default:true;comment:'是否启用'" json:"enabled"`
	Description    string     `gorm:"column:description;type:varchar(500);comment:'规则描述'" json:"description"`
	CreateTime     util.HTime `gorm:"column:create_time;NOT NULL" json:"createTime"`
	UpdateTime     util.HTime `gorm:"column:update_time;NOT NULL" json:"updateTime"`
}

func (AlertRule) TableName() string {
	return "n9e_alert_rule"
}

// AlertEvent 告警事件
type AlertEvent struct {
	ID           uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RuleID       uint       `gorm:"column:rule_id;index;comment:'关联规则ID'" json:"ruleId"`
	RuleName     string     `gorm:"column:rule_name;type:varchar(200);comment:'规则名称'" json:"ruleName"`
	AlertName    string     `gorm:"column:alert_name;type:varchar(200);NOT NULL;comment:'告警名称'" json:"alertName"`
	Severity     string     `gorm:"column:severity;type:varchar(20);default:'warning'" json:"severity"`
	Status       string     `gorm:"column:status;type:varchar(20);default:'firing';comment:'firing/resolved'" json:"status"`
	Labels       string     `gorm:"column:labels;type:text;comment:'标签 JSON'" json:"labels"`
	Annotations  string     `gorm:"column:annotations;type:text;comment:'注解 JSON'" json:"annotations"`
	StartsAt     time.Time  `gorm:"column:starts_at;NOT NULL" json:"startsAt"`
	EndsAt       *time.Time `gorm:"column:ends_at" json:"endsAt"`
	NotifyStatus string     `gorm:"column:notify_status;type:varchar(20);default:'pending';comment:'pending/sent/failed'" json:"notifyStatus"`
	NotifyResult string     `gorm:"column:notify_result;type:text;comment:'通知结果'" json:"notifyResult"`
	CreateTime   util.HTime `gorm:"column:create_time;NOT NULL" json:"createTime"`
}

func (AlertEvent) TableName() string {
	return "n9e_alert_event"
}

// NotifyChannel 通知渠道配置
type NotifyChannel struct {
	ID         uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name       string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:'渠道名称'" json:"name" binding:"required"`
	Type       string     `gorm:"column:type;type:varchar(20);NOT NULL;comment:'wechat/dingtalk/email'" json:"type" binding:"required"`
	Config     string     `gorm:"column:config;type:text;NOT NULL;comment:'渠道配置 JSON'" json:"config" binding:"required"`
	Enabled    bool       `gorm:"column:enabled;default:true" json:"enabled"`
	CreateTime util.HTime `gorm:"column:create_time;NOT NULL" json:"createTime"`
	UpdateTime util.HTime `gorm:"column:update_time;NOT NULL" json:"updateTime"`
}

func (NotifyChannel) TableName() string {
	return "n9e_notify_channel"
}

// ---- DTOs ----

type CreateAlertRuleDto struct {
	Name           string `json:"name" binding:"required"`
	Severity       string `json:"severity"`
	Source         string `json:"source"`
	MatchLabels    string `json:"matchLabels"`
	NotifyChannels string `json:"notifyChannels"`
	NotifyTarget   string `json:"notifyTarget"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
}

type UpdateAlertRuleDto struct {
	ID             uint   `json:"id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Severity       string `json:"severity"`
	Source         string `json:"source"`
	MatchLabels    string `json:"matchLabels"`
	NotifyChannels string `json:"notifyChannels"`
	NotifyTarget   string `json:"notifyTarget"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
}

type CreateNotifyChannelDto struct {
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Config  string `json:"config" binding:"required"`
	Enabled bool   `json:"enabled"`
}

type UpdateNotifyChannelDto struct {
	ID      uint   `json:"id" binding:"required"`
	Name    string `json:"name" binding:"required"`
	Type    string `json:"type" binding:"required"`
	Config  string `json:"config" binding:"required"`
	Enabled bool   `json:"enabled"`
}

// N9E Webhook 告警推送格式（兼容 Alertmanager）
type WebhookPayload struct {
	Receiver string         `json:"receiver"`
	Status   string         `json:"status"`
	Alerts   []WebhookAlert `json:"alerts"`
}

type WebhookAlert struct {
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
	Fingerprint string            `json:"fingerprint"`
}

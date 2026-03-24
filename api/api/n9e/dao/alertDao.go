package dao

import (
	"dodevops-api/api/n9e/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	"time"
)

// ==================== AlertRule ====================

func CreateAlertRule(rule *model.AlertRule) error {
	rule.CreateTime = util.HTime{Time: time.Now()}
	rule.UpdateTime = util.HTime{Time: time.Now()}
	return common.GetDB().Create(rule).Error
}

func UpdateAlertRule(rule *model.AlertRule) error {
	rule.UpdateTime = util.HTime{Time: time.Now()}
	return common.GetDB().Save(rule).Error
}

func DeleteAlertRule(id uint) error {
	return common.GetDB().Delete(&model.AlertRule{}, id).Error
}

func GetAlertRuleByID(id uint) (*model.AlertRule, error) {
	var rule model.AlertRule
	err := common.GetDB().First(&rule, id).Error
	return &rule, err
}

func GetAlertRuleList(page, pageSize int) ([]model.AlertRule, int64, error) {
	var rules []model.AlertRule
	var total int64
	db := common.GetDB().Model(&model.AlertRule{})
	db.Count(&total)
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rules).Error
	return rules, total, err
}

func GetEnabledAlertRules() ([]model.AlertRule, error) {
	var rules []model.AlertRule
	err := common.GetDB().Where("enabled = ?", true).Find(&rules).Error
	return rules, err
}

// ==================== AlertEvent ====================

func CreateAlertEvent(event *model.AlertEvent) error {
	event.CreateTime = util.HTime{Time: time.Now()}
	return common.GetDB().Create(event).Error
}

func UpdateAlertEventStatus(id uint, status, notifyStatus, notifyResult string) error {
	updates := map[string]interface{}{
		"status":        status,
		"notify_status": notifyStatus,
		"notify_result": notifyResult,
	}
	if status == "resolved" {
		now := time.Now()
		updates["ends_at"] = &now
	}
	return common.GetDB().Model(&model.AlertEvent{}).Where("id = ?", id).Updates(updates).Error
}

func GetAlertEventList(page, pageSize int, severity, status string) ([]model.AlertEvent, int64, error) {
	var events []model.AlertEvent
	var total int64
	db := common.GetDB().Model(&model.AlertEvent{})
	if severity != "" {
		db = db.Where("severity = ?", severity)
	}
	if status != "" {
		db = db.Where("status = ?", status)
	}
	db.Count(&total)
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error
	return events, total, err
}

func GetAlertEventStats() (map[string]int64, error) {
	var total, firing, resolved, critical, warning int64
	db := common.GetDB().Model(&model.AlertEvent{})
	db.Count(&total)
	common.GetDB().Model(&model.AlertEvent{}).Where("status = ?", "firing").Count(&firing)
	common.GetDB().Model(&model.AlertEvent{}).Where("status = ?", "resolved").Count(&resolved)
	common.GetDB().Model(&model.AlertEvent{}).Where("severity = ?", "critical").Count(&critical)
	common.GetDB().Model(&model.AlertEvent{}).Where("severity = ?", "warning").Count(&warning)
	return map[string]int64{
		"total":    total,
		"firing":   firing,
		"resolved": resolved,
		"critical": critical,
		"warning":  warning,
	}, nil
}

// ==================== NotifyChannel ====================

func CreateNotifyChannel(channel *model.NotifyChannel) error {
	channel.CreateTime = util.HTime{Time: time.Now()}
	channel.UpdateTime = util.HTime{Time: time.Now()}
	return common.GetDB().Create(channel).Error
}

func UpdateNotifyChannel(channel *model.NotifyChannel) error {
	channel.UpdateTime = util.HTime{Time: time.Now()}
	return common.GetDB().Save(channel).Error
}

func DeleteNotifyChannel(id uint) error {
	return common.GetDB().Delete(&model.NotifyChannel{}, id).Error
}

func GetNotifyChannelByID(id uint) (*model.NotifyChannel, error) {
	var channel model.NotifyChannel
	err := common.GetDB().First(&channel, id).Error
	return &channel, err
}

func GetNotifyChannelList() ([]model.NotifyChannel, error) {
	var channels []model.NotifyChannel
	err := common.GetDB().Order("id DESC").Find(&channels).Error
	return channels, err
}

func GetEnabledNotifyChannelsByType(channelType string) ([]model.NotifyChannel, error) {
	var channels []model.NotifyChannel
	err := common.GetDB().Where("type = ? AND enabled = ?", channelType, true).Find(&channels).Error
	return channels, err
}

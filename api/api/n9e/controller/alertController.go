package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/api/n9e/model"
	"dodevops-api/api/n9e/service"
	"dodevops-api/common/result"
	"dodevops-api/common/util"

	"github.com/gin-gonic/gin"
)

type AlertController struct {
	notifier *service.Notifier
}

func NewAlertController() *AlertController {
	return &AlertController{
		notifier: service.NewNotifier(),
	}
}

// ==================== 告警规则 ====================

func (ac *AlertController) GetAlertRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	rules, total, err := dao.GetAlertRuleList(page, pageSize)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取告警规则失败: "+err.Error())
		return
	}
	result.SuccessWithPage(c, rules, total, page, pageSize)
}

func (ac *AlertController) CreateAlertRule(c *gin.Context) {
	var dto model.CreateAlertRuleDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	rule := model.AlertRule{
		Name:           dto.Name,
		Severity:       dto.Severity,
		Source:         dto.Source,
		MatchLabels:    dto.MatchLabels,
		NotifyChannels: dto.NotifyChannels,
		NotifyTarget:   dto.NotifyTarget,
		Enabled:        dto.Enabled,
		Description:    dto.Description,
	}
	if rule.Severity == "" {
		rule.Severity = "warning"
	}
	if rule.Source == "" {
		rule.Source = "n9e"
	}
	if err := dao.CreateAlertRule(&rule); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建告警规则失败: "+err.Error())
		return
	}
	result.Success(c, rule)
}

func (ac *AlertController) UpdateAlertRule(c *gin.Context) {
	var dto model.UpdateAlertRuleDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	existing, err := dao.GetAlertRuleByID(dto.ID)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "规则不存在")
		return
	}
	existing.Name = dto.Name
	existing.Severity = dto.Severity
	existing.Source = dto.Source
	existing.MatchLabels = dto.MatchLabels
	existing.NotifyChannels = dto.NotifyChannels
	existing.NotifyTarget = dto.NotifyTarget
	existing.Enabled = dto.Enabled
	existing.Description = dto.Description
	if err := dao.UpdateAlertRule(existing); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新告警规则失败: "+err.Error())
		return
	}
	result.Success(c, existing)
}

func (ac *AlertController) DeleteAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	if err := dao.DeleteAlertRule(uint(id)); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除告警规则失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

// ==================== 告警事件 ====================

func (ac *AlertController) GetAlertEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	severity := c.Query("severity")
	status := c.Query("status")
	events, total, err := dao.GetAlertEventList(page, pageSize, severity, status)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取告警事件失败: "+err.Error())
		return
	}
	result.SuccessWithPage(c, events, total, page, pageSize)
}

func (ac *AlertController) GetAlertEventStats(c *gin.Context) {
	stats, err := dao.GetAlertEventStats()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取告警统计失败: "+err.Error())
		return
	}
	result.Success(c, stats)
}

// ==================== 通知渠道 ====================

func (ac *AlertController) GetNotifyChannels(c *gin.Context) {
	channels, err := dao.GetNotifyChannelList()
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取通知渠道失败: "+err.Error())
		return
	}
	result.Success(c, channels)
}

func (ac *AlertController) CreateNotifyChannel(c *gin.Context) {
	var dto model.CreateNotifyChannelDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	channel := model.NotifyChannel{
		Name:    dto.Name,
		Type:    dto.Type,
		Config:  dto.Config,
		Enabled: dto.Enabled,
	}
	if err := dao.CreateNotifyChannel(&channel); err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建通知渠道失败: "+err.Error())
		return
	}
	result.Success(c, channel)
}

func (ac *AlertController) UpdateNotifyChannel(c *gin.Context) {
	var dto model.UpdateNotifyChannelDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	existing, err := dao.GetNotifyChannelByID(dto.ID)
	if err != nil {
		result.Failed(c, http.StatusNotFound, "渠道不存在")
		return
	}
	existing.Name = dto.Name
	existing.Type = dto.Type
	existing.Config = dto.Config
	existing.Enabled = dto.Enabled
	if err := dao.UpdateNotifyChannel(existing); err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新通知渠道失败: "+err.Error())
		return
	}
	result.Success(c, existing)
}

func (ac *AlertController) DeleteNotifyChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	if err := dao.DeleteNotifyChannel(uint(id)); err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除通知渠道失败: "+err.Error())
		return
	}
	result.Success(c, nil)
}

func (ac *AlertController) TestNotifyChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的ID")
		return
	}
	channel, err := dao.GetNotifyChannelByID(uint(id))
	if err != nil {
		result.Failed(c, http.StatusNotFound, "渠道不存在")
		return
	}
	msg := service.AlertMessage{
		AlertName: "测试告警",
		Severity:  "info",
		Status:    "firing",
		StartsAt:  time.Now(),
		Annotations: map[string]string{
			"description": "这是一条测试告警通知，来自 AutoOps 平台",
		},
	}
	if err := ac.notifier.Dispatch(channel.Type, channel.Config, msg); err != nil {
		result.Failed(c, http.StatusInternalServerError, "发送测试通知失败: "+err.Error())
		return
	}
	result.Success(c, gin.H{"message": "测试通知已发送"})
}

// ==================== Webhook 接收 ====================

func (ac *AlertController) ReceiveWebhook(c *gin.Context) {
	// Token 校验
	token := c.GetHeader("X-Webhook-Token")
	if token == "" {
		token = c.Query("token")
	}
	// 从配置读取 expected token（简化：使用环境变量或 config）
	expectedToken := "webhook-notify-token-2024"
	if token != expectedToken {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "invalid webhook token"})
		return
	}

	var payload model.WebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid payload: " + err.Error()})
		return
	}

	log.Printf("[Webhook] 收到告警: receiver=%s status=%s alerts=%d", payload.Receiver, payload.Status, len(payload.Alerts))

	// 获取启用的告警规则
	rules, _ := dao.GetEnabledAlertRules()

	for _, alert := range payload.Alerts {
		// 创建事件记录
		labelsJSON, _ := json.Marshal(alert.Labels)
		annotationsJSON, _ := json.Marshal(alert.Annotations)

		alertName := alert.Labels["alertname"]
		severity := alert.Labels["severity"]
		if severity == "" {
			severity = "warning"
		}

		event := model.AlertEvent{
			AlertName:    alertName,
			Severity:     severity,
			Status:       alert.Status,
			Labels:       string(labelsJSON),
			Annotations:  string(annotationsJSON),
			StartsAt:     alert.StartsAt,
			NotifyStatus: "pending",
			CreateTime:   util.HTime{Time: time.Now()},
		}

		if alert.Status == "resolved" && !alert.EndsAt.IsZero() {
			event.EndsAt = &alert.EndsAt
		}

		// 匹配规则
		matchedRule := ac.matchRule(rules, alert)
		if matchedRule != nil {
			event.RuleID = matchedRule.ID
			event.RuleName = matchedRule.Name
		}

		if err := dao.CreateAlertEvent(&event); err != nil {
			log.Printf("[Webhook] 保存告警事件失败: %v", err)
			continue
		}

		// 异步发送通知
		if matchedRule != nil {
			go ac.sendNotifications(event, matchedRule)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "ok", "received": len(payload.Alerts)})
}

// matchRule 匹配告警规则
func (ac *AlertController) matchRule(rules []model.AlertRule, alert model.WebhookAlert) *model.AlertRule {
	for i := range rules {
		rule := &rules[i]
		if rule.MatchLabels == "" || rule.MatchLabels == "{}" {
			return rule // 空匹配 = 匹配所有
		}
		var matchLabels map[string]string
		if err := json.Unmarshal([]byte(rule.MatchLabels), &matchLabels); err != nil {
			continue
		}
		matched := true
		for k, v := range matchLabels {
			if alert.Labels[k] != v {
				matched = false
				break
			}
		}
		if matched {
			return rule
		}
	}
	return nil
}

// sendNotifications 发送通知到所有配置的渠道
func (ac *AlertController) sendNotifications(event model.AlertEvent, rule *model.AlertRule) {
	msg := service.AlertMessage{
		AlertName: event.AlertName,
		Severity:  event.Severity,
		Status:    event.Status,
		StartsAt:  event.StartsAt,
	}
	if event.Annotations != "" {
		json.Unmarshal([]byte(event.Annotations), &msg.Annotations)
	}

	var channelTypes []string
	if err := json.Unmarshal([]byte(rule.NotifyChannels), &channelTypes); err != nil {
		log.Printf("[Notify] 解析通知渠道失败: %v", err)
		dao.UpdateAlertEventStatus(event.ID, event.Status, "failed", "解析渠道失败")
		return
	}

	var results []string
	allSuccess := true
	for _, chType := range channelTypes {
		channels, err := dao.GetEnabledNotifyChannelsByType(chType)
		if err != nil || len(channels) == 0 {
			results = append(results, chType+": 无可用渠道")
			allSuccess = false
			continue
		}
		for _, ch := range channels {
			if err := ac.notifier.Dispatch(ch.Type, ch.Config, msg); err != nil {
				log.Printf("[Notify] 发送失败 channel=%s: %v", ch.Name, err)
				results = append(results, ch.Name+": "+err.Error())
				allSuccess = false
			} else {
				results = append(results, ch.Name+": 成功")
			}
		}
	}

	notifyStatus := "sent"
	if !allSuccess {
		notifyStatus = "failed"
	}
	resultJSON, _ := json.Marshal(results)
	dao.UpdateAlertEventStatus(event.ID, event.Status, notifyStatus, string(resultJSON))
}

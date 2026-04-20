package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"dodevops-api/api/deploy/dao"
	"dodevops-api/api/deploy/model"
	"dodevops-api/common/config"
	"dodevops-api/pkg/dingtalkbot"

	"gorm.io/gorm"
)

type ChatContext struct {
	Provider         string                 `json:"provider"`
	ChatID           string                 `json:"chat_id"`
	AtMobiles        []string               `json:"at_mobiles"`
	AtUserIDs        []string               `json:"at_user_ids"`
	SenderExternalID string                 `json:"sender_external_id"`
	OriginMessageID  string                 `json:"origin_message_id"`
	Extra            map[string]interface{} `json:"extra"`
}

type IDeployNotifier interface {
	NotifyExecutionResult(req *model.DeployRequest, exec *model.ExecutionRecord) error
}

type DeployNotifier struct {
	store deployNotificationStore
}

func NewDeployNotifier(db *gorm.DB) IDeployNotifier {
	return &DeployNotifier{store: dao.NewDeployDao(db)}
}

type deployNotificationStore interface {
	CreateDeployNotification(notification *model.DeployNotification) error
	UpdateDeployNotification(id uint, updates map[string]interface{}) error
}

func (n *DeployNotifier) NotifyExecutionResult(req *model.DeployRequest, exec *model.ExecutionRecord) error {
	if req == nil || exec == nil {
		return fmt.Errorf("deploy notifier requires request and execution record")
	}

	botCfg := config.Config.Integrations.DeployBot
	stage := notificationStageFromExecution(exec.Status)
	title, text := n.buildMarkdown(req, exec)

	var chatCtx ChatContext
	if strings.TrimSpace(req.ChatContextJSON) != "" {
		_ = json.Unmarshal([]byte(req.ChatContextJSON), &chatCtx)
	}

	payload := map[string]interface{}{
		"title":       title,
		"text":        text,
		"provider":    chatCtx.Provider,
		"chat_id":     chatCtx.ChatID,
		"at_mobiles":  chatCtx.AtMobiles,
		"at_user_ids": chatCtx.AtUserIDs,
	}
	payloadJSON, _ := json.Marshal(payload)

	record := &model.DeployNotification{
		RequestID:   req.ID,
		Channel:     model.NotificationChannelDingtalkRobot,
		Stage:       stage,
		PayloadJSON: string(payloadJSON),
		Status:      model.NotificationStatusPending,
	}
	if err := n.store.CreateDeployNotification(record); err != nil {
		return fmt.Errorf("创建部署通知记录失败: %w", err)
	}

	if !botCfg.Enabled || strings.TrimSpace(botCfg.Provider) != "dingtalk" || strings.TrimSpace(botCfg.WebhookURL) == "" {
		return n.store.UpdateDeployNotification(record.ID, map[string]interface{}{
			"status":        model.NotificationStatusSkipped,
			"error_message": "deploy bot disabled or not configured",
			"updated_at":    time.Now(),
		})
	}

	client := dingtalkbot.NewClient(dingtalkbot.Config{
		WebhookURL: botCfg.WebhookURL,
		Secret:     botCfg.Secret,
	})
	err := client.SendMarkdown(title, text, chatCtx.AtMobiles, chatCtx.AtUserIDs)

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if err != nil {
		updates["status"] = model.NotificationStatusFailed
		updates["error_message"] = err.Error()
	} else {
		now := time.Now()
		updates["status"] = model.NotificationStatusSent
		updates["sent_at"] = now
		updates["error_message"] = ""
	}
	if updateErr := n.store.UpdateDeployNotification(record.ID, updates); updateErr != nil {
		if err != nil {
			return fmt.Errorf("%v; update notification status failed: %w", err, updateErr)
		}
		return fmt.Errorf("更新部署通知记录失败: %w", updateErr)
	}
	return err
}

func (n *DeployNotifier) buildMarkdown(req *model.DeployRequest, exec *model.ExecutionRecord) (string, string) {
	emoji := map[string]string{
		model.ExecutionStatusSucceeded:  "✅",
		model.ExecutionStatusFailed:     "❌",
		model.ExecutionStatusRolledBack: "↩️",
		model.ExecutionStatusCleaned:    "🧹",
	}[exec.Status]
	if emoji == "" {
		emoji = "ℹ️"
	}

	title := fmt.Sprintf("%s AutoOps 部署结果 - %s", emoji, req.RequestNo)

	var b strings.Builder
	fmt.Fprintf(&b, "#### %s\n\n", title)
	fmt.Fprintf(&b, "- **申请号**: %s\n", req.RequestNo)
	fmt.Fprintf(&b, "- **模式**: %s\n", req.Mode)
	fmt.Fprintf(&b, "- **发布名**: %s\n", req.ReleaseName)
	fmt.Fprintf(&b, "- **命名空间**: %s\n", req.Namespace)
	fmt.Fprintf(&b, "- **状态**: %s\n", exec.Status)
	fmt.Fprintf(&b, "- **执行阶段**: %s\n", exec.Phase)
	fmt.Fprintf(&b, "- **完成时间**: %s", formatNotificationTime(exec.EndedAt))

	// Enrich with image and service access when execution succeeded.
	if exec.Status == model.ExecutionStatusSucceeded {
		access := buildAccessInfo(req)
		if access != nil && strings.TrimSpace(access.Image) != "" {
			fmt.Fprintf(&b, "\n- **镜像**: %s", access.Image)
		}
		if access != nil && access.ServiceEnabled {
			fmt.Fprintf(&b, "\n- **Service 类型**: %s", access.ServiceType)
			if access.ServicePort > 0 {
				fmt.Fprintf(&b, "\n- **Service 端口**: %d → %d", access.ServicePort, access.TargetPort)
			}
		}
	}

	// Append error detail when execution failed.
	if exec.Status == model.ExecutionStatusFailed {
		if errMsg := execErrorMessage(exec); errMsg != "" {
			fmt.Fprintf(&b, "\n- **错误信息**: %s", errMsg)
		}
	}

	return title, b.String()
}

func notificationStageFromExecution(status string) string {
	switch status {
	case model.ExecutionStatusSucceeded:
		return model.NotificationStageExecuted
	case model.ExecutionStatusFailed:
		return model.NotificationStageFailed
	case model.ExecutionStatusRolledBack:
		return model.NotificationStageRolledBack
	case model.ExecutionStatusCleaned:
		return model.NotificationStageCleaned
	default:
		return model.NotificationStageExecuted
	}
}

func formatNotificationTime(ts *time.Time) string {
	if ts == nil {
		return "-"
	}
	return ts.Format(time.RFC3339)
}

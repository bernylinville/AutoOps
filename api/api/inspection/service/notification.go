package service

import (
	"fmt"
	"strings"
	"time"

	"dodevops-api/api/inspection/dao"
	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/dingtalkbot"
	"dodevops-api/pkg/log"

	"gorm.io/gorm"
)

// InspectionNotifier sends DingTalk notifications for inspection runs.
type InspectionNotifier struct {
	db        *gorm.DB
	notifyDAO *dao.NotificationDAO
}

// NewInspectionNotifier creates an InspectionNotifier.
func NewInspectionNotifier(db *gorm.DB) *InspectionNotifier {
	return &InspectionNotifier{
		db:        db,
		notifyDAO: dao.NewNotificationDAO(db),
	}
}

// NotifyRunResult sends a DingTalk notification for an inspection run result.
// Never blocks on failure — logs and returns.
func (n *InspectionNotifier) NotifyRunResult(run *model.InspectionRun, task *model.InspectionTask) error {
	// Check notification rules.
	hasCritical := run.CriticalHosts > 0 || run.TotalAlerts > 0
	hasWarning := run.WarningHosts > 0
	hasFailure := run.FailedHosts > 0

	shouldNotify := false
	if hasCritical && task.NotifyOnCritical {
		shouldNotify = true
	}
	if hasWarning && task.NotifyOnWarning {
		shouldNotify = true
	}
	if hasFailure && task.NotifyOnFailure {
		shouldNotify = true
	}

	if !shouldNotify {
		log.Log().Infof("[InspectionNotifier] run %d: notification skipped (no conditions met)", run.ID)
		return nil
	}

	// Build Markdown payload.
	title, text := n.buildMarkdown(run, task)
	payload := fmt.Sprintf("title=%s, text=%s", title, text)

	// Create notification record (pending).
	record := &model.InspectionNotification{
		RunID:   run.ID,
		Channel: "dingtalk",
		Payload: payload,
		Status:  model.NotifyStatusPending,
	}
	if err := n.notifyDAO.Create(record); err != nil {
		log.Log().Errorf("[InspectionNotifier] run %d: failed to create notification record: %v", run.ID, err)
		return fmt.Errorf("创建通知记录失败: %w", err)
	}

	// Check webhook configured.
	if strings.TrimSpace(task.NotifyWebhookURL) == "" {
		log.Log().Infof("[InspectionNotifier] run %d: no webhook URL, marking skipped", run.ID)
		n.notifyDAO.Update(record.ID, map[string]interface{}{
			"status":    model.NotifyStatusSkipped,
			"error_msg": "webhook URL not configured",
		})
		return nil
	}

	// Send via DingTalk.
	client := dingtalkbot.NewClient(dingtalkbot.Config{
		WebhookURL: task.NotifyWebhookURL,
		Secret:     task.NotifySecret,
	})
	err := client.SendMarkdown(title, text, nil, nil)

	// Update notification record.
	updates := map[string]interface{}{}
	if err != nil {
		updates["status"] = model.NotifyStatusFailed
		updates["error_msg"] = err.Error()
		log.Log().Errorf("[InspectionNotifier] run %d: send failed: %v", run.ID, err)
	} else {
		now := time.Now()
		updates["status"] = model.NotifyStatusSent
		updates["sent_at"] = now
		updates["error_msg"] = ""
		log.Log().Infof("[InspectionNotifier] run %d: notification sent", run.ID)
	}

	if updateErr := n.notifyDAO.Update(record.ID, updates); updateErr != nil {
		log.Log().Errorf("[InspectionNotifier] run %d: failed to update notification status: %v", run.ID, updateErr)
	}

	return err
}

// buildMarkdown builds the DingTalk Markdown message for an inspection report.
func (n *InspectionNotifier) buildMarkdown(run *model.InspectionRun, task *model.InspectionTask) (string, string) {
	emoji := "ℹ️"
	if run.Status == model.RunStatusSuccess {
		if run.CriticalHosts > 0 || run.WarningHosts > 0 {
			emoji = "⚠️"
		} else {
			emoji = "✅"
		}
	} else if run.Status == model.RunStatusFailed {
		emoji = "❌"
	} else if run.Status == model.RunStatusPartial {
		emoji = "⚠️"
	}

	title := fmt.Sprintf("%s 巡检报告 — %s", emoji, task.Name)

	duration := formatDurationMs(run.DurationMs)
	timeStr := "-"
	if run.EndedAt != nil {
		timeStr = run.EndedAt.Format("2006-01-02 15:04:05")
	}

	var b strings.Builder
	fmt.Fprintf(&b, "### %s\n\n", title)
	fmt.Fprintf(&b, "- **任务**: %s (ID: %d)\n", task.Name, task.ID)
	fmt.Fprintf(&b, "- **触发**: %s (%s)\n", run.TriggerType, timeStr)
	fmt.Fprintf(&b, "- **耗时**: %s\n", duration)
	fmt.Fprintf(&b, "- **主机**: %d 台 (正常 %d, 警告 %d, 严重 %d, 失败 %d)\n",
		run.TotalHosts, run.NormalHosts, run.WarningHosts, run.CriticalHosts, run.FailedHosts)
	fmt.Fprintf(&b, "- **异常**: %d 条\n", run.TotalAlerts)

	// Add error message for failed runs.
	if run.Status == model.RunStatusFailed && run.ErrorMessage != "" {
		fmt.Fprintf(&b, "- **错误**: %s\n", run.ErrorMessage)
	}

	return title, b.String()
}

// formatDurationMs formats milliseconds into a human-readable duration string.
func formatDurationMs(ms int64) string {
	if ms <= 0 {
		return "-"
	}
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	return d.Truncate(time.Second).String()
}

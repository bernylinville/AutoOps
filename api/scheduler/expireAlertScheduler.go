package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	cmdbmodel "dodevops-api/api/cmdb/model"
	n9emodel "dodevops-api/api/n9e/model"
	"dodevops-api/common"
	"dodevops-api/common/config"

	"github.com/robfig/cron/v3"
)

// ExpireAlertScheduler 每日扫描即将到期的主机，生成告警事件
type ExpireAlertScheduler struct {
	cron   *cron.Cron
	cronID cron.EntryID
}

func NewExpireAlertScheduler() *ExpireAlertScheduler {
	return &ExpireAlertScheduler{cron: cron.New()}
}

// Start 启动调度器，每日 09:00 执行
func (s *ExpireAlertScheduler) Start() error {
	id, err := s.cron.AddFunc("0 9 * * *", s.scanExpiringHosts)
	if err != nil {
		return fmt.Errorf("注册到期预警 cron 任务失败: %v", err)
	}
	s.cronID = id
	s.cron.Start()
	log.Println("主机到期预警调度器已启动（每日 09:00 执行）")
	return nil
}

// Stop 停止调度器
func (s *ExpireAlertScheduler) Stop() {
	s.cron.Stop()
	log.Println("主机到期预警调度器已停止")
}

// scanExpiringHosts 扫描 30 天内到期的主机，写入告警事件
func (s *ExpireAlertScheduler) scanExpiringHosts() {
	db := common.GetDB()
	if db == nil {
		log.Println("[到期预警] 数据库未就绪，跳过本次扫描")
		return
	}

	now := time.Now()
	deadline := now.AddDate(0, 0, 30)

	var hosts []cmdbmodel.CmdbHost
	db.Where("expire_time > ? AND expire_time <= ?", now, deadline).Find(&hosts)

	if len(hosts) == 0 {
		return
	}

	log.Printf("[到期预警] 检测到 %d 台主机将在 30 天内到期", len(hosts))

	var events []n9emodel.AlertEvent
	for _, host := range hosts {
		daysLeft := int(host.ExpireTime.Sub(now).Hours() / 24)
		log.Printf("[到期预警] 主机 %s (ID:%d) 还有 %d 天到期（%s）",
			host.HostName, host.ID, daysLeft, host.ExpireTime.Format("2006-01-02"))

		events = append(events, n9emodel.AlertEvent{
			RuleName:  "主机到期预警",
			AlertName: fmt.Sprintf("主机 [%s] 还有 %d 天到期", host.HostName, daysLeft),
			Severity:  "warning",
			Status:    "firing",
			Labels: fmt.Sprintf(
				`{"host_id":"%d","host_name":"%s","expire_date":"%s"}`,
				host.ID, host.HostName, host.ExpireTime.Format("2006-01-02"),
			),
			Annotations: fmt.Sprintf(
				`{"summary":"主机 %s 将于 %s 到期（剩余 %d 天），请及时续费或申请下线","days_left":"%d"}`,
				host.HostName, host.ExpireTime.Format("2006-01-02"), daysLeft, daysLeft,
			),
			StartsAt:     now,
			NotifyStatus: "pending",
		})
	}

	if err := db.Create(&events).Error; err != nil {
		log.Printf("[到期预警] 写入告警事件失败: %v", err)
	}

	// 推送钉钉通知（webhook 为空则跳过）
	if config.Config == nil || config.Config.Dingtalk.WebhookURL == "" {
		return
	}
	sendDingtalkExpireAlert(config.Config.Dingtalk.WebhookURL, hosts, now)
}

// sendDingtalkExpireAlert 向钉钉机器人推送主机到期预警 markdown 消息
func sendDingtalkExpireAlert(webhookURL string, hosts []cmdbmodel.CmdbHost, now time.Time) {
	var rows []string
	for _, host := range hosts {
		daysLeft := int(host.ExpireTime.Sub(now).Hours() / 24)
		rows = append(rows, fmt.Sprintf("| %s | %s | %d |",
			host.HostName,
			host.ExpireTime.Format("2006-01-02"),
			daysLeft,
		))
	}

	text := fmt.Sprintf(
		"### ⚠️ 主机到期预警\n\n共 **%d** 台主机将在 30 天内到期\n\n| 主机名 | 到期日期 | 剩余天数 |\n|---|---|---|\n%s\n",
		len(hosts),
		strings.Join(rows, "\n"),
	)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": "主机到期预警",
			"text":  text,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[到期预警] 序列化钉钉消息失败: %v", err)
		return
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		log.Printf("[到期预警] 发送钉钉通知失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("[到期预警] 钉钉通知发送成功，共 %d 台主机", len(hosts))
	} else {
		log.Printf("[到期预警] 钉钉通知返回 HTTP %d", resp.StatusCode)
	}
}

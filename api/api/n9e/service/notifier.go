package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Notifier 通知发送器
type Notifier struct {
	httpClient *http.Client
}

// NewNotifier 创建通知器
func NewNotifier() *Notifier {
	return &Notifier{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// WechatConfig 企业微信机器人配置
type WechatConfig struct {
	WebhookURL string `json:"webhookUrl"`
}

// DingtalkConfig 钉钉机器人配置
type DingtalkConfig struct {
	WebhookURL string `json:"webhookUrl"`
	Secret     string `json:"secret,omitempty"`
}

// EmailConfig 邮件 SMTP 配置
type EmailConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	From     string   `json:"from"`
	To       []string `json:"to"`
}

// AlertMessage 告警消息（统一格式）
type AlertMessage struct {
	AlertName   string
	Severity    string
	Status      string
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    time.Time
}

// SendWechat 发送企业微信机器人消息
func (n *Notifier) SendWechat(configJSON string, msg AlertMessage) error {
	var config WechatConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析企业微信配置失败: %w", err)
	}
	if config.WebhookURL == "" {
		return fmt.Errorf("企业微信 webhookUrl 为空")
	}

	statusEmoji := "🔴"
	if msg.Status == "resolved" {
		statusEmoji = "✅"
	}

	content := fmt.Sprintf(`### %s 告警通知
> **告警名称**: %s
> **严重级别**: %s
> **状态**: %s %s
> **时间**: %s`,
		statusEmoji, msg.AlertName, msg.Severity,
		msg.Status, statusEmoji, msg.StartsAt.Format("2006-01-02 15:04:05"))

	if desc, ok := msg.Annotations["description"]; ok {
		content += fmt.Sprintf("\n> **描述**: %s", desc)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	return n.postJSON(config.WebhookURL, payload)
}

// SendDingtalk 发送钉钉机器人消息
func (n *Notifier) SendDingtalk(configJSON string, msg AlertMessage) error {
	var config DingtalkConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析钉钉配置失败: %w", err)
	}
	if config.WebhookURL == "" {
		return fmt.Errorf("钉钉 webhookUrl 为空")
	}

	statusEmoji := "🔴"
	if msg.Status == "resolved" {
		statusEmoji = "✅"
	}

	title := fmt.Sprintf("%s [%s] %s", statusEmoji, strings.ToUpper(msg.Severity), msg.AlertName)
	text := fmt.Sprintf("### %s\n\n- **状态**: %s\n- **级别**: %s\n- **时间**: %s\n",
		title, msg.Status, msg.Severity, msg.StartsAt.Format("2006-01-02 15:04:05"))

	if desc, ok := msg.Annotations["description"]; ok {
		text += fmt.Sprintf("- **描述**: %s\n", desc)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]interface{}{
			"title": title,
			"text":  text,
		},
	}

	return n.postJSON(config.WebhookURL, payload)
}

// SendEmail 发送邮件通知
func (n *Notifier) SendEmail(configJSON string, msg AlertMessage) error {
	var config EmailConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析邮件配置失败: %w", err)
	}

	subject := fmt.Sprintf("[%s] %s - %s", strings.ToUpper(msg.Severity), msg.AlertName, msg.Status)
	body := fmt.Sprintf("告警名称: %s\n级别: %s\n状态: %s\n时间: %s\n",
		msg.AlertName, msg.Severity, msg.Status, msg.StartsAt.Format("2006-01-02 15:04:05"))

	if desc, ok := msg.Annotations["description"]; ok {
		body += fmt.Sprintf("描述: %s\n", desc)
	}

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		config.From, strings.Join(config.To, ","), subject, body)

	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	auth := smtp.PlainAuth("", config.Username, config.Password, config.Host)

	return smtp.SendMail(addr, auth, config.From, config.To, []byte(message))
}

// Dispatch 根据渠道类型分发通知
func (n *Notifier) Dispatch(channelType, configJSON string, msg AlertMessage) error {
	switch channelType {
	case "wechat":
		return n.SendWechat(configJSON, msg)
	case "dingtalk":
		return n.SendDingtalk(configJSON, msg)
	case "email":
		return n.SendEmail(configJSON, msg)
	default:
		return fmt.Errorf("不支持的通知渠道类型: %s", channelType)
	}
}

// postJSON 发送 JSON POST 请求
func (n *Notifier) postJSON(url string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化 JSON 失败: %w", err)
	}

	resp, err := n.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("通知发送响应: status=%d body=%s", resp.StatusCode, string(body))
		return fmt.Errorf("通知发送失败, HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

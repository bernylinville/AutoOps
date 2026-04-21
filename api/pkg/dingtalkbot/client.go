package dingtalkbot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	WebhookURL string
	Secret     string
	Timeout    time.Duration
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) SendText(content string, atMobiles, atUserIDs []string) error {
	payload := textPayload{
		MsgType: "text",
		Text: textContent{
			Content: content,
		},
		At: atConfig{
			AtMobiles: atMobiles,
			AtUserIDs: atUserIDs,
		},
	}
	return c.send(payload)
}

func (c *Client) SendMarkdown(title, text string, atMobiles, atUserIDs []string) error {
	payload := markdownPayload{
		MsgType: "markdown",
		Markdown: markdownContent{
			Title: title,
			Text:  text,
		},
		At: atConfig{
			AtMobiles: atMobiles,
			AtUserIDs: atUserIDs,
		},
	}
	return c.send(payload)
}

type textPayload struct {
	MsgType string      `json:"msgtype"`
	Text    textContent `json:"text"`
	At      atConfig    `json:"at"`
}

type textContent struct {
	Content string `json:"content"`
}

type markdownPayload struct {
	MsgType  string          `json:"msgtype"`
	Markdown markdownContent `json:"markdown"`
	At       atConfig        `json:"at"`
}

type markdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type atConfig struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIDs []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

func (c *Client) send(payload interface{}) error {
	if strings.TrimSpace(c.cfg.WebhookURL) == "" {
		return fmt.Errorf("钉钉机器人 webhookURL 为空")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化钉钉机器人消息失败: %w", err)
	}

	endpoint := c.signedWebhookURL()
	resp, err := c.httpClient.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("发送钉钉机器人消息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("钉钉机器人返回 HTTP %d", resp.StatusCode)
	}

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析钉钉机器人响应失败: %w", err)
	}
	if result.ErrCode != 0 {
		return fmt.Errorf("钉钉机器人返回 errcode=%d errmsg=%s", result.ErrCode, result.ErrMsg)
	}
	return nil
}

func (c *Client) signedWebhookURL() string {
	webhookURL := strings.TrimSpace(c.cfg.WebhookURL)
	secret := strings.TrimSpace(c.cfg.Secret)
	if secret == "" {
		return webhookURL
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(stringToSign))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	values := url.Values{}
	values.Set("timestamp", timestamp)
	values.Set("sign", sign)
	if strings.Contains(webhookURL, "?") {
		return webhookURL + "&" + values.Encode()
	}
	return webhookURL + "?" + values.Encode()
}

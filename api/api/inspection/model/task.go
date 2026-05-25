// Package model provides GORM models for the inspection feature.
package model

import (
	"net/url"
	"strings"
	"time"

	"dodevops-api/common/util"
)

// InspectionTask 巡检任务定义（按 N9E 业务组）
type InspectionTask struct {
	ID               uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	N9EGroupID       int64      `gorm:"column:n9e_group_id;uniqueIndex:idx_task_group;NOT NULL;comment:N9E 业务组 ID" json:"n9eGroupId"`
	N9EGroupName     string     `gorm:"column:n9e_group_name;type:varchar(200);comment:业务组名称快照" json:"n9eGroupName"`
	Name             string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:任务名称" json:"name"`
	Enabled          bool       `gorm:"column:enabled;default:true;comment:是否启用" json:"enabled"`
	Cron             string     `gorm:"column:cron;type:varchar(100);default:'CRON_TZ=Asia/Shanghai 0 10 * * *';comment:Cron 表达式" json:"cron"`
	TargetQuery      string     `gorm:"column:target_query;type:varchar(500);comment:目标标签过滤，如 items=业务标签" json:"targetQuery"`
	NotifyWebhookURL string     `gorm:"column:notify_webhook_url;type:varchar(500);comment:钉钉 Webhook URL" json:"notifyWebhookUrl"`
	NotifySecret     string     `gorm:"column:notify_secret;type:varchar(200);comment:钉钉 Secret" json:"-"` // GET 不返回
	NotifyOnWarning  bool       `gorm:"column:notify_on_warning;default:true;comment:Warning 是否通知" json:"notifyOnWarning"`
	NotifyOnCritical bool       `gorm:"column:notify_on_critical;default:true;comment:Critical 是否通知" json:"notifyOnCritical"`
	NotifyOnFailure  bool       `gorm:"column:notify_on_failure;default:true;comment:失败是否通知" json:"notifyOnFailure"`
	CreateTime       util.HTime `gorm:"column:create_time;NOT NULL" json:"createTime"`
	UpdateTime       util.HTime `gorm:"column:update_time;NOT NULL" json:"updateTime"`
}

func (InspectionTask) TableName() string { return "inspection_task" }

// TaskVO GET 接口返回的脱敏视图
type TaskVO struct {
	ID               uint       `json:"id"`
	N9EGroupID       int64      `json:"n9eGroupId"`
	N9EGroupName     string     `json:"n9eGroupName"`
	Name             string     `json:"name"`
	Enabled          bool       `json:"enabled"`
	Cron             string     `json:"cron"`
	TargetQuery      string     `json:"targetQuery"`
	NotifyWebhookURL string     `json:"notifyWebhookUrl"` // 脱敏: token=***
	NotifySecret     string     `json:"notifySecret"`     // 始终返回空
	NotifyOnWarning  bool       `json:"notifyOnWarning"`
	NotifyOnCritical bool       `json:"notifyOnCritical"`
	NotifyOnFailure  bool       `json:"notifyOnFailure"`
	CreateTime       util.HTime `json:"createTime"`
	UpdateTime       util.HTime `json:"updateTime"`
}

// MaskWebhookURL 脱敏 Webhook URL（token 部分掩码为 "***"）
func (t *InspectionTask) MaskWebhookURL() string {
	if t.NotifyWebhookURL == "" {
		return ""
	}

	u := t.NotifyWebhookURL
	// 替换 access_token=xxx 中的 xxx 为 ***
	idx := 0
	for {
		idx = findTokenIndex(u, idx)
		if idx == -1 {
			break
		}
		// 找到 token 值的结束位置
		end := idx
		for end < len(u) && u[end] != '&' && u[end] != ' ' {
			end++
		}
		u = u[:idx] + "***" + u[end:]
		idx += 3 // 跳过 ***
	}

	return u
}

func findTokenIndex(s string, start int) int {
	variants := []string{"access_token=", "token="}
	for _, v := range variants {
		idx := subIndex(s[start:], v)
		if idx != -1 {
			return start + idx + len(v)
		}
	}
	return -1
}

func subIndex(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func (t *InspectionTask) ToVO() *TaskVO {
	if t == nil {
		return nil
	}
	return &TaskVO{
		ID:               t.ID,
		N9EGroupID:       t.N9EGroupID,
		N9EGroupName:     t.N9EGroupName,
		Name:             t.Name,
		Enabled:          t.Enabled,
		Cron:             t.Cron,
		TargetQuery:      t.TargetQuery,
		NotifyWebhookURL: t.MaskWebhookURL(),
		NotifySecret:     "",
		NotifyOnWarning:  t.NotifyOnWarning,
		NotifyOnCritical: t.NotifyOnCritical,
		NotifyOnFailure:  t.NotifyOnFailure,
		CreateTime:       t.CreateTime,
		UpdateTime:       t.UpdateTime,
	}
}

// UpdateTaskDto PUT 接口入参，空字符串保留原值。
type UpdateTaskDto struct {
	Enabled          *bool   `json:"enabled,omitempty"`
	Cron             string  `json:"cron,omitempty"`
	TargetQuery      *string `json:"targetQuery,omitempty"`
	NotifyWebhookURL string  `json:"notifyWebhookUrl,omitempty"`
	NotifySecret     string  `json:"notifySecret,omitempty"`
	NotifyOnWarning  *bool   `json:"notifyOnWarning,omitempty"`
	NotifyOnCritical *bool   `json:"notifyOnCritical,omitempty"`
	NotifyOnFailure  *bool   `json:"notifyOnFailure,omitempty"`
}

func (t *InspectionTask) ApplyUpdate(dto *UpdateTaskDto) {
	if dto.Enabled != nil {
		t.Enabled = *dto.Enabled
	}
	if dto.Cron != "" {
		t.Cron = dto.Cron
	}
	if dto.TargetQuery != nil {
		t.TargetQuery = strings.TrimSpace(*dto.TargetQuery)
	}
	if dto.NotifyWebhookURL != "" && !isMaskedURL(dto.NotifyWebhookURL) {
		t.NotifyWebhookURL = dto.NotifyWebhookURL
	}
	if dto.NotifySecret != "" {
		t.NotifySecret = dto.NotifySecret
	}
	if dto.NotifyOnWarning != nil {
		t.NotifyOnWarning = *dto.NotifyOnWarning
	}
	if dto.NotifyOnCritical != nil {
		t.NotifyOnCritical = *dto.NotifyOnCritical
	}
	if dto.NotifyOnFailure != nil {
		t.NotifyOnFailure = *dto.NotifyOnFailure
	}
	t.UpdateTime = util.HTime{Time: time.Now()}
}

// TaskListQuery 任务列表查询参数
type TaskListQuery struct {
	Enabled  *bool  `form:"enabled"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// isMaskedURL returns true if the URL contains a masked token (literal or URL-encoded ***).
func isMaskedURL(raw string) bool {
	decoded, err := url.QueryUnescape(raw)
	if err == nil && strings.Contains(decoded, "***") {
		return true
	}
	return strings.Contains(raw, "***")
}

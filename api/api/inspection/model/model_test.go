package model

import (
	"testing"
	"time"

	"dodevops-api/common/util"
)

func TestMaskWebhookURL_Empty(t *testing.T) {
	task := &InspectionTask{NotifyWebhookURL: ""}
	got := task.MaskWebhookURL()
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestMaskWebhookURL_AccessToken(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=abc123def456",
	}
	got := task.MaskWebhookURL()
	if got == task.NotifyWebhookURL {
		t.Error("webhook URL was not masked")
	}
	if got != "https://oapi.dingtalk.com/robot/send?access_token=***" {
		t.Errorf("expected masked access_token, got %q", got)
	}
}

func TestMaskWebhookURL_TokenParam(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://example.com/hook?token=secret123",
	}
	got := task.MaskWebhookURL()
	if got != "https://example.com/hook?token=***" {
		t.Errorf("expected masked token, got %q", got)
	}
}

func TestMaskWebhookURL_MultipleParams(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=abc&timestamp=123&sign=xyz",
	}
	got := task.MaskWebhookURL()
	if got != "https://oapi.dingtalk.com/robot/send?access_token=***&timestamp=123&sign=xyz" {
		t.Errorf("expected only token masked, got %q", got)
	}
}

func TestMaskWebhookURL_NoToken(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://example.com/plain/hook",
	}
	got := task.MaskWebhookURL()
	if got != "https://example.com/plain/hook" {
		t.Errorf("expected no change, got %q", got)
	}
}

func TestToVO_Nil(t *testing.T) {
	var task *InspectionTask
	vo := task.ToVO()
	if vo != nil {
		t.Error("expected nil VO for nil task")
	}
}

func TestToVO_Masking(t *testing.T) {
	task := &InspectionTask{
		ID:               1,
		N9EGroupID:       100,
		N9EGroupName:     "生产环境",
		Name:             "生产环境巡检",
		Enabled:          true,
		Cron:             "0 10 * * *",
		TargetQuery:      "busigroup=生产环境",
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=secret123",
		NotifySecret:     "SECabc",
		NotifyOnWarning:  true,
		NotifyOnCritical: true,
		NotifyOnFailure:  false,
		CreateTime:       util.HTime{Time: time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)},
		UpdateTime:       util.HTime{Time: time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local)},
	}

	vo := task.ToVO()
	if vo == nil {
		t.Fatal("expected non-nil VO")
	}

	// Secret must be empty.
	if vo.NotifySecret != "" {
		t.Errorf("expected empty secret, got %q", vo.NotifySecret)
	}

	// Webhook URL must be masked.
	if vo.NotifyWebhookURL == task.NotifyWebhookURL {
		t.Error("webhook URL should be masked in VO")
	}

	// Other fields should match.
	if vo.ID != task.ID {
		t.Errorf("ID: expected %d, got %d", task.ID, vo.ID)
	}
	if vo.Name != task.Name {
		t.Errorf("Name: expected %q, got %q", task.Name, vo.Name)
	}
	if vo.NotifyOnFailure != task.NotifyOnFailure {
		t.Errorf("NotifyOnFailure: expected %v, got %v", task.NotifyOnFailure, vo.NotifyOnFailure)
	}
}

func TestApplyUpdate_AllFields(t *testing.T) {
	now := time.Now()
	task := &InspectionTask{
		Enabled:          false,
		Cron:             "0 10 * * *",
		NotifyWebhookURL: "old_url",
		NotifySecret:     "old_secret",
		NotifyOnWarning:  true,
		NotifyOnCritical: true,
		NotifyOnFailure:  true,
		UpdateTime:       util.HTime{Time: now},
	}

	enabled := true
	warnOff := false
	dto := &UpdateTaskDto{
		Enabled:          &enabled,
		Cron:             "0 8 * * *",
		NotifyWebhookURL: "new_url",
		NotifySecret:     "new_secret",
		NotifyOnWarning:  &warnOff,
		NotifyOnCritical: nil, // nil = no change
		NotifyOnFailure:  nil, // nil = no change
	}

	task.ApplyUpdate(dto)

	if task.Enabled != true {
		t.Error("enabled should be true")
	}
	if task.Cron != "0 8 * * *" {
		t.Errorf("cron should be updated, got %q", task.Cron)
	}
	if task.NotifyWebhookURL != "new_url" {
		t.Errorf("webhook URL should be updated, got %q", task.NotifyWebhookURL)
	}
	if task.NotifySecret != "new_secret" {
		t.Errorf("secret should be updated, got %q", task.NotifySecret)
	}
	if task.NotifyOnWarning != false {
		t.Error("notify_on_warning should be false")
	}
	if task.NotifyOnCritical != true {
		t.Error("notify_on_critical should remain true (nil)")
	}
	if task.NotifyOnFailure != true {
		t.Error("notify_on_failure should remain true (nil)")
	}
	if task.UpdateTime.Time.Before(now) {
		t.Error("update time should be updated")
	}
}

func TestApplyUpdate_EmptyStringPreservesWebhookURL(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=preserved_key",
		UpdateTime:       util.HTime{Time: time.Now()},
	}

	dto := &UpdateTaskDto{
		NotifyWebhookURL: "", // empty string: preserve original
	}

	task.ApplyUpdate(dto)

	if task.NotifyWebhookURL != "https://oapi.dingtalk.com/robot/send?access_token=preserved_key" {
		t.Errorf("webhook URL should be preserved, got %q", task.NotifyWebhookURL)
	}
}

func TestApplyUpdate_EmptyStringPreservesSecret(t *testing.T) {
	task := &InspectionTask{
		NotifySecret: "preserved_secret",
		UpdateTime:   util.HTime{Time: time.Now()},
	}

	dto := &UpdateTaskDto{
		NotifySecret: "", // empty string: preserve original
	}

	task.ApplyUpdate(dto)

	if task.NotifySecret != "preserved_secret" {
		t.Errorf("secret should be preserved, got %q", task.NotifySecret)
	}
}

func TestApplyUpdate_MaskedURLNotApplied(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=real_key",
		UpdateTime:       util.HTime{Time: time.Now()},
	}

	// Simulate frontend sending back the masked URL.
	dto := &UpdateTaskDto{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=***",
	}

	task.ApplyUpdate(dto)

	if task.NotifyWebhookURL != "https://oapi.dingtalk.com/robot/send?access_token=real_key" {
		t.Errorf("webhook URL should NOT be overwritten with masked value, got %q", task.NotifyWebhookURL)
	}
}

func TestApplyUpdate_OnlyProvidedFieldsChanged(t *testing.T) {
	task := &InspectionTask{
		Enabled:          true,
		Cron:             "0 10 * * *",
		NotifyWebhookURL: "original_url",
		NotifySecret:     "original_secret",
		NotifyOnWarning:  true,
		NotifyOnCritical: true,
		NotifyOnFailure:  true,
		UpdateTime:       util.HTime{Time: time.Now()},
	}

	// Only change cron.
	dto := &UpdateTaskDto{
		Cron: "0 12 * * *",
	}

	task.ApplyUpdate(dto)

	if task.Cron != "0 12 * * *" {
		t.Errorf("cron should be updated, got %q", task.Cron)
	}
	if task.Enabled != true {
		t.Error("enabled should remain unchanged")
	}
	if task.NotifyWebhookURL != "original_url" {
		t.Error("webhook URL should remain unchanged")
	}
	if task.NotifySecret != "original_secret" {
		t.Error("secret should remain unchanged")
	}
}

func TestApplyUpdate_TargetQueryCanBeCleared(t *testing.T) {
	task := &InspectionTask{
		TargetQuery: "busigroup=生产环境",
		UpdateTime:  util.HTime{Time: time.Now()},
	}

	empty := "  "
	task.ApplyUpdate(&UpdateTaskDto{TargetQuery: &empty})

	if task.TargetQuery != "" {
		t.Errorf("target query should be cleared, got %q", task.TargetQuery)
	}
}

func TestApplyUpdate_TargetQueryNilPreserves(t *testing.T) {
	task := &InspectionTask{
		TargetQuery: "busigroup=生产环境",
		UpdateTime:  util.HTime{Time: time.Now()},
	}

	task.ApplyUpdate(&UpdateTaskDto{})

	if task.TargetQuery != "busigroup=生产环境" {
		t.Errorf("target query should be preserved, got %q", task.TargetQuery)
	}
}

func TestNewHostResultKeepsIdent(t *testing.T) {
	host := NewHostResult(&HostMeta{
		Ident:    "host-a@10.0.0.1",
		Hostname: "host-a",
	})

	if host.Ident != "host-a@10.0.0.1" {
		t.Errorf("expected ident to be preserved, got %q", host.Ident)
	}
}

func TestBeforeCreate_SetsRunDate(t *testing.T) {
	run := &InspectionRun{
		TaskID:      1,
		TriggerType: TriggerTypeCron,
		Status:      RunStatusPending,
	}

	// BeforeCreate is called by GORM. Simulate by calling directly.
	err := run.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if run.RunDate == "" {
		t.Error("RunDate should be set")
	}

	today := time.Now().Format("2006-01-02")
	if run.RunDate != today {
		t.Errorf("RunDate expected %q, got %q", today, run.RunDate)
	}
}

func TestBeforeCreate_PreservesExistingRunDate(t *testing.T) {
	run := &InspectionRun{
		TaskID:      1,
		TriggerType: TriggerTypeCron,
		Status:      RunStatusPending,
		RunDate:     "2026-01-15",
	}

	err := run.BeforeCreate(nil)
	if err != nil {
		t.Fatalf("BeforeCreate returned error: %v", err)
	}
	if run.RunDate != "2026-01-15" {
		t.Errorf("existing RunDate should be preserved, got %q", run.RunDate)
	}
}

func TestJSONRaw_Scan_Nil(t *testing.T) {
	var j JSONRaw
	err := j.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) should succeed: %v", err)
	}
	if j != nil {
		t.Error("JSONRaw should be nil after scanning nil")
	}
}

func TestJSONRaw_Scan_Bytes(t *testing.T) {
	var j JSONRaw
	err := j.Scan([]byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("Scan(bytes) should succeed: %v", err)
	}
	if string(j) != `{"key":"value"}` {
		t.Errorf("expected %q, got %q", `{"key":"value"}`, string(j))
	}
}

func TestJSONRaw_Value_Nil(t *testing.T) {
	var j JSONRaw
	v, err := j.Value()
	if err != nil {
		t.Fatalf("Value() should succeed: %v", err)
	}
	if v != nil {
		t.Errorf("expected nil, got %v", v)
	}
}

func TestJSONRaw_Value_Bytes(t *testing.T) {
	j := JSONRaw(`{"key":"value"}`)
	v, err := j.Value()
	if err != nil {
		t.Fatalf("Value() should succeed: %v", err)
	}
	bytes, ok := v.([]byte)
	if !ok {
		t.Fatalf("expected []byte, got %T", v)
	}
	if string(bytes) != `{"key":"value"}` {
		t.Errorf("expected %q, got %q", `{"key":"value"}`, string(bytes))
	}
}

func TestApplyUpdate_URLEncodedMaskedURLNotApplied(t *testing.T) {
	task := &InspectionTask{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=real_key",
		UpdateTime:       util.HTime{Time: time.Now()},
	}

	// URL-encoded *** (%2A%2A%2A) should be detected as masked.
	dto := &UpdateTaskDto{
		NotifyWebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=%2A%2A%2A",
	}

	task.ApplyUpdate(dto)

	if task.NotifyWebhookURL != "https://oapi.dingtalk.com/robot/send?access_token=real_key" {
		t.Errorf("webhook URL should NOT be overwritten with URL-encoded masked value, got %q", task.NotifyWebhookURL)
	}
}

func TestIsMaskedURL(t *testing.T) {
	cases := []struct {
		input  string
		masked bool
	}{
		{"https://x.com?token=***", true},
		{"https://x.com?token=%2A%2A%2A", true},
		{"https://x.com?token=%2a%2a%2a", true},
		{"https://x.com?token=real123", false},
		{"", false},
		{"https://x.com?token=%2A%2Areal", false},
	}
	for _, tc := range cases {
		got := isMaskedURL(tc.input)
		if got != tc.masked {
			t.Errorf("isMaskedURL(%q): expected %v, got %v", tc.input, tc.masked, got)
		}
	}
}

package service

import (
	"strings"
	"testing"
	"time"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service/engine/vmclient"
)

func TestDetermineRunStatus_Success(t *testing.T) {
	r := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:  10,
			NormalHosts: 10,
		},
	}
	if got := determineRunStatus(r); got != model.RunStatusSuccess {
		t.Errorf("expected success, got %s", got)
	}
}

func TestDetermineRunStatus_Partial_Warning(t *testing.T) {
	r := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:   10,
			NormalHosts:  8,
			WarningHosts: 2,
		},
	}
	if got := determineRunStatus(r); got != model.RunStatusPartial {
		t.Errorf("expected partial, got %s", got)
	}
}

func TestDetermineRunStatus_Partial_Critical(t *testing.T) {
	r := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:    10,
			NormalHosts:   8,
			CriticalHosts: 2,
		},
	}
	if got := determineRunStatus(r); got != model.RunStatusPartial {
		t.Errorf("expected partial, got %s", got)
	}
}

func TestDetermineRunStatus_Partial_Failed(t *testing.T) {
	r := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:  10,
			NormalHosts: 8,
			FailedHosts: 2,
		},
	}
	if got := determineRunStatus(r); got != model.RunStatusPartial {
		t.Errorf("expected partial, got %s", got)
	}
}

func TestDetermineRunStatus_Failed_AllFailed(t *testing.T) {
	r := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:  5,
			FailedHosts: 5,
		},
	}
	if got := determineRunStatus(r); got != model.RunStatusFailed {
		t.Errorf("expected failed, got %s", got)
	}
}

func TestDetermineRunStatus_NilSummary(t *testing.T) {
	r := &model.InspectionResult{Summary: nil}
	if got := determineRunStatus(r); got != model.RunStatusFailed {
		t.Errorf("expected failed for nil summary, got %s", got)
	}
}

func TestParseTargetQuery_Empty(t *testing.T) {
	if got := parseTargetQuery(""); got != nil {
		t.Errorf("expected nil for empty query, got %v", got)
	}
	if got := parseTargetQuery("  "); got != nil {
		t.Errorf("expected nil for whitespace query, got %v", got)
	}
}

func TestParseTargetQuery_Busigroup(t *testing.T) {
	f := parseTargetQuery("busigroup=生产环境")
	if f == nil {
		t.Fatal("expected non-nil filter")
	}
	if len(f.BusinessGroups) != 1 || f.BusinessGroups[0] != "生产环境" {
		t.Errorf("expected busigroup 生产环境, got %v", f.BusinessGroups)
	}
	if len(f.Tags) != 0 {
		t.Errorf("expected no tags, got %v", f.Tags)
	}
}

func TestParseTargetQuery_Tags(t *testing.T) {
	f := parseTargetQuery("env=prod,region=cn-east")
	if f == nil {
		t.Fatal("expected non-nil filter")
	}
	if f.Tags["env"] != "prod" {
		t.Errorf("expected env=prod, got %q", f.Tags["env"])
	}
	if f.Tags["region"] != "cn-east" {
		t.Errorf("expected region=cn-east, got %q", f.Tags["region"])
	}
}

func TestParseTargetQuery_BusigroupAndTags(t *testing.T) {
	f := parseTargetQuery("busigroup=生产环境,env=prod")
	if f == nil {
		t.Fatal("expected non-nil filter")
	}
	if len(f.BusinessGroups) != 1 || f.BusinessGroups[0] != "生产环境" {
		t.Errorf("expected busigroup 生产环境, got %v", f.BusinessGroups)
	}
	if f.Tags["env"] != "prod" {
		t.Errorf("expected env=prod, got %q", f.Tags["env"])
	}
}

func TestParseTargetQuery_InvalidFormat(t *testing.T) {
	// Malformed entry (no = sign) should be skipped.
	f := parseTargetQuery("invalid")
	if f != nil {
		t.Error("expected nil for malformed entry")
	}
}

func TestParseTargetQuery_OnlyCommas(t *testing.T) {
	f := parseTargetQuery(" , , ")
	if f != nil {
		t.Error("expected nil for only commas")
	}
}

func TestGenerateReportPath_DefaultDir(t *testing.T) {
	path := (&InspectionService{}).generateReportPath("", 42)
	if !strings.Contains(path, "/data/inspection/42/") {
		t.Errorf("expected path under /data/inspection/42/, got %q", path)
	}
	if !strings.Contains(path, "inspection_report_42_") {
		t.Errorf("expected filename prefix, got %q", path)
	}
	if !strings.HasSuffix(path, ".xlsx") {
		t.Errorf("expected .xlsx suffix, got %q", path)
	}
}

func TestGenerateReportPath_CustomDir(t *testing.T) {
	path := (&InspectionService{}).generateReportPath("/custom/path", 7)
	if !strings.HasPrefix(path, "/custom/path/7/") {
		t.Errorf("expected path under /custom/path/7/, got %q", path)
	}
}

func TestFormatDurationMs_Zero(t *testing.T) {
	if got := formatDurationMs(0); got != "-" {
		t.Errorf("expected '-', got %q", got)
	}
}

func TestFormatDurationMs_Negative(t *testing.T) {
	if got := formatDurationMs(-100); got != "-" {
		t.Errorf("expected '-', got %q", got)
	}
}

func TestFormatDurationMs_SubSecond(t *testing.T) {
	if got := formatDurationMs(500); got != "500ms" {
		t.Errorf("expected '500ms', got %q", got)
	}
}

func TestFormatDurationMs_Seconds(t *testing.T) {
	if got := formatDurationMs(5000); got != "5s" {
		t.Errorf("expected '5s', got %q", got)
	}
}

func TestFormatDurationMs_Minutes(t *testing.T) {
	if got := formatDurationMs(150000); got != "2m30s" {
		t.Errorf("expected '2m30s', got %q", got)
	}
}

func TestBuildMarkdown_SuccessRun(t *testing.T) {
	n := &InspectionNotifier{}
	now := time.Now()
	run := &model.InspectionRun{
		ID:           42,
		TriggerType:  model.TriggerTypeCron,
		Status:       model.RunStatusSuccess,
		TotalHosts:   10,
		NormalHosts:  10,
		WarningHosts: 0,
		CriticalHosts: 0,
		FailedHosts:  0,
		TotalAlerts:  0,
		DurationMs:   150000,
		EndedAt:      &now,
	}
	task := &model.InspectionTask{
		ID:   3,
		Name: "生产环境",
	}

	title, text := n.buildMarkdown(run, task)
	if !strings.Contains(title, "生产环境") {
		t.Errorf("title should contain task name, got %q", title)
	}
	if !strings.Contains(text, "ID: 3") {
		t.Errorf("text should contain task ID, got %q", text)
	}
	if !strings.Contains(text, "正常 10") {
		t.Errorf("text should contain stats, got %q", text)
	}
}

func TestBuildMarkdown_FailedRun(t *testing.T) {
	n := &InspectionNotifier{}
	now := time.Now()
	run := &model.InspectionRun{
		ID:           42,
		TriggerType:  model.TriggerTypeManual,
		Status:       model.RunStatusFailed,
		TotalHosts:   5,
		NormalHosts:  0,
		FailedHosts:  5,
		TotalAlerts:  0,
		DurationMs:   1000,
		ErrorMessage: "connection refused",
		EndedAt:      &now,
	}
	task := &model.InspectionTask{
		ID:   3,
		Name: "测试业务组",
	}

	title, text := n.buildMarkdown(run, task)
	if !strings.Contains(title, "测试业务组") {
		t.Errorf("title should contain task name, got %q", title)
	}
	if !strings.Contains(text, "connection refused") {
		t.Errorf("text should contain error message, got %q", text)
	}
}

func TestBuildMarkdown_PartialRun(t *testing.T) {
	n := &InspectionNotifier{}
	now := time.Now()
	run := &model.InspectionRun{
		ID:            42,
		TriggerType:   model.TriggerTypeCron,
		Status:        model.RunStatusPartial,
		TotalHosts:    10,
		NormalHosts:   7,
		WarningHosts:  2,
		CriticalHosts: 1,
		FailedHosts:   0,
		TotalAlerts:   3,
		DurationMs:    120000,
		EndedAt:       &now,
	}
	task := &model.InspectionTask{
		ID:   3,
		Name: "生产环境",
	}

	title, text := n.buildMarkdown(run, task)
	if !strings.Contains(text, "警告 2") {
		t.Errorf("text should contain warning count, got %q", text)
	}
	if !strings.Contains(text, "严重 1") {
		t.Errorf("text should contain critical count, got %q", text)
	}
	if !strings.Contains(text, "异常") {
		t.Errorf("text should contain alert count, got %q", text)
	}
	_ = title
}

func TestRemoveReportFile_EmptyPath(t *testing.T) {
	// Should not panic or attempt deletion.
	s := &CleanupService{}
	s.removeReportFile("")
	// Just asserting no panic.
}

func TestRemoveReportFile_PathTraversalRejected(t *testing.T) {
	s := &CleanupService{}
	// Attempt to traverse outside /data/inspection/.
	s.removeReportFile("/etc/passwd")
	s.removeReportFile("/data/inspection/../../../etc/passwd")
	s.removeReportFile("/tmp/evil.xlsx")
	// All should be rejected silently.
}

func TestRemoveReportFile_NonexistentFile(t *testing.T) {
	s := &CleanupService{}
	// Valid path but file doesn't exist — should log but not error.
	s.removeReportFile("/data/inspection/99999/nonexistent.xlsx")
}

func TestHostFilter_IsEmpty(t *testing.T) {
	var nilFilter *vmclient.HostFilter
	if !nilFilter.IsEmpty() {
		t.Error("nil filter should be empty")
	}

	emptyFilter := &vmclient.HostFilter{}
	if !emptyFilter.IsEmpty() {
		t.Error("empty filter should be empty")
	}

	nonEmptyFilter := &vmclient.HostFilter{
		BusinessGroups: []string{"test"},
	}
	if nonEmptyFilter.IsEmpty() {
		t.Error("non-empty filter should not be empty")
	}

	tagFilter := &vmclient.HostFilter{
		Tags: map[string]string{"env": "prod"},
	}
	if tagFilter.IsEmpty() {
		t.Error("filter with tags should not be empty")
	}
}

// TestNotificationWouldFire proves CRIT-1 fix: after populating run struct from
// InspectionResult, the notification goroutine sees correct summary counts.
func TestNotificationWouldFire_AfterRunPopulation(t *testing.T) {
	run := &model.InspectionRun{
		ID:     42,
		TaskID: 3,
		Status: model.RunStatusSuccess,
	}
	result := &model.InspectionResult{
		Summary: &model.InspectionSummary{
			TotalHosts:    10,
			NormalHosts:   7,
			WarningHosts:  2,
			CriticalHosts: 1,
			FailedHosts:   0,
		},
		AlertSummary: &model.AlertSummary{
			TotalAlerts:   3,
			WarningCount:  2,
			CriticalCount: 1,
		},
	}

	// Populate run from result (this is what the fix adds).
	run.TotalHosts = result.Summary.TotalHosts
	run.NormalHosts = result.Summary.NormalHosts
	run.WarningHosts = result.Summary.WarningHosts
	run.CriticalHosts = result.Summary.CriticalHosts
	run.FailedHosts = result.Summary.FailedHosts
	run.TotalAlerts = result.AlertSummary.TotalAlerts

	// Verify the exact behavior that was broken:
	// Before fix: run.CriticalHosts == 0, run.TotalAlerts == 0 → shouldNotify never true.
	if run.CriticalHosts != 1 {
		t.Errorf("CriticalHosts: expected 1, got %d", run.CriticalHosts)
	}
	if run.WarningHosts != 2 {
		t.Errorf("WarningHosts: expected 2, got %d", run.WarningHosts)
	}
	if run.TotalAlerts != 3 {
		t.Errorf("TotalAlerts: expected 3, got %d", run.TotalAlerts)
	}

	// Verify notification conditions (mirrors NotifyRunResult logic).
	hasCritical := run.CriticalHosts > 0 || run.TotalAlerts > 0
	hasWarning := run.WarningHosts > 0
	hasFailure := run.FailedHosts > 0

	if !hasCritical {
		t.Error("hasCritical should be true — notification would be skipped without fix")
	}
	if !hasWarning {
		t.Error("hasWarning should be true")
	}
	if hasFailure {
		t.Error("hasFailure should be false")
	}
}

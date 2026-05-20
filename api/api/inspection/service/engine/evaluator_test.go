package engine

import (
	"testing"

	"dodevops-api/api/inspection/model"
)

func defs() []*model.MetricDefinition { return HostMetricDefinitions() }

func TestEvaluateHost_Normal(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 30.0))
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))

	result := e.EvaluateHost("test-host", hm)
	if result.Status != model.HostStatusNormal {
		t.Fatalf("expected normal, got %s", result.Status)
	}
	if len(result.Alerts) != 0 {
		t.Fatalf("expected 0 alerts, got %d", len(result.Alerts))
	}
	for _, mv := range result.Metrics {
		if mv.Status != model.MetricStatusNormal {
			t.Errorf("metric %s expected normal, got %s", mv.Name, mv.Status)
		}
	}
}

func TestEvaluateHost_Warning(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 75.0)) // warning ≥ 70
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))

	result := e.EvaluateHost("test-host", hm)
	if result.Status != model.HostStatusWarning {
		t.Fatalf("expected warning, got %s", result.Status)
	}
	if len(result.Alerts) == 0 {
		t.Fatal("expected at least 1 alert")
	}
	alert := result.Alerts[0]
	if alert.Level != model.AlertLevelWarning {
		t.Fatalf("expected warning alert, got %s", alert.Level)
	}
	if alert.MetricName != "cpu_usage" {
		t.Fatalf("expected cpu_usage alert, got %s", alert.MetricName)
	}
}

func TestEvaluateHost_Critical(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 95.0)) // critical ≥ 90
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))

	result := e.EvaluateHost("test-host", hm)
	if result.Status != model.HostStatusCritical {
		t.Fatalf("expected critical, got %s", result.Status)
	}
	alert := result.Alerts[0]
	if alert.Level != model.AlertLevelCritical {
		t.Fatalf("expected critical alert, got %s", alert.Level)
	}
}

func TestEvaluateHost_MultipleAlerts_DominantStatus(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 75.0))     // warning
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 95.0)) // critical
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))

	result := e.EvaluateHost("test-host", hm)
	if result.Status != model.HostStatusCritical {
		t.Fatalf("expected critical (dominant), got %s", result.Status)
	}
	if len(result.Alerts) < 2 {
		t.Fatalf("expected at least 2 alerts, got %d", len(result.Alerts))
	}
}

func TestEvaluateHost_NTPStratumZero(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 30.0))
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))

	ntp := model.NewMetricValue("ntp_offset", 0.5)
	ntp.Labels = map[string]string{"stratum": "0"}
	hm.SetMetric(ntp)

	result := e.EvaluateHost("test-host", hm)
	if result.Status != model.HostStatusCritical {
		t.Fatalf("expected critical (NTP unsynced), got %s", result.Status)
	}
	if len(result.Alerts) == 0 {
		t.Fatal("expected at least 1 alert")
	}
	alert := result.Alerts[0]
	if alert.Level != model.AlertLevelCritical {
		t.Fatalf("expected critical alert, got %s", alert.Level)
	}
	if alert.FormattedValue != "N/A (未同步)" {
		t.Fatalf("expected N/A (未同步), got %s", alert.FormattedValue)
	}
}

func TestEvaluateHost_NTPOffsetAbsoluteValue(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")

	// Negative offset should use abs value for threshold comparison.
	ntp := model.NewMetricValue("ntp_offset", -0.8)
	ntp.Labels = map[string]string{"stratum": "1"}
	hm.SetMetric(ntp)

	result := e.EvaluateHost("test-host", hm)
	// |−0.8| = 0.8 ≥ 0.7 warning, < 1.0 critical
	if result.Status != model.HostStatusWarning {
		t.Fatalf("expected warning for abs(offset)=0.8, got %s", result.Status)
	}
}

func TestEvaluateHost_ExpandedMetricsNoAlert(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewMetricValue("cpu_usage", 30.0))
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))
	// Expanded metrics should NOT generate alerts.
	hm.SetMetric(model.NewMetricValue("disk_usage:/home", 85.0))
	hm.SetMetric(model.NewMetricValue("disk_usage:/data", 92.0))

	result := e.EvaluateHost("test-host", hm)
	if len(result.Alerts) != 0 {
		t.Errorf("expanded metrics should not generate alerts, got %d", len(result.Alerts))
	}
}

func TestEvaluateHost_NAMetrics(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())
	hm := model.NewHostMetrics("test-host")
	hm.SetMetric(model.NewNAMetricValue("processes_zombies"))

	result := e.EvaluateHost("test-host", hm)
	if len(result.Alerts) > 0 {
		t.Errorf("N/A metrics should not generate alerts, got %d", len(result.Alerts))
	}
	if mv := result.Metrics["processes_zombies"]; mv == nil {
		t.Fatal("expected N/A metric to be present")
	} else if mv.Status != model.MetricStatusPending {
		t.Errorf("expected pending status for N/A, got %s", mv.Status)
	}
}

func TestEvaluateAll_Summary(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())

	hostMetrics := map[string]*model.HostMetrics{
		"host-normal":   makeHostMetrics("host-normal", 30.0),
		"host-warning":  makeHostMetrics("host-warning", 75.0),
		"host-critical": makeHostMetrics("host-critical", 95.0),
	}

	result := e.EvaluateAll(hostMetrics)
	if result.Summary.TotalAlerts != 2 {
		t.Fatalf("expected 2 total alerts, got %d", result.Summary.TotalAlerts)
	}
	if result.Summary.WarningCount != 1 {
		t.Fatalf("expected 1 warning, got %d", result.Summary.WarningCount)
	}
	if result.Summary.CriticalCount != 1 {
		t.Fatalf("expected 1 critical, got %d", result.Summary.CriticalCount)
	}
	if len(result.HostResults) != 3 {
		t.Fatalf("expected 3 host results, got %d", len(result.HostResults))
	}
}

func makeHostMetrics(hostname string, cpuValue float64) *model.HostMetrics {
	hm := model.NewHostMetrics(hostname)
	hm.SetMetric(model.NewMetricValue("cpu_usage", cpuValue))
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))
	return hm
}

func TestEvaluateThreshold_Boundary(t *testing.T) {
	e := NewEvaluator(DefaultThresholds(), defs())

	cases := []struct {
		value float64
		level model.AlertLevel
	}{
		{69.9, model.AlertLevelNormal},
		{70.0, model.AlertLevelWarning},
		{89.9, model.AlertLevelWarning},
		{90.0, model.AlertLevelCritical},
		{100.0, model.AlertLevelCritical},
	}

	thresh := &ThresholdPair{Warning: 70, Critical: 90}
	for _, tc := range cases {
		got := e.evaluateThreshold(tc.value, thresh)
		if got != tc.level {
			t.Errorf("value %.1f: expected %s, got %s", tc.value, tc.level, got)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1099511627776, "1.00 TB"},
	}
	for _, tc := range cases {
		got := formatBytes(tc.bytes)
		if got != tc.expected {
			t.Errorf("formatBytes(%d): expected %q, got %q", tc.bytes, tc.expected, got)
		}
	}
}

func TestFormatUptime(t *testing.T) {
	cases := []struct {
		seconds  float64
		expected string
	}{
		{0, "0分钟"},
		{60, "1分钟"},
		{3661, "1时1分"},   // 3600+61
		{90061, "1天1时1分"}, // 86400+3600+61
	}
	for _, tc := range cases {
		got := formatUptime(tc.seconds)
		if got != tc.expected {
			t.Errorf("formatUptime(%.0f): expected %q, got %q", tc.seconds, tc.expected, got)
		}
	}
}

func TestFormatNTPOffset(t *testing.T) {
	cases := []struct {
		seconds  float64
		expected string
	}{
		{0.0, "+0.0ms"},
		{0.001, "+1.0ms"},
		{-0.002, "-2.0ms"},
		{0.0005, "+0.5ms"},
	}
	for _, tc := range cases {
		got := formatNTPOffset(tc.seconds)
		if got != tc.expected {
			t.Errorf("formatNTPOffset(%.4f): expected %q, got %q", tc.seconds, tc.expected, got)
		}
	}
}

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()
	if th.CPUUsage.Warning != 70 {
		t.Errorf("cpu warning: expected 70, got %.0f", th.CPUUsage.Warning)
	}
	if th.CPUUsage.Critical != 90 {
		t.Errorf("cpu critical: expected 90, got %.0f", th.CPUUsage.Critical)
	}
	if th.NTPOffset.Warning != 0.5 {
		t.Errorf("ntp warning: expected 0.5, got %.1f", th.NTPOffset.Warning)
	}
}

func TestMetricThresholdMap_Coverage(t *testing.T) {
	for _, name := range AlertableMetricNames() {
		if _, ok := metricThresholdMap[name]; !ok {
			t.Errorf("alertable metric %q missing from metricThresholdMap", name)
		}
	}
}

func TestHostMetricDefinitions_NoNil(t *testing.T) {
	defs := HostMetricDefinitions()
	if len(defs) == 0 {
		t.Fatal("no metric definitions")
	}
	for _, d := range defs {
		if d == nil {
			t.Fatal("nil metric definition")
		}
		if d.Name == "" {
			t.Error("metric with empty name")
		}
		if d.DisplayName == "" {
			t.Errorf("metric %s has empty display name", d.Name)
		}
	}
}

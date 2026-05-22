package engine

import (
	"testing"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service/engine/vmclient"
)

func TestParseMemoryToKB(t *testing.T) {
	cases := []struct {
		input    string
		expected int64
	}{
		{"", 0},
		{"0", 0},
		{"16000000kB", 16000000},
		{"16000000KB", 16000000},
		{"16000000kb", 16000000},
		{"1GB", 1 * 1024 * 1024},
		{"2 GB", 2 * 1024 * 1024},
		{"2048MB", 2048 * 1024},
		{" 2048 MB ", 2048 * 1024},
		{"512K", 512},
		{"1024", 1024}, // bare number, assumed kB
		{"16.5GB", int64(16.5 * 1024 * 1024)},
	}
	for _, tc := range cases {
		got := parseMemoryToKB(tc.input)
		if got != tc.expected {
			t.Errorf("parseMemoryToKB(%q): expected %d, got %d", tc.input, tc.expected, got)
		}
	}
}

func TestTagsOnlyHostFilter(t *testing.T) {
	filter := &vmclient.HostFilter{
		BusinessGroups: []string{"prod"},
		Tags:           map[string]string{"region": "cq", "env": "prod"},
	}

	got := tagsOnlyHostFilter(filter)
	if got == nil {
		t.Fatal("expected tags-only filter")
	}
	if len(got.BusinessGroups) != 0 {
		t.Fatalf("expected business groups to be dropped, got %v", got.BusinessGroups)
	}
	if got.Tags["region"] != "cq" || got.Tags["env"] != "prod" {
		t.Fatalf("unexpected tags: %#v", got.Tags)
	}

	got.Tags["region"] = "changed"
	if filter.Tags["region"] != "cq" {
		t.Fatalf("tagsOnlyHostFilter should copy tags, original mutated to %#v", filter.Tags)
	}
}

func TestHasAnyCollectedMetric(t *testing.T) {
	if hasAnyCollectedMetric(nil) {
		t.Fatal("nil metrics should not count as collected")
	}
	metrics := map[string]*model.HostMetrics{
		"empty": model.NewHostMetrics("empty"),
		"pending": {
			Hostname: "pending",
			Metrics:  map[string]*model.MetricValue{"password_expiry": model.NewNAMetricValue("password_expiry")},
		},
	}
	if hasAnyCollectedMetric(metrics) {
		t.Fatal("N/A-only metrics should not count as collected")
	}
	metrics["host-1"] = model.NewHostMetrics("host-1")
	metrics["host-1"].SetMetric(model.NewMetricValue("cpu_usage", 1))
	if !hasAnyCollectedMetric(metrics) {
		t.Fatal("active metric should count as collected")
	}
}

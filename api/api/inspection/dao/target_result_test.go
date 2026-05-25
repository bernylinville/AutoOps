package dao

import (
	"testing"

	"dodevops-api/api/inspection/model"
)

func TestBuildMetricSummary(t *testing.T) {
	raw := model.JSONRaw(`{
		"cpu_usage":{"formatted_value":"12.3%","raw_value":12.3},
		"memory_usage":{"formatted_value":"45.6%","raw_value":45.6},
		"disk_usage_max":{"formatted_value":"78.9%","raw_value":78.9},
		"load_per_core":{"formatted_value":"0.42","raw_value":0.42}
	}`)

	got := BuildMetricSummary(raw)
	want := "CPU 12.3% / 内存 45.6% / 磁盘 78.9% / 负载 0.42"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildMetricSummarySkipsNA(t *testing.T) {
	raw := model.JSONRaw(`{
		"cpu_usage":{"formatted_value":"N/A","raw_value":0,"is_na":true},
		"memory_usage":{"formatted_value":"45.6%","raw_value":45.6}
	}`)

	got := BuildMetricSummary(raw)
	want := "内存 45.6%"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildMetricSummaryInvalidJSON(t *testing.T) {
	if got := BuildMetricSummary(model.JSONRaw(`not-json`)); got != "" {
		t.Fatalf("expected empty summary for invalid JSON, got %q", got)
	}
}

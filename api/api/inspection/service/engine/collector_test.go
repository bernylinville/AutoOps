package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service/engine/vmclient"
	n9eservice "dodevops-api/api/n9e/service"
)

func TestCollectorPreservesN9ETagMatchedHostsWithoutMetrics(t *testing.T) {
	const targetTag = "重数传媒数字乡村-电信侧"

	n9eServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/n9e/targets" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, map[string]any{
			"err": "",
			"dat": map[string]any{
				"total": 2,
				"list": []map[string]any{
					{
						"ident":      "cq-gdhy-eleack01",
						"hostname":   "cq-gdhy-eleack01",
						"host_ip":    "10.0.0.1",
						"os":         "linux",
						"tags_maps":  map[string]string{"items": targetTag},
						"group_ids":  []int64{1001},
						"group_objs": []map[string]any{{"id": 1001, "name": "重数传媒数字乡村电信侧"}},
					},
					{
						"ident":      "cq-gdhy-web01",
						"hostname":   "cq-gdhy-web01",
						"host_ip":    "10.0.0.2",
						"os":         "linux",
						"tags_maps":  map[string]string{"items": targetTag},
						"group_ids":  []int64{1001},
						"group_objs": []map[string]any{{"id": 1001, "name": "重数传媒数字乡村电信侧"}},
					},
				},
			},
		})
	}))
	defer n9eServer.Close()

	vmQueries := 0
	vmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			http.NotFound(w, r)
			return
		}
		vmQueries++
		if strings.Contains(r.URL.Query().Get("query"), "items") {
			t.Fatalf("N9E target tag must not be injected into VM query, got %q", r.URL.Query().Get("query"))
		}
		writeJSON(t, w, map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []map[string]any{
					{
						"metric": map[string]string{"ident": "cq-gdhy-eleack01", "items": targetTag},
						"value":  []any{float64(1), "10"},
					},
				},
			},
		})
	}))
	defer vmServer.Close()

	collector := NewCollector(
		n9eservice.NewN9EClient(n9eServer.URL, "token", 5),
		vmclient.NewClient(vmServer.URL, 5),
		[]*model.MetricDefinition{{Name: "cpu_usage", Query: "cpu_usage"}},
		&vmclient.HostFilter{TargetTags: map[string]string{"items": targetTag}},
		1,
	)

	result, err := collector.CollectAll(context.Background())
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}
	if vmQueries == 0 {
		t.Fatal("expected collector to query VM for metrics")
	}
	if len(result.Hosts) != 2 {
		t.Fatalf("expected both N9E tag-matched hosts to be retained, got %d", len(result.Hosts))
	}
	if len(result.FailedHosts) != 1 || result.FailedHosts[0].Hostname != "cq-gdhy-web01" {
		t.Fatalf("expected cq-gdhy-web01 to be marked failed for missing metrics, got %#v", result.FailedHosts)
	}
}

func TestCollectorMatchesVMIdentToN9EHostnameWithIP(t *testing.T) {
	n9eServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/n9e/targets" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, map[string]any{
			"err": "",
			"dat": map[string]any{
				"total": 1,
				"list": []map[string]any{
					{
						"ident":     "cq-gdhy-api1@192.168.7.102",
						"hostname":  "cq-gdhy-api1@192.168.7.102",
						"host_ip":   "192.168.7.102",
						"os":        "linux",
						"tags_maps": map[string]string{"items": "重数传媒数字乡村-电信侧"},
					},
				},
			},
		})
	}))
	defer n9eServer.Close()

	vmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			http.NotFound(w, r)
			return
		}
		writeJSON(t, w, map[string]any{
			"status": "success",
			"data": map[string]any{
				"resultType": "vector",
				"result": []map[string]any{
					{
						"metric": map[string]string{"ident": "cq-gdhy-api1"},
						"value":  []any{float64(1), "10"},
					},
				},
			},
		})
	}))
	defer vmServer.Close()

	collector := NewCollector(
		n9eservice.NewN9EClient(n9eServer.URL, "token", 5),
		vmclient.NewClient(vmServer.URL, 5),
		[]*model.MetricDefinition{{Name: "cpu_usage", Query: "cpu_usage"}},
		&vmclient.HostFilter{TargetTags: map[string]string{"items": "重数传媒数字乡村-电信侧"}},
		1,
	)

	result, err := collector.CollectAll(context.Background())
	if err != nil {
		t.Fatalf("CollectAll returned error: %v", err)
	}
	if len(result.FailedHosts) != 0 {
		t.Fatalf("expected host@IP N9E target to match clean VM ident, got failed hosts %#v", result.FailedHosts)
	}
	hostMetrics := result.HostMetrics["cq-gdhy-api1"]
	if hostMetrics == nil || hostMetrics.Metrics["cpu_usage"] == nil {
		t.Fatalf("expected cpu_usage metric under clean hostname, got %#v", result.HostMetrics)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode JSON response: %v", err)
	}
}

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

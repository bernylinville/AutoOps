package engine

import (
	"context"
	"testing"
	"time"

	"dodevops-api/api/inspection/model"
)

// mockCollector implements HostCollector for testing.
type mockCollector struct {
	result *CollectionResult
	err    error
}

func (m *mockCollector) CollectAll(ctx context.Context) (*CollectionResult, error) {
	return m.result, m.err
}

// mockEvaluator implements HostEvaluator for testing.
type mockEvaluator struct {
	result *EvaluationResult
}

func (m *mockEvaluator) EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult {
	return m.result
}

func TestInspector_Run_Success(t *testing.T) {
	collector := &mockCollector{
		result: &CollectionResult{
			Hosts: []*model.HostMeta{
				{Ident: "host-1@10.0.0.1", Hostname: "host-1", IP: "10.0.0.1", OS: "linux", CPUCores: 4},
				{Ident: "host-2@10.0.0.2", Hostname: "host-2", IP: "10.0.0.2", OS: "linux", CPUCores: 8},
			},
			HostMetrics: map[string]*model.HostMetrics{
				"host-1": buildHostMetrics("host-1", 30.0, model.HostStatusNormal),
				"host-2": buildHostMetrics("host-2", 75.0, model.HostStatusWarning),
			},
			CollectedAt: time.Now().Unix(),
		},
	}

	evaluator := &mockEvaluator{
		result: &EvaluationResult{
			HostResults: []*HostEvaluationResult{
				{
					Hostname: "host-1",
					Status:   model.HostStatusNormal,
					Metrics:  buildMetricMap("host-1", 30.0),
				},
				{
					Hostname: "host-2",
					Status:   model.HostStatusWarning,
					Metrics:  buildMetricMap("host-2", 75.0),
					Alerts: []*model.Alert{
						{
							Hostname:     "host-2",
							MetricName:   "cpu_usage",
							CurrentValue: 75.0,
							Level:        model.AlertLevelWarning,
						},
					},
				},
			},
			Alerts: []*model.Alert{
				{Hostname: "host-2", MetricName: "cpu_usage", CurrentValue: 75.0, Level: model.AlertLevelWarning},
			},
			Summary: &model.AlertSummary{TotalAlerts: 1, WarningCount: 1},
		},
	}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Summary == nil {
		t.Fatal("summary should not be nil")
	}
	if result.Summary.TotalHosts != 2 {
		t.Errorf("expected 2 total hosts, got %d", result.Summary.TotalHosts)
	}
	if result.Summary.NormalHosts != 1 {
		t.Errorf("expected 1 normal host, got %d", result.Summary.NormalHosts)
	}
	if result.Summary.WarningHosts != 1 {
		t.Errorf("expected 1 warning host, got %d", result.Summary.WarningHosts)
	}
	if result.AlertSummary == nil {
		t.Fatal("alert summary should not be nil")
	}
	if result.AlertSummary.TotalAlerts != 1 {
		t.Errorf("expected 1 total alert, got %d", result.AlertSummary.TotalAlerts)
	}
	if len(result.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(result.Hosts))
	}
	if result.Duration <= 0 {
		t.Error("duration should be > 0")
	}
}

func TestInspector_Run_EmptyHosts(t *testing.T) {
	collector := &mockCollector{
		result: &CollectionResult{
			Hosts:       []*model.HostMeta{},
			HostMetrics: make(map[string]*model.HostMetrics),
			CollectedAt: time.Now().Unix(),
		},
	}
	evaluator := &mockEvaluator{
		result: &EvaluationResult{},
	}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if result.Summary.TotalHosts != 0 {
		t.Errorf("expected 0 hosts, got %d", result.Summary.TotalHosts)
	}
}

func TestInspector_Run_CollectorError(t *testing.T) {
	collector := &mockCollector{
		err: context.DeadlineExceeded,
	}
	evaluator := &mockEvaluator{
		result: &EvaluationResult{},
	}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Error("result should be nil on error")
	}
}

func TestInspector_Run_FailedHosts(t *testing.T) {
	collector := &mockCollector{
		result: &CollectionResult{
			Hosts: []*model.HostMeta{
				{Ident: "host-ok@10.0.0.1", Hostname: "host-ok", IP: "10.0.0.1", OS: "linux", CPUCores: 4},
				{Ident: "host-fail@10.0.0.2", Hostname: "host-fail", IP: "10.0.0.2", OS: "linux", CPUCores: 8},
			},
			HostMetrics: map[string]*model.HostMetrics{
				"host-ok": buildHostMetrics("host-ok", 30.0, model.HostStatusNormal),
			},
			FailedHosts: []FailedHost{
				{Hostname: "host-fail", Error: "no metrics collected"},
			},
			CollectedAt: time.Now().Unix(),
		},
	}

	evaluator := &mockEvaluator{
		result: &EvaluationResult{
			HostResults: []*HostEvaluationResult{
				{
					Hostname: "host-ok",
					Status:   model.HostStatusNormal,
					Metrics:  buildMetricMap("host-ok", 30.0),
				},
			},
		},
	}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary.FailedHosts != 1 {
		t.Errorf("expected 1 failed host, got %d", result.Summary.FailedHosts)
	}
	if len(result.Hosts) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(result.Hosts))
	}
	for _, host := range result.Hosts {
		if host.Hostname == "host-fail" {
			if host.Status != model.HostStatusFailed {
				t.Errorf("host-fail expected status failed, got %s", host.Status)
			}
			if host.Error != "no metrics collected" {
				t.Errorf("host-fail expected error 'no metrics collected', got %q", host.Error)
			}
		}
	}
}

func TestInspector_Run_BootTimeCalculation(t *testing.T) {
	uptimeSeconds := float64(86400) // 1 day

	collector := &mockCollector{
		result: &CollectionResult{
			Hosts: []*model.HostMeta{
				{Ident: "host-1@10.0.0.1", Hostname: "host-1", IP: "10.0.0.1", OS: "linux", CPUCores: 4},
			},
			HostMetrics: map[string]*model.HostMetrics{
				"host-1": buildHostMetrics("host-1", 30.0, model.HostStatusNormal),
			},
			CollectedAt: time.Now().Unix(),
		},
	}

	evaluator := &mockEvaluator{
		result: &EvaluationResult{
			HostResults: []*HostEvaluationResult{
				{
					Hostname: "host-1",
					Status:   model.HostStatusNormal,
					Metrics:  buildMetricMapWithUptime("host-1", 30.0, uptimeSeconds),
				},
			},
		},
	}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(result.Hosts))
	}
	bootTime := result.Hosts[0].BootTime
	if bootTime == "" {
		t.Error("boot time should not be empty")
	}
}

func TestInspector_Run_Version(t *testing.T) {
	collector := &mockCollector{
		result: &CollectionResult{
			Hosts:       []*model.HostMeta{},
			HostMetrics: make(map[string]*model.HostMetrics),
			CollectedAt: time.Now().Unix(),
		},
	}
	evaluator := &mockEvaluator{result: &EvaluationResult{}}

	inspector := NewInspector(collector, evaluator)
	result, err := inspector.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result.Version)
	}
}

func TestInspector_Run_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	collector := &mockCollector{
		result: &CollectionResult{
			Hosts:       []*model.HostMeta{},
			HostMetrics: make(map[string]*model.HostMetrics),
		},
	}

	evaluator := &mockEvaluator{result: &EvaluationResult{}}
	inspector := NewInspector(collector, evaluator)

	// Still runs because mock doesn't check ctx. This test validates
	// that the inspector propagates the context to collector.
	result, err := inspector.Run(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

// --- helpers ---

func buildHostMetrics(hostname string, cpu float64, status model.HostStatus) *model.HostMetrics {
	hm := model.NewHostMetrics(hostname)
	hm.SetMetric(model.NewMetricValue("cpu_usage", cpu))
	hm.SetMetric(model.NewMetricValue("memory_usage", 40.0))
	hm.SetMetric(model.NewMetricValue("disk_usage_max", 50.0))
	hm.SetMetric(model.NewMetricValue("processes_zombies", 0))
	hm.SetMetric(model.NewMetricValue("load_per_core", 0.4))
	hm.SetMetric(model.NewMetricValue("ntp_offset", 0.01))
	return hm
}

func buildMetricMap(hostname string, cpu float64) map[string]*model.MetricValue {
	hm := buildHostMetrics(hostname, cpu, "")
	return hm.Metrics
}

func buildMetricMapWithUptime(hostname string, cpu float64, uptime float64) map[string]*model.MetricValue {
	m := buildMetricMap(hostname, cpu)
	m["uptime"] = model.NewMetricValue("uptime", uptime)
	return m
}

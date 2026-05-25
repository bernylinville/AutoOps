package vmclient

import "testing"

func TestInjectMatchersToQuery_Simple(t *testing.T) {
	got := injectMatchersToQuery("cpu_usage", []string{`busigroup=~"prod"`})
	want := `cpu_usage{busigroup=~"prod"}`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestInjectMatchersToQuery_ExistingLabels(t *testing.T) {
	got := injectMatchersToQuery(`cpu_usage{cpu="cpu-total"}`, []string{`busigroup=~"prod"`})
	want := `cpu_usage{cpu="cpu-total", busigroup=~"prod"}`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestInjectMatchersToQuery_ExpressionQuery(t *testing.T) {
	// Expression-based query like memory_usage: should NOT inject into the number 100.
	got := injectMatchersToQuery("100 - mem_available_percent", []string{`busigroup=~"prod"`})
	// 100 should be left alone (no braces injected); mem_available_percent should get matchers.
	want := `100 - mem_available_percent{busigroup=~"prod"}`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestInjectMatchersToQuery_FunctionCall(t *testing.T) {
	// rate() is a function call and should NOT get label matchers injected.
	got := injectMatchersToQuery("rate(cpu_usage[5m])", []string{`busigroup=~"prod"`})
	want := "rate(cpu_usage{busigroup=~\"prod\"}[5m])"
	if got != want {
		t.Errorf("expected %s, got %s", want, got)
	}
}

func TestInjectMatchersToQuery_MultipleMetrics(t *testing.T) {
	got := injectMatchersToQuery("cpu_usage + memory_usage", []string{`busigroup=~"prod"`})
	want := `cpu_usage{busigroup=~"prod"} + memory_usage{busigroup=~"prod"}`
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestInjectMatchersToQuery_NilFilter(t *testing.T) {
	got := injectMatchersToQuery("cpu_usage", nil)
	if got != "cpu_usage" {
		t.Errorf("expected no change, got %q", got)
	}
}

func TestInjectMatchersToQuery_EmptyFilter(t *testing.T) {
	got := injectMatchersToQuery("cpu_usage", []string{})
	if got != "cpu_usage" {
		t.Errorf("expected no change, got %q", got)
	}
}

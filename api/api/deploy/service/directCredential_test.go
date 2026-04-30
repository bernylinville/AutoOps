package service

import (
	"strings"
	"testing"
)

func TestIsDirectInClusterRef(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want bool
	}{
		{name: "exact", ref: "in-cluster", want: true},
		{name: "trimmed", ref: "  in-cluster  ", want: true},
		{name: "case insensitive", ref: "IN-CLUSTER", want: true},
		{name: "account ref", ref: "account:dev", want: false},
		{name: "empty", ref: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDirectInClusterRef(tt.ref); got != tt.want {
				t.Fatalf("isDirectInClusterRef(%q) = %v, want %v", tt.ref, got, tt.want)
			}
		})
	}
}

func TestResolveDirectKubeconfigMentionsSupportedRefs(t *testing.T) {
	_, err := ResolveDirectKubeconfig("kubeconfig-secret")
	if err == nil {
		t.Fatal("expected unsupported ref error")
	}

	msg := err.Error()
	for _, want := range []string{"account:<id|alias>", "in-cluster"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected error %q to mention %q", msg, want)
		}
	}
}

func TestDirectRESTConfigFromRefUsesInClusterPath(t *testing.T) {
	_, err := directRESTConfigFromRef("in-cluster")
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "加载 in-cluster kubeconfig 失败") {
		t.Fatalf("expected in-cluster load error, got %q", err.Error())
	}
}

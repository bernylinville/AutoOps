package service

import (
	"testing"

	deploymodel "dodevops-api/api/deploy/model"
)

func TestBuildAccessInfoNilRequest(t *testing.T) {
	if got := buildAccessInfo(nil); got != nil {
		t.Fatalf("expected nil for nil request, got %+v", got)
	}
}

func TestBuildAccessInfoBasicFields(t *testing.T) {
	req := &deploymodel.DeployRequest{
		Image:       "registry.example.com/myapp:v1.2.3",
		Namespace:   "ao-direct-myapp",
		ReleaseName: "myapp",
	}
	info := buildAccessInfo(req)
	if info == nil {
		t.Fatal("expected non-nil AccessInfo")
	}
	if info.Image != req.Image {
		t.Errorf("Image: got %q, want %q", info.Image, req.Image)
	}
	if info.Namespace != req.Namespace {
		t.Errorf("Namespace: got %q, want %q", info.Namespace, req.Namespace)
	}
	if info.ReleaseName != req.ReleaseName {
		t.Errorf("ReleaseName: got %q, want %q", info.ReleaseName, req.ReleaseName)
	}
	if info.ServiceEnabled {
		t.Error("ServiceEnabled should be false when not set")
	}
	if info.ServiceType != "" || info.ServicePort != 0 || info.TargetPort != 0 {
		t.Error("service fields should be empty when ServiceEnabled is false")
	}
}

func TestBuildAccessInfoWithService(t *testing.T) {
	req := &deploymodel.DeployRequest{
		Image:          "registry.example.com/api:latest",
		Namespace:      "ao-direct-api",
		ReleaseName:    "api",
		ServiceEnabled: true,
		ServiceType:    "NodePort",
		ServicePort:    80,
		TargetPort:     8080,
	}
	info := buildAccessInfo(req)
	if info == nil {
		t.Fatal("expected non-nil AccessInfo")
	}
	if !info.ServiceEnabled {
		t.Error("ServiceEnabled should be true")
	}
	if info.ServiceType != "NodePort" {
		t.Errorf("ServiceType: got %q, want %q", info.ServiceType, "NodePort")
	}
	if info.ServicePort != 80 {
		t.Errorf("ServicePort: got %d, want 80", info.ServicePort)
	}
	if info.TargetPort != 8080 {
		t.Errorf("TargetPort: got %d, want 8080", info.TargetPort)
	}
}

func TestBuildAccessInfoServiceDisabledOmitsServiceFields(t *testing.T) {
	req := &deploymodel.DeployRequest{
		Image:          "registry.example.com/app:v1",
		Namespace:      "ao-direct-app",
		ReleaseName:    "app",
		ServiceEnabled: false,
		ServiceType:    "ClusterIP", // set but should be ignored
		ServicePort:    9090,
	}
	info := buildAccessInfo(req)
	if info.ServiceEnabled {
		t.Error("ServiceEnabled should be false")
	}
	if info.ServiceType != "" {
		t.Errorf("ServiceType should be empty when ServiceEnabled=false, got %q", info.ServiceType)
	}
	if info.ServicePort != 0 {
		t.Errorf("ServicePort should be 0 when ServiceEnabled=false, got %d", info.ServicePort)
	}
}

func TestBuildAccessInfoServiceEnabledAppliesDefaults(t *testing.T) {
	req := &deploymodel.DeployRequest{
		Image:          "registry.example.com/app:v1",
		Namespace:      "ao-direct-app",
		ReleaseName:    "app",
		ServiceEnabled: true,
		ServiceType:    "",
		ServicePort:    0,
		TargetPort:     0,
	}
	info := buildAccessInfo(req)
	if info.ServiceType != "ClusterIP" {
		t.Errorf("ServiceType: got %q, want \"ClusterIP\"", info.ServiceType)
	}
	if info.ServicePort != 80 {
		t.Errorf("ServicePort: got %d, want 80", info.ServicePort)
	}
	if info.TargetPort != 80 {
		t.Errorf("TargetPort: got %d, want 80 (= ServicePort default)", info.TargetPort)
	}
}

func TestExecErrorMessageNilRecord(t *testing.T) {
	if got := execErrorMessage(nil); got != "" {
		t.Errorf("expected empty string for nil record, got %q", got)
	}
}

func TestExecErrorMessageEmptyDetailJSON(t *testing.T) {
	rec := &deploymodel.ExecutionRecord{DetailJSON: ""}
	if got := execErrorMessage(rec); got != "" {
		t.Errorf("expected empty string for empty DetailJSON, got %q", got)
	}
}

func TestExecErrorMessageNoErrorKey(t *testing.T) {
	rec := &deploymodel.ExecutionRecord{
		DetailJSON: `{"comment":"auto","note":"direct resources applied"}`,
	}
	if got := execErrorMessage(rec); got != "" {
		t.Errorf("expected empty string when no error key, got %q", got)
	}
}

func TestExecErrorMessageWithError(t *testing.T) {
	rec := &deploymodel.ExecutionRecord{
		DetailJSON: `{"comment":"auto","note":"execution failed","error":"context deadline exceeded"}`,
	}
	got := execErrorMessage(rec)
	if got != "context deadline exceeded" {
		t.Errorf("got %q, want %q", got, "context deadline exceeded")
	}
}

func TestExecErrorMessageInvalidJSON(t *testing.T) {
	rec := &deploymodel.ExecutionRecord{DetailJSON: `not-json`}
	if got := execErrorMessage(rec); got != "" {
		t.Errorf("expected empty string for invalid JSON, got %q", got)
	}
}

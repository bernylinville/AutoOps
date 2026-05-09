package service

import (
	"strings"
	"testing"

	deploymodel "dodevops-api/api/deploy/model"
)

func TestCanTakeOverResourceOwnerAllowsSameAppEnvSlice(t *testing.T) {
	appID := uint(42)
	existingReq := &deploymodel.DeployRequest{
		Mode:            deploymodel.DeployModeDirect,
		ApplicationID:   &appID,
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		ReleaseName:     "java-demo",
		EnvJSON:         `{"name":"test","env":"test"}`,
	}
	currentReq := &deploymodel.DeployRequest{
		Mode:            deploymodel.DeployModeDirect,
		ApplicationID:   &appID,
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		ReleaseName:     "java-demo",
		EnvJSON:         `{"SPRING_PROFILES_ACTIVE":"test"}`,
	}
	existingOwner := &deploymodel.ResourceOwner{
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		Kind:            "Deployment",
		Name:            "java-demo",
		OwnerSystem:     deploymodel.ResourceOwnerSystemDirect,
	}
	desiredOwner := &deploymodel.ResourceOwner{
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		Kind:            "Deployment",
		Name:            "java-demo",
		OwnerSystem:     deploymodel.ResourceOwnerSystemDirect,
	}

	if !canTakeOverResourceOwner(existingReq, currentReq, existingOwner, desiredOwner) {
		t.Fatal("expected same app/env/namespace/release to allow takeover")
	}
}

func TestCanTakeOverResourceOwnerRejectsDifferentEnv(t *testing.T) {
	appID := uint(42)
	existingReq := &deploymodel.DeployRequest{
		Mode:            deploymodel.DeployModeDirect,
		ApplicationID:   &appID,
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		ReleaseName:     "java-demo",
		EnvJSON:         `{"name":"dev"}`,
	}
	currentReq := &deploymodel.DeployRequest{
		Mode:            deploymodel.DeployModeDirect,
		ApplicationID:   &appID,
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		ReleaseName:     "java-demo",
		EnvJSON:         `{"name":"test"}`,
	}
	existingOwner := &deploymodel.ResourceOwner{
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		Kind:            "Deployment",
		Name:            "java-demo",
		OwnerSystem:     deploymodel.ResourceOwnerSystemDirect,
	}
	desiredOwner := &deploymodel.ResourceOwner{
		ClusterTargetID: 2,
		Namespace:       "ao-direct-java-demo-test",
		Kind:            "Deployment",
		Name:            "java-demo",
		OwnerSystem:     deploymodel.ResourceOwnerSystemDirect,
	}

	if canTakeOverResourceOwner(existingReq, currentReq, existingOwner, desiredOwner) {
		t.Fatal("expected different logical env to block takeover")
	}
}

func TestDeployRequestLogicalEnv(t *testing.T) {
	if got := deployRequestLogicalEnv(&deploymodel.DeployRequest{EnvJSON: `{"env":"test"}`}); got != "test" {
		t.Fatalf("deployRequestLogicalEnv() = %q, want %q", got, "test")
	}
	if got := deployRequestLogicalEnv(&deploymodel.DeployRequest{EnvJSON: `{"SPRING_PROFILES_ACTIVE":"DEV"}`}); got != "dev" {
		t.Fatalf("deployRequestLogicalEnv() should normalize profile env, got %q", got)
	}
	if got := deployRequestLogicalEnv(&deploymodel.DeployRequest{EnvJSON: `not-json`}); got != "" {
		t.Fatalf("deployRequestLogicalEnv() should ignore invalid JSON, got %q", got)
	}
}

func TestBuildAgentPipelineInfo(t *testing.T) {
	run := &deploymodel.PipelineRun{
		Status:             deploymodel.PipelineStatusScanning,
		CurrentStage:       deploymodel.PipelineStageScan,
		GitRef:             "main",
		JenkinsQueueID:     7,
		JenkinsBuildNumber: 28,
		JenkinsBuildURL:    "http://10.0.17.204/job/java-demo-build/28/",
		HarborProject:      "java-demo",
		HarborRepository:   "java-demo",
		ArtifactTag:        "main-28",
		PlannedImageRef:    "10.0.17.205:80/java-demo/java-demo:main-28",
		LastError:          "",
	}
	stages := []deploymodel.PipelineStageRecord{
		{Stage: deploymodel.PipelineStageBuild, Status: deploymodel.PipelineStageStatusSucceeded},
		{Stage: deploymodel.PipelineStageScan, Status: deploymodel.PipelineStageStatusRunning},
	}

	info := buildAgentPipelineInfo(run, stages)
	if info == nil {
		t.Fatal("expected non-nil pipeline info")
	}
	if info["currentStage"] != deploymodel.PipelineStageScan || info["scanStatus"] != deploymodel.PipelineStageStatusRunning {
		t.Fatalf("unexpected stage summary: %+v", info)
	}
	if info["imageRef"] != run.PlannedImageRef {
		t.Fatalf("expected imageRef to fall back to planned image, got %+v", info["imageRef"])
	}
	stageSummaries, ok := info["stages"].([]map[string]interface{})
	if !ok || len(stageSummaries) != 2 {
		t.Fatalf("expected simplified stage summaries, got %#v", info["stages"])
	}
}

func TestBuildAgentPipelineInfoNilRun(t *testing.T) {
	if got := buildAgentPipelineInfo(nil, nil); got != nil {
		t.Fatalf("expected nil pipeline info for nil run, got %#v", got)
	}
}

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

func TestBuildAccessInfoIncludesDirectApplyAccessURLs(t *testing.T) {
	req := &deploymodel.DeployRequest{
		Image:          "10.0.17.205:80/java-demo/java-demo:v1",
		Namespace:      "ao-direct-java-demo",
		ReleaseName:    "java-demo",
		ServiceEnabled: true,
		ServiceType:    "NodePort",
		ServicePort:    80,
		TargetPort:     8080,
	}
	applyResult := &DirectApplyResult{
		NodeIPs:    []string{"10.0.17.40"},
		AccessURLs: []string{"http://10.0.17.40:30278/"},
		Service: &DirectServiceResult{
			Name: "java-demo",
			Type: "NodePort",
			Ports: []DirectServicePort{
				{Port: 80, TargetPort: "8080", NodePort: 30278},
			},
		},
	}

	info := buildAccessInfo(req, applyResult)
	if info.NodeIP != "10.0.17.40" || info.NodePort != 30278 {
		t.Fatalf("node access not mapped: %+v", info)
	}
	if len(info.AccessURLs) != 1 || info.AccessURLs[0] != "http://10.0.17.40:30278/" {
		t.Fatalf("access urls not mapped: %+v", info.AccessURLs)
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

func TestDirectApplyResultFromExecutionParsesEncodedResult(t *testing.T) {
	applyResult := &DirectApplyResult{
		Namespace:  "ao-direct-java-demo",
		AccessURLs: []string{"http://10.0.17.40:30278/"},
		Service: &DirectServiceResult{
			Name: "java-demo",
			Type: "NodePort",
			Ports: []DirectServicePort{
				{Port: 80, TargetPort: "8080", NodePort: 30278},
			},
		},
	}
	rec := &deploymodel.ExecutionRecord{
		DetailJSON: executionDetailWithDirectApplyResult("ok", "preview", applyResult),
	}

	got := directApplyResultFromExecution(rec)
	if got == nil {
		t.Fatal("expected DirectApplyResult")
	}
	if len(got.AccessURLs) != 1 || got.AccessURLs[0] != "http://10.0.17.40:30278/" {
		t.Fatalf("unexpected access urls: %+v", got)
	}
	if got.Service == nil || len(got.Service.Ports) != 1 || got.Service.Ports[0].NodePort != 30278 {
		t.Fatalf("unexpected service result: %+v", got.Service)
	}
}

func TestApprovalImageFormValueUsesExistingImage(t *testing.T) {
	req := &deploymodel.DeployRequest{
		WorkflowKind: deploymodel.WorkflowKindBuildDeploy,
		Image:        "10.0.17.205:80/java-demo/java-demo:v1",
	}

	if got := approvalImageFormValue(req, nil); got != req.Image {
		t.Fatalf("approvalImageFormValue() = %q, want %q", got, req.Image)
	}
}

func TestApprovalImageFormValueDescribesPendingBuildImage(t *testing.T) {
	req := &deploymodel.DeployRequest{
		WorkflowKind: deploymodel.WorkflowKindBuildDeploy,
	}
	run := &deploymodel.PipelineRun{
		GitRef:           "main",
		HarborProject:    "java-demo",
		HarborRepository: "java-demo",
	}

	got := approvalImageFormValue(req, run)
	if got == "" {
		t.Fatal("expected non-empty placeholder for build-deploy approval image field")
	}
	for _, want := range []string{"构建后由 Jenkins 生成", "Harbor: java-demo/java-demo", "Git: main"} {
		if !strings.Contains(got, want) {
			t.Fatalf("approval image placeholder %q missing %q", got, want)
		}
	}
}

func TestApprovalImageFormValueDescribesPendingBuildImageWithoutPipelineMetadata(t *testing.T) {
	req := &deploymodel.DeployRequest{WorkflowKind: deploymodel.WorkflowKindBuildDeploy}
	if got := approvalImageFormValue(req, nil); got != "构建后由 Jenkins 生成" {
		t.Fatalf("approvalImageFormValue() = %q, want pending build placeholder", got)
	}
}

func TestApprovalImageFormValueLeavesEmptyNonBuildDeployImageEmpty(t *testing.T) {
	req := &deploymodel.DeployRequest{WorkflowKind: deploymodel.WorkflowKindDeployOnly}
	if got := approvalImageFormValue(req, nil); got != "" {
		t.Fatalf("approvalImageFormValue() = %q, want empty", got)
	}
}

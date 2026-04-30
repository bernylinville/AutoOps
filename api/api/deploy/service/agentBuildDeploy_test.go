package service

import (
	"testing"

	appmodel "dodevops-api/api/app/model"
	deploymodel "dodevops-api/api/deploy/model"
)

func TestBuildAgentDeployRequestFromProfileMapsProfileAndOverrides(t *testing.T) {
	appID := uint(42)
	profile := &appmodel.AppDeployProfile{
		Env:              appmodel.DeployProfileEnvDev,
		ClusterTargetID:  7,
		Namespace:        "ao-direct-java-demo",
		ReleaseName:      "java-demo",
		ResourceType:     deploymodel.DeployResourceTypeDeployment,
		JenkinsServerID:  3,
		JenkinsJobName:   "java-demo-build",
		HarborServerID:   4,
		HarborProject:    "java-demo",
		HarborRepository: "java-demo",
		DefaultGitRef:    "main",
		ApproverAdminID:  9,
		Replicas:         2,
		ServiceEnabled:   true,
		ServiceType:      "ClusterIP",
		ServicePort:      80,
		TargetPort:       8080,
		EnvJSON:          `{"JAVA_OPTS":"-Xmx512m"}`,
		ResourcesJSON:    `{"limits":{"cpu":"1"}}`,
		BuildParamsJSON:  `{"MAVEN_PROFILE":"dev"}`,
		ScanPolicyJSON:   `{"blockOnCritical":true}`,
	}
	req := &deploymodel.CreateAgentBuildDeployRequest{
		RequesterExternalType: "dingtalk",
		RequesterExternalID:   "ding-user-1",
		RequesterDisplayName:  "Alice",
		GitRef:                "feature/demo",
		Reason:                "deploy demo",
		BuildParams:           map[string]interface{}{"MAVEN_PROFILE": "fast"},
		ChatContext:           map[string]interface{}{"chatId": "cid"},
	}

	got := buildAgentDeployRequestFromProfile(appID, profile, req)

	if got.Mode != deploymodel.DeployModeDirect || got.WorkflowKind != deploymodel.WorkflowKindBuildDeploy {
		t.Fatalf("unexpected mode/workflow: %s/%s", got.Mode, got.WorkflowKind)
	}
	if got.ApplicationID == nil || *got.ApplicationID != appID {
		t.Fatalf("application id not mapped: %#v", got.ApplicationID)
	}
	if got.ApproverAdminID == nil || *got.ApproverAdminID != profile.ApproverAdminID {
		t.Fatalf("approver not mapped: %#v", got.ApproverAdminID)
	}
	if got.JenkinsServerID == nil || *got.JenkinsServerID != profile.JenkinsServerID || got.JenkinsJobName != profile.JenkinsJobName {
		t.Fatalf("jenkins not mapped: server=%#v job=%s", got.JenkinsServerID, got.JenkinsJobName)
	}
	if got.HarborServerID == nil || *got.HarborServerID != profile.HarborServerID || got.HarborProject != profile.HarborProject || got.HarborRepository != profile.HarborRepository {
		t.Fatalf("harbor not mapped: server=%#v project=%s repo=%s", got.HarborServerID, got.HarborProject, got.HarborRepository)
	}
	if got.GitRef != "feature/demo" || got.BuildParams["GIT_REF"] != "feature/demo" {
		t.Fatalf("git ref override not applied: gitRef=%s params=%v", got.GitRef, got.BuildParams)
	}
	if got.BuildParams["MAVEN_PROFILE"] != "fast" {
		t.Fatalf("request build params should override profile params: %v", got.BuildParams)
	}
	if got.Env["name"] != appmodel.DeployProfileEnvDev || got.Env["JAVA_OPTS"] != "-Xmx512m" {
		t.Fatalf("env not merged: %v", got.Env)
	}
	if got.ScanPolicy["blockOnCritical"] != true {
		t.Fatalf("scan policy not mapped: %v", got.ScanPolicy)
	}
}

func TestDefaultAgentBuildGitRef(t *testing.T) {
	cases := []struct {
		name       string
		requestRef string
		profileRef string
		want       string
	}{
		{name: "request wins", requestRef: " feature/x ", profileRef: "main", want: "feature/x"},
		{name: "profile fallback", profileRef: " develop ", want: "develop"},
		{name: "main fallback", want: "main"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := defaultAgentBuildGitRef(tt.requestRef, tt.profileRef); got != tt.want {
				t.Fatalf("defaultAgentBuildGitRef() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateDeployRequestFromAgentPreservesBuildDeployFields(t *testing.T) {
	jenkinsID := uint(11)
	harborID := uint(12)
	appID := uint(13)
	approverID := uint(14)
	req := &deploymodel.CreateAgentDeployRequest{
		Mode:             deploymodel.DeployModeDirect,
		WorkflowKind:     deploymodel.WorkflowKindBuildDeploy,
		ResourceType:     deploymodel.DeployResourceTypeDeployment,
		ClusterTargetID:  15,
		ApplicationID:    &appID,
		ApproverAdminID:  &approverID,
		GitRef:           "main",
		BuildParams:      map[string]interface{}{"ENV": "dev"},
		JenkinsServerID:  &jenkinsID,
		JenkinsJobName:   "java-demo-build",
		HarborServerID:   &harborID,
		HarborProject:    "java-demo",
		HarborRepository: "java-demo",
		ArtifactTag:      "main-1",
		ScanPolicy:       map[string]interface{}{"blockOnCritical": true},
	}

	got := createDeployRequestFromAgent(req)

	if got.JenkinsServerID == nil || *got.JenkinsServerID != jenkinsID || got.JenkinsJobName != req.JenkinsJobName {
		t.Fatalf("jenkins fields dropped: server=%#v job=%q", got.JenkinsServerID, got.JenkinsJobName)
	}
	if got.HarborServerID == nil || *got.HarborServerID != harborID || got.HarborProject != req.HarborProject || got.HarborRepository != req.HarborRepository {
		t.Fatalf("harbor fields dropped: server=%#v project=%q repo=%q", got.HarborServerID, got.HarborProject, got.HarborRepository)
	}
	if got.GitRef != req.GitRef || got.BuildParams["ENV"] != "dev" || got.ScanPolicy["blockOnCritical"] != true {
		t.Fatalf("build fields dropped: gitRef=%q build=%v scan=%v", got.GitRef, got.BuildParams, got.ScanPolicy)
	}
}

func TestCloneCreateDeployRequestPreservesBuildDeployFields(t *testing.T) {
	jenkinsID := uint(21)
	harborID := uint(22)
	req := &deploymodel.CreateDeployRequest{
		WorkflowKind:     deploymodel.WorkflowKindBuildDeploy,
		GitRef:           "release/v1",
		BuildParams:      map[string]interface{}{"SKIP_TESTS": false},
		JenkinsServerID:  &jenkinsID,
		JenkinsJobName:   "release-job",
		HarborServerID:   &harborID,
		HarborProject:    "library",
		HarborRepository: "java-demo",
		ArtifactTag:      "v1",
		ScanPolicy:       map[string]interface{}{"maxSeverity": "high"},
	}

	got := cloneCreateDeployRequest(req)

	if got.JenkinsServerID == nil || *got.JenkinsServerID != jenkinsID || got.JenkinsJobName != req.JenkinsJobName {
		t.Fatalf("jenkins fields dropped: server=%#v job=%q", got.JenkinsServerID, got.JenkinsJobName)
	}
	if got.HarborServerID == nil || *got.HarborServerID != harborID || got.HarborProject != req.HarborProject || got.HarborRepository != req.HarborRepository {
		t.Fatalf("harbor fields dropped: server=%#v project=%q repo=%q", got.HarborServerID, got.HarborProject, got.HarborRepository)
	}
	if got.GitRef != req.GitRef || got.BuildParams["SKIP_TESTS"] != false || got.ScanPolicy["maxSeverity"] != "high" {
		t.Fatalf("build fields dropped: gitRef=%q build=%v scan=%v", got.GitRef, got.BuildParams, got.ScanPolicy)
	}
}

func TestParseAgentGitRepoURL(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		wantHost string
		wantPath string
		wantRepo string
	}{
		{
			name:     "scp style ssh",
			raw:      "git@gayhub.seeingtv.com:demo/springboot-demo.git",
			wantHost: "gayhub.seeingtv.com",
			wantPath: "demo/springboot-demo",
			wantRepo: "springboot-demo",
		},
		{
			name:     "https with nested group",
			raw:      "https://gayhub.seeingtv.com/platform/demo/springboot-demo.git",
			wantHost: "gayhub.seeingtv.com",
			wantPath: "platform/demo/springboot-demo",
			wantRepo: "springboot-demo",
		},
		{
			name:     "ssh url",
			raw:      "ssh://git@gayhub.seeingtv.com/demo/springboot-demo.git",
			wantHost: "gayhub.seeingtv.com",
			wantPath: "demo/springboot-demo",
			wantRepo: "springboot-demo",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAgentGitRepoURL(tt.raw)
			if err != nil {
				t.Fatalf("parseAgentGitRepoURL() error = %v", err)
			}
			if got.Host != tt.wantHost || got.Path != tt.wantPath || got.RepoName != tt.wantRepo {
				t.Fatalf("unexpected repo parse: %+v", got)
			}
		})
	}
}

func TestParseAgentGitRepoURLRejectsInvalidInputs(t *testing.T) {
	cases := []string{
		"",
		"not-a-git-url",
		"https://user:secret@gayhub.seeingtv.com/demo/springboot-demo.git",
		"ftp://gayhub.seeingtv.com/demo/springboot-demo.git",
		"https://gayhub.seeingtv.com/springboot-demo.git",
	}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			if _, err := parseAgentGitRepoURL(raw); err == nil {
				t.Fatalf("expected parse error for %q", raw)
			}
		})
	}
}

func TestAgentGitHostAllowlistNormalizesHost(t *testing.T) {
	if !isAgentGitHostAllowed("gayhub.seeingtv.com", []string{"https://gayhub.seeingtv.com"}) {
		t.Fatal("expected URL allowlist entry to match bare host")
	}
	if !isAgentGitHostAllowed("gayhub.seeingtv.com:443", []string{"gayhub.seeingtv.com"}) {
		t.Fatal("expected host:port to match bare host")
	}
	if isAgentGitHostAllowed("evil.example.com", []string{"gayhub.seeingtv.com"}) {
		t.Fatal("unexpected allowlist match")
	}
}

func TestAgentServiceTypeFromExposureMode(t *testing.T) {
	cases := []struct {
		mode string
		want string
	}{
		{mode: "", want: "ClusterIP"},
		{mode: "clusterip", want: "ClusterIP"},
		{mode: "nodeport", want: "NodePort"},
		{mode: "NodePort", want: "NodePort"},
	}
	for _, tt := range cases {
		t.Run(tt.mode, func(t *testing.T) {
			got, err := agentServiceTypeFromExposureMode(tt.mode)
			if err != nil {
				t.Fatalf("agentServiceTypeFromExposureMode() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("agentServiceTypeFromExposureMode() = %q, want %q", got, tt.want)
			}
		})
	}
	for _, unsupported := range []string{"gateway", "metallb"} {
		t.Run(unsupported, func(t *testing.T) {
			if _, err := agentServiceTypeFromExposureMode(unsupported); err == nil {
				t.Fatalf("expected unsupported exposure mode %q to fail", unsupported)
			}
		})
	}
}

func TestBuildAgentOnboardingProfileDefaults(t *testing.T) {
	defaults := &agentProjectOnboardingDefaults{
		SharedJenkinsJobName:   "autoops-springboot-build",
		DefaultJenkinsServerID: 3,
		DefaultHarborServerID:  4,
		DefaultHarborProject:   "library",
		DefaultApproverAdminID: 5,
		NamespacePrefix:        "ao-direct",
		DefaultServicePort:     80,
		DefaultTargetPort:      8080,
	}
	app := &appmodel.Application{ID: 42, Code: "springboot-demo"}
	repo := agentGitRepo{RawURL: "git@gayhub.seeingtv.com:demo/springboot-demo.git", Host: "gayhub.seeingtv.com", Path: "demo/springboot-demo", RepoName: "springboot-demo"}

	got := buildAgentOnboardingProfileDefaults(defaults, app, repo, appmodel.DeployProfileEnvDev, 7, "NodePort")

	if got.Namespace != "ao-direct-springboot-demo" || got.ReleaseName != "springboot-demo" {
		t.Fatalf("namespace/release defaults wrong: namespace=%q release=%q", got.Namespace, got.ReleaseName)
	}
	if got.ServiceType != "NodePort" || got.ServicePort != 80 || got.TargetPort != 8080 {
		t.Fatalf("service defaults wrong: type=%q port=%d target=%d", got.ServiceType, got.ServicePort, got.TargetPort)
	}
	if got.JenkinsJobName != "autoops-springboot-build" || got.HarborProject != "library" || got.HarborRepository != "springboot-demo" {
		t.Fatalf("ci registry defaults wrong: job=%q harbor=%s/%s", got.JenkinsJobName, got.HarborProject, got.HarborRepository)
	}
	resources := mergeProfileMaps(got.ResourcesJSON, nil)
	limits, ok := resources["limits"].(map[string]interface{})
	if !ok || limits["memory"] != "768Mi" || limits["cpu"] != "1000m" {
		t.Fatalf("resource limits should default to Spring Boot safe values, got %v", resources)
	}
	requests, ok := resources["requests"].(map[string]interface{})
	if !ok || requests["memory"] != "256Mi" || requests["cpu"] != "100m" {
		t.Fatalf("resource requests should default to Spring Boot safe values, got %v", resources)
	}
	params := mergeProfileMaps(got.BuildParamsJSON, nil)
	if params["GIT_URL"] != repo.RawURL || params["APPLICATION_CODE"] != app.Code || params["HARBOR_PROJECT"] != "library" || params["HARBOR_REPOSITORY"] != app.Code {
		t.Fatalf("build params missing required values: %v", params)
	}
}

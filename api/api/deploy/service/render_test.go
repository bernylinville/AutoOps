package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	deploymodel "dodevops-api/api/deploy/model"
	commonconfig "dodevops-api/common/config"

	appsv1 "k8s.io/api/apps/v1"
)

func TestRenderDirectManifestAddsOwnerLabelsAndTTL(t *testing.T) {
	ttl := 24
	req := &deploymodel.DeployRequest{
		RequestNo:      "DRTEST001",
		Mode:           deploymodel.DeployModeDirect,
		ResourceType:   deploymodel.DeployResourceTypeDeployment,
		ReleaseName:    "nginx-demo",
		Namespace:      "ao-direct-nginx-demo",
		Image:          "nginx:1.27.4-alpine",
		Replicas:       1,
		ServiceEnabled: true,
		ServiceType:    "ClusterIP",
		ServicePort:    80,
		TargetPort:     80,
		TTLHours:       &ttl,
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		t.Fatalf("RenderDirectManifest() error = %v", err)
	}

	if !strings.Contains(rendered.YAML, "app.kubernetes.io/managed-by: autoops") {
		t.Fatalf("expected managed-by label in manifest, got:\n%s", rendered.YAML)
	}
	if !strings.Contains(rendered.YAML, "autoops.io/owner-system: direct") {
		t.Fatalf("expected owner-system label in manifest, got:\n%s", rendered.YAML)
	}
	if !strings.Contains(rendered.YAML, "autoops.io/deploy-mode: direct") {
		t.Fatalf("expected deploy-mode label in manifest, got:\n%s", rendered.YAML)
	}
	if !strings.Contains(rendered.YAML, "autoops.io/request-id: DRTEST001") {
		t.Fatalf("expected request-id label in manifest, got:\n%s", rendered.YAML)
	}
	if !strings.Contains(rendered.YAML, "autoops.io/ttl-expire-at:") {
		t.Fatalf("expected ttl annotation in manifest, got:\n%s", rendered.YAML)
	}
	if strings.Contains(rendered.YAML, "imagePullSecrets:") {
		t.Fatalf("did not expect imagePullSecrets for public image, got:\n%s", rendered.YAML)
	}
}

func TestRenderDirectManifestAddsVolcesImagePullSecret(t *testing.T) {
	req := &deploymodel.DeployRequest{
		RequestNo:      "DRTEST004",
		Mode:           deploymodel.DeployModeDirect,
		ResourceType:   deploymodel.DeployResourceTypeDeployment,
		ReleaseName:    "nginx-demo",
		Namespace:      "ao-direct-nginx-demo",
		Image:          "pukka-all-images-cn-shanghai.cr.volces.com/proxy/nginx:1.27.4-alpine",
		Replicas:       1,
		ServiceEnabled: false,
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		t.Fatalf("RenderDirectManifest() error = %v", err)
	}

	if !strings.Contains(rendered.YAML, "imagePullSecrets:") {
		t.Fatalf("expected imagePullSecrets for volces image, got:\n%s", rendered.YAML)
	}
	if !strings.Contains(rendered.YAML, "- name: volces-registry") {
		t.Fatalf("expected volces-registry pull secret, got:\n%s", rendered.YAML)
	}
}

func TestRenderDirectManifestAddsHarborImagePullSecret(t *testing.T) {
	req := &deploymodel.DeployRequest{
		RequestNo:    "DRTEST005",
		Mode:         deploymodel.DeployModeDirect,
		ResourceType: deploymodel.DeployResourceTypeDeployment,
		ReleaseName:  "java-demo",
		Namespace:    "ao-direct-java-demo",
		Image:        "10.0.17.205:80/java-demo/java-demo:20260430040417-470cfcd",
		Replicas:     1,
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		t.Fatalf("RenderDirectManifest() error = %v", err)
	}
	if !strings.Contains(rendered.YAML, "- name: harbor-pull-secret") {
		t.Fatalf("expected harbor-pull-secret for Harbor image, got:\n%s", rendered.YAML)
	}
}

func TestRenderDirectManifestAppliesEnvAndResourceDefaults(t *testing.T) {
	req := &deploymodel.DeployRequest{
		RequestNo:      "DRTEST006",
		Mode:           deploymodel.DeployModeDirect,
		ResourceType:   deploymodel.DeployResourceTypeDeployment,
		ReleaseName:    "java-demo",
		Namespace:      "ao-direct-java-demo",
		Image:          "java-demo:latest",
		Replicas:       1,
		EnvJSON:        `{"SPRING_PROFILES_ACTIVE":"dev","JAVA_TOOL_OPTIONS":"-XX:MaxRAMPercentage=75"}`,
		ResourcesJSON:  `{}`,
		ServiceEnabled: true,
		ServicePort:    80,
		TargetPort:     8080,
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		t.Fatalf("RenderDirectManifest() error = %v", err)
	}
	deployment, ok := rendered.Objects[0].(*appsv1.Deployment)
	if !ok {
		t.Fatalf("expected deployment object, got %T", rendered.Objects[0])
	}
	container := deployment.Spec.Template.Spec.Containers[0]
	if len(container.Env) != 2 {
		t.Fatalf("expected env vars from EnvJSON, got %#v", container.Env)
	}
	if got := container.Resources.Requests.Memory().String(); got != "256Mi" {
		t.Fatalf("default memory request = %s, want 256Mi", got)
	}
	if got := container.Resources.Limits.Cpu().String(); got != "1" {
		t.Fatalf("default cpu limit = %s, want 1", got)
	}
	if got := container.Resources.Limits.Memory().String(); got != "768Mi" {
		t.Fatalf("default memory limit = %s, want 768Mi", got)
	}
}

func TestRenderDirectManifestOverridesResourceJSON(t *testing.T) {
	req := &deploymodel.DeployRequest{
		RequestNo:     "DRTEST007",
		Mode:          deploymodel.DeployModeDirect,
		ResourceType:  deploymodel.DeployResourceTypeDeployment,
		ReleaseName:   "java-demo",
		Namespace:     "ao-direct-java-demo",
		Image:         "java-demo:latest",
		Replicas:      1,
		ResourcesJSON: `{"requests":{"memory":"512Mi"},"limits":{"cpu":"1500m"}}`,
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		t.Fatalf("RenderDirectManifest() error = %v", err)
	}
	deployment := rendered.Objects[0].(*appsv1.Deployment)
	container := deployment.Spec.Template.Spec.Containers[0]
	if got := container.Resources.Requests.Memory().String(); got != "512Mi" {
		t.Fatalf("memory request = %s, want 512Mi", got)
	}
	if got := container.Resources.Limits.Cpu().String(); got != "1500m" {
		t.Fatalf("cpu limit = %s, want 1500m", got)
	}
	if got := container.Resources.Requests.Cpu().String(); got != "100m" {
		t.Fatalf("default cpu request = %s, want 100m", got)
	}
}

func TestRenderGitOpsReleaseFileIncludesOwnerMetadata(t *testing.T) {
	req := &deploymodel.DeployRequest{
		RequestNo:      "DRTEST002",
		Mode:           deploymodel.DeployModeGitOps,
		ResourceType:   deploymodel.DeployResourceTypeDeployment,
		ReleaseName:    "nginx-gitops",
		Namespace:      "ao-gitops-nginx-gitops",
		Image:          "nginx:1.27.4-alpine",
		Replicas:       2,
		ServiceEnabled: true,
		ServiceType:    "ClusterIP",
		ServicePort:    80,
		TargetPort:     80,
	}

	filePath, content, err := RenderGitOpsReleaseFile(req, "pukka-devtest")
	if err != nil {
		t.Fatalf("RenderGitOpsReleaseFile() error = %v", err)
	}

	expectedPath := "apps/autoops-managed-releases/releases/nginx-gitops.yaml"
	if filePath != expectedPath {
		t.Fatalf("expected file path %q, got %q", expectedPath, filePath)
	}
	for _, want := range []string{
		"autoops.io/owner-system: gitops",
		"autoops.io/deploy-mode: gitops",
		"autoops.io/request-id: DRTEST002",
		"clusterTarget: pukka-devtest",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in release file, got:\n%s", want, content)
		}
	}
}

func TestWriteAndCommitGitOpsReleaseToWorkingTree(t *testing.T) {
	repoDir := t.TempDir()
	mustWriteFile(t, filepath.Join(repoDir, "apps/autoops-managed-releases/Chart.yaml"), "apiVersion: v2\nname: test\n")
	mustWriteFile(t, filepath.Join(repoDir, "argocd-apps/templates/autoops-managed-releases.yaml"), "kind: Application\n")
	if err := os.MkdirAll(filepath.Join(repoDir, "apps/autoops-managed-releases/releases"), 0o755); err != nil {
		t.Fatalf("mkdir releases dir: %v", err)
	}

	runGitForTest(t, repoDir, "init")
	runGitForTest(t, repoDir, "checkout", "-b", "main")
	runGitForTest(t, repoDir, "config", "user.email", "test@example.com")
	runGitForTest(t, repoDir, "config", "user.name", "AutoOps Test")
	runGitForTest(t, repoDir, "add", ".")
	runGitForTest(t, repoDir, "commit", "-m", "initial")

	configPath := filepath.Join(t.TempDir(), "config.yaml")
	mustWriteFile(t, configPath, "integrations:\n  gitops:\n    local_checkout_path: "+repoDir+"\n")
	if err := commonconfig.LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	req := &deploymodel.DeployRequest{
		RequestNo:    "DRTEST003",
		Mode:         deploymodel.DeployModeGitOps,
		ResourceType: deploymodel.DeployResourceTypeDeployment,
		ReleaseName:  "writer-demo",
		Namespace:    "ao-gitops-writer-demo",
		Image:        "nginx:1.27.4-alpine",
	}

	writeResult, err := WriteGitOpsReleaseToWorkingTree(req, "pukka-devtest", "")
	if err != nil {
		t.Fatalf("WriteGitOpsReleaseToWorkingTree() error = %v", err)
	}
	if !writeResult.Written {
		t.Fatalf("expected write result to be written")
	}
	if _, err := os.Stat(filepath.Join(repoDir, filepath.FromSlash(writeResult.FilePath))); err != nil {
		t.Fatalf("expected written file to exist: %v", err)
	}

	commitResult, err := CommitGitOpsWorkingTree("main", writeResult.FilePath, req.RequestNo)
	if err != nil {
		t.Fatalf("CommitGitOpsWorkingTree() error = %v", err)
	}
	if commitResult.CommitSHA == "" {
		t.Fatalf("expected commit sha to be set")
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runGitForTest(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repoDir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}

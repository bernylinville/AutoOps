package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	deploymodel "dodevops-api/api/deploy/model"
	commonconfig "dodevops-api/common/config"
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

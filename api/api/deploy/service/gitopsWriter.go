package service

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dodevops-api/api/deploy/model"
	"dodevops-api/common/config"
)

type GitOpsWriteResult struct {
	RepoPath string `json:"repoPath"`
	FilePath string `json:"filePath"`
	Written  bool   `json:"written"`
	Message  string `json:"message"`
}

type GitOpsWorkingTreeValidationResult struct {
	Valid           bool   `json:"valid"`
	RepoPath        string `json:"repoPath"`
	ChartPath       string `json:"chartPath"`
	ApplicationPath string `json:"applicationPath"`
	ReleasesDir     string `json:"releasesDir"`
	Message         string `json:"message"`
}

type GitOpsRepoValidationResult struct {
	Valid    bool   `json:"valid"`
	RepoPath string `json:"repoPath"`
	Branch   string `json:"branch"`
	Message  string `json:"message"`
}

type GitOpsCommitResult struct {
	RepoPath  string `json:"repoPath"`
	Branch    string `json:"branch"`
	CommitSHA string `json:"commitSha"`
	Message   string `json:"message"`
}

type GitOpsPushResult struct {
	RepoPath string `json:"repoPath"`
	Branch   string `json:"branch"`
	Message  string `json:"message"`
}

type GitOpsDeleteResult struct {
	RepoPath  string `json:"repoPath"`
	FilePath  string `json:"filePath"`
	Branch    string `json:"branch"`
	CommitSHA string `json:"commitSha"`
	Message   string `json:"message"`
}

func WriteGitOpsReleaseToWorkingTree(req *model.DeployRequest, clusterTargetName string, releaseDir string) (*GitOpsWriteResult, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}
	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return nil, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	defaultFilePath, content, err := RenderGitOpsReleaseFile(req, clusterTargetName)
	if err != nil {
		return nil, err
	}

	filePath := defaultFilePath
	if strings.TrimSpace(releaseDir) != "" {
		filePath = filepath.ToSlash(filepath.Join(strings.TrimSpace(releaseDir), req.ReleaseName+".yaml"))
	}

	absolutePath := filepath.Join(repoRoot, filepath.FromSlash(filePath))
	cleanRoot := filepath.Clean(repoRoot) + string(filepath.Separator)
	cleanTarget := filepath.Clean(absolutePath)
	if !strings.HasPrefix(cleanTarget+string(filepath.Separator), cleanRoot) && cleanTarget != filepath.Clean(repoRoot) {
		return nil, fmt.Errorf("目标路径超出 gitops 工作树")
	}

	if err := os.MkdirAll(filepath.Dir(absolutePath), 0755); err != nil {
		return nil, fmt.Errorf("创建 gitops 目录失败: %v", err)
	}
	if err := os.WriteFile(absolutePath, []byte(content), 0644); err != nil {
		return nil, fmt.Errorf("写入 gitops release 文件失败: %v", err)
	}

	return &GitOpsWriteResult{
		RepoPath: repoRoot,
		FilePath: filePath,
		Written:  true,
		Message:  "GitOps release 已写入本地工作树，尚未 commit/push",
	}, nil
}

func ValidateGitOpsWorkingTree() (*GitOpsWorkingTreeValidationResult, error) {
	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return &GitOpsWorkingTreeValidationResult{Valid: false, Message: "integrations.gitops.local_checkout_path 未配置"}, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	chartPath := filepath.Join(repoRoot, "apps/autoops-managed-releases/Chart.yaml")
	applicationPath := filepath.Join(repoRoot, "argocd-apps/templates/autoops-managed-releases.yaml")
	releasesDir := filepath.Join(repoRoot, "apps/autoops-managed-releases/releases")

	missing := []string{}
	if !pathExists(repoRoot) {
		missing = append(missing, "repoRoot")
	}
	if !pathExists(chartPath) {
		missing = append(missing, "chartPath")
	}
	if !pathExists(applicationPath) {
		missing = append(missing, "applicationPath")
	}
	if !pathExists(releasesDir) {
		missing = append(missing, "releasesDir")
	}

	if len(missing) > 0 {
		return &GitOpsWorkingTreeValidationResult{
			Valid:           false,
			RepoPath:        repoRoot,
			ChartPath:       chartPath,
			ApplicationPath: applicationPath,
			ReleasesDir:     releasesDir,
			Message:         "GitOps 工作树不完整: " + strings.Join(missing, ", "),
		}, fmt.Errorf("GitOps 工作树不完整")
	}

	return &GitOpsWorkingTreeValidationResult{
		Valid:           true,
		RepoPath:        repoRoot,
		ChartPath:       chartPath,
		ApplicationPath: applicationPath,
		ReleasesDir:     releasesDir,
		Message:         "GitOps 本地工作树校验通过",
	}, nil
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ValidateGitOpsRepoState(branch string) (*GitOpsRepoValidationResult, error) {
	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return &GitOpsRepoValidationResult{Valid: false, Message: "integrations.gitops.local_checkout_path 未配置"}, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	targetBranch := strings.TrimSpace(branch)
	if targetBranch == "" {
		targetBranch = "HEAD"
	}

	if _, err := runGit(repoRoot, "rev-parse", "--is-inside-work-tree"); err != nil {
		return &GitOpsRepoValidationResult{
			Valid:    false,
			RepoPath: repoRoot,
			Branch:   targetBranch,
			Message:  "目标目录不是有效的 Git 仓库",
		}, err
	}

	if targetBranch != "HEAD" {
		if _, err := runGit(repoRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+targetBranch); err != nil {
			return &GitOpsRepoValidationResult{
				Valid:    false,
				RepoPath: repoRoot,
				Branch:   targetBranch,
				Message:  "本地仓库不存在目标分支: " + targetBranch,
			}, err
		}
	}

	return &GitOpsRepoValidationResult{
		Valid:    true,
		RepoPath: repoRoot,
		Branch:   targetBranch,
		Message:  "GitOps 本地仓库状态校验通过",
	}, nil
}

func CommitGitOpsWorkingTree(branch string, filePath string, requestNo string) (*GitOpsCommitResult, error) {
	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return nil, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	targetBranch := strings.TrimSpace(branch)
	if targetBranch == "" {
		currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return nil, err
		}
		targetBranch = currentBranch
	}

	currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	if currentBranch != targetBranch {
		return nil, fmt.Errorf("当前工作树分支为 %s，目标分支为 %s，请手动切换后再执行", currentBranch, targetBranch)
	}

	addTarget := filePath
	if strings.TrimSpace(addTarget) == "" {
		addTarget = "."
	}
	if _, err := runGit(repoRoot, "add", "--", addTarget); err != nil {
		return nil, err
	}

	statusOutput, err := runGit(repoRoot, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(statusOutput) == "" {
		return &GitOpsCommitResult{
			RepoPath: repoRoot,
			Branch:   targetBranch,
			Message:  "GitOps 工作树无可提交变更",
		}, nil
	}

	commitMessage := fmt.Sprintf("Create AutoOps release request %s", requestNo)
	if _, err := runGit(repoRoot, "commit", "-m", commitMessage, "--", addTarget); err != nil {
		return nil, err
	}

	commitSHA, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	return &GitOpsCommitResult{
		RepoPath:  repoRoot,
		Branch:    targetBranch,
		CommitSHA: commitSHA,
		Message:   "GitOps 本地 commit 已创建，尚未 push",
	}, nil
}

func PushGitOpsBranch(branch string) (*GitOpsPushResult, error) {
	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return nil, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	targetBranch := strings.TrimSpace(branch)
	if targetBranch == "" {
		currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return nil, err
		}
		targetBranch = currentBranch
	}

	currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	if currentBranch != targetBranch {
		return nil, fmt.Errorf("当前工作树分支为 %s，目标分支为 %s，请手动切换后再执行", currentBranch, targetBranch)
	}

	if _, err := runGit(repoRoot, "push", "origin", targetBranch); err != nil {
		return nil, err
	}

	return &GitOpsPushResult{
		RepoPath: repoRoot,
		Branch:   targetBranch,
		Message:  "GitOps 本地提交已推送到远端分支",
	}, nil
}

func DeleteGitOpsRelease(req *model.DeployRequest, releaseDir string, branch string) (*GitOpsDeleteResult, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}

	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return nil, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	filePath := filepath.ToSlash(filepath.Join(strings.TrimSpace(defaultGitOpsReleaseDir(releaseDir)), req.ReleaseName+".yaml"))
	absolutePath := filepath.Join(repoRoot, filepath.FromSlash(filePath))
	if !pathExists(absolutePath) {
		return nil, fmt.Errorf("GitOps release 文件不存在: %s", filePath)
	}
	if err := os.Remove(absolutePath); err != nil {
		return nil, fmt.Errorf("删除 GitOps release 文件失败: %v", err)
	}

	targetBranch := strings.TrimSpace(branch)
	if targetBranch == "" {
		currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
		if err != nil {
			return nil, err
		}
		targetBranch = currentBranch
	}

	currentBranch, err := runGit(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	if currentBranch != targetBranch {
		return nil, fmt.Errorf("当前工作树分支为 %s，目标分支为 %s，请手动切换后再执行", currentBranch, targetBranch)
	}

	if _, err := runGit(repoRoot, "add", "-A", "--", filePath); err != nil {
		return nil, err
	}
	commitMessage := fmt.Sprintf("Delete AutoOps release request %s", req.RequestNo)
	if _, err := runGit(repoRoot, "commit", "-m", commitMessage, "--", filePath); err != nil {
		return nil, err
	}
	if _, err := runGit(repoRoot, "push", "origin", targetBranch); err != nil {
		return nil, err
	}
	commitSHA, err := runGit(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}

	return &GitOpsDeleteResult{
		RepoPath:  repoRoot,
		FilePath:  filePath,
		Branch:    targetBranch,
		CommitSHA: commitSHA,
		Message:   "GitOps release 文件已删除并推送",
	}, nil
}

func defaultGitOpsReleaseDir(releaseDir string) string {
	if strings.TrimSpace(releaseDir) != "" {
		return strings.TrimSpace(releaseDir)
	}
	return "apps/autoops-managed-releases/releases"
}

func runGit(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git %s 失败: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

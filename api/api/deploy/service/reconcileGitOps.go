package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"dodevops-api/api/deploy/model"
	"dodevops-api/common/config"

	"gorm.io/gorm"
)

type GitOpsReconcileItem struct {
	ReleaseName     string `json:"releaseName"`
	RequestNo       string `json:"requestNo"`
	FilePath        string `json:"filePath"`
	RequestStatus   string `json:"requestStatus"`
	ExecutionStatus string `json:"executionStatus"`
	OwnerSystem     string `json:"ownerSystem"`
	State           string `json:"state"`
}

type GitOpsReconcileReport struct {
	RepoPath string                `json:"repoPath"`
	Items    []GitOpsReconcileItem `json:"items"`
}

func ReconcileGitOpsWorkingTree(db *gorm.DB) (*GitOpsReconcileReport, error) {
	if db == nil {
		return nil, fmt.Errorf("database is required")
	}

	repoRoot := strings.TrimSpace(config.Config.Integrations.GitOps.LocalCheckoutPath)
	if repoRoot == "" {
		return nil, fmt.Errorf("integrations.gitops.local_checkout_path 未配置")
	}

	releasesDir := filepath.Join(repoRoot, "apps/autoops-managed-releases/releases")
	entries, err := os.ReadDir(releasesDir)
	if err != nil {
		return nil, fmt.Errorf("读取 GitOps releases 目录失败: %v", err)
	}

	var requests []model.DeployRequest
	if err := db.Where("mode = ?", model.DeployModeGitOps).Order("id DESC").Find(&requests).Error; err != nil {
		return nil, fmt.Errorf("读取 GitOps 部署申请失败: %v", err)
	}

	var owners []model.ResourceOwner
	if err := db.Where("owner_system = ?", model.ResourceOwnerSystemGitOps).Where("active = ?", true).Find(&owners).Error; err != nil {
		return nil, fmt.Errorf("读取 GitOps 资源 owner 失败: %v", err)
	}

	requestByRelease := map[string]model.DeployRequest{}
	for _, req := range requests {
		if _, exists := requestByRelease[req.ReleaseName]; !exists {
			requestByRelease[req.ReleaseName] = req
		}
	}
	ownerByRelease := map[string]string{}
	for _, owner := range owners {
		if owner.ReleaseName != "" {
			ownerByRelease[owner.ReleaseName] = owner.OwnerSystem
		}
	}

	items := make([]GitOpsReconcileItem, 0)
	seenReleases := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		releaseName := strings.TrimSuffix(entry.Name(), ".yaml")
		seenReleases[releaseName] = struct{}{}
		item := GitOpsReconcileItem{
			ReleaseName: releaseName,
			FilePath:    filepath.ToSlash(filepath.Join("apps/autoops-managed-releases/releases", entry.Name())),
			OwnerSystem: ownerByRelease[releaseName],
			State:       "orphan_file",
		}
		if req, ok := requestByRelease[releaseName]; ok {
			item.RequestNo = req.RequestNo
			item.RequestStatus = req.RequestStatus
			item.ExecutionStatus = req.ExecutionStatus
			item.State = "matched"
		}
		items = append(items, item)
	}

	for releaseName, req := range requestByRelease {
		if _, exists := seenReleases[releaseName]; exists {
			continue
		}
		items = append(items, GitOpsReconcileItem{
			ReleaseName:     releaseName,
			RequestNo:       req.RequestNo,
			RequestStatus:   req.RequestStatus,
			ExecutionStatus: req.ExecutionStatus,
			OwnerSystem:     ownerByRelease[releaseName],
			State:           "missing_file",
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].State == items[j].State {
			return items[i].ReleaseName < items[j].ReleaseName
		}
		return items[i].State < items[j].State
	})

	return &GitOpsReconcileReport{
		RepoPath: repoRoot,
		Items:    items,
	}, nil
}

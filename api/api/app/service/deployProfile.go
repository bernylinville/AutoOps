package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	appmodel "dodevops-api/api/app/model"
	ccmodel "dodevops-api/api/configcenter/model"
	deploymodel "dodevops-api/api/deploy/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (s *ApplicationService) ListDeployProfiles(c *gin.Context, appID uint) {
	if _, err := s.appDao.GetApplicationByID(appID); err != nil {
		result.Failed(c, 404, "应用不存在")
		return
	}
	var profiles []appmodel.AppDeployProfile
	if err := s.db.Where("app_id = ?", appID).Order("env").Find(&profiles).Error; err != nil {
		result.Failed(c, 500, "获取部署配置失败: "+err.Error())
		return
	}
	result.Success(c, profiles)
}

func (s *ApplicationService) CreateDeployProfile(c *gin.Context, appID uint, req *appmodel.CreateAppDeployProfileRequest) {
	app, err := s.appDao.GetApplicationByID(appID)
	if err != nil {
		result.Failed(c, 404, "应用不存在")
		return
	}
	if err := s.validateDeployProfileRefs(req.ClusterTargetID, req.JenkinsServerID, req.HarborServerID, req.ApproverAdminID, req.Env); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	profile := &appmodel.AppDeployProfile{
		AppID:             appID,
		ApplicationCode:   app.Code,
		Env:               strings.TrimSpace(req.Env),
		Enabled:           enabled,
		ClusterTargetID:   req.ClusterTargetID,
		Namespace:         strings.TrimSpace(req.Namespace),
		ReleaseName:       strings.TrimSpace(req.ReleaseName),
		ResourceType:      defaultProfileString(req.ResourceType, deploymodel.DeployResourceTypeDeployment),
		JenkinsServerID:   req.JenkinsServerID,
		JenkinsJobName:    strings.TrimSpace(req.JenkinsJobName),
		HarborServerID:    req.HarborServerID,
		HarborProject:     strings.TrimSpace(req.HarborProject),
		HarborRepository:  strings.TrimSpace(req.HarborRepository),
		DefaultGitRef:     defaultProfileString(req.DefaultGitRef, "main"),
		ApproverAdminID:   req.ApproverAdminID,
		Replicas:          defaultProfileReplicas(req.Replicas),
		ServiceEnabled:    req.ServiceEnabled,
		ServiceType:       defaultProfileString(req.ServiceType, "ClusterIP"),
		ServicePort:       defaultProfilePort(req.ServicePort, 80),
		TargetPort:        defaultProfilePort(req.TargetPort, 8080),
		EnvJSON:           marshalProfileJSON(req.EnvVars),
		ResourcesJSON:     marshalProfileJSON(req.Resources),
		BuildParamsJSON:   marshalProfileJSON(req.BuildParams),
		ScanPolicyJSON:    marshalProfileJSON(req.ScanPolicy),
		AccessURLTemplate: strings.TrimSpace(req.AccessURLTemplate),
		HealthCheckPath:   strings.TrimSpace(req.HealthCheckPath),
		Description:       strings.TrimSpace(req.Description),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(profile).Error; err != nil {
			return err
		}
		return syncProfileSideEffects(tx, profile)
	}); err != nil {
		result.Failed(c, 500, "创建部署配置失败: "+err.Error())
		return
	}
	result.Success(c, profile)
}

func (s *ApplicationService) UpdateDeployProfile(c *gin.Context, appID, profileID uint, req *appmodel.UpdateAppDeployProfileRequest) {
	var profile appmodel.AppDeployProfile
	if err := s.db.Where("id = ? AND app_id = ?", profileID, appID).First(&profile).Error; err != nil {
		result.Failed(c, 404, "部署配置不存在")
		return
	}
	updates, candidate, err := buildDeployProfileUpdates(profile, req)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	if err := s.validateDeployProfileRefs(candidate.ClusterTargetID, candidate.JenkinsServerID, candidate.HarborServerID, candidate.ApproverAdminID, candidate.Env); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&profile).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := tx.First(&profile, profileID).Error; err != nil {
			return err
		}
		return syncProfileSideEffects(tx, &profile)
	}); err != nil {
		result.Failed(c, 500, "更新部署配置失败: "+err.Error())
		return
	}
	result.Success(c, profile)
}

func buildDeployProfileUpdates(profile appmodel.AppDeployProfile, req *appmodel.UpdateAppDeployProfileRequest) (map[string]interface{}, appmodel.AppDeployProfile, error) {
	updates := map[string]interface{}{}
	candidate := profile
	if req.Enabled != nil {
		candidate.Enabled = *req.Enabled
		updates["enabled"] = candidate.Enabled
	}
	if req.ClusterTargetID != nil {
		candidate.ClusterTargetID = *req.ClusterTargetID
		updates["cluster_target_id"] = candidate.ClusterTargetID
	}
	if req.Namespace != nil {
		candidate.Namespace = strings.TrimSpace(*req.Namespace)
		updates["namespace"] = candidate.Namespace
	}
	if req.ReleaseName != nil {
		candidate.ReleaseName = strings.TrimSpace(*req.ReleaseName)
		updates["release_name"] = candidate.ReleaseName
	}
	if req.ResourceType != nil {
		candidate.ResourceType = defaultProfileString(*req.ResourceType, deploymodel.DeployResourceTypeDeployment)
		if candidate.ResourceType != deploymodel.DeployResourceTypeDeployment && candidate.ResourceType != deploymodel.DeployResourceTypePod {
			return nil, candidate, fmt.Errorf("resourceType 仅支持 deployment 或 pod")
		}
		updates["resource_type"] = candidate.ResourceType
	}
	if req.JenkinsServerID != nil {
		candidate.JenkinsServerID = *req.JenkinsServerID
		updates["jenkins_server_id"] = candidate.JenkinsServerID
	}
	if req.JenkinsJobName != nil {
		candidate.JenkinsJobName = strings.TrimSpace(*req.JenkinsJobName)
		updates["jenkins_job_name"] = candidate.JenkinsJobName
	}
	if req.HarborServerID != nil {
		candidate.HarborServerID = *req.HarborServerID
		updates["harbor_server_id"] = candidate.HarborServerID
	}
	if req.HarborProject != nil {
		candidate.HarborProject = strings.TrimSpace(*req.HarborProject)
		updates["harbor_project"] = candidate.HarborProject
	}
	if req.HarborRepository != nil {
		candidate.HarborRepository = strings.TrimSpace(*req.HarborRepository)
		updates["harbor_repository"] = candidate.HarborRepository
	}
	if req.DefaultGitRef != nil {
		candidate.DefaultGitRef = defaultProfileString(*req.DefaultGitRef, "main")
		updates["default_git_ref"] = candidate.DefaultGitRef
	}
	if req.ApproverAdminID != nil {
		candidate.ApproverAdminID = *req.ApproverAdminID
		updates["approver_admin_id"] = candidate.ApproverAdminID
	}
	if req.Replicas != nil {
		candidate.Replicas = defaultProfileReplicas(*req.Replicas)
		updates["replicas"] = candidate.Replicas
	}
	if req.ServiceEnabled != nil {
		candidate.ServiceEnabled = *req.ServiceEnabled
		updates["service_enabled"] = candidate.ServiceEnabled
	}
	if req.ServiceType != nil {
		candidate.ServiceType = defaultProfileString(*req.ServiceType, "ClusterIP")
		updates["service_type"] = candidate.ServiceType
	}
	if req.ServicePort != nil {
		candidate.ServicePort = defaultProfilePort(*req.ServicePort, 80)
		updates["service_port"] = candidate.ServicePort
	}
	if req.TargetPort != nil {
		candidate.TargetPort = defaultProfilePort(*req.TargetPort, 8080)
		updates["target_port"] = candidate.TargetPort
	}
	if req.EnvVars != nil {
		candidate.EnvJSON = marshalProfileJSON(req.EnvVars)
		updates["env_json"] = candidate.EnvJSON
	}
	if req.Resources != nil {
		candidate.ResourcesJSON = marshalProfileJSON(req.Resources)
		updates["resources_json"] = candidate.ResourcesJSON
	}
	if req.BuildParams != nil {
		candidate.BuildParamsJSON = marshalProfileJSON(req.BuildParams)
		updates["build_params_json"] = candidate.BuildParamsJSON
	}
	if req.ScanPolicy != nil {
		candidate.ScanPolicyJSON = marshalProfileJSON(req.ScanPolicy)
		updates["scan_policy_json"] = candidate.ScanPolicyJSON
	}
	if req.AccessURLTemplate != nil {
		candidate.AccessURLTemplate = strings.TrimSpace(*req.AccessURLTemplate)
		updates["access_url_template"] = candidate.AccessURLTemplate
	}
	if req.HealthCheckPath != nil {
		candidate.HealthCheckPath = strings.TrimSpace(*req.HealthCheckPath)
		updates["health_check_path"] = candidate.HealthCheckPath
	}
	if req.Description != nil {
		candidate.Description = strings.TrimSpace(*req.Description)
		updates["description"] = candidate.Description
	}
	return updates, candidate, nil
}

func (s *ApplicationService) DeleteDeployProfile(c *gin.Context, appID, profileID uint) {
	var profile appmodel.AppDeployProfile
	if err := s.db.Where("id = ? AND app_id = ?", profileID, appID).First(&profile).Error; err != nil {
		result.Failed(c, 404, "部署配置不存在")
		return
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&profile).Error; err != nil {
			return err
		}
		return deleteProfileManagedSideEffects(tx, &profile)
	}); err != nil {
		result.Failed(c, 500, "删除部署配置失败: "+err.Error())
		return
	}
	result.Success(c, "删除成功")
}

func (s *ApplicationService) ValidateDeployProfile(c *gin.Context, appID, profileID uint) {
	var profile appmodel.AppDeployProfile
	if err := s.db.Where("id = ? AND app_id = ?", profileID, appID).First(&profile).Error; err != nil {
		result.Failed(c, 404, "部署配置不存在")
		return
	}
	messages := []string{}
	if err := s.validateDeployProfileRefs(profile.ClusterTargetID, profile.JenkinsServerID, profile.HarborServerID, profile.ApproverAdminID, profile.Env); err != nil {
		messages = append(messages, err.Error())
	}
	if strings.TrimSpace(profile.JenkinsJobName) == "" {
		messages = append(messages, "Jenkins job 未配置")
	}
	if strings.TrimSpace(profile.Namespace) == "" {
		messages = append(messages, "namespace 未配置")
	}
	result.Success(c, appmodel.AppDeployProfileValidation{Valid: len(messages) == 0, Messages: messages})
}

func (s *ApplicationService) validateDeployProfileRefs(clusterTargetID, jenkinsServerID, harborServerID, approverAdminID uint, env string) error {
	if clusterTargetID == 0 || jenkinsServerID == 0 || harborServerID == 0 || approverAdminID == 0 {
		return fmt.Errorf("clusterTargetId、jenkinsServerId、harborServerId、approverAdminId 均为必填")
	}
	var clusterTarget deploymodel.ClusterTarget
	if err := s.db.First(&clusterTarget, clusterTargetID).Error; err != nil {
		return fmt.Errorf("部署目标不存在")
	}
	if err := checkClusterTargetEnvType(clusterTarget.EnvType, env); err != nil {
		return err
	}
	if err := s.validateAccountType(jenkinsServerID, appmodel.JenkinsAccountType, "Jenkins"); err != nil {
		return err
	}
	if err := s.validateAccountType(harborServerID, appmodel.HarborAccountType, "Harbor"); err != nil {
		return err
	}
	var approver systemmodel.SysAdmin
	if err := s.db.First(&approver, approverAdminID).Error; err != nil {
		return fmt.Errorf("审批人不存在")
	}
	if strings.TrimSpace(approver.DingtalkUserID) == "" {
		return fmt.Errorf("审批人未配置钉钉 UserID")
	}
	return nil
}

func (s *ApplicationService) validateAccountType(id uint, want int, label string) error {
	var account ccmodel.AccountAuth
	if err := s.db.First(&account, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%s 凭据不存在", label)
		}
		return err
	}
	if account.Type != want {
		return fmt.Errorf("%s 凭据类型不匹配", label)
	}
	return nil
}

func syncProfileSideEffects(db *gorm.DB, profile *appmodel.AppDeployProfile) error {
	var env appmodel.JenkinsEnv
	if err := db.Where("app_id = ? AND env_name = ?", profile.AppID, profile.Env).First(&env).Error; err == nil {
		if err := db.Model(&env).Updates(map[string]interface{}{"jenkins_server_id": profile.JenkinsServerID, "job_name": profile.JenkinsJobName}).Error; err != nil {
			return err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&appmodel.JenkinsEnv{AppID: profile.AppID, EnvName: profile.Env, JenkinsServerID: &profile.JenkinsServerID, JobName: profile.JenkinsJobName}).Error; err != nil {
			return err
		}
	} else {
		return err
	}

	// Keep the existing agent allow-list gate in sync with productized profile configuration.
	allow := deploymodel.AgentApproverAllowlist{
		ApproverAdminID: profile.ApproverAdminID,
		ApplicationCode: profile.ApplicationCode,
		Env:             profile.Env,
		CreatedBy:       "deploy-profile",
	}
	if err := db.Where("application_code = ? AND env = ? AND created_by = ? AND approver_admin_id <> ?", allow.ApplicationCode, allow.Env, "deploy-profile", allow.ApproverAdminID).
		Delete(&deploymodel.AgentApproverAllowlist{}).Error; err != nil {
		return err
	}
	if err := db.Where("approver_admin_id = ? AND application_code = ? AND env = ?", allow.ApproverAdminID, allow.ApplicationCode, allow.Env).
		FirstOrCreate(&allow).Error; err != nil {
		return err
	}
	return nil
}

func deleteProfileManagedSideEffects(db *gorm.DB, profile *appmodel.AppDeployProfile) error {
	if err := db.Where("application_code = ? AND env = ? AND created_by = ?", profile.ApplicationCode, profile.Env, "deploy-profile").
		Delete(&deploymodel.AgentApproverAllowlist{}).Error; err != nil {
		return err
	}
	return db.Where("app_id = ? AND env_name = ?", profile.AppID, profile.Env).
		Delete(&appmodel.JenkinsEnv{}).Error
}

// checkClusterTargetEnvType returns an error when the cluster target's env type
// does not match the profile's env. Extracted as a pure function for testability.
func checkClusterTargetEnvType(clusterEnvType, profileEnv string) error {
	if strings.ToLower(strings.TrimSpace(clusterEnvType)) != strings.ToLower(strings.TrimSpace(profileEnv)) {
		return fmt.Errorf("部署目标与环境不匹配（集群目标 envType=%s，profile env=%s 不匹配）", strings.TrimSpace(clusterEnvType), strings.TrimSpace(profileEnv))
	}
	return nil
}

func marshalProfileJSON(v map[string]interface{}) string {
	if len(v) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func defaultProfileString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func defaultProfileReplicas(v int32) int32 {
	if v <= 0 {
		return 1
	}
	return v
}
func defaultProfilePort(v int32, fallback int32) int32 {
	if v <= 0 {
		return fallback
	}
	return v
}

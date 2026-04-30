package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	appmodel "dodevops-api/api/app/model"
	"dodevops-api/api/deploy/model"
	"dodevops-api/common/config"
	"dodevops-api/common/result"
	"dodevops-api/common/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateAgentBuildDeployRequest lets Hermes submit natural project/env requests while
// AutoOps resolves the internal deployment profile and keeps authoritative state here.
func (s *DeployService) CreateAgentBuildDeployRequest(c *gin.Context, req *model.CreateAgentBuildDeployRequest) {
	if strings.TrimSpace(req.RequesterExternalType) != "dingtalk" {
		result.Failed(c, 400, "当前仅支持 dingtalk 外部身份类型")
		return
	}

	var app appmodel.Application
	if err := s.db.Where("code = ?", strings.TrimSpace(req.ApplicationCode)).First(&app).Error; err != nil {
		result.Failed(c, 404, "应用不存在或未接入 AutoOps")
		return
	}

	var profile appmodel.AppDeployProfile
	if err := s.db.Where("app_id = ? AND env = ?", app.ID, strings.TrimSpace(req.Env)).First(&profile).Error; err != nil {
		result.Failed(c, 404, "应用环境未配置部署 Profile")
		return
	}
	if !profile.Enabled {
		result.Failed(c, 400, "应用环境部署 Profile 已禁用")
		return
	}

	s.CreateAgentDeployRequest(c, buildAgentDeployRequestFromProfile(app.ID, &profile, req))
}

// CreateAgentProjectOnboardBuildDeployRequest lets Hermes accept a GitLab URL
// from the user, create/reuse the AutoOps Application + dev/test deploy Profile,
// then run the same build-deploy flow as the profile-driven endpoint.
func (s *DeployService) CreateAgentProjectOnboardBuildDeployRequest(c *gin.Context, req *model.CreateAgentProjectOnboardBuildDeployRequest) {
	if strings.TrimSpace(req.RequesterExternalType) != "dingtalk" {
		result.Failed(c, 400, "当前仅支持 dingtalk 外部身份类型")
		return
	}
	if _, err := s.getAdminByDingtalkUserID(req.RequesterExternalID); err != nil {
		result.Failed(c, 400, "该钉钉用户未绑定 AutoOps 账号")
		return
	}

	defaults, err := loadAgentProjectOnboardingDefaults()
	if err != nil {
		result.Failed(c, 500, "项目自动接入配置不可用: "+err.Error())
		return
	}

	repo, err := parseAgentGitRepoURL(req.GitRepoURL)
	if err != nil {
		result.Failed(c, 400, "GitLab 仓库地址无效: "+err.Error())
		return
	}
	if !isAgentGitHostAllowed(repo.Host, defaults.AllowedGitHosts) {
		result.Failed(c, 400, fmt.Sprintf("GitLab host %q 不在 AutoOps 允许列表中", repo.Host))
		return
	}

	appCode := strings.TrimSpace(req.ApplicationCode)
	if appCode == "" {
		appCode = util.GenerateAppCode(repo.RepoName)
	}
	if !util.ValidateAppCode(appCode) {
		result.Failed(c, 400, "应用代号无效：仅允许小写字母、数字和连字符，长度不超过 32")
		return
	}

	env := strings.ToLower(strings.TrimSpace(req.Env))
	serviceType, err := agentServiceTypeFromExposureMode(req.ExposureMode)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	exposureSpecified := strings.TrimSpace(req.ExposureMode) != ""

	app, _, err := s.ensureAgentProjectOnboarded(defaults, repo, appCode, env, serviceType, exposureSpecified)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}

	s.CreateAgentBuildDeployRequest(c, &model.CreateAgentBuildDeployRequest{
		RequesterExternalType: req.RequesterExternalType,
		RequesterExternalID:   req.RequesterExternalID,
		RequesterDisplayName:  req.RequesterDisplayName,
		ApplicationCode:       app.Code,
		Env:                   env,
		GitRef:                req.GitRef,
		Reason:                req.Reason,
		BuildParams:           req.BuildParams,
		ChatContext:           req.ChatContext,
	})
}

func buildAgentDeployRequestFromProfile(appID uint, profile *appmodel.AppDeployProfile, req *model.CreateAgentBuildDeployRequest) *model.CreateAgentDeployRequest {
	gitRef := defaultAgentBuildGitRef(req.GitRef, profile.DefaultGitRef)
	buildParams := mergeProfileMaps(profile.BuildParamsJSON, req.BuildParams)
	buildParams["GIT_REF"] = gitRef
	buildParams["ENV"] = profile.Env
	buildParams["RELEASE_NAME"] = profile.ReleaseName

	approverID := profile.ApproverAdminID
	jenkinsServerID := profile.JenkinsServerID
	harborServerID := profile.HarborServerID
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "钉钉机器人触发构建部署"
	}

	return &model.CreateAgentDeployRequest{
		RequesterExternalType: req.RequesterExternalType,
		RequesterExternalID:   req.RequesterExternalID,
		RequesterDisplayName:  req.RequesterDisplayName,
		Mode:                  model.DeployModeDirect,
		WorkflowKind:          model.WorkflowKindBuildDeploy,
		ApplicationID:         &appID,
		ResourceType:          defaultProfileResourceType(profile.ResourceType),
		ClusterTargetID:       profile.ClusterTargetID,
		ReleaseName:           profile.ReleaseName,
		Namespace:             profile.Namespace,
		Replicas:              profile.Replicas,
		ServiceEnabled:        profile.ServiceEnabled,
		ServiceType:           profile.ServiceType,
		ServicePort:           profile.ServicePort,
		TargetPort:            profile.TargetPort,
		Env:                   mergeProfileMaps(profile.EnvJSON, map[string]interface{}{"name": profile.Env, "env": profile.Env}),
		Resources:             mergeProfileMaps(profile.ResourcesJSON, nil),
		ApproverAdminID:       &approverID,
		Reason:                reason,
		ChatContext:           req.ChatContext,
		GitRef:                gitRef,
		BuildParams:           buildParams,
		JenkinsServerID:       &jenkinsServerID,
		JenkinsJobName:        profile.JenkinsJobName,
		HarborServerID:        &harborServerID,
		HarborProject:         profile.HarborProject,
		HarborRepository:      profile.HarborRepository,
		ScanPolicy:            mergeProfileMaps(profile.ScanPolicyJSON, nil),
	}
}

func defaultAgentBuildGitRef(requestGitRef, profileGitRef string) string {
	if gitRef := strings.TrimSpace(requestGitRef); gitRef != "" {
		return gitRef
	}
	if gitRef := strings.TrimSpace(profileGitRef); gitRef != "" {
		return gitRef
	}
	return "main"
}

func mergeProfileMaps(raw string, override map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func defaultProfileResourceType(value string) string {
	if strings.TrimSpace(value) == "" {
		return model.DeployResourceTypeDeployment
	}
	return strings.TrimSpace(value)
}

type agentProjectOnboardingDefaults struct {
	AllowedGitHosts        []string
	SharedJenkinsJobName   string
	DefaultBusinessGroupID uint
	DefaultBusinessDeptID  uint
	DefaultJenkinsServerID uint
	DefaultHarborServerID  uint
	DefaultHarborProject   string
	DefaultApproverAdminID uint
	DevClusterTargetID     uint
	TestClusterTargetID    uint
	NamespacePrefix        string
	DefaultServicePort     int32
	DefaultTargetPort      int32
}

type agentGitRepo struct {
	RawURL   string
	Host     string
	Path     string
	RepoName string
}

func loadAgentProjectOnboardingDefaults() (*agentProjectOnboardingDefaults, error) {
	if config.Config == nil {
		return nil, fmt.Errorf("config 未初始化")
	}
	cfg := config.Config.Integrations.Agent.ProjectOnboarding
	if !cfg.Enabled {
		return nil, fmt.Errorf("integrations.agent.project_onboarding.enabled 未开启")
	}
	defaults := &agentProjectOnboardingDefaults{
		AllowedGitHosts:        cfg.AllowedGitHosts,
		SharedJenkinsJobName:   strings.TrimSpace(cfg.SharedJenkinsJobName),
		DefaultBusinessGroupID: cfg.DefaultBusinessGroupID,
		DefaultBusinessDeptID:  cfg.DefaultBusinessDeptID,
		DefaultJenkinsServerID: cfg.DefaultJenkinsServerID,
		DefaultHarborServerID:  cfg.DefaultHarborServerID,
		DefaultHarborProject:   strings.TrimSpace(cfg.DefaultHarborProject),
		DefaultApproverAdminID: cfg.DefaultApproverAdminID,
		DevClusterTargetID:     cfg.DevClusterTargetID,
		TestClusterTargetID:    cfg.TestClusterTargetID,
		NamespacePrefix:        defaultString(cfg.NamespacePrefix, "ao-direct"),
		DefaultServicePort:     defaultAgentProjectPort(cfg.DefaultServicePort, 80),
		DefaultTargetPort:      defaultAgentProjectPort(cfg.DefaultTargetPort, 8080),
	}
	if len(defaults.AllowedGitHosts) == 0 {
		return nil, fmt.Errorf("allowed_git_hosts 未配置")
	}
	if defaults.SharedJenkinsJobName == "" {
		return nil, fmt.Errorf("shared_jenkins_job_name 未配置")
	}
	if defaults.DefaultBusinessGroupID == 0 || defaults.DefaultBusinessDeptID == 0 {
		return nil, fmt.Errorf("default_business_group_id/default_business_dept_id 未配置")
	}
	if defaults.DefaultJenkinsServerID == 0 || defaults.DefaultHarborServerID == 0 {
		return nil, fmt.Errorf("default_jenkins_server_id/default_harbor_server_id 未配置")
	}
	if defaults.DefaultHarborProject == "" {
		return nil, fmt.Errorf("default_harbor_project 未配置")
	}
	if defaults.DefaultApproverAdminID == 0 {
		return nil, fmt.Errorf("default_approver_admin_id 未配置")
	}
	return defaults, nil
}

func (d *agentProjectOnboardingDefaults) clusterTargetID(env string) (uint, error) {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case appmodel.DeployProfileEnvDev:
		if d.DevClusterTargetID == 0 {
			return 0, fmt.Errorf("dev_cluster_target_id 未配置")
		}
		return d.DevClusterTargetID, nil
	case appmodel.DeployProfileEnvTest:
		if d.TestClusterTargetID == 0 {
			return 0, fmt.Errorf("test_cluster_target_id 未配置")
		}
		return d.TestClusterTargetID, nil
	default:
		return 0, fmt.Errorf("当前仅支持 dev/test 环境")
	}
}

func (s *DeployService) ensureAgentProjectOnboarded(defaults *agentProjectOnboardingDefaults, repo agentGitRepo, appCode, env, serviceType string, exposureSpecified bool) (*appmodel.Application, *appmodel.AppDeployProfile, error) {
	var app appmodel.Application
	var profile appmodel.AppDeployProfile

	err := s.db.Transaction(func(tx *gorm.DB) error {
		clusterTargetID, err := defaults.clusterTargetID(env)
		if err != nil {
			return err
		}
		if err := validateAgentOnboardingClusterTarget(tx, clusterTargetID, env); err != nil {
			return err
		}

		app, err = firstOrCreateAgentOnboardingApplication(tx, defaults, repo, appCode)
		if err != nil {
			return err
		}

		profileDefaults := buildAgentOnboardingProfileDefaults(defaults, &app, repo, env, clusterTargetID, serviceType)
		if err := tx.Where("app_id = ? AND env = ?", app.ID, env).First(&profile).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			profile = profileDefaults
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
			return syncAgentOnboardedProfileSideEffects(tx, &profile)
		}

		if err := validateAgentOnboardingClusterTarget(tx, profile.ClusterTargetID, env); err != nil {
			return err
		}

		effectiveHarborProject := defaultString(profile.HarborProject, defaults.DefaultHarborProject)
		effectiveHarborRepository := defaultString(profile.HarborRepository, app.Code)
		effectiveServicePort := defaultAgentProjectPort(profile.ServicePort, defaults.DefaultServicePort)
		effectiveTargetPort := defaultAgentProjectPort(profile.TargetPort, defaults.DefaultTargetPort)
		updates := map[string]interface{}{
			"build_params_json": mergeAgentProjectBuildParamsJSON(profile.BuildParamsJSON, requiredAgentProjectBuildParams(repo.RawURL, app.Code, effectiveHarborProject, effectiveHarborRepository, effectiveServicePort, effectiveTargetPort)),
		}
		if exposureSpecified && profile.Enabled {
			updates["service_enabled"] = true
			updates["service_type"] = serviceType
			updates["service_port"] = effectiveServicePort
			updates["target_port"] = effectiveTargetPort
		}
		if strings.TrimSpace(profile.JenkinsJobName) == "" {
			updates["jenkins_job_name"] = defaults.SharedJenkinsJobName
		}
		if profile.JenkinsServerID == 0 {
			updates["jenkins_server_id"] = defaults.DefaultJenkinsServerID
		}
		if profile.HarborServerID == 0 {
			updates["harbor_server_id"] = defaults.DefaultHarborServerID
		}
		if strings.TrimSpace(profile.HarborProject) == "" {
			updates["harbor_project"] = effectiveHarborProject
		}
		if strings.TrimSpace(profile.HarborRepository) == "" {
			updates["harbor_repository"] = effectiveHarborRepository
		}
		if profile.ApproverAdminID == 0 {
			updates["approver_admin_id"] = defaults.DefaultApproverAdminID
		}
		if strings.TrimSpace(profile.ApplicationCode) == "" {
			updates["application_code"] = app.Code
		}
		if strings.TrimSpace(profile.DefaultGitRef) == "" {
			updates["default_git_ref"] = "main"
		}
		if isEmptyProfileJSON(profile.ResourcesJSON) {
			updates["resources_json"] = marshalProfileMap(defaultAgentProjectResources())
		}
		if err := tx.Model(&profile).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.First(&profile, profile.ID).Error; err != nil {
			return err
		}
		return syncAgentOnboardedProfileSideEffects(tx, &profile)
	})
	if err != nil {
		return nil, nil, err
	}
	return &app, &profile, nil
}

func firstOrCreateAgentOnboardingApplication(tx *gorm.DB, defaults *agentProjectOnboardingDefaults, repo agentGitRepo, appCode string) (appmodel.Application, error) {
	var app appmodel.Application
	if err := tx.Where("code = ?", appCode).First(&app).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return app, err
		}
		app = appmodel.Application{
			Name:            appCode,
			Code:            appCode,
			BusinessGroupID: defaults.DefaultBusinessGroupID,
			BusinessDeptID:  defaults.DefaultBusinessDeptID,
			Description:     "Hermes 自动接入的 Spring Boot GitLab 项目",
			RepoURL:         repo.RawURL,
			ProgrammingLang: "Java",
			StartCommand:    "java -jar app.jar",
			HealthAPI:       "/actuator/health",
			Status:          1,
		}
		return app, tx.Create(&app).Error
	}
	if strings.TrimSpace(app.RepoURL) == "" {
		app.RepoURL = repo.RawURL
		return app, tx.Model(&app).Update("repo_url", repo.RawURL).Error
	}
	if !sameAgentGitRepo(app.RepoURL, repo.RawURL) {
		return app, fmt.Errorf("应用代号 %q 已存在，但仓库地址为 %q；为避免误部署，请更换 applicationCode 或先在 AutoOps 中确认应用归属", appCode, app.RepoURL)
	}
	return app, nil
}

func buildAgentOnboardingProfileDefaults(defaults *agentProjectOnboardingDefaults, app *appmodel.Application, repo agentGitRepo, env string, clusterTargetID uint, serviceType string) appmodel.AppDeployProfile {
	servicePort := defaults.DefaultServicePort
	targetPort := defaults.DefaultTargetPort
	harborRepository := app.Code
	return appmodel.AppDeployProfile{
		AppID:            app.ID,
		ApplicationCode:  app.Code,
		Env:              env,
		Enabled:          true,
		ClusterTargetID:  clusterTargetID,
		Namespace:        strings.Trim(strings.TrimSpace(defaults.NamespacePrefix), "-") + "-" + app.Code,
		ReleaseName:      app.Code,
		ResourceType:     model.DeployResourceTypeDeployment,
		JenkinsServerID:  defaults.DefaultJenkinsServerID,
		JenkinsJobName:   defaults.SharedJenkinsJobName,
		HarborServerID:   defaults.DefaultHarborServerID,
		HarborProject:    defaults.DefaultHarborProject,
		HarborRepository: harborRepository,
		DefaultGitRef:    "main",
		ApproverAdminID:  defaults.DefaultApproverAdminID,
		Replicas:         1,
		ServiceEnabled:   true,
		ServiceType:      serviceType,
		ServicePort:      servicePort,
		TargetPort:       targetPort,
		EnvJSON:          marshalProfileMap(map[string]interface{}{"SPRING_PROFILES_ACTIVE": env}),
		ResourcesJSON:    marshalProfileMap(defaultAgentProjectResources()),
		BuildParamsJSON:  mergeAgentProjectBuildParamsJSON("{}", requiredAgentProjectBuildParams(repo.RawURL, app.Code, defaults.DefaultHarborProject, harborRepository, servicePort, targetPort)),
		ScanPolicyJSON:   "{}",
		HealthCheckPath:  "/actuator/health",
		Description:      "Hermes GitLab 项目自动接入生成的部署 Profile",
	}
}

func validateAgentOnboardingClusterTarget(tx *gorm.DB, clusterTargetID uint, env string) error {
	var target model.ClusterTarget
	if err := tx.First(&target, clusterTargetID).Error; err != nil {
		return fmt.Errorf("部署目标不存在: %d", clusterTargetID)
	}
	if !target.DirectEnabled {
		return fmt.Errorf("部署目标 %s 未启用 Direct 模式", target.Name)
	}
	if strings.TrimSpace(target.DirectKubeconfigRef) == "" {
		return fmt.Errorf("部署目标 %s 未配置 directKubeconfigRef，无法 Direct 部署", target.Name)
	}
	if strings.ToLower(strings.TrimSpace(target.EnvType)) != strings.ToLower(strings.TrimSpace(env)) {
		return fmt.Errorf("部署目标与环境不匹配（集群目标 envType=%s，profile env=%s 不匹配）", strings.TrimSpace(target.EnvType), strings.TrimSpace(env))
	}
	return nil
}

func syncAgentOnboardedProfileSideEffects(tx *gorm.DB, profile *appmodel.AppDeployProfile) error {
	var env appmodel.JenkinsEnv
	if err := tx.Where("app_id = ? AND env_name = ?", profile.AppID, profile.Env).First(&env).Error; err == nil {
		if err := tx.Model(&env).Updates(map[string]interface{}{"jenkins_server_id": profile.JenkinsServerID, "job_name": profile.JenkinsJobName}).Error; err != nil {
			return err
		}
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&appmodel.JenkinsEnv{AppID: profile.AppID, EnvName: profile.Env, JenkinsServerID: &profile.JenkinsServerID, JobName: profile.JenkinsJobName}).Error; err != nil {
			return err
		}
	} else {
		return err
	}

	allow := model.AgentApproverAllowlist{
		ApproverAdminID: profile.ApproverAdminID,
		ApplicationCode: profile.ApplicationCode,
		Env:             profile.Env,
		CreatedBy:       "project-onboarding",
	}
	if err := tx.Where("application_code = ? AND env = ? AND created_by IN ?", allow.ApplicationCode, allow.Env, []string{"project-onboarding", "deploy-profile"}).
		Where("approver_admin_id <> ?", allow.ApproverAdminID).
		Delete(&model.AgentApproverAllowlist{}).Error; err != nil {
		return err
	}
	return tx.Where("approver_admin_id = ? AND application_code = ? AND env = ?", allow.ApproverAdminID, allow.ApplicationCode, allow.Env).
		FirstOrCreate(&allow).Error
}

func requiredAgentProjectBuildParams(gitRepoURL, appCode, harborProject, harborRepository string, servicePort, targetPort int32) map[string]interface{} {
	return map[string]interface{}{
		"GIT_URL":           gitRepoURL,
		"GIT_REPO_URL":      gitRepoURL,
		"APPLICATION_CODE":  appCode,
		"HARBOR_PROJECT":    harborProject,
		"HARBOR_REPOSITORY": harborRepository,
		"SERVICE_PORT":      servicePort,
		"TARGET_PORT":       targetPort,
	}
}

func mergeAgentProjectBuildParamsJSON(raw string, required map[string]interface{}) string {
	params := mergeProfileMaps(raw, required)
	return marshalProfileMap(params)
}

func marshalProfileMap(v map[string]interface{}) string {
	if len(v) == 0 {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func defaultAgentProjectResources() map[string]interface{} {
	return map[string]interface{}{
		"requests": map[string]interface{}{
			"cpu":    "100m",
			"memory": "256Mi",
		},
		"limits": map[string]interface{}{
			"cpu":    "1000m",
			"memory": "768Mi",
		},
	}
}

func isEmptyProfileJSON(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == "" || trimmed == "{}" || trimmed == "null"
}

func parseAgentGitRepoURL(raw string) (agentGitRepo, error) {
	repo := agentGitRepo{RawURL: strings.TrimSpace(raw)}
	if repo.RawURL == "" {
		return repo, fmt.Errorf("不能为空")
	}

	if strings.Contains(repo.RawURL, "://") {
		parsed, err := url.Parse(repo.RawURL)
		if err != nil {
			return repo, err
		}
		switch strings.ToLower(parsed.Scheme) {
		case "http", "https", "ssh", "git":
		default:
			return repo, fmt.Errorf("仅支持 http/https/ssh/git 仓库地址")
		}
		if (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.User != nil {
			return repo, fmt.Errorf("URL 中不能包含用户名或密码")
		}
		if parsed.User != nil {
			if _, hasPassword := parsed.User.Password(); hasPassword {
				return repo, fmt.Errorf("URL 中不能包含密码")
			}
		}
		repo.Host = normalizeAgentGitHost(parsed.Hostname())
		repo.Path = strings.Trim(parsed.Path, "/")
	} else {
		before, path, ok := strings.Cut(repo.RawURL, ":")
		if !ok || strings.TrimSpace(path) == "" {
			return repo, fmt.Errorf("请使用 https://host/group/repo.git 或 git@host:group/repo.git 格式")
		}
		if at := strings.LastIndex(before, "@"); at >= 0 {
			before = before[at+1:]
		}
		repo.Host = normalizeAgentGitHost(before)
		repo.Path = strings.Trim(path, "/")
	}

	repo.Path = strings.TrimSuffix(repo.Path, ".git")
	if repo.Host == "" || repo.Path == "" || !strings.Contains(repo.Path, "/") {
		return repo, fmt.Errorf("必须包含 host 与 group/repo 路径")
	}
	parts := strings.Split(repo.Path, "/")
	repo.RepoName = strings.TrimSpace(parts[len(parts)-1])
	if repo.RepoName == "" {
		return repo, fmt.Errorf("无法从仓库地址解析 repo 名称")
	}
	return repo, nil
}

func sameAgentGitRepo(a, b string) bool {
	parsedA, errA := parseAgentGitRepoURL(a)
	parsedB, errB := parseAgentGitRepoURL(b)
	if errA == nil && errB == nil {
		return parsedA.Host == parsedB.Host && strings.EqualFold(parsedA.Path, parsedB.Path)
	}
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func isAgentGitHostAllowed(host string, allowed []string) bool {
	host = normalizeAgentGitHost(host)
	for _, item := range allowed {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "://") {
			if parsed, err := url.Parse(item); err == nil {
				item = parsed.Hostname()
			}
		}
		if host == normalizeAgentGitHost(item) {
			return true
		}
	}
	return false
}

func normalizeAgentGitHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.Trim(host, "[]")
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	} else if strings.Count(host, ":") == 1 {
		if h, _, ok := strings.Cut(host, ":"); ok {
			host = h
		}
	}
	return strings.Trim(host, "[]")
}

func agentServiceTypeFromExposureMode(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "clusterip":
		return "ClusterIP", nil
	case "nodeport":
		return "NodePort", nil
	case "gateway", "metallb":
		return "", fmt.Errorf("当前自动接入 MVP 仅支持 clusterip/nodeport，%s 暴露模式将在后续版本支持", strings.ToLower(strings.TrimSpace(mode)))
	default:
		return "", fmt.Errorf("exposureMode 仅支持 clusterip/nodeport（gateway/metallb 后续支持）")
	}
}

func defaultAgentProjectPort(value, fallback int32) int32 {
	if value <= 0 {
		return fallback
	}
	return value
}

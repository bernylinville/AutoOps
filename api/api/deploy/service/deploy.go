package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	appmodel "dodevops-api/api/app/model"
	"dodevops-api/api/deploy/dao"
	"dodevops-api/api/deploy/model"
	k8smodel "dodevops-api/api/k8s/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/config"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IDeployService interface {
	ListClusterTargets(c *gin.Context)
	GetClusterTarget(c *gin.Context, id uint)
	CreateClusterTarget(c *gin.Context, req *model.CreateClusterTargetRequest)
	UpdateClusterTarget(c *gin.Context, id uint, req *model.UpdateClusterTargetRequest)
	CreateDeployRequest(c *gin.Context, req *model.CreateDeployRequest)
	CreateAgentDeployRequest(c *gin.Context, req *model.CreateAgentDeployRequest)
	CreateAgentBuildDeployRequest(c *gin.Context, req *model.CreateAgentBuildDeployRequest)
	CreateAgentProjectOnboardBuildDeployRequest(c *gin.Context, req *model.CreateAgentProjectOnboardBuildDeployRequest)
	ListDeployRequests(c *gin.Context, query *model.DeployRequestListQuery)
	GetDeployRequest(c *gin.Context, id uint)
	GetDeployRequestByRequestNo(c *gin.Context, requestNo string)
	RetryApprovalDispatch(c *gin.Context, id uint)
	RetryApprovalDispatchByRequestNo(c *gin.Context, requestNo string)
	SyncApprovalStatus(c *gin.Context, id uint)
	SyncApprovalStatusByRequestNo(c *gin.Context, requestNo string)
	ValidateDirectCredential(c *gin.Context, targetID uint)
	ValidateGitOpsWorkingTree(c *gin.Context)
	ValidateGitOpsRepo(c *gin.Context, targetID uint)
	CleanupDirectRequestByID(id uint) error
	CleanupDirectRequest(c *gin.Context, id uint)
	ExecuteDeployRequest(c *gin.Context, id uint, req *model.ExecuteDeployRequest)
	ListExecutionRecords(c *gin.Context, requestID uint)
	ListNotifications(c *gin.Context, requestID uint)
	ApproveDeployRequest(c *gin.Context, id uint, req *model.ApproveDeployRequest)
	RejectDeployRequest(c *gin.Context, id uint, req *model.RejectDeployRequest)
	GetAgentStatusByRequestNo(c *gin.Context, requestNo string)
	RollbackDeployRequest(c *gin.Context, id uint)
}

type DeployService struct {
	dao              dao.IDeployDao
	db               *gorm.DB
	dingtalkApproval IDingtalkApprovalService
	notifier         IDeployNotifier
}

func newDeployService(db *gorm.DB) *DeployService {
	return &DeployService{
		dao:              dao.NewDeployDao(db),
		db:               db,
		dingtalkApproval: NewDingtalkApprovalService(),
		notifier:         NewDeployNotifier(db),
	}
}

func NewDeployService(db *gorm.DB) IDeployService {
	return newDeployService(db)
}

func (s *DeployService) ListClusterTargets(c *gin.Context) {
	targets, err := s.dao.ListClusterTargets()
	if err != nil {
		result.Failed(c, 500, "获取部署目标失败: "+err.Error())
		return
	}
	result.Success(c, targets)
}

func (s *DeployService) GetClusterTarget(c *gin.Context, id uint) {
	target, err := s.dao.GetClusterTargetByID(id)
	if err != nil {
		result.Failed(c, 404, "部署目标不存在")
		return
	}
	result.Success(c, target)
}

func (s *DeployService) CreateClusterTarget(c *gin.Context, req *model.CreateClusterTargetRequest) {
	if err := s.validateKubeCluster(req.KubeClusterID); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	if req.DefaultApproverAdminID != nil {
		if _, err := s.getApproverByID(*req.DefaultApproverAdminID); err != nil {
			result.Failed(c, 400, "默认审批人不存在")
			return
		}
	}

	target := &model.ClusterTarget{
		Name:                   strings.TrimSpace(req.Name),
		KubeClusterID:          req.KubeClusterID,
		EnvType:                defaultString(req.EnvType, "test"),
		GitOpsEnabled:          defaultBool(req.GitOpsEnabled, true),
		DirectEnabled:          defaultBool(req.DirectEnabled, true),
		HarborServerID:         req.HarborServerID,
		JenkinsServerID:        req.JenkinsServerID,
		GitOpsRepo:             req.GitOpsRepo,
		GitOpsBranch:           req.GitOpsBranch,
		GitOpsReleaseDir:       req.GitOpsReleaseDir,
		DirectNamespacePrefix:  defaultString(req.DirectNamespacePrefix, "ao-direct"),
		DefaultTTLHours:        defaultInt(req.DefaultTTLHours, 72),
		DirectKubeconfigRef:    req.DirectKubeconfigRef,
		DefaultApproverAdminID: req.DefaultApproverAdminID,
		Description:            req.Description,
	}
	if err := s.dao.CreateClusterTarget(target); err != nil {
		result.Failed(c, 500, "创建部署目标失败: "+err.Error())
		return
	}
	result.Success(c, target)
}

func (s *DeployService) UpdateClusterTarget(c *gin.Context, id uint, req *model.UpdateClusterTargetRequest) {
	if _, err := s.dao.GetClusterTargetByID(id); err != nil {
		result.Failed(c, 404, "部署目标不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.KubeClusterID != nil {
		if err := s.validateKubeCluster(*req.KubeClusterID); err != nil {
			result.Failed(c, 400, err.Error())
			return
		}
		updates["kube_cluster_id"] = *req.KubeClusterID
	}
	if req.EnvType != nil {
		updates["env_type"] = *req.EnvType
	}
	if req.GitOpsEnabled != nil {
		updates["git_ops_enabled"] = *req.GitOpsEnabled
	}
	if req.DirectEnabled != nil {
		updates["direct_enabled"] = *req.DirectEnabled
	}
	if req.GitOpsRepo != nil {
		updates["git_ops_repo"] = *req.GitOpsRepo
	}
	if req.GitOpsBranch != nil {
		updates["git_ops_branch"] = *req.GitOpsBranch
	}
	if req.GitOpsReleaseDir != nil {
		updates["git_ops_release_dir"] = *req.GitOpsReleaseDir
	}
	if req.DirectNamespacePrefix != nil {
		updates["direct_namespace_prefix"] = *req.DirectNamespacePrefix
	}
	if req.DefaultTTLHours != nil {
		updates["default_ttl_hours"] = *req.DefaultTTLHours
	}
	if req.DirectKubeconfigRef != nil {
		updates["direct_kubeconfig_ref"] = *req.DirectKubeconfigRef
	}
	if req.HarborServerID != nil {
		updates["harbor_server_id"] = *req.HarborServerID
	}
	if req.JenkinsServerID != nil {
		updates["jenkins_server_id"] = *req.JenkinsServerID
	}
	if req.DefaultApproverAdminID != nil {
		if _, err := s.getApproverByID(*req.DefaultApproverAdminID); err != nil {
			result.Failed(c, 400, "默认审批人不存在")
			return
		}
		updates["default_approver_admin_id"] = *req.DefaultApproverAdminID
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	updates["updated_at"] = time.Now()

	if err := s.dao.UpdateClusterTarget(id, updates); err != nil {
		result.Failed(c, 500, "更新部署目标失败: "+err.Error())
		return
	}
	target, err := s.dao.GetClusterTargetByID(id)
	if err != nil {
		result.Failed(c, 500, "获取更新后的部署目标失败")
		return
	}
	result.Success(c, target)
}

func (s *DeployService) CreateDeployRequest(c *gin.Context, req *model.CreateDeployRequest) {
	requestedBy, err := jwt.GetAdminId(c)
	if err != nil {
		result.Failed(c, 401, "获取当前用户失败")
		return
	}

	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, 401, "获取当前用户详情失败")
		return
	}

	s.createDeployRequestWithIdentity(c, &createDeployIdentity{
		Source:                model.DeploySourceUI,
		RequestedBy:           requestedBy,
		RequesterDisplayName:  admin.Nickname,
		RequesterExternalType: "",
		RequesterExternalID:   "",
		Request: &model.CreateDeployRequest{
			Mode:            req.Mode,
			WorkflowKind:    req.WorkflowKind,
			ResourceType:    req.ResourceType,
			ClusterTargetID: req.ClusterTargetID,
			ApplicationID:   req.ApplicationID,
			ReleaseName:     req.ReleaseName,
			Namespace:       req.Namespace,
			Image:           req.Image,
			Replicas:        req.Replicas,
			ServiceEnabled:  req.ServiceEnabled,
			ServiceType:     req.ServiceType,
			ServicePort:     req.ServicePort,
			TargetPort:      req.TargetPort,
			Env:             req.Env,
			Resources:       req.Resources,
			TTLHours:        req.TTLHours,
			ApproverAdminID: req.ApproverAdminID,
			Reason:          req.Reason,
			ChatContext:     req.ChatContext,
		},
	})
}

func (s *DeployService) CreateAgentDeployRequest(c *gin.Context, req *model.CreateAgentDeployRequest) {
	if strings.TrimSpace(req.RequesterExternalType) != "dingtalk" {
		result.Failed(c, 400, "当前仅支持 dingtalk 外部身份类型")
		return
	}
	requester, err := s.getAdminByDingtalkUserID(req.RequesterExternalID)
	if err != nil {
		result.Failed(c, 400, "该钉钉用户未绑定 AutoOps 账号")
		return
	}

	s.createDeployRequestWithIdentity(c, &createDeployIdentity{
		Source:                model.DeploySourceAgent,
		RequestedBy:           requester.ID,
		RequesterDisplayName:  defaultString(req.RequesterDisplayName, requester.Nickname),
		RequesterExternalType: strings.TrimSpace(req.RequesterExternalType),
		RequesterExternalID:   strings.TrimSpace(req.RequesterExternalID),
		Request:               createDeployRequestFromAgent(req),
	})
}

func createDeployRequestFromAgent(req *model.CreateAgentDeployRequest) *model.CreateDeployRequest {
	if req == nil {
		return nil
	}
	return &model.CreateDeployRequest{
		Mode:             req.Mode,
		WorkflowKind:     req.WorkflowKind,
		ApplicationID:    req.ApplicationID,
		ResourceType:     req.ResourceType,
		ClusterTargetID:  req.ClusterTargetID,
		ReleaseName:      req.ReleaseName,
		Namespace:        req.Namespace,
		Image:            req.Image,
		Replicas:         req.Replicas,
		ServiceEnabled:   req.ServiceEnabled,
		ServiceType:      req.ServiceType,
		ServicePort:      req.ServicePort,
		TargetPort:       req.TargetPort,
		Env:              cloneInterfaceMap(req.Env),
		Resources:        cloneInterfaceMap(req.Resources),
		TTLHours:         req.TTLHours,
		ApproverAdminID:  req.ApproverAdminID,
		Reason:           req.Reason,
		ChatContext:      cloneInterfaceMap(req.ChatContext),
		GitRef:           req.GitRef,
		BuildParams:      cloneInterfaceMap(req.BuildParams),
		JenkinsServerID:  req.JenkinsServerID,
		JenkinsJobName:   req.JenkinsJobName,
		HarborServerID:   req.HarborServerID,
		HarborProject:    req.HarborProject,
		HarborRepository: req.HarborRepository,
		ArtifactTag:      req.ArtifactTag,
		ScanPolicy:       cloneInterfaceMap(req.ScanPolicy),
	}
}

func cloneCreateDeployRequest(req *model.CreateDeployRequest) *model.CreateDeployRequest {
	if req == nil {
		return nil
	}
	return &model.CreateDeployRequest{
		Mode:             req.Mode,
		WorkflowKind:     req.WorkflowKind,
		ApplicationID:    req.ApplicationID,
		ResourceType:     req.ResourceType,
		ClusterTargetID:  req.ClusterTargetID,
		ReleaseName:      req.ReleaseName,
		Namespace:        req.Namespace,
		Image:            req.Image,
		Replicas:         req.Replicas,
		ServiceEnabled:   req.ServiceEnabled,
		ServiceType:      req.ServiceType,
		ServicePort:      req.ServicePort,
		TargetPort:       req.TargetPort,
		Env:              cloneInterfaceMap(req.Env),
		Resources:        cloneInterfaceMap(req.Resources),
		TTLHours:         req.TTLHours,
		ApproverAdminID:  req.ApproverAdminID,
		Reason:           req.Reason,
		ChatContext:      cloneInterfaceMap(req.ChatContext),
		GitRef:           req.GitRef,
		BuildParams:      cloneInterfaceMap(req.BuildParams),
		JenkinsServerID:  req.JenkinsServerID,
		JenkinsJobName:   req.JenkinsJobName,
		HarborServerID:   req.HarborServerID,
		HarborProject:    req.HarborProject,
		HarborRepository: req.HarborRepository,
		ArtifactTag:      req.ArtifactTag,
		ScanPolicy:       cloneInterfaceMap(req.ScanPolicy),
	}
}

func cloneInterfaceMap(in map[string]interface{}) map[string]interface{} {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

type createDeployIdentity struct {
	Source                string
	RequestedBy           uint
	RequesterDisplayName  string
	RequesterExternalType string
	RequesterExternalID   string
	Request               *model.CreateDeployRequest
}

func (s *DeployService) createDeployRequestWithIdentity(c *gin.Context, identity *createDeployIdentity) {
	req := identity.Request

	target, err := s.dao.GetClusterTargetByID(req.ClusterTargetID)
	if err != nil {
		result.Failed(c, 404, "部署目标不存在")
		return
	}

	approverID := req.ApproverAdminID
	if approverID == nil {
		approverID = target.DefaultApproverAdminID
	}
	if approverID == nil {
		result.Failed(c, 400, "未配置审批人")
		return
	}

	approver, err := s.getApproverByID(*approverID)
	if err != nil {
		result.Failed(c, 400, "审批人不存在")
		return
	}
	if approver.DingtalkUserID == "" {
		result.Failed(c, 400, "审批人未配置钉钉用户ID")
		return
	}

	releaseName := sanitizeName(req.ReleaseName)
	if releaseName == "" {
		result.Failed(c, 400, "发布名称无效")
		return
	}

	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		namespace = s.defaultNamespace(req.Mode, releaseName)
	}

	envJSON, _ := json.Marshal(req.Env)
	resourcesJSON, _ := json.Marshal(req.Resources)
	chatContextJSON, _ := json.Marshal(req.ChatContext)

	approvalDispatchStatus := model.ApprovalDispatchStatusPending
	approvalDispatchMessage := ""
	if strings.TrimSpace(config.Config.DingtalkApproval.ProcessCode) == "" {
		approvalDispatchStatus = model.ApprovalDispatchStatusSkipped
		approvalDispatchMessage = "钉钉原生审批流程 processCode 未配置，审批实例尚未发起"
	}

	requestNo := s.generateRequestNo()
	workflowKind := strings.TrimSpace(req.WorkflowKind)
	if workflowKind == "" {
		workflowKind = model.WorkflowKindDeployOnly
	}

	if workflowKind == model.WorkflowKindBuildDeploy {
		jenkinsServerID := defaultUint(req.JenkinsServerID)
		if jenkinsServerID == 0 {
			jenkinsServerID = defaultUint(target.JenkinsServerID)
		}
		harborServerID := defaultUint(req.HarborServerID)
		if harborServerID == 0 {
			harborServerID = defaultUint(target.HarborServerID)
		}
		if req.ApplicationID == nil || *req.ApplicationID == 0 {
			result.Failed(c, 400, "build_deploy 工作流需要指定应用ID")
			return
		}
		if strings.TrimSpace(req.GitRef) == "" {
			result.Failed(c, 400, "build_deploy 工作流需要指定 Git 引用")
			return
		}
		if jenkinsServerID == 0 {
			result.Failed(c, 400, "build_deploy 工作流需要配置 Jenkins 服务器")
			return
		}
		if harborServerID == 0 {
			result.Failed(c, 400, "build_deploy 工作流需要配置 Harbor 服务器")
			return
		}
		if strings.TrimSpace(req.HarborProject) == "" || strings.TrimSpace(req.HarborRepository) == "" {
			result.Failed(c, 400, "build_deploy 工作流需要指定 Harbor 项目和仓库")
			return
		}
	}

	deployRequest := &model.DeployRequest{
		RequestNo:                      requestNo,
		Source:                         identity.Source,
		RequesterExternalType:          identity.RequesterExternalType,
		RequesterExternalID:            identity.RequesterExternalID,
		Mode:                           req.Mode,
		WorkflowKind:                   workflowKind,
		ResourceType:                   req.ResourceType,
		ClusterTargetID:                req.ClusterTargetID,
		ApplicationID:                  req.ApplicationID,
		ReleaseName:                    releaseName,
		Namespace:                      namespace,
		Image:                          req.Image,
		Replicas:                       defaultReplicas(req.Replicas),
		ServiceEnabled:                 req.ServiceEnabled,
		ServiceType:                    req.ServiceType,
		ServicePort:                    req.ServicePort,
		TargetPort:                     req.TargetPort,
		EnvJSON:                        string(envJSON),
		ResourcesJSON:                  string(resourcesJSON),
		TTLHours:                       req.TTLHours,
		Reason:                         req.Reason,
		RequestStatus:                  model.DeployRequestStatusPendingApproval,
		ApprovalStatus:                 model.ApprovalStatusPending,
		ExecutionStatus:                model.ExecutionStatusPending,
		PipelineStatus:                 model.PipelineStatusPending,
		RequestedBy:                    identity.RequestedBy,
		RequesterDisplayName:           identity.RequesterDisplayName,
		ApproverAdminID:                approverID,
		ApproverDingtalkUserIDSnapshot: approver.DingtalkUserID,
		ApprovalChannel:                model.ApprovalChannelDingtalkOA,
		ApprovalDispatchStatus:         approvalDispatchStatus,
		ApprovalDispatchMessage:        approvalDispatchMessage,
		DingtalkProcessCodeSnapshot:    config.Config.DingtalkApproval.ProcessCode,
		ChatContextJSON:                string(chatContextJSON),
	}

	if err := s.dao.CreateDeployRequest(deployRequest); err != nil {
		result.Failed(c, 500, "创建部署申请失败: "+err.Error())
		return
	}

	if deployRequest.WorkflowKind == model.WorkflowKindBuildDeploy {
		if err := s.createPipelineRunForRequest(deployRequest, req); err != nil {
			log.Printf("[DeployService] 创建流水线运行失败 requestNo=%s err=%v", deployRequest.RequestNo, err)
			_ = s.dao.UpdateDeployRequest(deployRequest.ID, map[string]interface{}{
				"request_status":   model.DeployRequestStatusFailed,
				"execution_status": model.ExecutionStatusFailed,
				"pipeline_status":  model.PipelineStatusFailed,
				"updated_at":       time.Now(),
			})
			result.Failed(c, 500, "创建流水线运行失败: "+err.Error())
			return
		}
	}

	deployRequest = s.tryDispatchApproval(deployRequest)
	result.Success(c, deployRequest)
}

func (s *DeployService) createPipelineRunForRequest(req *model.DeployRequest, buildReq *model.CreateDeployRequest) error {
	target, err := s.dao.GetClusterTargetByID(req.ClusterTargetID)
	if err != nil {
		return fmt.Errorf("读取部署目标失败: %v", err)
	}

	pipelineSvc := NewPipelineService(s.db)
	jenkinsServerID := defaultUint(buildReq.JenkinsServerID)
	if jenkinsServerID == 0 {
		jenkinsServerID = defaultUint(target.JenkinsServerID)
	}
	harborServerID := defaultUint(buildReq.HarborServerID)
	if harborServerID == 0 {
		harborServerID = defaultUint(target.HarborServerID)
	}
	createReq := &model.CreatePipelineRunRequest{
		RequestID:        req.ID,
		ApplicationID:    defaultUint(req.ApplicationID),
		JenkinsServerID:  jenkinsServerID,
		HarborServerID:   harborServerID,
		GitRef:           strings.TrimSpace(buildReq.GitRef),
		BuildParams:      buildReq.BuildParams,
		HarborProject:    strings.TrimSpace(buildReq.HarborProject),
		HarborRepository: strings.TrimSpace(buildReq.HarborRepository),
		ArtifactTag:      strings.TrimSpace(buildReq.ArtifactTag),
		ScanPolicy:       buildReq.ScanPolicy,
	}
	if jobName := strings.TrimSpace(buildReq.JenkinsJobName); jobName != "" {
		createReq.JenkinsJobNameSnapshot = jobName
	}
	if req.ApplicationID != nil && *req.ApplicationID > 0 {
		createReq.ApplicationID = *req.ApplicationID
		if jobName := s.getJenkinsJobNameForApp(*req.ApplicationID, target.EnvType); jobName != "" && createReq.JenkinsJobNameSnapshot == "" {
			createReq.JenkinsJobNameSnapshot = jobName
		}
	}
	_, err = pipelineSvc.CreatePipelineRun(createReq)
	return err
}

func (s *DeployService) getJenkinsJobNameForApp(appID uint, envType string) string {
	if appID == 0 {
		return ""
	}
	var jenkinsEnv appmodel.JenkinsEnv
	if err := s.db.Where("app_id = ? AND env_name = ?", appID, envType).First(&jenkinsEnv).Error; err != nil {
		return ""
	}
	return strings.TrimSpace(jenkinsEnv.JobName)
}

func (s *DeployService) ListDeployRequests(c *gin.Context, query *model.DeployRequestListQuery) {
	requests, total, err := s.dao.ListDeployRequests(query)
	if err != nil {
		result.Failed(c, 500, "获取部署申请列表失败: "+err.Error())
		return
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	result.SuccessWithPage(c, requests, total, page, pageSize)
}

func (s *DeployService) GetDeployRequest(c *gin.Context, id uint) {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	result.Success(c, req)
}

func (s *DeployService) GetDeployRequestByRequestNo(c *gin.Context, requestNo string) {
	req, err := s.dao.GetDeployRequestByRequestNo(strings.TrimSpace(requestNo))
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	result.Success(c, req)
}

func (s *DeployService) RetryApprovalDispatch(c *gin.Context, id uint) {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	s.retryApprovalDispatchInternal(c, req)
}

func (s *DeployService) RetryApprovalDispatchByRequestNo(c *gin.Context, requestNo string) {
	req, err := s.dao.GetDeployRequestByRequestNo(strings.TrimSpace(requestNo))
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	s.retryApprovalDispatchInternal(c, req)
}

func (s *DeployService) SyncApprovalStatus(c *gin.Context, id uint) {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	s.syncApprovalStatusInternal(c, req)
}

func (s *DeployService) SyncApprovalStatusByRequestNo(c *gin.Context, requestNo string) {
	req, err := s.dao.GetDeployRequestByRequestNo(strings.TrimSpace(requestNo))
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	s.syncApprovalStatusInternal(c, req)
}

func (s *DeployService) ValidateDirectCredential(c *gin.Context, targetID uint) {
	target, err := s.dao.GetClusterTargetByID(targetID)
	if err != nil {
		result.Failed(c, 404, "部署目标不存在")
		return
	}
	if strings.TrimSpace(target.DirectKubeconfigRef) == "" {
		result.Failed(c, 400, "部署目标未配置 direct_kubeconfig_ref")
		return
	}
	validation, err := ValidateDirectKubeconfigAccess(target.DirectKubeconfigRef, target.DirectNamespacePrefix+"-probe")
	if err != nil {
		result.Failed(c, 400, validation.Message)
		return
	}
	result.Success(c, validation)
}

func (s *DeployService) ValidateGitOpsWorkingTree(c *gin.Context) {
	validation, err := ValidateGitOpsWorkingTree()
	if err != nil {
		result.Failed(c, 400, validation.Message)
		return
	}
	result.Success(c, validation)
}

func (s *DeployService) ValidateGitOpsRepo(c *gin.Context, targetID uint) {
	target, err := s.dao.GetClusterTargetByID(targetID)
	if err != nil {
		result.Failed(c, 404, "部署目标不存在")
		return
	}
	validation, err := ValidateGitOpsRepoState(target.GitOpsBranch)
	if err != nil {
		result.Failed(c, 400, validation.Message)
		return
	}
	result.Success(c, validation)
}

func (s *DeployService) CleanupDirectRequest(c *gin.Context, id uint) {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	if err := s.cleanupDirectRequestInternal(req); err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	updated, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 500, "获取更新后的部署申请失败")
		return
	}
	result.Success(c, updated)
}

func (s *DeployService) CleanupDirectRequestByID(id uint) error {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		return err
	}
	return s.cleanupDirectRequestInternal(req)
}

func (s *DeployService) ExecuteDeployRequest(c *gin.Context, id uint, req *model.ExecuteDeployRequest) {
	deployRequest, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	if deployRequest.WorkflowKind == model.WorkflowKindBuildDeploy {
		result.Failed(c, 400, "build_deploy 工作流必须通过流水线执行，禁止直接部署")
		return
	}
	updated, record, err := s.executeDeployRequestInternal(deployRequest, req.Comment, false)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}
	result.Success(c, map[string]interface{}{
		"request":   updated,
		"execution": record,
	})
}

func AutoExecuteApprovedDeployRequest(db *gorm.DB, requestID uint, comment string) (*model.DeployRequest, *model.ExecutionRecord, error) {
	service := newDeployService(db)
	deployRequest, err := service.dao.GetDeployRequestByID(requestID)
	if err != nil {
		return nil, nil, err
	}
	if deployRequest.WorkflowKind == model.WorkflowKindBuildDeploy {
		return nil, nil, fmt.Errorf("build_deploy 工作流必须通过流水线执行")
	}
	return service.executeDeployRequestInternal(deployRequest, comment, false)
}

func (s *DeployService) executeDeployRequestInternal(deployRequest *model.DeployRequest, comment string, skipNotification bool) (*model.DeployRequest, *model.ExecutionRecord, error) {
	if deployRequest == nil {
		return nil, nil, fmt.Errorf("部署申请不存在")
	}
	if deployRequest.ApprovalStatus != model.ApprovalStatusApproved {
		return nil, nil, fmt.Errorf("部署申请未审批通过，不能执行")
	}
	if deployRequest.ExecutionStatus == model.ExecutionStatusRunning {
		return nil, nil, fmt.Errorf("部署申请已在执行中")
	}
	if deployRequest.ExecutionStatus == model.ExecutionStatusSucceeded {
		return nil, nil, fmt.Errorf("部署申请已执行成功")
	}

	now := time.Now()
	detailJSON := executionDetailJSON(comment)
	if deployRequest.Mode == model.DeployModeDirect {
		rendered, err := RenderDirectManifest(deployRequest)
		if err != nil {
			return nil, nil, fmt.Errorf("渲染 direct manifest 失败: %v", err)
		}
		detailJSON = executionDetailWithManifest(comment, rendered.YAML)
	}
	if deployRequest.Mode == model.DeployModeGitOps {
		target, err := s.dao.GetClusterTargetByID(deployRequest.ClusterTargetID)
		if err != nil {
			return nil, nil, fmt.Errorf("读取部署目标失败: %v", err)
		}
		filePath, content, err := RenderGitOpsReleaseFile(deployRequest, target.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("渲染 GitOps release 失败: %v", err)
		}
		detailJSON = executionDetailWithGitOpsPreview(comment, filePath, content)
	}
	started, err := s.dao.TryStartExecution(deployRequest.ID, now)
	if err != nil {
		return nil, nil, fmt.Errorf("抢占部署执行失败: %v", err)
	}
	if !started {
		latest, latestErr := s.dao.GetDeployRequestByID(deployRequest.ID)
		if latestErr == nil && latest != nil {
			switch latest.ExecutionStatus {
			case model.ExecutionStatusRunning:
				return nil, nil, fmt.Errorf("部署申请已在执行中")
			case model.ExecutionStatusSucceeded:
				return nil, nil, fmt.Errorf("部署申请已执行成功")
			}
		}
		return nil, nil, fmt.Errorf("部署申请已被其他执行流接管")
	}
	if err := s.reserveResourceOwners(deployRequest); err != nil {
		_ = s.dao.UpdateDeployRequest(deployRequest.ID, map[string]interface{}{
			"request_status":   deployRequest.RequestStatus,
			"execution_status": deployRequest.ExecutionStatus,
			"started_at":       deployRequest.StartedAt,
			"updated_at":       time.Now(),
		})
		return nil, nil, err
	}

	finalExecutionStatus := model.ExecutionStatusRunning
	finalRequestStatus := model.DeployRequestStatusExecuting
	var finishedAt *time.Time
	if deployRequest.Mode == model.DeployModeDirect {
		target, err := s.dao.GetClusterTargetByID(deployRequest.ClusterTargetID)
		if err != nil {
			return nil, nil, fmt.Errorf("读取部署目标失败: %v", err)
		}
		applyResult, err := ApplyDirectResources(deployRequest, target.DirectKubeconfigRef)
		doneAt := time.Now()
		finishedAt = &doneAt
		if err != nil {
			finalExecutionStatus = model.ExecutionStatusFailed
			finalRequestStatus = model.DeployRequestStatusFailed
			detailJSON = executionDetailWithError(comment, detailJSON, err.Error())
		} else {
			finalExecutionStatus = model.ExecutionStatusSucceeded
			finalRequestStatus = model.DeployRequestStatusSucceeded
			detailJSON = executionDetailWithDirectApplyResult(comment, detailJSON, applyResult)
		}
	}
	if deployRequest.Mode == model.DeployModeGitOps {
		target, err := s.dao.GetClusterTargetByID(deployRequest.ClusterTargetID)
		if err != nil {
			return nil, nil, fmt.Errorf("读取部署目标失败: %v", err)
		}
		writeResult, err := WriteGitOpsReleaseToWorkingTree(deployRequest, target.Name, target.GitOpsReleaseDir)
		doneAt := time.Now()
		finishedAt = &doneAt
		if err != nil {
			finalExecutionStatus = model.ExecutionStatusFailed
			finalRequestStatus = model.DeployRequestStatusFailed
			detailJSON = executionDetailWithError(comment, detailJSON, err.Error())
		} else {
			commitResult, commitErr := CommitGitOpsWorkingTree(target.GitOpsBranch, writeResult.FilePath, deployRequest.RequestNo)
			if commitErr != nil {
				finalExecutionStatus = model.ExecutionStatusFailed
				finalRequestStatus = model.DeployRequestStatusFailed
				detailJSON = executionDetailWithError(comment, detailJSON, commitErr.Error())
			} else {
				pushResult, pushErr := PushGitOpsBranch(target.GitOpsBranch)
				if pushErr != nil {
					finalExecutionStatus = model.ExecutionStatusFailed
					finalRequestStatus = model.DeployRequestStatusFailed
					detailJSON = executionDetailWithError(comment, detailJSON, pushErr.Error())
				} else {
					finalExecutionStatus = model.ExecutionStatusSucceeded
					finalRequestStatus = model.DeployRequestStatusSucceeded
					detailJSON = executionDetailWithGitOpsWriteCommitPushResult(comment, detailJSON, writeResult, commitResult, pushResult)
				}
			}
		}
	}

	record := &model.ExecutionRecord{
		RequestID:    deployRequest.ID,
		ExecutorType: deployRequest.Mode,
		Phase:        model.ExecutionPhaseQueued,
		Status:       finalExecutionStatus,
		DetailJSON:   detailJSON,
		K8sNamespace: deployRequest.Namespace,
		StartedAt:    &now,
		EndedAt:      finishedAt,
	}
	if err := s.dao.CreateExecutionRecord(record); err != nil {
		return nil, nil, fmt.Errorf("创建执行记录失败: %v", err)
	}

	updates := map[string]interface{}{
		"request_status":   finalRequestStatus,
		"execution_status": finalExecutionStatus,
		"updated_at":       now,
	}
	if finishedAt != nil {
		updates["finished_at"] = *finishedAt
	}
	if err := s.dao.UpdateDeployRequest(deployRequest.ID, updates); err != nil {
		return nil, nil, fmt.Errorf("更新部署申请执行状态失败: %v", err)
	}
	if finalExecutionStatus == model.ExecutionStatusFailed {
		_ = s.dao.DeactivateResourceOwnersByRequestID(deployRequest.ID)
	}

	updated, err := s.dao.GetDeployRequestByID(deployRequest.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("获取更新后的部署申请失败")
	}
	if !skipNotification && s.notifier != nil {
		if notifyErr := s.notifier.NotifyExecutionResult(updated, record); notifyErr != nil {
			log.Printf("[DeployNotifier] requestNo=%s notify failed: %v", updated.RequestNo, notifyErr)
		}
	}
	return updated, record, nil
}

func (s *DeployService) ListExecutionRecords(c *gin.Context, requestID uint) {
	if _, err := s.dao.GetDeployRequestByID(requestID); err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	records, err := s.dao.ListExecutionRecordsByRequestID(requestID)
	if err != nil {
		result.Failed(c, 500, "获取执行记录失败: "+err.Error())
		return
	}
	result.Success(c, records)
}

func (s *DeployService) ListNotifications(c *gin.Context, requestID uint) {
	if _, err := s.dao.GetDeployRequestByID(requestID); err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	notifications, err := s.dao.ListNotificationsByRequestID(requestID)
	if err != nil {
		result.Failed(c, 500, "获取通知记录失败: "+err.Error())
		return
	}
	result.Success(c, notifications)
}

func (s *DeployService) ApproveDeployRequest(c *gin.Context, id uint, req *model.ApproveDeployRequest) {
	s.handleApprovalAction(c, id, "approve", req.Comment)
}

func (s *DeployService) RejectDeployRequest(c *gin.Context, id uint, req *model.RejectDeployRequest) {
	s.handleApprovalAction(c, id, "reject", req.Comment)
}

func (s *DeployService) GetAgentStatusByRequestNo(c *gin.Context, requestNo string) {
	req, err := s.dao.GetDeployRequestByRequestNo(strings.TrimSpace(requestNo))
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}

	// Only surface an error message when the request is currently in a failed state
	// to avoid returning stale errors from a previous attempt that is being retried.
	var errMsg string
	if req.ExecutionStatus == model.ExecutionStatusFailed {
		record, recordErr := s.dao.GetLatestExecutionRecordByRequestID(req.ID)
		if recordErr == nil {
			errMsg = execErrorMessage(record)
		}
	}

	var accessInfo *AccessInfo
	if req.ExecutionStatus == model.ExecutionStatusSucceeded {
		record, recordErr := s.dao.GetLatestExecutionRecordByRequestID(req.ID)
		if recordErr == nil {
			accessInfo = buildAccessInfo(req, directApplyResultFromExecution(record))
		} else {
			accessInfo = buildAccessInfo(req)
		}
	}

	result.Success(c, map[string]interface{}{
		"requestNo":        req.RequestNo,
		"requestStatus":    req.RequestStatus,
		"approvalStatus":   req.ApprovalStatus,
		"executionStatus":  req.ExecutionStatus,
		"finishedAt":       req.FinishedAt,
		"executionSummary": buildExecutionSummary(req),
		"accessInfo":       accessInfo,
		"errorMessage":     errMsg,
	})
}

func (s *DeployService) RollbackDeployRequest(c *gin.Context, id uint) {
	req, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	if req.Mode != model.DeployModeGitOps {
		result.Failed(c, 400, "仅 gitops 模式支持下线")
		return
	}
	if req.ExecutionStatus != model.ExecutionStatusSucceeded {
		result.Failed(c, 400, "仅已执行成功的申请支持下线")
		return
	}

	target, err := s.dao.GetClusterTargetByID(req.ClusterTargetID)
	if err != nil {
		result.Failed(c, 400, "读取部署目标失败: "+err.Error())
		return
	}

	now := time.Now()
	deleteResult, deleteErr := DeleteGitOpsRelease(req, target.GitOpsReleaseDir, target.GitOpsBranch)
	record := &model.ExecutionRecord{
		RequestID:    req.ID,
		ExecutorType: req.Mode,
		Phase:        model.ExecutionPhaseRollback,
		Status:       model.ExecutionStatusRolledBack,
		K8sNamespace: req.Namespace,
		StartedAt:    &now,
		EndedAt:      &now,
	}
	if deleteErr != nil {
		record.Status = model.ExecutionStatusFailed
		record.DetailJSON = executionDetailWithError("rollback", "", deleteErr.Error())
	} else {
		record.GitCommitSHA = deleteResult.CommitSHA
		record.GitFilePath = deleteResult.FilePath
		record.DetailJSON = executionDetailWithGitOpsWriteCommitPushResult("rollback", "", &GitOpsWriteResult{
			RepoPath: deleteResult.RepoPath,
			FilePath: deleteResult.FilePath,
			Written:  false,
			Message:  deleteResult.Message,
		}, &GitOpsCommitResult{
			RepoPath:  deleteResult.RepoPath,
			Branch:    deleteResult.Branch,
			CommitSHA: deleteResult.CommitSHA,
			Message:   deleteResult.Message,
		}, &GitOpsPushResult{
			RepoPath: deleteResult.RepoPath,
			Branch:   deleteResult.Branch,
			Message:  deleteResult.Message,
		})
	}
	if err := s.dao.CreateExecutionRecord(record); err != nil {
		result.Failed(c, 500, "创建回滚执行记录失败: "+err.Error())
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if deleteErr == nil {
		updates["request_status"] = model.DeployRequestStatusRolledBack
		updates["execution_status"] = model.ExecutionStatusRolledBack
		updates["finished_at"] = time.Now()
		if err := s.dao.DeactivateResourceOwnersByRequestID(req.ID); err != nil {
			result.Failed(c, 500, "释放资源 owner 失败: "+err.Error())
			return
		}
	}
	if err := s.dao.UpdateDeployRequest(req.ID, updates); err != nil {
		result.Failed(c, 500, "更新部署申请状态失败: "+err.Error())
		return
	}
	updated, err := s.dao.GetDeployRequestByID(req.ID)
	if err != nil {
		result.Failed(c, 500, "获取更新后的部署申请失败")
		return
	}
	if s.notifier != nil {
		if notifyErr := s.notifier.NotifyExecutionResult(updated, record); notifyErr != nil {
			log.Printf("[DeployNotifier] rollback notify failed requestNo=%s err=%v", updated.RequestNo, notifyErr)
		}
	}
	if deleteErr != nil {
		result.Failed(c, 500, "GitOps 下线失败: "+deleteErr.Error())
		return
	}
	result.Success(c, map[string]interface{}{
		"request":   updated,
		"execution": record,
		"result":    deleteResult,
	})
}

func (s *DeployService) handleApprovalAction(c *gin.Context, id uint, action, comment string) {
	admin, err := jwt.GetAdmin(c)
	if err != nil {
		result.Failed(c, 401, "获取当前用户失败")
		return
	}

	deployRequest, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 404, "部署申请不存在")
		return
	}
	if deployRequest.ApprovalStatus != model.ApprovalStatusPending {
		result.Failed(c, 400, "当前申请不处于待审批状态")
		return
	}
	if deployRequest.ApproverAdminID == nil || *deployRequest.ApproverAdminID != admin.ID {
		result.Failed(c, 403, "当前用户不是该申请的审批人")
		return
	}

	now := time.Now()
	record := &model.ApprovalRecord{
		RequestID:       deployRequest.ID,
		ApproverAdminID: deployRequest.ApproverAdminID,
		ApproverName:    admin.Nickname,
		Source:          model.ApprovalSourceUI,
		Action:          action,
		Comment:         comment,
		ActedAt:         now,
	}
	if err := s.dao.CreateApprovalRecord(record); err != nil {
		result.Failed(c, 500, "写入审批记录失败: "+err.Error())
		return
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if action == "approve" {
		updates["approval_status"] = model.ApprovalStatusApproved
		updates["request_status"] = model.DeployRequestStatusApproved
		updates["approved_at"] = now
	} else {
		updates["approval_status"] = model.ApprovalStatusRejected
		updates["request_status"] = model.DeployRequestStatusRejected
		updates["rejected_at"] = now
	}

	if err := s.dao.UpdateDeployRequest(id, updates); err != nil {
		result.Failed(c, 500, "更新部署申请状态失败: "+err.Error())
		return
	}

	updated, err := s.dao.GetDeployRequestByID(id)
	if err != nil {
		result.Failed(c, 500, "获取更新后的部署申请失败")
		return
	}
	result.Success(c, updated)
}

func (s *DeployService) getApproverByID(id uint) (*systemmodel.SysAdmin, error) {
	var admin systemmodel.SysAdmin
	if err := s.db.First(&admin, id).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (s *DeployService) getAdminByDingtalkUserID(dingtalkUserID string) (*systemmodel.SysAdmin, error) {
	var admin systemmodel.SysAdmin
	if err := s.db.Where("dingtalk_user_id = ?", strings.TrimSpace(dingtalkUserID)).First(&admin).Error; err != nil {
		return nil, err
	}
	return &admin, nil
}

func (s *DeployService) tryDispatchApproval(deployRequest *model.DeployRequest) *model.DeployRequest {
	if deployRequest == nil {
		return deployRequest
	}
	processCode := strings.TrimSpace(config.Config.DingtalkApproval.ProcessCode)
	if processCode == "" {
		return deployRequest
	}

	requester, err := s.getApproverByID(deployRequest.RequestedBy)
	if err != nil || strings.TrimSpace(requester.DingtalkUserID) == "" {
		return s.markApprovalDispatch(deployRequest, model.ApprovalDispatchStatusFailed, "申请人未配置钉钉用户ID，无法发起审批实例", "")
	}
	if strings.TrimSpace(deployRequest.ApproverDingtalkUserIDSnapshot) == "" {
		return s.markApprovalDispatch(deployRequest, model.ApprovalDispatchStatusFailed, "approver dingtalk_user_id empty", "")
	}

	formValues, err := s.buildApprovalFormValues(deployRequest)
	if err != nil {
		return s.markApprovalDispatch(deployRequest, model.ApprovalDispatchStatusSkipped, err.Error(), "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := s.dingtalkApproval.CreateProcessInstance(ctx, &DingtalkCreateProcessInstanceRequest{
		ProcessCode:         processCode,
		OriginatorUserID:    requester.DingtalkUserID,
		DeptID:              config.Config.DingtalkApproval.OriginatorDeptID,
		Approvers:           buildDingtalkApprovers(deployRequest.ApproverDingtalkUserIDSnapshot),
		FormComponentValues: formValues,
	})
	if err != nil {
		return s.markApprovalDispatch(deployRequest, model.ApprovalDispatchStatusFailed, "发起钉钉审批实例失败: "+err.Error(), "")
	}

	return s.markApprovalDispatch(deployRequest, model.ApprovalDispatchStatusDispatched, "已发起钉钉审批实例", resp.ProcessInstanceID)
}

func (s *DeployService) syncApprovalStatusInternal(c *gin.Context, deployRequest *model.DeployRequest) {
	if strings.TrimSpace(deployRequest.DingtalkProcessInstanceID) == "" {
		result.Failed(c, 400, "当前申请未关联钉钉审批实例")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	detail, err := s.dingtalkApproval.GetProcessInstance(ctx, deployRequest.DingtalkProcessInstanceID)
	if err != nil {
		result.Failed(c, 500, "查询钉钉审批实例失败: "+err.Error())
		return
	}

	approvalStatus, requestStatus, actedAt := mapApprovalFromDingtalk(detail)
	updates := map[string]interface{}{
		"updated_at":                time.Now(),
		"approval_dispatch_status":  model.ApprovalDispatchStatusDispatched,
		"approval_dispatch_message": fmt.Sprintf("钉钉审批实例状态: status=%s result=%s", detail.Status, detail.Result),
	}
	if approvalStatus != "" {
		updates["approval_status"] = approvalStatus
	}
	if requestStatus != "" {
		updates["request_status"] = requestStatus
	}
	if approvalStatus == model.ApprovalStatusApproved && actedAt != nil {
		updates["approved_at"] = *actedAt
	}
	if approvalStatus == model.ApprovalStatusRejected && actedAt != nil {
		updates["rejected_at"] = *actedAt
	}

	if err := s.dao.UpdateDeployRequest(deployRequest.ID, updates); err != nil {
		result.Failed(c, 500, "回写审批状态失败: "+err.Error())
		return
	}

	updated, err := s.dao.GetDeployRequestByID(deployRequest.ID)
	if err != nil {
		result.Failed(c, 500, "获取更新后的部署申请失败")
		return
	}
	var execution *model.ExecutionRecord
	if updated.ApprovalStatus == model.ApprovalStatusApproved && updated.ExecutionStatus == model.ExecutionStatusPending {
		if updated.WorkflowKind == model.WorkflowKindBuildDeploy {
			log.Printf("[DeployService] build_deploy 工作流审批通过，交由流水线调度器执行 requestNo=%s", updated.RequestNo)
		} else {
			executionComment := "auto execute after dingtalk approval sync"
			updated, execution, err = s.executeDeployRequestInternal(updated, executionComment, false)
			if err != nil {
				result.Failed(c, 500, "审批已通过但自动执行失败: "+err.Error())
				return
			}
		}
	}
	result.Success(c, map[string]interface{}{
		"request":   updated,
		"execution": execution,
		"dingtalk":  detail,
	})
}

func (s *DeployService) retryApprovalDispatchInternal(c *gin.Context, deployRequest *model.DeployRequest) {
	if deployRequest.ApprovalStatus != model.ApprovalStatusPending {
		result.Failed(c, 400, "当前申请不处于待审批状态，不能重发审批")
		return
	}
	if deployRequest.ApprovalDispatchStatus == model.ApprovalDispatchStatusDispatched {
		result.Failed(c, 400, "当前申请已成功发起审批实例")
		return
	}

	updated := s.tryDispatchApproval(deployRequest)
	result.Success(c, updated)
}

func (s *DeployService) reserveResourceOwners(req *model.DeployRequest) error {
	owners := []model.ResourceOwner{}
	switch req.Mode {
	case model.DeployModeDirect:
		owners = append(owners, model.ResourceOwner{
			ClusterTargetID: req.ClusterTargetID,
			Namespace:       req.Namespace,
			Kind:            strings.Title(req.ResourceType),
			Name:            req.ReleaseName,
			OwnerSystem:     model.ResourceOwnerSystemDirect,
			RequestID:       req.ID,
			ReleaseName:     req.ReleaseName,
			Active:          true,
		})
		if req.ServiceEnabled {
			owners = append(owners, model.ResourceOwner{
				ClusterTargetID: req.ClusterTargetID,
				Namespace:       req.Namespace,
				Kind:            "Service",
				Name:            req.ReleaseName,
				OwnerSystem:     model.ResourceOwnerSystemDirect,
				RequestID:       req.ID,
				ReleaseName:     req.ReleaseName,
				Active:          true,
			})
		}
	case model.DeployModeGitOps:
		owners = append(owners, model.ResourceOwner{
			ClusterTargetID: req.ClusterTargetID,
			Namespace:       req.Namespace,
			Kind:            "Deployment",
			Name:            req.ReleaseName,
			OwnerSystem:     model.ResourceOwnerSystemGitOps,
			RequestID:       req.ID,
			ReleaseName:     req.ReleaseName,
			Active:          true,
		})
		if req.ServiceEnabled {
			owners = append(owners, model.ResourceOwner{
				ClusterTargetID: req.ClusterTargetID,
				Namespace:       req.Namespace,
				Kind:            "Service",
				Name:            req.ReleaseName,
				OwnerSystem:     model.ResourceOwnerSystemGitOps,
				RequestID:       req.ID,
				ReleaseName:     req.ReleaseName,
				Active:          true,
			})
		}
	}

	for _, owner := range owners {
		existing, err := s.dao.GetActiveResourceOwner(owner.ClusterTargetID, owner.Namespace, owner.Kind, owner.Name)
		if err == nil {
			if existing.RequestID != req.ID || existing.OwnerSystem != owner.OwnerSystem {
				return fmt.Errorf("资源已被其他 owner 占用: %s/%s %s (%s)", owner.Namespace, owner.Name, owner.Kind, existing.OwnerSystem)
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("检查资源 owner 失败: %v", err)
		}
		if err := s.dao.CreateResourceOwner(&owner); err != nil {
			return fmt.Errorf("创建资源 owner 失败: %v", err)
		}
	}
	return nil
}

func (s *DeployService) cleanupDirectRequestInternal(req *model.DeployRequest) error {
	if req == nil {
		return fmt.Errorf("部署申请不能为空")
	}
	if req.Mode != model.DeployModeDirect {
		return fmt.Errorf("仅 direct mode 支持清理")
	}
	if req.ExecutionStatus == model.ExecutionStatusCleaned {
		return fmt.Errorf("当前申请已清理")
	}

	target, err := s.dao.GetClusterTargetByID(req.ClusterTargetID)
	if err != nil {
		return fmt.Errorf("读取部署目标失败: %v", err)
	}
	clientset, err := NewDirectKubeClient(target.DirectKubeconfigRef)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = clientset.CoreV1().Namespaces().Delete(ctx, req.Namespace, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("删除 direct namespace 失败: %v", err)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"request_status":   model.DeployRequestStatusExpired,
		"execution_status": model.ExecutionStatusCleaned,
		"finished_at":      now,
		"updated_at":       now,
	}
	if err := s.dao.UpdateDeployRequest(req.ID, updates); err != nil {
		return fmt.Errorf("更新 direct 清理状态失败: %v", err)
	}
	if err := s.dao.DeactivateResourceOwnersByRequestID(req.ID); err != nil {
		return fmt.Errorf("释放资源 owner 失败: %v", err)
	}
	return nil
}

func (s *DeployService) buildApprovalFormValues(deployRequest *model.DeployRequest) ([]DingtalkFormComponentValue, error) {
	mappings := config.Config.DingtalkApproval.FieldMappings
	if strings.TrimSpace(mappings.RequestNo) == "" ||
		strings.TrimSpace(mappings.ReleaseName) == "" ||
		strings.TrimSpace(mappings.ClusterTarget) == "" ||
		strings.TrimSpace(mappings.DeployMode) == "" ||
		strings.TrimSpace(mappings.ResourceType) == "" ||
		strings.TrimSpace(mappings.Image) == "" ||
		strings.TrimSpace(mappings.Namespace) == "" ||
		strings.TrimSpace(mappings.Reason) == "" {
		return nil, fmt.Errorf("钉钉审批表单字段映射未配置完整，审批实例尚未发起")
	}

	target, err := s.dao.GetClusterTargetByID(deployRequest.ClusterTargetID)
	if err != nil {
		return nil, fmt.Errorf("读取部署目标失败")
	}

	pipelineRun := s.pipelineRunForApproval(deployRequest)
	values := []DingtalkFormComponentValue{
		{Name: mappings.RequestNo, Value: deployRequest.RequestNo},
		{Name: mappings.ReleaseName, Value: deployRequest.ReleaseName},
		{Name: mappings.ClusterTarget, Value: target.Name},
		{Name: mappings.DeployMode, Value: deployRequest.Mode},
		{Name: mappings.ResourceType, Value: deployRequest.ResourceType},
		{Name: mappings.Image, Value: approvalImageFormValue(deployRequest, pipelineRun)},
		{Name: mappings.Namespace, Value: deployRequest.Namespace},
		{Name: mappings.Reason, Value: deployRequest.Reason},
	}
	if strings.TrimSpace(mappings.TTLHours) != "" && deployRequest.TTLHours != nil {
		values = append(values, DingtalkFormComponentValue{
			Name:  mappings.TTLHours,
			Value: strconv.Itoa(*deployRequest.TTLHours),
		})
	}
	return values, nil
}

func (s *DeployService) pipelineRunForApproval(deployRequest *model.DeployRequest) *model.PipelineRun {
	if deployRequest == nil ||
		deployRequest.WorkflowKind != model.WorkflowKindBuildDeploy ||
		deployRequest.ID == 0 ||
		s.db == nil {
		return nil
	}

	pipelineRun, err := dao.NewPipelineDao(s.db).GetPipelineRunByRequestID(deployRequest.ID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[DeployService] 读取审批流水线信息失败 requestNo=%s err=%v", deployRequest.RequestNo, err)
		}
		return nil
	}
	return pipelineRun
}

func approvalImageFormValue(deployRequest *model.DeployRequest, pipelineRun *model.PipelineRun) string {
	if deployRequest == nil {
		return ""
	}
	if image := strings.TrimSpace(deployRequest.Image); image != "" {
		return image
	}
	if deployRequest.WorkflowKind != model.WorkflowKindBuildDeploy {
		return ""
	}

	details := []string{"构建后由 Jenkins 生成"}
	if pipelineRun != nil {
		repository := strings.Trim(strings.TrimSpace(pipelineRun.HarborProject)+"/"+strings.TrimSpace(pipelineRun.HarborRepository), "/")
		if repository != "" {
			details = append(details, "Harbor: "+repository)
		}
		if gitRef := strings.TrimSpace(pipelineRun.GitRef); gitRef != "" {
			details = append(details, "Git: "+gitRef)
		}
	}
	return strings.Join(details, "；")
}

func (s *DeployService) markApprovalDispatch(deployRequest *model.DeployRequest, status, message, processInstanceID string) *model.DeployRequest {
	updates := map[string]interface{}{
		"approval_dispatch_status":       status,
		"approval_dispatch_message":      message,
		"dingtalk_process_code_snapshot": config.Config.DingtalkApproval.ProcessCode,
		"updated_at":                     time.Now(),
	}
	if processInstanceID != "" {
		updates["dingtalk_process_instance_id"] = processInstanceID
	}
	_ = s.dao.UpdateDeployRequest(deployRequest.ID, updates)
	if updated, err := s.dao.GetDeployRequestByID(deployRequest.ID); err == nil {
		return updated
	}
	return deployRequest
}

func (s *DeployService) validateKubeCluster(id uint) error {
	var cluster k8smodel.KubeCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		return fmt.Errorf("关联K8s集群不存在")
	}
	return nil
}

func (s *DeployService) generateRequestNo() string {
	return fmt.Sprintf("DR%s", time.Now().Format("20060102150405.000000000"))
}

func (s *DeployService) defaultNamespace(mode, releaseName string) string {
	if mode == model.DeployModeGitOps {
		return "ao-gitops-" + releaseName
	}
	return "ao-direct-" + releaseName
}

func defaultReplicas(replicas int32) int32 {
	if replicas <= 0 {
		return 1
	}
	return replicas
}

func defaultBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultInt(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultUint(value *uint) uint {
	if value == nil {
		return 0
	}
	return *value
}

func executionDetailJSON(comment string) string {
	detail := map[string]string{
		"comment": comment,
		"note":    "execution queued; real executor is not wired yet",
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithManifest(comment, manifest string) string {
	detail := map[string]string{
		"comment":  comment,
		"note":     "direct manifest rendered; real executor is not wired yet",
		"manifest": manifest,
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithGitOpsPreview(comment, filePath, content string) string {
	detail := map[string]string{
		"comment":  comment,
		"note":     "gitops release rendered; repo writer is not wired yet",
		"filePath": filePath,
		"content":  content,
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithGitOpsWriteResult(comment, preview string, writeResult *GitOpsWriteResult) string {
	resultJSON, _ := json.Marshal(writeResult)
	detail := map[string]string{
		"comment": comment,
		"note":    "gitops release written to local working tree",
		"preview": preview,
		"result":  string(resultJSON),
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithGitOpsWriteAndCommitResult(comment, preview string, writeResult *GitOpsWriteResult, commitResult *GitOpsCommitResult) string {
	writeJSON, _ := json.Marshal(writeResult)
	commitJSON, _ := json.Marshal(commitResult)
	detail := map[string]string{
		"comment": comment,
		"note":    "gitops release written and committed to local working tree",
		"preview": preview,
		"write":   string(writeJSON),
		"commit":  string(commitJSON),
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithGitOpsWriteCommitPushResult(comment, preview string, writeResult *GitOpsWriteResult, commitResult *GitOpsCommitResult, pushResult *GitOpsPushResult) string {
	writeJSON, _ := json.Marshal(writeResult)
	commitJSON, _ := json.Marshal(commitResult)
	pushJSON, _ := json.Marshal(pushResult)
	detail := map[string]string{
		"comment": comment,
		"note":    "gitops release written, committed and pushed",
		"preview": preview,
		"write":   string(writeJSON),
		"commit":  string(commitJSON),
		"push":    string(pushJSON),
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithDirectApplyResult(comment, preview string, applyResult *DirectApplyResult) string {
	appliedJSON, _ := json.Marshal(applyResult)
	detail := map[string]string{
		"comment": comment,
		"note":    "direct resources applied",
		"preview": preview,
		"result":  string(appliedJSON),
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func executionDetailWithError(comment, preview, errMsg string) string {
	detail := map[string]string{
		"comment": comment,
		"note":    "execution failed",
		"preview": preview,
		"error":   errMsg,
	}
	data, _ := json.Marshal(detail)
	return string(data)
}

func mapApprovalFromDingtalk(detail *DingtalkProcessInstanceDetailResponse) (string, string, *time.Time) {
	if detail == nil {
		return "", "", nil
	}
	status := strings.ToUpper(strings.TrimSpace(detail.Status))
	resultValue := strings.ToUpper(strings.TrimSpace(detail.Result))
	now := time.Now()

	if resultValue == "AGREE" || resultValue == "APPROVED" || resultValue == "PASS" {
		return model.ApprovalStatusApproved, model.DeployRequestStatusApproved, &now
	}
	if resultValue == "REFUSE" || resultValue == "REFUSED" || resultValue == "REJECT" || resultValue == "REJECTED" {
		return model.ApprovalStatusRejected, model.DeployRequestStatusRejected, &now
	}

	if status == "COMPLETED" && resultValue == "" {
		return model.ApprovalStatusApproved, model.DeployRequestStatusApproved, &now
	}
	if status == "TERMINATED" || status == "CANCELED" || status == "CANCELLED" {
		return model.ApprovalStatusRejected, model.DeployRequestStatusRejected, &now
	}
	return "", "", nil
}

func buildDingtalkApprovers(userID string) []DingtalkApprovalNode {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}
	return []DingtalkApprovalNode{
		{
			ActionType: "AND",
			UserIDs:    []string{userID},
		},
	}
}

func buildExecutionSummary(req *model.DeployRequest) string {
	if req == nil {
		return ""
	}
	if req.ExecutionStatus == model.ExecutionStatusSucceeded {
		switch req.Mode {
		case model.DeployModeGitOps:
			return "GitOps 执行成功"
		case model.DeployModeDirect:
			return "Direct 执行成功"
		}
	}
	if req.ExecutionStatus == model.ExecutionStatusRolledBack {
		return "GitOps 下线完成"
	}
	if req.ExecutionStatus == model.ExecutionStatusCleaned {
		return "Direct 资源已回收"
	}
	if req.ExecutionStatus == model.ExecutionStatusFailed {
		return "执行失败"
	}
	return "执行中或待执行"
}

var validNamePattern = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "_", "-")
	name = validNamePattern.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

// AccessInfo holds the runtime access details for a deployed workload.
type AccessInfo struct {
	Image       string `json:"image"`
	Namespace   string `json:"namespace"`
	ReleaseName string `json:"releaseName"`
	// Service fields — present only when ServiceEnabled is true
	ServiceEnabled bool     `json:"serviceEnabled"`
	ServiceType    string   `json:"serviceType,omitempty"`
	ServicePort    int32    `json:"servicePort,omitempty"`
	TargetPort     int32    `json:"targetPort,omitempty"`
	NodeIP         string   `json:"nodeIp,omitempty"`
	NodePort       int32    `json:"nodePort,omitempty"`
	NodeIPs        []string `json:"nodeIps,omitempty"`
	NodePorts      []int32  `json:"nodePorts,omitempty"`
	AccessURLs     []string `json:"accessUrls,omitempty"`
	Warnings       []string `json:"warnings,omitempty"`
}

// buildAccessInfo constructs an AccessInfo from the fields stored on the
// DeployRequest and enriches it with live Direct apply results when available.
func buildAccessInfo(req *model.DeployRequest, applyResults ...*DirectApplyResult) *AccessInfo {
	if req == nil {
		return nil
	}
	info := &AccessInfo{
		Image:          req.Image,
		Namespace:      req.Namespace,
		ReleaseName:    req.ReleaseName,
		ServiceEnabled: req.ServiceEnabled,
	}
	if req.ServiceEnabled {
		info.ServiceType = req.ServiceType
		if info.ServiceType == "" {
			info.ServiceType = "ClusterIP"
		}
		info.ServicePort = req.ServicePort
		if info.ServicePort <= 0 {
			info.ServicePort = 80
		}
		info.TargetPort = req.TargetPort
		if info.TargetPort <= 0 {
			info.TargetPort = info.ServicePort
		}
	}
	for _, applyResult := range applyResults {
		if applyResult == nil {
			continue
		}
		if applyResult.Service != nil {
			info.ServiceEnabled = true
			if applyResult.Service.Type != "" {
				info.ServiceType = applyResult.Service.Type
			}
			if len(applyResult.Service.Ports) > 0 {
				firstPort := applyResult.Service.Ports[0]
				if firstPort.Port > 0 {
					info.ServicePort = firstPort.Port
				}
				if firstPort.TargetPort != "" {
					if parsed, err := strconv.ParseInt(firstPort.TargetPort, 10, 32); err == nil {
						info.TargetPort = int32(parsed)
					}
				}
				for _, port := range applyResult.Service.Ports {
					if port.NodePort > 0 {
						info.NodePorts = append(info.NodePorts, port.NodePort)
					}
				}
				info.NodePorts = dedupeInt32s(info.NodePorts)
				if len(info.NodePorts) > 0 {
					info.NodePort = info.NodePorts[0]
				}
			}
		}
		info.NodeIPs = dedupeStrings(append(info.NodeIPs, applyResult.NodeIPs...))
		if len(info.NodeIPs) > 0 {
			info.NodeIP = info.NodeIPs[0]
		}
		info.AccessURLs = dedupeStrings(append(info.AccessURLs, applyResult.AccessURLs...))
		info.Warnings = dedupeStrings(append(info.Warnings, applyResult.Warnings...))
	}
	return info
}

func directApplyResultFromExecution(record *model.ExecutionRecord) *DirectApplyResult {
	if record == nil || strings.TrimSpace(record.DetailJSON) == "" {
		return nil
	}

	var detail map[string]json.RawMessage
	if err := json.Unmarshal([]byte(record.DetailJSON), &detail); err != nil {
		return nil
	}
	raw, ok := detail["result"]
	if !ok || len(raw) == 0 {
		return nil
	}

	var result DirectApplyResult
	if raw[0] == '"' {
		var encoded string
		if err := json.Unmarshal(raw, &encoded); err != nil || strings.TrimSpace(encoded) == "" {
			return nil
		}
		if err := json.Unmarshal([]byte(encoded), &result); err != nil {
			return nil
		}
		return &result
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return &result
}

func dedupeInt32s(values []int32) []int32 {
	seen := map[int32]struct{}{}
	out := make([]int32, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

// execErrorMessage extracts the "error" field from an ExecutionRecord's
// DetailJSON. Returns an empty string when there is no error or the JSON
// cannot be parsed.
func execErrorMessage(record *model.ExecutionRecord) string {
	if record == nil || strings.TrimSpace(record.DetailJSON) == "" {
		return ""
	}
	var detail map[string]string
	if err := json.Unmarshal([]byte(record.DetailJSON), &detail); err != nil {
		return ""
	}
	return detail["error"]
}

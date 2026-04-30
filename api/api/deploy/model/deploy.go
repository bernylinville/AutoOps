package model

import "time"

const (
	DeploySourceUI    = "ui"
	DeploySourceAgent = "agent"
)

const (
	DeployModeGitOps = "gitops"
	DeployModeDirect = "direct"
)

const (
	DeployResourceTypePod        = "pod"
	DeployResourceTypeDeployment = "deployment"
	DeployResourceTypeService    = "service"
)

const (
	DeployRequestStatusSubmitted       = "submitted"
	DeployRequestStatusPendingApproval = "pending_approval"
	DeployRequestStatusApproved        = "approved"
	DeployRequestStatusRejected        = "rejected"
	DeployRequestStatusExecuting       = "executing"
	DeployRequestStatusSucceeded       = "succeeded"
	DeployRequestStatusFailed          = "failed"
	DeployRequestStatusRolledBack      = "rolled_back"
	DeployRequestStatusCancelled       = "cancelled"
	DeployRequestStatusExpired         = "expired"
)

const (
	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
	ApprovalStatusExpired  = "expired"
)

const (
	ApprovalDispatchStatusPending    = "pending"
	ApprovalDispatchStatusDispatched = "dispatched"
	ApprovalDispatchStatusSkipped    = "skipped"
	ApprovalDispatchStatusFailed     = "failed"
)

const (
	ExecutionStatusPending    = "pending"
	ExecutionStatusRunning    = "running"
	ExecutionStatusSucceeded  = "succeeded"
	ExecutionStatusFailed     = "failed"
	ExecutionStatusRolledBack = "rolled_back"
	ExecutionStatusCleaned    = "cleaned"
)

const (
	ExecutionPhaseQueued   = "queued"
	ExecutionPhaseRollback = "rollback"
)

const (
	ApprovalChannelDingtalkOA = "dingtalk_oa"
)

const (
	ApprovalSourceDingtalkCard = "dingtalk_card"
	ApprovalSourceUI           = "ui"
)

const (
	ResourceOwnerSystemGitOps = "gitops"
	ResourceOwnerSystemDirect = "direct"
)

const (
	NotificationChannelDingtalkRobot = "dingtalk_robot"
	NotificationStageExecuted        = "executed"
	NotificationStageFailed          = "failed"
	NotificationStageRolledBack      = "rolled_back"
	NotificationStageCleaned         = "cleaned"
	NotificationStatusPending        = "pending"
	NotificationStatusSent           = "sent"
	NotificationStatusFailed         = "failed"
	NotificationStatusSkipped        = "skipped"
)

// ClusterTarget 部署目标集群配置
type ClusterTarget struct {
	ID                     uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                   string    `gorm:"size:100;not null;uniqueIndex;comment:'目标名称'" json:"name"`
	KubeClusterID          uint      `gorm:"not null;index;comment:'关联K8s集群ID'" json:"kubeClusterId"`
	EnvType                string    `gorm:"size:32;not null;default:'test';comment:'环境类型'" json:"envType"`
	GitOpsEnabled          bool      `gorm:"not null;default:true;comment:'是否启用GitOps模式'" json:"gitOpsEnabled"`
	DirectEnabled          bool      `gorm:"not null;default:true;comment:'是否启用直连模式'" json:"directEnabled"`
	HarborServerID         *uint     `gorm:"index;comment:'Harbor服务器ID'" json:"harborServerId"`
	JenkinsServerID        *uint     `gorm:"index;comment:'Jenkins服务器ID'" json:"jenkinsServerId"`
	GitOpsRepo             string    `gorm:"size:255;comment:'GitOps仓库地址'" json:"gitOpsRepo"`
	GitOpsBranch           string    `gorm:"size:128;comment:'GitOps分支'" json:"gitOpsBranch"`
	GitOpsReleaseDir       string    `gorm:"size:255;comment:'GitOps发布目录'" json:"gitOpsReleaseDir"`
	DirectNamespacePrefix  string    `gorm:"size:64;not null;default:'ao-direct';comment:'直连命名空间前缀'" json:"directNamespacePrefix"`
	DefaultTTLHours        int       `gorm:"not null;default:72;comment:'默认TTL小时'" json:"defaultTtlHours"`
	DirectKubeconfigRef    string    `gorm:"size:255;comment:'直连模式凭据引用'" json:"directKubeconfigRef"`
	DefaultApproverAdminID *uint     `gorm:"index;comment:'默认审批人ID'" json:"defaultApproverAdminId"`
	Description            string    `gorm:"type:text;comment:'说明'" json:"description"`
	CreatedAt              time.Time `json:"createdAt"`
	UpdatedAt              time.Time `json:"updatedAt"`
}

func (ClusterTarget) TableName() string {
	return "deploy_cluster_target"
}

// DeployRequest 部署申请主表
type DeployRequest struct {
	ID                             uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestNo                      string     `gorm:"size:64;not null;uniqueIndex;comment:'申请单号'" json:"requestNo"`
	Source                         string     `gorm:"size:32;not null;default:'ui';comment:'请求来源'" json:"source"`
	RequesterExternalType          string     `gorm:"size:32;comment:'外部身份类型'" json:"requesterExternalType"`
	RequesterExternalID            string     `gorm:"size:128;index;comment:'外部身份ID'" json:"requesterExternalId"`
	Mode                           string     `gorm:"size:32;not null;comment:'部署模式'" json:"mode"`
	WorkflowKind                   string     `gorm:"size:32;not null;default:'deploy_only';comment:'工作流类型'" json:"workflowKind"`
	ResourceType                   string     `gorm:"size:32;not null;comment:'资源类型'" json:"resourceType"`
	ClusterTargetID                uint       `gorm:"not null;index;comment:'目标集群ID'" json:"clusterTargetId"`
	ApplicationID                  *uint      `gorm:"index;comment:'关联应用ID'" json:"applicationId"`
	ReleaseName                    string     `gorm:"size:128;not null;comment:'发布名称'" json:"releaseName"`
	Namespace                      string     `gorm:"size:128;not null;comment:'命名空间'" json:"namespace"`
	Image                          string     `gorm:"size:255;comment:'镜像地址'" json:"image"`
	Replicas                       int32      `gorm:"not null;default:1;comment:'副本数'" json:"replicas"`
	ServiceEnabled                 bool       `gorm:"not null;default:false;comment:'是否创建Service'" json:"serviceEnabled"`
	ServiceType                    string     `gorm:"size:32;comment:'Service类型'" json:"serviceType"`
	ServicePort                    int32      `gorm:"default:0;comment:'Service端口'" json:"servicePort"`
	TargetPort                     int32      `gorm:"default:0;comment:'目标端口'" json:"targetPort"`
	EnvJSON                        string     `gorm:"type:text;comment:'环境变量快照JSON'" json:"envJson"`
	ResourcesJSON                  string     `gorm:"type:text;comment:'资源配置JSON'" json:"resourcesJson"`
	TTLHours                       *int       `gorm:"comment:'TTL小时'" json:"ttlHours"`
	Reason                         string     `gorm:"type:text;comment:'申请原因'" json:"reason"`
	RequestStatus                  string     `gorm:"size:32;not null;default:'submitted';index;comment:'申请状态'" json:"requestStatus"`
	ApprovalStatus                 string     `gorm:"size:32;not null;default:'pending';index;comment:'审批状态'" json:"approvalStatus"`
	ExecutionStatus                string     `gorm:"size:32;not null;default:'pending';index;comment:'执行状态'" json:"executionStatus"`
	PipelineStatus                 string     `gorm:"size:32;not null;default:'pending';index;comment:'流水线状态'" json:"pipelineStatus"`
	CurrentPipelineStage           string     `gorm:"size:32;comment:'当前流水线阶段'" json:"currentPipelineStage"`
	PipelineErrorMessage           string     `gorm:"type:text;comment:'流水线错误信息'" json:"pipelineErrorMessage"`
	RequestedBy                    uint       `gorm:"not null;index;comment:'申请人ID'" json:"requestedBy"`
	RequesterDisplayName           string     `gorm:"size:100;comment:'申请人展示名'" json:"requesterDisplayName"`
	ApproverAdminID                *uint      `gorm:"index;comment:'审批人ID'" json:"approverAdminId"`
	ApproverDingtalkUserIDSnapshot string     `gorm:"size:128;comment:'审批人钉钉ID快照'" json:"approverDingtalkUserIdSnapshot"`
	ApprovalChannel                string     `gorm:"size:64;comment:'审批通道'" json:"approvalChannel"`
	ApprovalDispatchStatus         string     `gorm:"size:32;not null;default:'pending';comment:'审批投递状态'" json:"approvalDispatchStatus"`
	ApprovalDispatchMessage        string     `gorm:"type:text;comment:'审批投递说明'" json:"approvalDispatchMessage"`
	DingtalkProcessCodeSnapshot    string     `gorm:"size:128;comment:'钉钉审批流程编码快照'" json:"dingtalkProcessCodeSnapshot"`
	DingtalkProcessInstanceID      string     `gorm:"size:128;index;comment:'钉钉审批实例ID'" json:"dingtalkProcessInstanceId"`
	ChatContextJSON                string     `gorm:"type:text;comment:'聊天上下文JSON'" json:"chatContextJson"`
	ApprovedAt                     *time.Time `gorm:"comment:'审批通过时间'" json:"approvedAt"`
	RejectedAt                     *time.Time `gorm:"comment:'审批拒绝时间'" json:"rejectedAt"`
	StartedAt                      *time.Time `gorm:"comment:'执行开始时间'" json:"startedAt"`
	FinishedAt                     *time.Time `gorm:"comment:'执行结束时间'" json:"finishedAt"`
	CreatedAt                      time.Time  `json:"createdAt"`
	UpdatedAt                      time.Time  `json:"updatedAt"`
}

func (DeployRequest) TableName() string {
	return "deploy_request"
}

// ApprovalRecord 审批记录
type ApprovalRecord struct {
	ID                  uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestID           uint      `gorm:"not null;index;comment:'申请ID'" json:"requestId"`
	ApproverAdminID     *uint     `gorm:"index;comment:'审批人ID'" json:"approverAdminId"`
	ApproverName        string    `gorm:"size:100;comment:'审批人名称'" json:"approverName"`
	ActorDingtalkUserID string    `gorm:"size:128;comment:'审批动作钉钉用户ID'" json:"actorDingtalkUserId"`
	Source              string    `gorm:"size:32;not null;comment:'审批来源'" json:"source"`
	Action              string    `gorm:"size:32;not null;comment:'审批动作'" json:"action"`
	Comment             string    `gorm:"type:text;comment:'审批意见'" json:"comment"`
	CardCallbackTraceID string    `gorm:"size:128;comment:'卡片回调追踪ID'" json:"cardCallbackTraceId"`
	ActedAt             time.Time `gorm:"not null;comment:'动作时间'" json:"actedAt"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

func (ApprovalRecord) TableName() string {
	return "deploy_approval_record"
}

// ExecutionRecord 执行记录
type ExecutionRecord struct {
	ID               uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestID        uint       `gorm:"not null;index;comment:'申请ID'" json:"requestId"`
	ApprovalRecordID *uint      `gorm:"index;comment:'审批记录ID'" json:"approvalRecordId"`
	ExecutorType     string     `gorm:"size:32;not null;comment:'执行器类型'" json:"executorType"`
	Phase            string     `gorm:"size:64;not null;comment:'执行阶段'" json:"phase"`
	Status           string     `gorm:"size:32;not null;default:'pending';index;comment:'执行状态'" json:"status"`
	DetailJSON       string     `gorm:"type:text;comment:'执行详情JSON'" json:"detailJson"`
	GitCommitSHA     string     `gorm:"size:128;comment:'Git提交SHA'" json:"gitCommitSha"`
	GitFilePath      string     `gorm:"size:255;comment:'Git变更文件路径'" json:"gitFilePath"`
	ReleaseRevision  string     `gorm:"size:128;comment:'发布版本'" json:"releaseRevision"`
	K8sNamespace     string     `gorm:"size:128;comment:'执行命名空间'" json:"k8sNamespace"`
	StartedAt        *time.Time `gorm:"comment:'开始时间'" json:"startedAt"`
	EndedAt          *time.Time `gorm:"comment:'结束时间'" json:"endedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (ExecutionRecord) TableName() string {
	return "deploy_execution_record"
}

// ResourceOwner 资源归属注册表
type ResourceOwner struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ClusterTargetID uint      `gorm:"not null;index;comment:'目标集群ID'" json:"clusterTargetId"`
	Namespace       string    `gorm:"size:128;not null;index:idx_deploy_owner_resource,priority:1;comment:'命名空间'" json:"namespace"`
	Kind            string    `gorm:"size:64;not null;index:idx_deploy_owner_resource,priority:2;comment:'资源类型'" json:"kind"`
	Name            string    `gorm:"size:128;not null;index:idx_deploy_owner_resource,priority:3;comment:'资源名称'" json:"name"`
	OwnerSystem     string    `gorm:"size:32;not null;index;comment:'归属系统'" json:"ownerSystem"`
	RequestID       uint      `gorm:"not null;index;comment:'关联申请ID'" json:"requestId"`
	ReleaseName     string    `gorm:"size:128;comment:'发布名称'" json:"releaseName"`
	Active          bool      `gorm:"not null;default:true;index;comment:'是否生效'" json:"active"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (ResourceOwner) TableName() string {
	return "deploy_resource_owner"
}

// DeployNotification 部署结果通知记录
type DeployNotification struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	RequestID    uint       `gorm:"not null;index;comment:'申请ID'" json:"requestId"`
	Channel      string     `gorm:"type:varchar(32);not null;comment:'通知渠道'" json:"channel"`
	Stage        string     `gorm:"type:varchar(32);not null;comment:'通知阶段'" json:"stage"`
	PayloadJSON  string     `gorm:"type:text;comment:'发送负载快照'" json:"payloadJson"`
	Status       string     `gorm:"type:varchar(32);not null;default:'pending';index;comment:'通知状态'" json:"status"`
	ErrorMessage string     `gorm:"type:text;comment:'错误信息'" json:"errorMessage,omitempty"`
	SentAt       *time.Time `gorm:"comment:'发送时间'" json:"sentAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

func (DeployNotification) TableName() string {
	return "deploy_notification"
}

// CreateDeployRequest 创建部署申请请求
type CreateDeployRequest struct {
	Mode            string                 `json:"mode" binding:"required,oneof=gitops direct"`
	WorkflowKind    string                 `json:"workflowKind" binding:"omitempty,oneof=deploy_only build_deploy"`
	ApplicationID   *uint                  `json:"applicationId"`
	ResourceType    string                 `json:"resourceType" binding:"required,oneof=pod deployment service"`
	ClusterTargetID uint                   `json:"clusterTargetId" binding:"required"`
	ReleaseName     string                 `json:"releaseName" binding:"required"`
	Namespace       string                 `json:"namespace"`
	Image           string                 `json:"image"`
	Replicas        int32                  `json:"replicas"`
	ServiceEnabled  bool                   `json:"serviceEnabled"`
	ServiceType     string                 `json:"serviceType"`
	ServicePort     int32                  `json:"servicePort"`
	TargetPort      int32                  `json:"targetPort"`
	Env             map[string]interface{} `json:"env"`
	Resources       map[string]interface{} `json:"resources"`
	TTLHours        *int                   `json:"ttlHours"`
	ApproverAdminID *uint                  `json:"approverAdminId"`
	Reason          string                 `json:"reason"`
	ChatContext     map[string]interface{} `json:"chatContext"`
	// BuildDeployFields 仅在 workflowKind=build_deploy 时生效
	GitRef           string                 `json:"gitRef"`
	BuildParams      map[string]interface{} `json:"buildParams"`
	JenkinsServerID  *uint                  `json:"jenkinsServerId"`
	JenkinsJobName   string                 `json:"jenkinsJobName"`
	HarborServerID   *uint                  `json:"harborServerId"`
	HarborProject    string                 `json:"harborProject"`
	HarborRepository string                 `json:"harborRepository"`
	ArtifactTag      string                 `json:"artifactTag"`
	ScanPolicy       map[string]interface{} `json:"scanPolicy"`
}

// CreateAgentDeployRequest Agent 创建部署申请请求
type CreateAgentDeployRequest struct {
	RequesterExternalType string                 `json:"requesterExternalType" binding:"required"`
	RequesterExternalID   string                 `json:"requesterExternalId" binding:"required"`
	RequesterDisplayName  string                 `json:"requesterDisplayName"`
	Mode                  string                 `json:"mode" binding:"required,oneof=gitops direct"`
	WorkflowKind          string                 `json:"workflowKind" binding:"omitempty,oneof=deploy_only build_deploy"`
	ApplicationID         *uint                  `json:"applicationId"`
	ResourceType          string                 `json:"resourceType" binding:"required,oneof=pod deployment service"`
	ClusterTargetID       uint                   `json:"clusterTargetId" binding:"required"`
	ReleaseName           string                 `json:"releaseName" binding:"required"`
	Namespace             string                 `json:"namespace"`
	Image                 string                 `json:"image"`
	Replicas              int32                  `json:"replicas"`
	ServiceEnabled        bool                   `json:"serviceEnabled"`
	ServiceType           string                 `json:"serviceType"`
	ServicePort           int32                  `json:"servicePort"`
	TargetPort            int32                  `json:"targetPort"`
	Env                   map[string]interface{} `json:"env"`
	Resources             map[string]interface{} `json:"resources"`
	TTLHours              *int                   `json:"ttlHours"`
	ApproverAdminID       *uint                  `json:"approverAdminId"`
	Reason                string                 `json:"reason"`
	ChatContext           map[string]interface{} `json:"chatContext"`
	// BuildDeployFields 仅在 workflowKind=build_deploy 时生效
	GitRef           string                 `json:"gitRef"`
	BuildParams      map[string]interface{} `json:"buildParams"`
	JenkinsServerID  *uint                  `json:"jenkinsServerId"`
	JenkinsJobName   string                 `json:"jenkinsJobName"`
	HarborServerID   *uint                  `json:"harborServerId"`
	HarborProject    string                 `json:"harborProject"`
	HarborRepository string                 `json:"harborRepository"`
	ArtifactTag      string                 `json:"artifactTag"`
	ScanPolicy       map[string]interface{} `json:"scanPolicy"`
}

// CreateAgentBuildDeployRequest lets Hermes submit a build+deploy request by
// application/env; AutoOps resolves deployment details from AppDeployProfile.
type CreateAgentBuildDeployRequest struct {
	RequesterExternalType string                 `json:"requesterExternalType" binding:"required"`
	RequesterExternalID   string                 `json:"requesterExternalId" binding:"required"`
	RequesterDisplayName  string                 `json:"requesterDisplayName"`
	ApplicationCode       string                 `json:"applicationCode" binding:"required"`
	Env                   string                 `json:"env" binding:"required,oneof=dev test"`
	GitRef                string                 `json:"gitRef"`
	Reason                string                 `json:"reason"`
	BuildParams           map[string]interface{} `json:"buildParams"`
	ChatContext           map[string]interface{} `json:"chatContext"`
}

// CreateAgentProjectOnboardBuildDeployRequest lets Hermes onboard a GitLab
// project into AutoOps and immediately create a profile-driven build+deploy
// request.
type CreateAgentProjectOnboardBuildDeployRequest struct {
	RequesterExternalType string                 `json:"requesterExternalType" binding:"required"`
	RequesterExternalID   string                 `json:"requesterExternalId" binding:"required"`
	RequesterDisplayName  string                 `json:"requesterDisplayName"`
	GitRepoURL            string                 `json:"gitRepoUrl" binding:"required"`
	ApplicationCode       string                 `json:"applicationCode"`
	Env                   string                 `json:"env" binding:"required,oneof=dev test"`
	GitRef                string                 `json:"gitRef"`
	Reason                string                 `json:"reason"`
	ExposureMode          string                 `json:"exposureMode"`
	BuildParams           map[string]interface{} `json:"buildParams"`
	ChatContext           map[string]interface{} `json:"chatContext"`
}

// ApproveDeployRequest 审批通过请求
type ApproveDeployRequest struct {
	Comment string `json:"comment"`
}

// RejectDeployRequest 审批拒绝请求
type RejectDeployRequest struct {
	Comment string `json:"comment"`
}

// ExecuteDeployRequest 触发部署执行请求
type ExecuteDeployRequest struct {
	Comment string `json:"comment"`
}

// DeployRequestListQuery 部署申请列表查询
type DeployRequestListQuery struct {
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
	RequestStatus   string `form:"requestStatus"`
	ApprovalStatus  string `form:"approvalStatus"`
	ExecutionStatus string `form:"executionStatus"`
	Mode            string `form:"mode"`
	ClusterTargetID *uint  `form:"clusterTargetId"`
	RequestedBy     *uint  `form:"requestedBy"`
}

// CreateClusterTargetRequest 创建部署目标请求
type CreateClusterTargetRequest struct {
	Name                   string `json:"name" binding:"required"`
	KubeClusterID          uint   `json:"kubeClusterId" binding:"required"`
	EnvType                string `json:"envType"`
	GitOpsEnabled          *bool  `json:"gitOpsEnabled"`
	DirectEnabled          *bool  `json:"directEnabled"`
	HarborServerID         *uint  `json:"harborServerId"`
	JenkinsServerID        *uint  `json:"jenkinsServerId"`
	GitOpsRepo             string `json:"gitOpsRepo"`
	GitOpsBranch           string `json:"gitOpsBranch"`
	GitOpsReleaseDir       string `json:"gitOpsReleaseDir"`
	DirectNamespacePrefix  string `json:"directNamespacePrefix"`
	DefaultTTLHours        *int   `json:"defaultTtlHours"`
	DirectKubeconfigRef    string `json:"directKubeconfigRef"`
	DefaultApproverAdminID *uint  `json:"defaultApproverAdminId"`
	Description            string `json:"description"`
}

// UpdateClusterTargetRequest 更新部署目标请求
type UpdateClusterTargetRequest struct {
	Name                   *string `json:"name"`
	KubeClusterID          *uint   `json:"kubeClusterId"`
	EnvType                *string `json:"envType"`
	GitOpsEnabled          *bool   `json:"gitOpsEnabled"`
	DirectEnabled          *bool   `json:"directEnabled"`
	HarborServerID         *uint   `json:"harborServerId"`
	JenkinsServerID        *uint   `json:"jenkinsServerId"`
	GitOpsRepo             *string `json:"gitOpsRepo"`
	GitOpsBranch           *string `json:"gitOpsBranch"`
	GitOpsReleaseDir       *string `json:"gitOpsReleaseDir"`
	DirectNamespacePrefix  *string `json:"directNamespacePrefix"`
	DefaultTTLHours        *int    `json:"defaultTtlHours"`
	DirectKubeconfigRef    *string `json:"directKubeconfigRef"`
	DefaultApproverAdminID *uint   `json:"defaultApproverAdminId"`
	Description            *string `json:"description"`
}

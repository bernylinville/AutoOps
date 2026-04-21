package model

import "time"

const (
	WorkflowKindDeployOnly  = "deploy_only"
	WorkflowKindBuildDeploy = "build_deploy"
)

const (
	PipelineStatusPending   = "pending"
	PipelineStatusBuilding  = "building"
	PipelineStatusScanning  = "scanning"
	PipelineStatusDeploying = "deploying"
	PipelineStatusSucceeded = "succeeded"
	PipelineStatusFailed    = "failed"
)

const (
	PipelineStageBuild  = "build"
	PipelineStageScan   = "scan"
	PipelineStageDeploy = "deploy"
	PipelineStageNotify = "notify"
)

const (
	PipelineStageStatusPending   = "pending"
	PipelineStageStatusRunning   = "running"
	PipelineStageStatusSucceeded = "succeeded"
	PipelineStageStatusFailed    = "failed"
)

// PipelineRun 流水线执行记录
type PipelineRun struct {
	ID                     uint       `gorm:"primaryKey;autoIncrement;comment:'流水线运行ID'" json:"id"`
	RequestID              uint       `gorm:"not null;uniqueIndex;comment:'关联部署申请ID'" json:"requestId"`
	ApplicationID          uint       `gorm:"index;comment:'关联应用ID'" json:"applicationId"`
	Status                 string     `gorm:"size:32;not null;index;comment:'流水线状态'" json:"status"`
	CurrentStage           string     `gorm:"size:32;comment:'当前执行阶段'" json:"currentStage"`
	JenkinsServerID        uint       `gorm:"index;comment:'Jenkins服务器ID'" json:"jenkinsServerId"`
	JenkinsJobNameSnapshot string     `gorm:"size:255;comment:'Jenkins任务名快照'" json:"jenkinsJobNameSnapshot"`
	GitRef                 string     `gorm:"size:128;comment:'Git引用'" json:"gitRef"`
	BuildParamsJSON        string     `gorm:"type:text;comment:'构建参数JSON'" json:"buildParamsJson"`
	JenkinsQueueID         int        `gorm:"comment:'Jenkins队列ID'" json:"jenkinsQueueId"`
	JenkinsBuildNumber     int        `gorm:"comment:'Jenkins构建编号'" json:"jenkinsBuildNumber"`
	JenkinsBuildURL        string     `gorm:"size:512;comment:'Jenkins构建URL'" json:"jenkinsBuildUrl"`
	HarborServerID         uint       `gorm:"index;comment:'Harbor服务器ID'" json:"harborServerId"`
	HarborProject          string     `gorm:"size:128;comment:'Harbor项目名'" json:"harborProject"`
	HarborRepository       string     `gorm:"size:255;comment:'Harbor仓库名'" json:"harborRepository"`
	ArtifactTag            string     `gorm:"size:128;comment:'制品标签'" json:"artifactTag"`
	ArtifactDigest         string     `gorm:"size:255;comment:'制品摘要'" json:"artifactDigest"`
	PlannedImageRef        string     `gorm:"size:512;comment:'计划镜像地址'" json:"plannedImageRef"`
	FinalImageRef          string     `gorm:"size:512;comment:'最终镜像地址'" json:"finalImageRef"`
	ScanPolicyJSON         string     `gorm:"type:text;comment:'扫描策略JSON'" json:"scanPolicyJson"`
	ScanReportJSON         string     `gorm:"type:text;comment:'扫描报告JSON'" json:"scanReportJson"`
	LastError              string     `gorm:"type:text;comment:'最后错误信息'" json:"lastError"`
	StartedAt              *time.Time `gorm:"comment:'开始时间'" json:"startedAt"`
	FinishedAt             *time.Time `gorm:"comment:'结束时间'" json:"finishedAt"`
	CreatedAt              time.Time  `gorm:"comment:'创建时间'" json:"createdAt"`
	UpdatedAt              time.Time  `gorm:"comment:'更新时间'" json:"updatedAt"`
}

func (PipelineRun) TableName() string {
	return "deploy_pipeline_run"
}

// PipelineStageRecord 流水线阶段执行记录
type PipelineStageRecord struct {
	ID            uint       `gorm:"primaryKey;autoIncrement;comment:'流水线阶段记录ID'" json:"id"`
	PipelineRunID uint       `gorm:"not null;uniqueIndex:idx_pipeline_stage;comment:'关联流水线运行ID'" json:"pipelineRunId"`
	Stage         string     `gorm:"size:32;not null;uniqueIndex:idx_pipeline_stage;comment:'流水线阶段'" json:"stage"`
	Status        string     `gorm:"size:32;not null;comment:'阶段状态'" json:"status"`
	ExternalID    string     `gorm:"size:128;comment:'外部系统ID'" json:"externalId"`
	ExternalURL   string     `gorm:"size:512;comment:'外部系统URL'" json:"externalUrl"`
	DetailJSON    string     `gorm:"type:text;comment:'阶段详情JSON'" json:"detailJson"`
	ErrorMessage  string     `gorm:"type:text;comment:'错误信息'" json:"errorMessage"`
	StartedAt     *time.Time `gorm:"comment:'开始时间'" json:"startedAt"`
	FinishedAt    *time.Time `gorm:"comment:'结束时间'" json:"finishedAt"`
	RetryCount    int        `gorm:"default:0;comment:'重试次数'" json:"retryCount"`
	CreatedAt     time.Time  `gorm:"comment:'创建时间'" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"comment:'更新时间'" json:"updatedAt"`
}

func (PipelineStageRecord) TableName() string {
	return "deploy_pipeline_stage_record"
}

// CreatePipelineRunRequest 创建流水线运行请求
type CreatePipelineRunRequest struct {
	RequestID              uint                   `json:"requestId" binding:"required"`
	ApplicationID          uint                   `json:"applicationId"`
	JenkinsServerID        uint                   `json:"jenkinsServerId"`
	JenkinsJobNameSnapshot string                 `json:"jenkinsJobNameSnapshot"`
	GitRef                 string                 `json:"gitRef"`
	BuildParams            map[string]interface{} `json:"buildParams"`
	HarborServerID         uint                   `json:"harborServerId"`
	HarborProject          string                 `json:"harborProject"`
	HarborRepository       string                 `json:"harborRepository"`
	ArtifactTag            string                 `json:"artifactTag"`
	PlannedImageRef        string                 `json:"plannedImageRef"`
	ScanPolicy             map[string]interface{} `json:"scanPolicy"`
}

// UpdatePipelineRunStatusRequest 更新流水线状态请求
type UpdatePipelineRunStatusRequest struct {
	Status             string                 `json:"status" binding:"required,oneof=pending building scanning deploying succeeded failed"`
	CurrentStage       string                 `json:"currentStage" binding:"omitempty,oneof=build scan deploy notify"`
	JenkinsQueueID     *int                   `json:"jenkinsQueueId"`
	JenkinsBuildNumber *int                   `json:"jenkinsBuildNumber"`
	JenkinsBuildURL    string                 `json:"jenkinsBuildUrl"`
	ArtifactTag        string                 `json:"artifactTag"`
	ArtifactDigest     string                 `json:"artifactDigest"`
	FinalImageRef      string                 `json:"finalImageRef"`
	ScanReport         map[string]interface{} `json:"scanReport"`
	LastError          string                 `json:"lastError"`
}

// CreatePipelineStageRecordRequest 创建流水线阶段记录请求
type CreatePipelineStageRecordRequest struct {
	PipelineRunID uint                   `json:"pipelineRunId" binding:"required"`
	Stage         string                 `json:"stage" binding:"required,oneof=build scan deploy notify"`
	Status        string                 `json:"status" binding:"required,oneof=pending running succeeded failed"`
	ExternalID    string                 `json:"externalId"`
	ExternalURL   string                 `json:"externalUrl"`
	Detail        map[string]interface{} `json:"detail"`
	ErrorMessage  string                 `json:"errorMessage"`
	RetryCount    int                    `json:"retryCount"`
}

// PipelineRunResponse 流水线运行响应
type PipelineRunResponse struct {
	ID                     uint                          `json:"id"`
	RequestID              uint                          `json:"requestId"`
	ApplicationID          uint                          `json:"applicationId"`
	Status                 string                        `json:"status"`
	CurrentStage           string                        `json:"currentStage"`
	JenkinsServerID        uint                          `json:"jenkinsServerId"`
	JenkinsJobNameSnapshot string                        `json:"jenkinsJobNameSnapshot"`
	GitRef                 string                        `json:"gitRef"`
	BuildParamsJSON        string                        `json:"buildParamsJson"`
	JenkinsQueueID         int                           `json:"jenkinsQueueId"`
	JenkinsBuildNumber     int                           `json:"jenkinsBuildNumber"`
	JenkinsBuildURL        string                        `json:"jenkinsBuildUrl"`
	HarborServerID         uint                          `json:"harborServerId"`
	HarborProject          string                        `json:"harborProject"`
	HarborRepository       string                        `json:"harborRepository"`
	ArtifactTag            string                        `json:"artifactTag"`
	ArtifactDigest         string                        `json:"artifactDigest"`
	PlannedImageRef        string                        `json:"plannedImageRef"`
	FinalImageRef          string                        `json:"finalImageRef"`
	ScanPolicyJSON         string                        `json:"scanPolicyJson"`
	ScanReportJSON         string                        `json:"scanReportJson"`
	LastError              string                        `json:"lastError"`
	StartedAt              *time.Time                    `json:"startedAt"`
	FinishedAt             *time.Time                    `json:"finishedAt"`
	CreatedAt              time.Time                     `json:"createdAt"`
	UpdatedAt              time.Time                     `json:"updatedAt"`
	Stages                 []PipelineStageRecordResponse `json:"stages,omitempty"`
}

// PipelineStageRecordResponse 流水线阶段记录响应
type PipelineStageRecordResponse struct {
	ID            uint       `json:"id"`
	PipelineRunID uint       `json:"pipelineRunId"`
	Stage         string     `json:"stage"`
	Status        string     `json:"status"`
	ExternalID    string     `json:"externalId"`
	ExternalURL   string     `json:"externalUrl"`
	DetailJSON    string     `json:"detailJson"`
	ErrorMessage  string     `json:"errorMessage"`
	StartedAt     *time.Time `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
	RetryCount    int        `json:"retryCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

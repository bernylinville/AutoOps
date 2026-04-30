package model

import "time"

const (
	DeployProfileEnvDev  = "dev"
	DeployProfileEnvTest = "test"
)

// AppDeployProfile stores the AutoOps-side deployment contract for one app/environment.
type AppDeployProfile struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	AppID             uint      `gorm:"not null;uniqueIndex:idx_app_deploy_profile_env;index" json:"appId"`
	ApplicationCode   string    `gorm:"size:128;not null;uniqueIndex:idx_app_deploy_profile_env;index" json:"applicationCode"`
	Env               string    `gorm:"size:32;not null;uniqueIndex:idx_app_deploy_profile_env;index" json:"env"`
	Enabled           bool      `gorm:"not null;default:true" json:"enabled"`
	ClusterTargetID   uint      `gorm:"not null;index" json:"clusterTargetId"`
	Namespace         string    `gorm:"size:128;not null" json:"namespace"`
	ReleaseName       string    `gorm:"size:128;not null" json:"releaseName"`
	ResourceType      string    `gorm:"size:32;not null;default:'deployment'" json:"resourceType"`
	JenkinsServerID   uint      `gorm:"not null;index" json:"jenkinsServerId"`
	JenkinsJobName    string    `gorm:"size:255;not null" json:"jenkinsJobName"`
	HarborServerID    uint      `gorm:"not null;index" json:"harborServerId"`
	HarborProject     string    `gorm:"size:128;not null" json:"harborProject"`
	HarborRepository  string    `gorm:"size:255;not null" json:"harborRepository"`
	DefaultGitRef     string    `gorm:"size:128;not null;default:'main'" json:"defaultGitRef"`
	ApproverAdminID   uint      `gorm:"not null;index" json:"approverAdminId"`
	Replicas          int32     `gorm:"not null;default:1" json:"replicas"`
	ServiceEnabled    bool      `gorm:"not null;default:true" json:"serviceEnabled"`
	ServiceType       string    `gorm:"size:32;default:'ClusterIP'" json:"serviceType"`
	ServicePort       int32     `gorm:"default:80" json:"servicePort"`
	TargetPort        int32     `gorm:"default:8080" json:"targetPort"`
	EnvJSON           string    `gorm:"type:text" json:"envJson"`
	ResourcesJSON     string    `gorm:"type:text" json:"resourcesJson"`
	BuildParamsJSON   string    `gorm:"type:text" json:"buildParamsJson"`
	ScanPolicyJSON    string    `gorm:"type:text" json:"scanPolicyJson"`
	AccessURLTemplate string    `gorm:"size:512" json:"accessUrlTemplate"`
	HealthCheckPath   string    `gorm:"size:255" json:"healthCheckPath"`
	Description       string    `gorm:"type:text" json:"description"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

func (AppDeployProfile) TableName() string { return "app_deploy_profile" }

type CreateAppDeployProfileRequest struct {
	Env               string                 `json:"env" binding:"required,oneof=dev test"`
	Enabled           *bool                  `json:"enabled"`
	ClusterTargetID   uint                   `json:"clusterTargetId" binding:"required"`
	Namespace         string                 `json:"namespace" binding:"required"`
	ReleaseName       string                 `json:"releaseName" binding:"required"`
	ResourceType      string                 `json:"resourceType" binding:"omitempty,oneof=deployment pod"`
	JenkinsServerID   uint                   `json:"jenkinsServerId" binding:"required"`
	JenkinsJobName    string                 `json:"jenkinsJobName" binding:"required"`
	HarborServerID    uint                   `json:"harborServerId" binding:"required"`
	HarborProject     string                 `json:"harborProject" binding:"required"`
	HarborRepository  string                 `json:"harborRepository" binding:"required"`
	DefaultGitRef     string                 `json:"defaultGitRef"`
	ApproverAdminID   uint                   `json:"approverAdminId" binding:"required"`
	Replicas          int32                  `json:"replicas"`
	ServiceEnabled    bool                   `json:"serviceEnabled"`
	ServiceType       string                 `json:"serviceType"`
	ServicePort       int32                  `json:"servicePort"`
	TargetPort        int32                  `json:"targetPort"`
	EnvVars           map[string]interface{} `json:"envVars"`
	Resources         map[string]interface{} `json:"resources"`
	BuildParams       map[string]interface{} `json:"buildParams"`
	ScanPolicy        map[string]interface{} `json:"scanPolicy"`
	AccessURLTemplate string                 `json:"accessUrlTemplate"`
	HealthCheckPath   string                 `json:"healthCheckPath"`
	Description       string                 `json:"description"`
}

type UpdateAppDeployProfileRequest struct {
	Enabled           *bool                  `json:"enabled"`
	ClusterTargetID   *uint                  `json:"clusterTargetId"`
	Namespace         *string                `json:"namespace"`
	ReleaseName       *string                `json:"releaseName"`
	ResourceType      *string                `json:"resourceType"`
	JenkinsServerID   *uint                  `json:"jenkinsServerId"`
	JenkinsJobName    *string                `json:"jenkinsJobName"`
	HarborServerID    *uint                  `json:"harborServerId"`
	HarborProject     *string                `json:"harborProject"`
	HarborRepository  *string                `json:"harborRepository"`
	DefaultGitRef     *string                `json:"defaultGitRef"`
	ApproverAdminID   *uint                  `json:"approverAdminId"`
	Replicas          *int32                 `json:"replicas"`
	ServiceEnabled    *bool                  `json:"serviceEnabled"`
	ServiceType       *string                `json:"serviceType"`
	ServicePort       *int32                 `json:"servicePort"`
	TargetPort        *int32                 `json:"targetPort"`
	EnvVars           map[string]interface{} `json:"envVars"`
	Resources         map[string]interface{} `json:"resources"`
	BuildParams       map[string]interface{} `json:"buildParams"`
	ScanPolicy        map[string]interface{} `json:"scanPolicy"`
	AccessURLTemplate *string                `json:"accessUrlTemplate"`
	HealthCheckPath   *string                `json:"healthCheckPath"`
	Description       *string                `json:"description"`
}

type AppDeployProfileValidation struct {
	Valid    bool     `json:"valid"`
	Messages []string `json:"messages"`
}

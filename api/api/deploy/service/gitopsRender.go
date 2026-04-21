package service

import (
	"fmt"
	"time"

	"dodevops-api/api/deploy/model"

	"sigs.k8s.io/yaml"
)

type GitOpsReleaseFile struct {
	APIVersion string                `yaml:"apiVersion" json:"apiVersion"`
	Kind       string                `yaml:"kind" json:"kind"`
	Metadata   GitOpsReleaseMetadata `yaml:"metadata" json:"metadata"`
	Spec       GitOpsReleaseSpec     `yaml:"spec" json:"spec"`
}

type GitOpsReleaseMetadata struct {
	Name string `yaml:"name" json:"name"`
}

type GitOpsReleaseSpec struct {
	ClusterTarget string                 `yaml:"clusterTarget" json:"clusterTarget"`
	Namespace     string                 `yaml:"namespace" json:"namespace"`
	Mode          string                 `yaml:"mode" json:"mode"`
	ResourceType  string                 `yaml:"resourceType" json:"resourceType"`
	Image         string                 `yaml:"image" json:"image"`
	Replicas      int32                  `yaml:"replicas" json:"replicas"`
	Service       GitOpsReleaseService   `yaml:"service" json:"service"`
	Labels        map[string]string      `yaml:"labels" json:"labels"`
	Annotations   map[string]string      `yaml:"annotations" json:"annotations"`
	Resources     map[string]interface{} `yaml:"resources,omitempty" json:"resources,omitempty"`
	Env           map[string]interface{} `yaml:"env,omitempty" json:"env,omitempty"`
}

type GitOpsReleaseService struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	Type       string `yaml:"type,omitempty" json:"type,omitempty"`
	Port       int32  `yaml:"port,omitempty" json:"port,omitempty"`
	TargetPort int32  `yaml:"targetPort,omitempty" json:"targetPort,omitempty"`
}

func RenderGitOpsReleaseFile(req *model.DeployRequest, clusterTargetName string) (string, string, error) {
	if req == nil {
		return "", "", fmt.Errorf("部署申请不能为空")
	}
	if req.Mode != model.DeployModeGitOps {
		return "", "", fmt.Errorf("仅 gitops mode 支持 release 渲染")
	}

	file := GitOpsReleaseFile{
		APIVersion: "autoops.io/v1alpha1",
		Kind:       "Release",
		Metadata: GitOpsReleaseMetadata{
			Name: req.ReleaseName,
		},
		Spec: GitOpsReleaseSpec{
			ClusterTarget: clusterTargetName,
			Namespace:     req.Namespace,
			Mode:          model.DeployModeGitOps,
			ResourceType:  req.ResourceType,
			Image:         req.Image,
			Replicas:      effectiveReplicas(req.Replicas),
			Service: GitOpsReleaseService{
				Enabled:    req.ServiceEnabled,
				Type:       req.ServiceType,
				Port:       req.ServicePort,
				TargetPort: req.TargetPort,
			},
			Labels: map[string]string{
				LabelManagedBy:   "argocd",
				LabelOwnerSystem: model.ResourceOwnerSystemGitOps,
				LabelDeployMode:  model.DeployModeGitOps,
				LabelRequestID:   req.RequestNo,
			},
			Annotations: map[string]string{
				AnnotationSource:          "autoops-gitops",
				"autoops.io/release-name": req.ReleaseName,
				"autoops.io/rendered-at":  time.Now().Format(time.RFC3339),
			},
		},
	}

	content, err := yaml.Marshal(file)
	if err != nil {
		return "", "", err
	}
	filePath := fmt.Sprintf("apps/autoops-managed-releases/releases/%s.yaml", req.ReleaseName)
	return filePath, string(content), nil
}

func effectiveReplicas(replicas int32) int32 {
	if replicas <= 0 {
		return 1
	}
	return replicas
}

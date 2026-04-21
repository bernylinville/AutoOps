package service

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/deploy/model"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

const (
	LabelManagedBy        = "app.kubernetes.io/managed-by"
	LabelOwnerSystem      = "autoops.io/owner-system"
	LabelDeployMode       = "autoops.io/deploy-mode"
	LabelRequestID        = "autoops.io/request-id"
	AnnotationTTLExpireAt = "autoops.io/ttl-expire-at"
	AnnotationSource      = "autoops.io/source"
	VolcesRegistryHost    = "pukka-all-images-cn-shanghai.cr.volces.com"
	VolcesPullSecretName  = "volces-registry"
)

type DirectManifestRenderResult struct {
	Objects []runtime.Object `json:"-"`
	YAML    string           `json:"yaml"`
}

func RenderDirectManifest(req *model.DeployRequest) (*DirectManifestRenderResult, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}
	if req.Mode != model.DeployModeDirect {
		return nil, fmt.Errorf("仅 direct mode 支持直连资源渲染")
	}

	objects := []runtime.Object{}
	switch req.ResourceType {
	case model.DeployResourceTypePod:
		objects = append(objects, buildDirectPod(req))
	case model.DeployResourceTypeDeployment:
		objects = append(objects, buildDirectDeployment(req))
	default:
		return nil, fmt.Errorf("direct mode 暂不支持资源类型: %s", req.ResourceType)
	}

	if req.ServiceEnabled {
		objects = append(objects, buildDirectService(req))
	}

	manifest, err := marshalObjects(objects)
	if err != nil {
		return nil, err
	}
	return &DirectManifestRenderResult{Objects: objects, YAML: manifest}, nil
}

func buildDirectPod(req *model.DeployRequest) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.ReleaseName,
			Namespace:   req.Namespace,
			Labels:      directLabels(req),
			Annotations: directAnnotations(req),
		},
		Spec: corev1.PodSpec{
			RestartPolicy:    corev1.RestartPolicyAlways,
			ImagePullSecrets: directImagePullSecrets(req),
			Containers: []corev1.Container{
				directContainer(req),
			},
		},
	}
}

func buildDirectDeployment(req *model.DeployRequest) *appsv1.Deployment {
	replicas := req.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	labels := directLabels(req)
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.ReleaseName,
			Namespace:   req.Namespace,
			Labels:      labels,
			Annotations: directAnnotations(req),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": req.ReleaseName}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: directAnnotations(req),
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: directImagePullSecrets(req),
					Containers: []corev1.Container{
						directContainer(req),
					},
				},
			},
		},
	}
}

func buildDirectService(req *model.DeployRequest) *corev1.Service {
	serviceType := corev1.ServiceTypeClusterIP
	if req.ServiceType != "" {
		serviceType = corev1.ServiceType(req.ServiceType)
	}
	port := req.ServicePort
	if port <= 0 {
		port = 80
	}
	targetPort := req.TargetPort
	if targetPort <= 0 {
		targetPort = port
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.ReleaseName,
			Namespace:   req.Namespace,
			Labels:      directLabels(req),
			Annotations: directAnnotations(req),
		},
		Spec: corev1.ServiceSpec{
			Type:     serviceType,
			Selector: map[string]string{"app": req.ReleaseName},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       port,
					TargetPort: intstr.FromInt(int(targetPort)),
				},
			},
		},
	}
}

func directContainer(req *model.DeployRequest) corev1.Container {
	container := corev1.Container{
		Name:  "main",
		Image: req.Image,
		Ports: []corev1.ContainerPort{
			{ContainerPort: defaultContainerPort(req), Protocol: corev1.ProtocolTCP},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("25m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
	}
	return container
}

func directImagePullSecrets(req *model.DeployRequest) []corev1.LocalObjectReference {
	if req == nil {
		return nil
	}
	image := strings.TrimSpace(req.Image)
	if image == "" {
		return nil
	}
	if strings.HasPrefix(image, VolcesRegistryHost+"/") {
		return []corev1.LocalObjectReference{{Name: VolcesPullSecretName}}
	}
	return nil
}

func defaultContainerPort(req *model.DeployRequest) int32 {
	if req.TargetPort > 0 {
		return req.TargetPort
	}
	if req.ServicePort > 0 {
		return req.ServicePort
	}
	return 80
}

func directLabels(req *model.DeployRequest) map[string]string {
	return map[string]string{
		"app":            req.ReleaseName,
		LabelManagedBy:   "autoops",
		LabelOwnerSystem: model.ResourceOwnerSystemDirect,
		LabelDeployMode:  model.DeployModeDirect,
		LabelRequestID:   req.RequestNo,
	}
}

func directAnnotations(req *model.DeployRequest) map[string]string {
	annotations := map[string]string{
		AnnotationSource: "autoops-direct",
	}
	if req.TTLHours != nil && *req.TTLHours > 0 {
		expireAt := time.Now().Add(time.Duration(*req.TTLHours) * time.Hour).Format(time.RFC3339)
		annotations[AnnotationTTLExpireAt] = expireAt
		annotations["autoops.io/ttl-hours"] = strconv.Itoa(*req.TTLHours)
	}
	return annotations
}

func marshalObjects(objects []runtime.Object) (string, error) {
	result := ""
	for index, obj := range objects {
		data, err := yaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		if index > 0 {
			result += "---\n"
		}
		result += string(data)
	}
	return result, nil
}

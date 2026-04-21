package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dodevops-api/api/deploy/model"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

type DirectApplyResult struct {
	Namespace string   `json:"namespace"`
	Applied   []string `json:"applied"`
}

func ApplyDirectResources(req *model.DeployRequest, kubeconfigRef string) (*DirectApplyResult, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}
	if req.Mode != model.DeployModeDirect {
		return nil, fmt.Errorf("仅 direct mode 支持直连执行")
	}
	if !strings.HasPrefix(req.Namespace, "ao-direct-") {
		return nil, fmt.Errorf("direct mode 命名空间必须以 ao-direct- 开头")
	}

	validation, err := ValidateDirectKubeconfigAccess(kubeconfigRef, req.Namespace)
	if err != nil {
		return nil, err
	}
	if !validation.Valid {
		return nil, fmt.Errorf("%s", validation.Message)
	}

	rendered, err := RenderDirectManifest(req)
	if err != nil {
		return nil, err
	}

	clientset, err := NewDirectKubeClient(kubeconfigRef)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ensureDirectNamespace(ctx, clientset, req); err != nil {
		return nil, err
	}
	if err := ensureDirectImagePullSecrets(ctx, clientset, req); err != nil {
		return nil, err
	}

	result := &DirectApplyResult{Namespace: req.Namespace}
	for _, obj := range rendered.Objects {
		applied, err := applyDirectObject(ctx, clientset, obj)
		if err != nil {
			return nil, err
		}
		result.Applied = append(result.Applied, applied)
	}
	return result, nil
}

func ensureDirectNamespace(ctx context.Context, clientset *kubernetes.Clientset, req *model.DeployRequest) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Namespace,
			Labels:      directLabels(req),
			Annotations: directAnnotations(req),
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err == nil || apierrors.IsAlreadyExists(err) {
		return nil
	}
	return fmt.Errorf("创建 direct namespace 失败: %v", err)
}

func ensureDirectImagePullSecrets(ctx context.Context, clientset *kubernetes.Clientset, req *model.DeployRequest) error {
	for _, ref := range directImagePullSecrets(req) {
		if ref.Name == "" {
			continue
		}
		if err := syncDirectImagePullSecret(ctx, clientset, "default", req.Namespace, ref.Name); err != nil {
			return fmt.Errorf("同步 imagePullSecret %s 到 namespace %s 失败: %v", ref.Name, req.Namespace, err)
		}
	}
	return nil
}

func syncDirectImagePullSecret(ctx context.Context, clientset *kubernetes.Clientset, sourceNamespace string, targetNamespace string, secretName string) error {
	if sourceNamespace == targetNamespace {
		return nil
	}
	source, err := clientset.CoreV1().Secrets(sourceNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("读取源 secret %s/%s 失败: %v", sourceNamespace, secretName, err)
	}

	target := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				LabelManagedBy:   "autoops",
				LabelOwnerSystem: model.ResourceOwnerSystemDirect,
			},
			Annotations: map[string]string{
				AnnotationSource: "autoops-direct",
			},
		},
		Type: source.Type,
		Data: copySecretData(source.Data),
	}

	existing, err := clientset.CoreV1().Secrets(targetNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = clientset.CoreV1().Secrets(targetNamespace).Create(ctx, target, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return fmt.Errorf("读取目标 secret %s/%s 失败: %v", targetNamespace, secretName, err)
	}

	target.ResourceVersion = existing.ResourceVersion
	_, err = clientset.CoreV1().Secrets(targetNamespace).Update(ctx, target, metav1.UpdateOptions{})
	return err
}

func copySecretData(data map[string][]byte) map[string][]byte {
	if len(data) == 0 {
		return nil
	}
	copied := make(map[string][]byte, len(data))
	for key, value := range data {
		copied[key] = append([]byte(nil), value...)
	}
	return copied
}

func applyDirectObject(ctx context.Context, clientset *kubernetes.Clientset, obj runtime.Object) (string, error) {
	switch resource := obj.(type) {
	case *corev1.Pod:
		_, err := clientset.CoreV1().Pods(resource.Namespace).Create(ctx, resource, metav1.CreateOptions{})
		if err != nil && apierrors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().Pods(resource.Namespace).Update(ctx, resource, metav1.UpdateOptions{})
		}
		if err != nil {
			return "", fmt.Errorf("应用 Pod 失败: %v", err)
		}
		return "Pod/" + resource.Name, nil
	case *appsv1.Deployment:
		_, err := clientset.AppsV1().Deployments(resource.Namespace).Create(ctx, resource, metav1.CreateOptions{})
		if err != nil && apierrors.IsAlreadyExists(err) {
			current, getErr := clientset.AppsV1().Deployments(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if getErr != nil {
				return "", fmt.Errorf("读取 Deployment 失败: %v", getErr)
			}
			resource.ResourceVersion = current.ResourceVersion
			_, err = clientset.AppsV1().Deployments(resource.Namespace).Update(ctx, resource, metav1.UpdateOptions{})
		}
		if err != nil {
			return "", fmt.Errorf("应用 Deployment 失败: %v", err)
		}
		return "Deployment/" + resource.Name, nil
	case *corev1.Service:
		_, err := clientset.CoreV1().Services(resource.Namespace).Create(ctx, resource, metav1.CreateOptions{})
		if err != nil && apierrors.IsAlreadyExists(err) {
			current, getErr := clientset.CoreV1().Services(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
			if getErr != nil {
				return "", fmt.Errorf("读取 Service 失败: %v", getErr)
			}
			resource.ResourceVersion = current.ResourceVersion
			resource.Spec.ClusterIP = current.Spec.ClusterIP
			resource.Spec.ClusterIPs = current.Spec.ClusterIPs
			_, err = clientset.CoreV1().Services(resource.Namespace).Update(ctx, resource, metav1.UpdateOptions{})
		}
		if err != nil {
			return "", fmt.Errorf("应用 Service 失败: %v", err)
		}
		return "Service/" + resource.Name, nil
	default:
		return "", fmt.Errorf("不支持的 direct 资源类型: %T", obj)
	}
}

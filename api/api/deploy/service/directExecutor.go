package service

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"dodevops-api/api/deploy/model"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

var (
	directApplyTimeout           = 30 * time.Second
	directReadinessTimeout       = 5 * time.Minute
	directReadinessPollInterval  = 2 * time.Second
	directImagePullSecretSources = []string{"default", "autoops", "harbor", "jenkins"}
)

type DirectApplyResult struct {
	Namespace  string               `json:"namespace"`
	Applied    []string             `json:"applied"`
	Ready      []string             `json:"ready,omitempty"`
	Service    *DirectServiceResult `json:"service,omitempty"`
	NodeIPs    []string             `json:"nodeIps,omitempty"`
	AccessURLs []string             `json:"accessUrls,omitempty"`
	Warnings   []string             `json:"warnings,omitempty"`
}

type DirectServiceResult struct {
	Name      string              `json:"name"`
	Type      string              `json:"type"`
	ClusterIP string              `json:"clusterIp,omitempty"`
	Ports     []DirectServicePort `json:"ports,omitempty"`
}

type DirectServicePort struct {
	Name       string `json:"name,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
	Port       int32  `json:"port,omitempty"`
	TargetPort string `json:"targetPort,omitempty"`
	NodePort   int32  `json:"nodePort,omitempty"`
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

	return applyDirectResourcesWithClient(context.Background(), clientset, req, rendered)
}

func applyDirectResourcesWithClient(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest, rendered *DirectManifestRenderResult) (*DirectApplyResult, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}
	if rendered == nil {
		var err error
		rendered, err = RenderDirectManifest(req)
		if err != nil {
			return nil, err
		}
	}

	applyCtx, cancel := context.WithTimeout(ctx, directApplyTimeout)
	defer cancel()

	if err := ensureDirectNamespace(applyCtx, clientset, req); err != nil {
		return nil, err
	}
	if err := ensureDirectImagePullSecrets(applyCtx, clientset, req); err != nil {
		return nil, err
	}

	result := &DirectApplyResult{Namespace: req.Namespace}
	for _, obj := range rendered.Objects {
		applied, err := applyDirectObject(applyCtx, clientset, obj)
		if err != nil {
			return nil, err
		}
		result.Applied = append(result.Applied, applied)
	}

	readyCtx, readyCancel := context.WithTimeout(ctx, directReadinessTimeout)
	defer readyCancel()
	ready, err := waitDirectWorkloadReady(readyCtx, clientset, req)
	if err != nil {
		return nil, err
	}
	result.Ready = ready

	service, nodeIPs, accessURLs, warnings, err := collectDirectServiceResult(ctx, clientset, req)
	if err != nil {
		return nil, err
	}
	result.Service = service
	result.NodeIPs = nodeIPs
	result.AccessURLs = accessURLs
	result.Warnings = warnings
	return result, nil
}

func ensureDirectNamespace(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest) error {
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

func ensureDirectImagePullSecrets(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest) error {
	for _, ref := range directImagePullSecrets(req) {
		if ref.Name == "" {
			continue
		}
		if _, err := clientset.CoreV1().Secrets(req.Namespace).Get(ctx, ref.Name, metav1.GetOptions{}); err == nil {
			continue
		} else if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("检查目标 imagePullSecret %s/%s 失败: %v", req.Namespace, ref.Name, err)
		}
		sourceNamespace, err := findDirectImagePullSecretSource(ctx, clientset, req.Namespace, ref.Name)
		if err != nil {
			return fmt.Errorf("查找 imagePullSecret %s 的源 namespace 失败: %v", ref.Name, err)
		}
		if err := syncDirectImagePullSecret(ctx, clientset, sourceNamespace, req.Namespace, ref.Name); err != nil {
			return fmt.Errorf("同步 imagePullSecret %s 到 namespace %s 失败: %v", ref.Name, req.Namespace, err)
		}
	}
	return nil
}

func findDirectImagePullSecretSource(ctx context.Context, clientset kubernetes.Interface, targetNamespace string, secretName string) (string, error) {
	for _, namespace := range directImagePullSecretSources {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" || namespace == targetNamespace {
			continue
		}
		if _, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{}); err == nil {
			return namespace, nil
		} else if err != nil && !apierrors.IsNotFound(err) {
			continue
		}
	}

	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + secretName,
	})
	if err != nil {
		return "", err
	}
	for _, secret := range secrets.Items {
		if secret.Namespace != "" && secret.Namespace != targetNamespace && secret.Name == secretName {
			return secret.Namespace, nil
		}
	}
	return "", fmt.Errorf("未找到可复制的源 secret；请在 default/autoops/harbor/jenkins 或任一可读 namespace 中预置 %s", secretName)
}

func syncDirectImagePullSecret(ctx context.Context, clientset kubernetes.Interface, sourceNamespace string, targetNamespace string, secretName string) error {
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

func applyDirectObject(ctx context.Context, clientset kubernetes.Interface, obj runtime.Object) (string, error) {
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

func waitDirectWorkloadReady(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest) ([]string, error) {
	if req == nil {
		return nil, fmt.Errorf("部署申请不能为空")
	}

	var lastMessage string
	for {
		ready, resources, message := directWorkloadReady(ctx, clientset, req)
		if ready {
			return resources, nil
		}
		if message != "" {
			lastMessage = message
		}

		select {
		case <-ctx.Done():
			if lastMessage == "" {
				lastMessage = ctx.Err().Error()
			}
			return nil, fmt.Errorf("等待 direct workload ready 超时: %s", lastMessage)
		case <-time.After(directReadinessPollInterval):
		}
	}
}

func directWorkloadReady(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest) (bool, []string, string) {
	switch req.ResourceType {
	case model.DeployResourceTypePod:
		pod, err := clientset.CoreV1().Pods(req.Namespace).Get(ctx, req.ReleaseName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil, fmt.Sprintf("Pod/%s 尚未创建", req.ReleaseName)
		}
		if err != nil {
			return false, nil, fmt.Sprintf("读取 Pod/%s 失败: %v", req.ReleaseName, err)
		}
		if isDirectPodReady(pod) {
			return true, []string{"Pod/" + pod.Name}, ""
		}
		return false, nil, directPodNotReadyMessage(pod)
	case model.DeployResourceTypeDeployment:
		deployment, err := clientset.AppsV1().Deployments(req.Namespace).Get(ctx, req.ReleaseName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil, fmt.Sprintf("Deployment/%s 尚未创建", req.ReleaseName)
		}
		if err != nil {
			return false, nil, fmt.Sprintf("读取 Deployment/%s 失败: %v", req.ReleaseName, err)
		}
		if isDirectDeploymentReady(deployment) {
			return true, []string{"Deployment/" + deployment.Name}, ""
		}
		return false, nil, directDeploymentNotReadyMessage(deployment)
	default:
		return true, nil, ""
	}
}

func isDirectPodReady(pod *corev1.Pod) bool {
	if pod == nil || pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func directPodNotReadyMessage(pod *corev1.Pod) string {
	if pod == nil {
		return "Pod 状态未知"
	}
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status != corev1.ConditionTrue {
			return fmt.Sprintf("Pod/%s phase=%s ready=%s reason=%s message=%s", pod.Name, pod.Status.Phase, condition.Status, condition.Reason, condition.Message)
		}
	}
	return fmt.Sprintf("Pod/%s phase=%s ready=false", pod.Name, pod.Status.Phase)
}

func isDirectDeploymentReady(deployment *appsv1.Deployment) bool {
	if deployment == nil {
		return false
	}
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	if desired <= 0 {
		return true
	}
	return deployment.Status.ReadyReplicas >= desired && deployment.Status.AvailableReplicas >= desired
}

func directDeploymentNotReadyMessage(deployment *appsv1.Deployment) string {
	if deployment == nil {
		return "Deployment 状态未知"
	}
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing && condition.Status == corev1.ConditionFalse {
			return fmt.Sprintf("Deployment/%s desired=%d ready=%d available=%d reason=%s message=%s", deployment.Name, desired, deployment.Status.ReadyReplicas, deployment.Status.AvailableReplicas, condition.Reason, condition.Message)
		}
	}
	return fmt.Sprintf("Deployment/%s desired=%d ready=%d available=%d updated=%d", deployment.Name, desired, deployment.Status.ReadyReplicas, deployment.Status.AvailableReplicas, deployment.Status.UpdatedReplicas)
}

func collectDirectServiceResult(ctx context.Context, clientset kubernetes.Interface, req *model.DeployRequest) (*DirectServiceResult, []string, []string, []string, error) {
	if req == nil || !req.ServiceEnabled {
		return nil, nil, nil, nil, nil
	}

	service, err := clientset.CoreV1().Services(req.Namespace).Get(ctx, req.ReleaseName, metav1.GetOptions{})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("读取 Service/%s 失败: %v", req.ReleaseName, err)
	}

	result := directServiceResult(service)
	var warnings []string
	nodeIPs, nodeErr := collectDirectNodeIPs(ctx, clientset)
	if nodeErr != nil {
		warnings = append(warnings, fmt.Sprintf("获取节点 IP 失败，无法生成 NodePort URL: %v", nodeErr))
	}
	accessURLs := directAccessURLs(service, nodeIPs)
	return result, nodeIPs, accessURLs, warnings, nil
}

func directServiceResult(service *corev1.Service) *DirectServiceResult {
	if service == nil {
		return nil
	}
	result := &DirectServiceResult{
		Name:      service.Name,
		Type:      string(service.Spec.Type),
		ClusterIP: service.Spec.ClusterIP,
	}
	for _, port := range service.Spec.Ports {
		result.Ports = append(result.Ports, DirectServicePort{
			Name:       port.Name,
			Protocol:   string(port.Protocol),
			Port:       port.Port,
			TargetPort: directTargetPortString(port.TargetPort),
			NodePort:   port.NodePort,
		})
	}
	return result
}

func directTargetPortString(targetPort intstr.IntOrString) string {
	if targetPort.Type == intstr.String {
		return targetPort.StrVal
	}
	if targetPort.IntVal > 0 {
		return fmt.Sprint(targetPort.IntVal)
	}
	return ""
}

func collectDirectNodeIPs(ctx context.Context, clientset kubernetes.Interface) ([]string, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	var external []string
	var internal []string
	for _, node := range nodes.Items {
		for _, address := range node.Status.Addresses {
			ip := strings.TrimSpace(address.Address)
			if ip == "" {
				continue
			}
			if _, ok := seen[ip]; ok {
				continue
			}
			switch address.Type {
			case corev1.NodeExternalIP:
				external = append(external, ip)
				seen[ip] = struct{}{}
			case corev1.NodeInternalIP:
				internal = append(internal, ip)
				seen[ip] = struct{}{}
			}
		}
	}
	return append(external, internal...), nil
}

func directAccessURLs(service *corev1.Service, nodeIPs []string) []string {
	if service == nil {
		return nil
	}
	var urls []string
	switch service.Spec.Type {
	case corev1.ServiceTypeNodePort, corev1.ServiceTypeLoadBalancer:
		for _, port := range service.Spec.Ports {
			if port.NodePort <= 0 {
				continue
			}
			for _, nodeIP := range nodeIPs {
				urls = append(urls, fmt.Sprintf("http://%s:%d/", hostForURL(nodeIP), port.NodePort))
			}
		}
	}
	return dedupeStrings(urls)
}

func hostForURL(host string) string {
	if ip := net.ParseIP(host); ip != nil && strings.Contains(host, ":") {
		return "[" + host + "]"
	}
	return host
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
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

package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	ccdao "dodevops-api/api/configcenter/dao"

	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const DirectKubeconfigRefPrefix = "account:"

type DirectKubeconfigValidationResult struct {
	Valid      bool                    `json:"valid"`
	Ref        string                  `json:"ref"`
	Message    string                  `json:"message"`
	Host       string                  `json:"host,omitempty"`
	Namespace  string                  `json:"namespace,omitempty"`
	Permission []DirectPermissionCheck `json:"permissions,omitempty"`
	Warnings   []string                `json:"warnings,omitempty"`
}

type DirectPermissionCheck struct {
	Verb      string `json:"verb"`
	Group     string `json:"group"`
	Resource  string `json:"resource"`
	Namespace string `json:"namespace,omitempty"`
	Allowed   bool   `json:"allowed"`
	Reason    string `json:"reason,omitempty"`
}

func ResolveDirectKubeconfig(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("direct_kubeconfig_ref 为空")
	}
	if !strings.HasPrefix(ref, DirectKubeconfigRefPrefix) {
		return "", fmt.Errorf("direct_kubeconfig_ref 仅支持 account:<id|alias>")
	}

	accountRef := strings.TrimSpace(strings.TrimPrefix(ref, DirectKubeconfigRefPrefix))
	if accountRef == "" {
		return "", fmt.Errorf("direct_kubeconfig_ref 缺少 account 引用")
	}

	accountDao := ccdao.NewAccountAuthDao()
	var accountPassword string
	if id, err := strconv.ParseUint(accountRef, 10, 64); err == nil {
		account, err := accountDao.GetByID(uint(id))
		if err != nil {
			return "", fmt.Errorf("读取 account 凭据失败: %v", err)
		}
		accountPassword, err = account.DecryptPassword()
		if err != nil {
			return "", fmt.Errorf("解密 account 凭据失败: %v", err)
		}
		return accountPassword, nil
	}

	account, err := accountDao.GetByAlias(accountRef)
	if err != nil {
		return "", fmt.Errorf("读取 account 凭据失败: %v", err)
	}
	if account == nil {
		return "", fmt.Errorf("account 别名不存在: %s", accountRef)
	}
	accountPassword, err = account.DecryptPassword()
	if err != nil {
		return "", fmt.Errorf("解密 account 凭据失败: %v", err)
	}
	return accountPassword, nil
}

func ValidateDirectKubeconfigRef(ref string) (*DirectKubeconfigValidationResult, error) {
	kubeconfig, err := ResolveDirectKubeconfig(ref)
	if err != nil {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Message: err.Error()}, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Message: "解析 kubeconfig 失败: " + err.Error()}, err
	}

	validation := &DirectKubeconfigValidationResult{
		Valid:   true,
		Ref:     ref,
		Message: "kubeconfig 解析成功；尚未执行集群 API 权限探测",
		Host:    restConfig.Host,
	}
	return validation, nil
}

func ValidateDirectKubeconfigAccess(ref string, namespace string) (*DirectKubeconfigValidationResult, error) {
	kubeconfig, err := ResolveDirectKubeconfig(ref)
	if err != nil {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Message: err.Error()}, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Message: "解析 kubeconfig 失败: " + err.Error()}, err
	}
	restConfig.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Message: "创建 Kubernetes 客户端失败: " + err.Error(), Host: restConfig.Host}, err
	}

	targetNamespace := strings.TrimSpace(namespace)
	if targetNamespace == "" {
		targetNamespace = "ao-direct-probe"
	}
	if !strings.HasPrefix(targetNamespace, "ao-direct-") {
		return &DirectKubeconfigValidationResult{Valid: false, Ref: ref, Host: restConfig.Host, Namespace: targetNamespace, Message: "direct mode 权限探测命名空间必须以 ao-direct- 开头"}, fmt.Errorf("direct mode 权限探测命名空间必须以 ao-direct- 开头")
	}

	checks := []DirectPermissionCheck{
		{Verb: "create", Group: "", Resource: "namespaces"},
		{Verb: "create", Group: "apps", Resource: "deployments", Namespace: targetNamespace},
		{Verb: "create", Group: "", Resource: "pods", Namespace: targetNamespace},
		{Verb: "create", Group: "", Resource: "services", Namespace: targetNamespace},
		{Verb: "delete", Group: "apps", Resource: "deployments", Namespace: targetNamespace},
		{Verb: "delete", Group: "", Resource: "pods", Namespace: targetNamespace},
		{Verb: "delete", Group: "", Resource: "services", Namespace: targetNamespace},
		{Verb: "create", Group: "", Resource: "persistentvolumeclaims", Namespace: targetNamespace},
		{Verb: "create", Group: "networking.k8s.io", Resource: "ingresses", Namespace: targetNamespace},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := &DirectKubeconfigValidationResult{
		Valid:     true,
		Ref:       ref,
		Message:   "Kubernetes API 权限探测完成",
		Host:      restConfig.Host,
		Namespace: targetNamespace,
	}

	for _, check := range checks {
		allowed, reason := selfSubjectAccessReview(ctx, clientset, check)
		check.Allowed = allowed
		check.Reason = reason
		result.Permission = append(result.Permission, check)
	}

	result.Valid = evaluateDirectPermissionChecks(result.Permission)
	if !result.Valid {
		result.Message = "direct kubeconfig 权限不满足最小边界要求"
	}
	return result, nil
}

func NewDirectKubeClient(ref string) (*kubernetes.Clientset, error) {
	kubeconfig, err := ResolveDirectKubeconfig(ref)
	if err != nil {
		return nil, err
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("解析 kubeconfig 失败: %v", err)
	}
	restConfig.Timeout = 15 * time.Second
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Kubernetes 客户端失败: %v", err)
	}
	return clientset, nil
}

func selfSubjectAccessReview(ctx context.Context, clientset *kubernetes.Clientset, check DirectPermissionCheck) (bool, string) {
	review := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace: check.Namespace,
				Verb:      check.Verb,
				Group:     check.Group,
				Resource:  check.Resource,
			},
		},
	}
	resp, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
	if err != nil {
		return false, err.Error()
	}
	return resp.Status.Allowed, resp.Status.Reason
}

func evaluateDirectPermissionChecks(checks []DirectPermissionCheck) bool {
	for _, check := range checks {
		switch check.Resource {
		case "persistentvolumeclaims", "ingresses":
			if check.Allowed {
				return false
			}
		default:
			if !check.Allowed {
				return false
			}
		}
	}
	return true
}

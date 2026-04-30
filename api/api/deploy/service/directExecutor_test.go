package service

import (
	"context"
	"strings"
	"testing"
	"time"

	deploymodel "dodevops-api/api/deploy/model"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestEnsureDirectImagePullSecretsFindsSourceAcrossNamespaces(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "ao-direct-java-demo"}},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: HarborPullSecretName, Namespace: "java-demo-devtest"},
			Type:       corev1.SecretTypeDockerConfigJson,
			Data:       map[string][]byte{corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`)},
		},
	)
	req := &deploymodel.DeployRequest{
		Namespace: "ao-direct-java-demo",
		Image:     "10.0.17.205:80/java-demo/java-demo:v1",
	}

	if err := ensureDirectImagePullSecrets(context.Background(), clientset, req); err != nil {
		t.Fatalf("ensureDirectImagePullSecrets() error = %v", err)
	}
	secret, err := clientset.CoreV1().Secrets(req.Namespace).Get(context.Background(), HarborPullSecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected synced target secret: %v", err)
	}
	if secret.Type != corev1.SecretTypeDockerConfigJson {
		t.Fatalf("secret type = %s, want %s", secret.Type, corev1.SecretTypeDockerConfigJson)
	}
}

func TestWaitDirectWorkloadReadyDeployment(t *testing.T) {
	replicas := int32(1)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "java-demo", Namespace: "ao-direct-java-demo"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
		},
	})
	req := &deploymodel.DeployRequest{
		ResourceType: deploymodel.DeployResourceTypeDeployment,
		Namespace:    "ao-direct-java-demo",
		ReleaseName:  "java-demo",
	}

	ready, err := waitDirectWorkloadReady(context.Background(), clientset, req)
	if err != nil {
		t.Fatalf("waitDirectWorkloadReady() error = %v", err)
	}
	if len(ready) != 1 || ready[0] != "Deployment/java-demo" {
		t.Fatalf("ready resources = %v", ready)
	}
}

func TestWaitDirectWorkloadReadyTimesOutWhenDeploymentNotReady(t *testing.T) {
	oldTimeout := directReadinessTimeout
	oldPollInterval := directReadinessPollInterval
	directReadinessTimeout = 2 * time.Millisecond
	directReadinessPollInterval = time.Millisecond
	t.Cleanup(func() {
		directReadinessTimeout = oldTimeout
		directReadinessPollInterval = oldPollInterval
	})

	replicas := int32(1)
	clientset := fake.NewSimpleClientset(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "java-demo", Namespace: "ao-direct-java-demo"},
		Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
	})
	req := &deploymodel.DeployRequest{
		ResourceType: deploymodel.DeployResourceTypeDeployment,
		Namespace:    "ao-direct-java-demo",
		ReleaseName:  "java-demo",
	}
	ctx, cancel := context.WithTimeout(context.Background(), directReadinessTimeout)
	defer cancel()

	_, err := waitDirectWorkloadReady(ctx, clientset, req)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "等待 direct workload ready 超时") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCollectDirectServiceResultBuildsNodePortURLs(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "java-demo", Namespace: "ao-direct-java-demo"},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{
					{Name: "http", Port: 80, NodePort: 30278},
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "10.0.17.40"},
			}},
		},
	)
	req := &deploymodel.DeployRequest{
		Namespace:      "ao-direct-java-demo",
		ReleaseName:    "java-demo",
		ServiceEnabled: true,
	}

	service, nodeIPs, accessURLs, warnings, err := collectDirectServiceResult(context.Background(), clientset, req)
	if err != nil {
		t.Fatalf("collectDirectServiceResult() error = %v", err)
	}
	if service == nil || len(service.Ports) != 1 || service.Ports[0].NodePort != 30278 {
		t.Fatalf("unexpected service result: %+v", service)
	}
	if len(nodeIPs) != 1 || nodeIPs[0] != "10.0.17.40" {
		t.Fatalf("nodeIPs = %v", nodeIPs)
	}
	if len(accessURLs) != 1 || accessURLs[0] != "http://10.0.17.40:30278/" {
		t.Fatalf("accessURLs = %v", accessURLs)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
}

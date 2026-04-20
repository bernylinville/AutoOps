package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	deploymodel "dodevops-api/api/deploy/model"
	commonconfig "dodevops-api/common/config"
)

func TestDeployNotifierSendsSuccessText(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()

	if err := os.WriteFile(configPath, []byte("integrations:\n  deploy_bot:\n    provider: dingtalk\n    enabled: true\n    webhook_url: "+server.URL+"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := commonconfig.LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	store := &fakeDeployNotificationStore{}
	notifier := &DeployNotifier{store: store}
	now := time.Now()
	req := &deploymodel.DeployRequest{
		ID:              1,
		RequestNo:       "DRTEST001",
		Mode:            deploymodel.DeployModeDirect,
		Namespace:       "ao-direct-nginx",
		ReleaseName:     "nginx",
		ChatContextJSON: `{"provider":"dingtalk","chat_id":"chat-1","at_mobiles":["13800138000"]}`,
	}
	exec := &deploymodel.ExecutionRecord{
		RequestID: 1,
		Status:    deploymodel.ExecutionStatusSucceeded,
		Phase:     deploymodel.ExecutionPhaseQueued,
		EndedAt:   &now,
	}

	if err := notifier.NotifyExecutionResult(req, exec); err != nil {
		t.Fatalf("NotifyExecutionResult() error = %v", err)
	}

	record := store.items[0]
	if record.Status != deploymodel.NotificationStatusSent {
		t.Fatalf("expected sent status, got %s", record.Status)
	}
	for _, want := range []string{"DRTEST001", "ao-direct-nginx", "succeeded"} {
		if !strings.Contains(record.PayloadJSON, want) {
			t.Fatalf("expected payload json to contain %q, got %s", want, record.PayloadJSON)
		}
	}
}

func TestBuildMarkdown_IncludesAccessInfo(t *testing.T) {
	n := &DeployNotifier{}
	req := &deploymodel.DeployRequest{
		RequestNo:      "DR999",
		Mode:           deploymodel.DeployModeDirect,
		ReleaseName:    "nginx-demo",
		Namespace:      "ao-direct-nginx",
		Image:          "nginx:1.27.4-alpine",
		ServiceEnabled: true,
		ServiceType:    "NodePort",
		ServicePort:    80,
		TargetPort:     80,
	}
	exec := &deploymodel.ExecutionRecord{
		Status: deploymodel.ExecutionStatusSucceeded,
		Phase:  deploymodel.ExecutionPhaseQueued,
	}
	_, text := n.buildMarkdown(req, exec)
	if !strings.Contains(text, "镜像") {
		t.Fatalf("expected image line in markdown, got:\n%s", text)
	}
	if !strings.Contains(text, "nginx:1.27.4-alpine") {
		t.Fatalf("expected image value in markdown, got:\n%s", text)
	}
	if !strings.Contains(text, "NodePort") {
		t.Fatalf("expected service type in markdown, got:\n%s", text)
	}
	if !strings.Contains(text, "80") {
		t.Fatalf("expected service port in markdown, got:\n%s", text)
	}
}

type fakeDeployNotificationStore struct {
	items []deploymodel.DeployNotification
}

func (f *fakeDeployNotificationStore) CreateDeployNotification(notification *deploymodel.DeployNotification) error {
	notification.ID = uint(len(f.items) + 1)
	f.items = append(f.items, *notification)
	return nil
}

func (f *fakeDeployNotificationStore) UpdateDeployNotification(id uint, updates map[string]interface{}) error {
	for idx := range f.items {
		if f.items[idx].ID != id {
			continue
		}
		if status, ok := updates["status"].(string); ok {
			f.items[idx].Status = status
		}
		if errorMessage, ok := updates["error_message"].(string); ok {
			f.items[idx].ErrorMessage = errorMessage
		}
		if sentAt, ok := updates["sent_at"].(time.Time); ok {
			f.items[idx].SentAt = &sentAt
		}
		if payloadJSON, ok := updates["payload_json"].(string); ok {
			f.items[idx].PayloadJSON = payloadJSON
		}
		return nil
	}
	return nil
}

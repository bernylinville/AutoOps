package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	commonconfig "dodevops-api/common/config"
)

func TestBuildDingtalkApproversBuildsApprovalNodes(t *testing.T) {
	approvers := buildDingtalkApprovers("ding-user-1")
	if len(approvers) != 1 {
		t.Fatalf("expected 1 approver node, got %d", len(approvers))
	}
	if approvers[0].ActionType != "AND" {
		t.Fatalf("expected action type AND, got %q", approvers[0].ActionType)
	}
	if len(approvers[0].UserIDs) != 1 || approvers[0].UserIDs[0] != "ding-user-1" {
		t.Fatalf("unexpected user ids: %#v", approvers[0].UserIDs)
	}
}

func TestDingtalkCreateProcessInstanceRequestMarshalsApproversAsJSONArray(t *testing.T) {
	payload, err := json.Marshal(&DingtalkCreateProcessInstanceRequest{
		ProcessCode:      "PROC-TEST",
		OriginatorUserID: "originator-1",
		DeptID:           155776048,
		Approvers:        buildDingtalkApprovers("ding-user-1"),
		FormComponentValues: []DingtalkFormComponentValue{
			{Name: "申请单号", Value: "DRTEST001"},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	expected := `{"processCode":"PROC-TEST","originatorUserId":"originator-1","deptId":155776048,"approvers":[{"actionType":"AND","userIds":["ding-user-1"]}],"formComponentValues":[{"name":"申请单号","value":"DRTEST001"}]}`
	if string(payload) != expected {
		t.Fatalf("unexpected payload: %s", string(payload))
	}
}

func TestGetAccessTokenCachesUntilExpiry(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(`dingtalkApproval:
  client_id: "app-key"
  client_secret: "app-secret"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := commonconfig.LoadConfig(configPath); err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1.0/oauth2/accessToken" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"accessToken":"token-1","expireIn":7200}`))
	}))
	defer server.Close()

	client := &DingtalkApprovalService{
		httpClient: server.Client(),
		baseURL:    server.URL,
	}

	first, err := client.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("first GetAccessToken() error = %v", err)
	}
	second, err := client.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("second GetAccessToken() error = %v", err)
	}
	if first != "token-1" || second != "token-1" {
		t.Fatalf("unexpected tokens: first=%q second=%q", first, second)
	}
	if calls != 1 {
		t.Fatalf("expected token endpoint to be called once, got %d", calls)
	}
}

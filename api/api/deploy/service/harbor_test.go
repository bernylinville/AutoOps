package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ccmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
)

func TestHarborAdapterGetArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		if r.URL.Path != "/api/v2.0/projects/library/repositories/nginx/demo/artifacts/v1.0.0" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("with_scan_overview") != "true" {
			t.Fatalf("expected with_scan_overview=true, got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{
			"digest":"sha256:abc",
			"size":12345,
			"push_time":"2026-04-21T10:00:00Z",
			"tags":[{"name":"v1.0.0"},{"name":"latest"}],
			"scan_overview":{
				"application/vnd.security.vulnerability.report; version=1.1":{
					"scan_status":"Success",
					"severity":"High",
					"complete_percent":100,
					"summary":{"summary":{"Critical":1,"High":2,"Medium":3,"Low":4}}
				}
			}
		}`))
	}))
	defer server.Close()

	adapter := newTestHarborAdapter(t, server.URL, true)
	artifact, err := adapter.GetArtifact(context.Background(), 1, "library", "nginx/demo", "v1.0.0")
	if err != nil {
		t.Fatalf("GetArtifact() error = %v", err)
	}
	if artifact.Digest != "sha256:abc" {
		t.Fatalf("Digest = %q, want sha256:abc", artifact.Digest)
	}
	if len(artifact.Tags) != 2 || artifact.Tags[0] != "v1.0.0" || artifact.Tags[1] != "latest" {
		t.Fatalf("unexpected tags: %#v", artifact.Tags)
	}
	if artifact.ScanOverview == nil {
		t.Fatal("expected scan overview")
	}
	if artifact.ScanOverview.ScanStatus != "Success" {
		t.Fatalf("ScanStatus = %q, want Success", artifact.ScanOverview.ScanStatus)
	}
	if artifact.ScanOverview.Summary["Critical"] != 1 || artifact.ScanOverview.Summary["High"] != 2 {
		t.Fatalf("unexpected summary: %#v", artifact.ScanOverview.Summary)
	}
}

func TestHarborAdapterPollScanUntilCompleteTriggersScan(t *testing.T) {
	var artifactCalls, triggerCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/scan"):
			triggerCalls++
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/artifacts/"):
			artifactCalls++
			status := "Pending"
			percent := 20
			if artifactCalls >= 2 {
				status = "Success"
				percent = 100
			}
			payload := map[string]interface{}{
				"digest": "sha256:poll",
				"scan_overview": map[string]interface{}{
					"application/vnd.security.vulnerability.report; version=1.1": map[string]interface{}{
						"scan_status":      status,
						"severity":         "Medium",
						"complete_percent": percent,
						"summary": map[string]interface{}{
							"summary": map[string]interface{}{"High": 1, "Medium": 2},
						},
					},
				},
			}
			if artifactCalls == 1 {
				payload["scan_overview"] = map[string]interface{}{}
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	adapter := NewHarborAdapter(HarborAdapterOptions{
		AccountDao:         fakeHarborAccountStore(t, server.URL),
		HTTPClient:         server.Client(),
		AllowInsecureHTTP:  true,
		PollInterval:       10 * time.Millisecond,
		DefaultScanTimeout: 500 * time.Millisecond,
	})

	overview, err := adapter.PollScanUntilComplete(context.Background(), 1, "library", "nginx/demo", "sha256:poll", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("PollScanUntilComplete() error = %v", err)
	}
	if triggerCalls != 1 {
		t.Fatalf("expected triggerCalls=1, got %d", triggerCalls)
	}
	if overview == nil || overview.ScanStatus != "Success" || overview.CompletePercent != 100 {
		t.Fatalf("unexpected overview: %#v", overview)
	}
}

func TestHarborAdapterGetVulnerabilities(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBasicAuth(t, r)
		if !strings.HasSuffix(r.URL.Path, "/additions/vulnerabilities") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Accept-Vulnerabilities"); got == "" {
			t.Fatal("expected X-Accept-Vulnerabilities header")
		}
		_, _ = w.Write([]byte(`{
			"generated_at":"2026-04-21T11:00:00Z",
			"vulnerabilities":[
				{
					"id":"CVE-2026-0001",
					"severity":"Critical",
					"package":"openssl",
					"version":"1.0.0",
					"fix_version":"1.0.1",
					"description":"critical vulnerability",
					"links":["https://cve.example/CVE-2026-0001"]
				},
				{
					"id":"CVE-2026-0002",
					"severity":"High",
					"package":{"name":"glibc","version":"2.31","fix_version":"2.32"},
					"description":"high vulnerability",
					"links":[{"href":"https://cve.example/CVE-2026-0002"}]
				}
			]
		}`))
	}))
	defer server.Close()

	adapter := newTestHarborAdapter(t, server.URL, true)
	vulns, err := adapter.GetVulnerabilities(context.Background(), 1, "library", "nginx/demo", "sha256:abc")
	if err != nil {
		t.Fatalf("GetVulnerabilities() error = %v", err)
	}
	if len(vulns) != 2 {
		t.Fatalf("expected 2 vulnerabilities, got %d", len(vulns))
	}
	if vulns[0].Package != "openssl" || vulns[1].Package != "glibc" {
		t.Fatalf("unexpected packages: %#v", vulns)
	}
	if vulns[1].FixVersion != "2.32" {
		t.Fatalf("unexpected fix version: %#v", vulns[1])
	}
}

func TestHarborAdapterEvaluateScanPolicy(t *testing.T) {
	adapter := &HarborAdapter{}
	overview := &HarborScanOverview{
		ScanStatus: "Success",
		Summary: map[string]int{
			"Critical": 1,
			"High":     3,
			"Medium":   5,
			"Low":      2,
		},
	}

	result, err := adapter.EvaluateScanPolicy(overview, &ScanPolicy{MaxCritical: 0, MaxHigh: 2, MaxMedium: 10})
	if err != nil {
		t.Fatalf("EvaluateScanPolicy() error = %v", err)
	}
	if result.Passed {
		t.Fatalf("expected policy to fail, got %+v", result)
	}
	if !strings.Contains(result.Message, "Critical") || !strings.Contains(result.Message, "High") {
		t.Fatalf("unexpected message: %s", result.Message)
	}

	passResult, err := adapter.EvaluateScanPolicy(overview, &ScanPolicy{MaxCritical: 1, MaxHigh: 3})
	if err != nil {
		t.Fatalf("EvaluateScanPolicy() second error = %v", err)
	}
	if !passResult.Passed {
		t.Fatalf("expected policy to pass, got %+v", passResult)
	}
}

func TestHarborAdapterNormalizeBaseURL(t *testing.T) {
	adapter := &HarborAdapter{allowInsecureHTTP: true}
	baseURL, err := adapter.normalizeHarborBaseURL("harbor.internal", 8080)
	if err != nil {
		t.Fatalf("normalizeHarborBaseURL() error = %v", err)
	}
	if baseURL != "http://harbor.internal:8080/api/v2.0" {
		t.Fatalf("baseURL = %q, want http://harbor.internal:8080/api/v2.0", baseURL)
	}

	secureAdapter := &HarborAdapter{allowInsecureHTTP: false}
	secureURL, err := secureAdapter.normalizeHarborBaseURL("https://harbor.example.com", 443)
	if err != nil {
		t.Fatalf("normalizeHarborBaseURL() secure error = %v", err)
	}
	if secureURL != "https://harbor.example.com/api/v2.0" {
		t.Fatalf("secureURL = %q, want https://harbor.example.com/api/v2.0", secureURL)
	}
}

func TestHarborAdapterHandlesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`[{"code":"NOT_FOUND","message":"artifact not found"}]`))
	}))
	defer server.Close()

	adapter := newTestHarborAdapter(t, server.URL, true)
	_, err := adapter.GetArtifact(context.Background(), 1, "library", "nginx/demo", "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "artifact not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeAccountAuthDao struct {
	account *ccmodel.AccountAuth
	err     error
}

func (f *fakeAccountAuthDao) GetByID(id uint) (*ccmodel.AccountAuth, error) {
	return f.account, f.err
}

func newTestHarborAdapter(t *testing.T, host string, allowInsecure bool) *HarborAdapter {
	t.Helper()
	adapter := NewHarborAdapter(HarborAdapterOptions{
		AccountDao:         fakeHarborAccountStore(t, host),
		HTTPClient:         http.DefaultClient,
		AllowInsecureHTTP:  allowInsecure,
		PollInterval:       10 * time.Millisecond,
		DefaultScanTimeout: 500 * time.Millisecond,
	}).(*HarborAdapter)
	adapter.httpClient = &http.Client{}
	return adapter
}

func fakeHarborAccountStore(t *testing.T, host string) harborAccountStore {
	t.Helper()
	encrypted, err := util.AESEncrypt("robot-secret")
	if err != nil {
		t.Fatalf("AESEncrypt() error = %v", err)
	}
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return &fakeAccountAuthDao{account: &ccmodel.AccountAuth{
		ID:       1,
		Alias:    "harbor-test",
		Host:     host,
		Port:     0,
		Name:     "robot$autoops",
		Password: encrypted,
	}}
}

func assertBasicAuth(t *testing.T, r *http.Request) {
	t.Helper()
	username, password, ok := r.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth")
	}
	if username != "robot$autoops" || password != "robot-secret" {
		t.Fatalf("unexpected auth: %q / %q", username, password)
	}
}

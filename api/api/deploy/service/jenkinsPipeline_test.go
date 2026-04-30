package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	appmodel "dodevops-api/api/app/model"
	appservice "dodevops-api/api/app/service"
	ccmodel "dodevops-api/api/configcenter/model"
)

func TestJenkinsPipelineTriggerBuildWithParameters(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/job/folder/job/demo/buildWithParameters" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("GIT_REF"); got != "refs/heads/main" {
			t.Fatalf("unexpected GIT_REF: %s", got)
		}
		if got := r.URL.Query().Get("IMAGE_TAG"); got != "release-1" {
			t.Fatalf("unexpected IMAGE_TAG: %s", got)
		}
		w.Header().Set("Location", "/queue/item/321/")
		w.WriteHeader(http.StatusCreated)
	}))

	queueID, err := adapter.TriggerBuild(context.Background(), 1, "folder/demo", map[string]string{
		"GIT_REF":   "refs/heads/main",
		"IMAGE_TAG": "release-1",
	})
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if queueID != 321 {
		t.Fatalf("expected queueID 321, got %d", queueID)
	}
}

func TestJenkinsPipelineTriggerBuildRetriesWithCrumb(t *testing.T) {
	var buildCalls int32
	var crumbCalls int32
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crumbIssuer/api/json":
			atomic.AddInt32(&crumbCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			http.SetCookie(w, &http.Cookie{Name: "JSESSIONID", Value: "jenkins-session-1", Path: "/"})
			_, _ = w.Write([]byte(`{"crumbRequestField":"Jenkins-Crumb","crumb":"crumb-123"}`))
			return
		case "/job/demo/buildWithParameters":
			call := atomic.AddInt32(&buildCalls, 1)
			if call == 1 {
				http.Error(w, "No valid crumb was included in the request", http.StatusForbidden)
				return
			}
			if got := r.Header.Get("Jenkins-Crumb"); got != "crumb-123" {
				t.Fatalf("expected Jenkins crumb header on retry, got %q", got)
			}
			if cookie, err := r.Cookie("JSESSIONID"); err != nil || cookie.Value != "jenkins-session-1" {
				t.Fatalf("expected Jenkins session cookie on crumb retry, cookie=%v err=%v", cookie, err)
			}
			w.Header().Set("Location", "/queue/item/322/")
			w.WriteHeader(http.StatusCreated)
			return
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))

	queueID, err := adapter.TriggerBuild(context.Background(), 1, "demo", map[string]string{
		"GIT_REF": "main",
	})
	if err != nil {
		t.Fatalf("TriggerBuild() error = %v", err)
	}
	if queueID != 322 {
		t.Fatalf("expected queueID 322, got %d", queueID)
	}
	if atomic.LoadInt32(&buildCalls) != 2 || atomic.LoadInt32(&crumbCalls) != 1 {
		t.Fatalf("unexpected calls: build=%d crumb=%d", buildCalls, crumbCalls)
	}
}

func TestJenkinsPipelineGetBuildNumberFromQueuePolls(t *testing.T) {
	var calls int32
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/queue/item/99/api/json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		call := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			_, _ = w.Write([]byte(`{"why":"Waiting for next available executor"}`))
			return
		}
		_, _ = w.Write([]byte(`{"executable":{"number":56}}`))
	}))

	buildNumber, err := adapter.GetBuildNumberFromQueue(context.Background(), 1, 99)
	if err != nil {
		t.Fatalf("GetBuildNumberFromQueue() error = %v", err)
	}
	if buildNumber != 56 {
		t.Fatalf("expected build number 56, got %d", buildNumber)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected polling, got %d call(s)", calls)
	}
}

func TestJenkinsPipelineGetBuildDetailParsesParameters(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/demo/18/api/json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"number":18,
			"url":"http://jenkins/job/demo/18/",
			"result":"SUCCESS",
			"building":false,
			"duration":1234,
			"timestamp":1710000000000,
			"actions":[
				{"parameters":[{"name":"GIT_REF","value":"refs/tags/v1.0.0"},{"name":"REPLICAS","value":3}]}
			]
		}`))
	}))

	detail, err := adapter.GetBuildDetail(context.Background(), 1, "demo", 18)
	if err != nil {
		t.Fatalf("GetBuildDetail() error = %v", err)
	}
	if detail.Number != 18 || detail.Result != "SUCCESS" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
	if detail.Parameters["GIT_REF"] != "refs/tags/v1.0.0" {
		t.Fatalf("unexpected GIT_REF parameter: %+v", detail.Parameters)
	}
	if detail.Parameters["REPLICAS"] != "3" {
		t.Fatalf("unexpected REPLICAS parameter: %+v", detail.Parameters)
	}
}

func TestJenkinsPipelinePollBuildUntilComplete(t *testing.T) {
	var calls int32
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/demo/7/api/json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		call := atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		if call == 1 {
			_, _ = w.Write([]byte(`{"number":7,"url":"http://jenkins/job/demo/7/","building":true,"result":"","duration":0,"timestamp":1710000000000}`))
			return
		}
		_, _ = w.Write([]byte(`{"number":7,"url":"http://jenkins/job/demo/7/","building":false,"result":"UNSTABLE","duration":4321,"timestamp":1710000000000}`))
	}))

	detail, err := adapter.PollBuildUntilComplete(context.Background(), 1, "demo", 7, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("PollBuildUntilComplete() error = %v", err)
	}
	if detail.Result != "UNSTABLE" {
		t.Fatalf("expected UNSTABLE result, got %+v", detail)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Fatalf("expected at least 2 polling calls, got %d", calls)
	}
}

func TestJenkinsPipelineExtractImageTagFromBuildLog(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/job/demo/11/consoleText" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("start\nIMAGE_TAG=release-20260421\ndocker push registry.example.com/demo/app:release-20260421\n"))
	}))

	imageTag, err := adapter.ExtractImageTagFromBuildLog(context.Background(), 1, "demo", 11)
	if err != nil {
		t.Fatalf("ExtractImageTagFromBuildLog() error = %v", err)
	}
	if imageTag != "release-20260421" {
		t.Fatalf("expected release-20260421, got %s", imageTag)
	}
}

func TestJenkinsPipelineGetBuildNumberFromQueueCancelled(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cancelled":true}`))
	}))

	_, err := adapter.GetBuildNumberFromQueue(context.Background(), 1, 101)
	if err == nil {
		t.Fatal("expected cancelled queue error")
	}
	if !strings.Contains(err.Error(), "已取消") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeJenkinsAccountStore struct {
	account *ccmodel.AccountAuth
	err     error
}

func (f *fakeJenkinsAccountStore) GetByID(id uint) (*ccmodel.AccountAuth, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.account, nil
}

func newTestJenkinsPipelineAdapter(t *testing.T, handler http.Handler) *JenkinsPipelineAdapter {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	host, port, splitErr := net.SplitHostPort(parsed.Host)
	if splitErr != nil {
		t.Fatalf("unexpected host format: %s", parsed.Host)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("parse port: %v", err)
	}

	account := &ccmodel.AccountAuth{
		ID:       1,
		Alias:    "jenkins-test",
		Host:     host,
		Port:     portNumber,
		Name:     "tester",
		Password: "secret",
		Type:     appmodel.JenkinsAccountType,
	}
	if err := account.EncryptPassword(); err != nil {
		t.Fatalf("EncryptPassword() error = %v", err)
	}

	return &JenkinsPipelineAdapter{
		accountDao: &fakeJenkinsAccountStore{account: account},
		newClient: func(host string, port int, username, password string) *appservice.JenkinsClient {
			return appservice.NewJenkinsClient(host, port, username, password)
		},
		pollInterval:     10 * time.Millisecond,
		defaultTimeout:   200 * time.Millisecond,
		imageTagPatterns: compileJenkinsImageTagPatterns(),
	}
}

func TestBuildJenkinsJobEndpoint(t *testing.T) {
	got, err := buildJenkinsJobEndpoint("folder/sub/demo")
	if err != nil {
		t.Fatalf("buildJenkinsJobEndpoint() error = %v", err)
	}
	want := "/job/folder/job/sub/job/demo"
	if got != want {
		t.Fatalf("expected %s, got %s", want, got)
	}
}

func TestParseJenkinsQueueID(t *testing.T) {
	queueID, err := parseJenkinsQueueID("http://jenkins.local/queue/item/654/")
	if err != nil {
		t.Fatalf("parseJenkinsQueueID() error = %v", err)
	}
	if queueID != 654 {
		t.Fatalf("expected 654, got %d", queueID)
	}
}

func TestJenkinsPipelineTriggerBuildMissingLocation(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	_, err := adapter.TriggerBuild(context.Background(), 1, "demo", nil)
	if err == nil {
		t.Fatal("expected missing location error")
	}
	if !strings.Contains(err.Error(), "Location") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelineUsesJenkinsAccountGuard(t *testing.T) {
	adapter := &JenkinsPipelineAdapter{
		accountDao: &fakeJenkinsAccountStore{account: &ccmodel.AccountAuth{ID: 1, Type: 99}},
		newClient:  func(host string, port int, username, password string) *appservice.JenkinsClient { return nil },
	}

	_, err := adapter.GetBuildDetail(context.Background(), 1, "demo", 1)
	if err == nil {
		t.Fatal("expected account type error")
	}
	if !strings.Contains(err.Error(), "账号类型不是 Jenkins") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelineDoRequestReturnsStatusBody(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "build missing", http.StatusNotFound)
	}))

	_, err := adapter.GetBuildDetail(context.Background(), 1, "demo", 33)
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "build missing") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelineExtractImageTagFallsBackToDockerPushLine(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("docker push registry.example.com/demo/app:v2.0.1\n"))
	}))

	got, err := adapter.ExtractImageTagFromBuildLog(context.Background(), 1, "demo", 12)
	if err != nil {
		t.Fatalf("ExtractImageTagFromBuildLog() error = %v", err)
	}
	if got != "registry.example.com/demo/app:v2.0.1" {
		t.Fatalf("unexpected image ref: %s", got)
	}
}

func TestJenkinsPipelineGetBuildNumberFromQueueHonorsContext(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"why":"still waiting"}`))
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := adapter.GetBuildNumberFromQueue(ctx, 1, 888)
	if err == nil {
		t.Fatal("expected context timeout error")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelinePollBuildUntilCompleteHonorsTimeout(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"number":7,"url":"http://jenkins/job/demo/7/","building":true,"result":"","duration":0,"timestamp":1710000000000}`))
	}))

	_, err := adapter.PollBuildUntilComplete(context.Background(), 1, "demo", 7, 25*time.Millisecond)
	if err == nil {
		t.Fatal("expected poll timeout error")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelineExtractImageTagMissing(t *testing.T) {
	adapter := newTestJenkinsPipelineAdapter(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("plain console output without image data\n"))
	}))

	_, err := adapter.ExtractImageTagFromBuildLog(context.Background(), 1, "demo", 13)
	if err == nil {
		t.Fatal("expected image tag extraction error")
	}
	if !strings.Contains(err.Error(), "未在 Jenkins 构建日志中提取到镜像标签") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJenkinsPipelineGetBuildDetailRejectsInvalidBuildNumber(t *testing.T) {
	adapter := &JenkinsPipelineAdapter{}
	_, err := adapter.getBuildDetailWithClient(context.Background(), &appservice.JenkinsClient{}, "demo", 0)
	if err == nil {
		t.Fatal("expected invalid buildNumber error")
	}
	if !strings.Contains(err.Error(), "buildNumber 非法") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContainsHTTPStatus(t *testing.T) {
	if !containsHTTPStatus([]int{http.StatusOK, http.StatusCreated}, http.StatusCreated) {
		t.Fatal("expected status to be contained")
	}
	if containsHTTPStatus([]int{http.StatusOK}, http.StatusAccepted) {
		t.Fatal("did not expect status to be contained")
	}
}

func TestNewJenkinsPipelineAdapterUsesDefaults(t *testing.T) {
	adapter, ok := NewJenkinsPipelineAdapter().(*JenkinsPipelineAdapter)
	if !ok {
		t.Fatal("expected concrete JenkinsPipelineAdapter")
	}
	if adapter.accountDao == nil || adapter.newClient == nil {
		t.Fatal("expected adapter dependencies to be initialized")
	}
	if adapter.pollInterval != defaultJenkinsPipelinePollInterval {
		t.Fatalf("unexpected poll interval: %s", adapter.pollInterval)
	}
	if adapter.defaultTimeout != defaultJenkinsPipelineTimeout {
		t.Fatalf("unexpected default timeout: %s", adapter.defaultTimeout)
	}
}

func TestEffectivePollIntervalFallsBackToDefault(t *testing.T) {
	adapter := &JenkinsPipelineAdapter{}
	if got := adapter.effectivePollInterval(); got != defaultJenkinsPipelinePollInterval {
		t.Fatalf("expected default poll interval %s, got %s", defaultJenkinsPipelinePollInterval, got)
	}
}

func TestIsJenkinsBuildComplete(t *testing.T) {
	if isJenkinsBuildComplete(&JenkinsBuildDetail{Building: true, Result: "SUCCESS"}) {
		t.Fatal("building build should not be complete")
	}
	if !isJenkinsBuildComplete(&JenkinsBuildDetail{Building: false, Result: "FAILURE"}) {
		t.Fatal("finished build should be complete")
	}
	if isJenkinsBuildComplete(nil) {
		t.Fatal("nil detail should not be complete")
	}
}

func TestParseJenkinsQueueIDRejectsInvalidValue(t *testing.T) {
	_, err := parseJenkinsQueueID("/queue/item/not-a-number/")
	if err == nil {
		t.Fatal("expected invalid queue id error")
	}
}

func TestBuildJenkinsJobEndpointRejectsEmptyValue(t *testing.T) {
	_, err := buildJenkinsJobEndpoint("  /  ")
	if err == nil {
		t.Fatal("expected empty job name error")
	}
}

func TestJenkinsPipelineGetJenkinsClientDecryptsPassword(t *testing.T) {
	account := &ccmodel.AccountAuth{
		ID:       1,
		Alias:    "jenkins-test",
		Host:     "localhost",
		Port:     8080,
		Name:     "tester",
		Password: "secret",
		Type:     appmodel.JenkinsAccountType,
	}
	if err := account.EncryptPassword(); err != nil {
		t.Fatalf("EncryptPassword() error = %v", err)
	}

	adapter := &JenkinsPipelineAdapter{
		accountDao: &fakeJenkinsAccountStore{account: account},
		newClient: func(host string, port int, username, password string) *appservice.JenkinsClient {
			if password != "secret" {
				t.Fatalf("expected decrypted password secret, got %s", password)
			}
			return &appservice.JenkinsClient{BaseURL: fmt.Sprintf("http://%s:%d", host, port), Username: username, Password: password, HTTPClient: http.DefaultClient}
		},
	}

	client, err := adapter.getJenkinsClient(1)
	if err != nil {
		t.Fatalf("getJenkinsClient() error = %v", err)
	}
	if client.Username != "tester" {
		t.Fatalf("unexpected username: %s", client.Username)
	}
}

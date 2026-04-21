package dingtalkbot

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientSendText(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()

	client := NewClient(Config{WebhookURL: server.URL})
	if err := client.SendText("hello", []string{"13800138000"}, []string{"user-1"}); err != nil {
		t.Fatalf("SendText() error = %v", err)
	}

	for _, want := range []string{"\"msgtype\":\"text\"", "hello", "13800138000", "user-1"} {
		if !strings.Contains(captured, want) {
			t.Fatalf("expected payload to contain %q, got %s", want, captured)
		}
	}
}

func TestClientSendMarkdown(t *testing.T) {
	var captured string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured = string(body)
		_, _ = w.Write([]byte(`{"errcode":0}`))
	}))
	defer server.Close()

	client := NewClient(Config{WebhookURL: server.URL, Secret: "secret"})
	if err := client.SendMarkdown("title", "markdown text", nil, nil); err != nil {
		t.Fatalf("SendMarkdown() error = %v", err)
	}

	for _, want := range []string{"\"msgtype\":\"markdown\"", "title", "markdown text"} {
		if !strings.Contains(captured, want) {
			t.Fatalf("expected payload to contain %q, got %s", want, captured)
		}
	}
}

func TestClientReturnsRobotError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"errcode":310000,"errmsg":"keyword not match"}`))
	}))
	defer server.Close()

	client := NewClient(Config{WebhookURL: server.URL})
	err := client.SendText("hello", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "errcode=310000") {
		t.Fatalf("expected robot error, got %v", err)
	}
}

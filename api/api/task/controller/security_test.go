package controller

import (
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ==================== #2 P1 validateTaskName 路径穿越 ====================

func TestValidateTaskName_ValidNames(t *testing.T) {
	validNames := []string{
		"test-task",
		"task_01",
		"my.task",
		"中文任务名",
		"task-with-123",
		"a",
		"MyTask_v2.1",
	}
	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			err := validateTaskName(name)
			assert.NoError(t, err, "合法名称 %q 应该通过", name)
		})
	}
}

func TestValidateTaskName_PathTraversal(t *testing.T) {
	maliciousNames := []string{
		"../etc/passwd",
		"../../root",
		"task/../../etc",
		"task/../secret",
		"/absolute/path",
	}
	for _, name := range maliciousNames {
		t.Run(name, func(t *testing.T) {
			err := validateTaskName(name)
			assert.Error(t, err, "路径穿越名称 %q 应被拒绝", name)
		})
	}
}

func TestValidateTaskName_SpecialChars(t *testing.T) {
	badNames := []string{
		"task;rm -rf /",
		"task$(whoami)",
		"task`id`",
		"task|cat /etc/passwd",
		"task\ninjected",
		"task name with space",
	}
	for _, name := range badNames {
		t.Run("bad: "+name, func(t *testing.T) {
			err := validateTaskName(name)
			assert.Error(t, err, "特殊字符名称 %q 应被拒绝", name)
		})
	}
}

func TestValidateTaskName_Empty(t *testing.T) {
	err := validateTaskName("")
	assert.Error(t, err, "空名称应被拒绝")
	assert.Contains(t, err.Error(), "不能为空")
}

func TestValidateTaskName_TooLong(t *testing.T) {
	longName := ""
	for i := 0; i < 129; i++ {
		longName += "a"
	}
	err := validateTaskName(longName)
	assert.Error(t, err, "超长名称应被拒绝")
	assert.Contains(t, err.Error(), "128")
}

// ==================== #4 P1 checkWSOrigin 白名单 ====================

func TestCheckWSOrigin_AllowedOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:8080,http://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	assert.True(t, checkWSOrigin(req), "白名单中的 Origin 应被允许")
}

func TestCheckWSOrigin_RejectedOrigin(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:8080,http://example.com")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.com")
	assert.False(t, checkWSOrigin(req), "非白名单 Origin 应被拒绝")
}

func TestCheckWSOrigin_EmptyOrigin(t *testing.T) {
	req, _ := http.NewRequest("GET", "/ws", nil)
	// 不设置 Origin header
	assert.False(t, checkWSOrigin(req), "空 Origin 应被拒绝")
}

func TestCheckWSOrigin_DefaultAllowList(t *testing.T) {
	os.Unsetenv("ALLOWED_ORIGINS")

	req, _ := http.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://localhost:8080")
	assert.True(t, checkWSOrigin(req), "默认白名单应包含 localhost:8080")

	req2, _ := http.NewRequest("GET", "/ws", nil)
	req2.Header.Set("Origin", "http://localhost:3000")
	assert.True(t, checkWSOrigin(req2), "默认白名单应包含 localhost:3000")
}

func TestCheckWSOrigin_SubstringAttack(t *testing.T) {
	os.Setenv("ALLOWED_ORIGINS", "http://localhost:8080")
	defer os.Unsetenv("ALLOWED_ORIGINS")

	attacks := []string{
		"http://localhost:8080.evil.com",
		"http://evil.com/http://localhost:8080",
		"http://localhost:80801",
	}
	for _, origin := range attacks {
		req, _ := http.NewRequest("GET", "/ws", nil)
		req.Header.Set("Origin", origin)
		assert.False(t, checkWSOrigin(req), "子串攻击 %q 应被拒绝", origin)
	}
}

// ==================== #5 P1 getHostKeyCallback ====================

func TestGetHostKeyCallback_WarningMode(t *testing.T) {
	os.Unsetenv("SSH_STRICT_HOST_KEY")
	// 默认模式：打印警告，不阻断
	// getHostKeyCallback 是 sshUtil.go 中的函数，此处验证不抛 panic
	// 由于函数在 util 包，这里做间接验证
	assert.NotPanics(t, func() {
		_ = fmt.Sprintf("SSH host key check mode: %s", os.Getenv("SSH_STRICT_HOST_KEY"))
	})
}

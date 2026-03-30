// 审计日志中间件
// 捕获高危操作的请求体、响应码、耗时等审计信息

package middleware

import (
	"bytes"
	"dodevops-api/api/system/dao"
	"dodevops-api/api/system/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/jwt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxRequestBodySize = 4096 // 请求体最大捕获 4KB

// responseCapture 用于捕获响应状态码
type responseCapture struct {
	gin.ResponseWriter
	statusCode int
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// AuditMiddleware 审计日志中间件
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := strings.ToUpper(c.Request.Method)

		// 仅记录写操作
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			c.Next()
			return
		}

		// 捕获请求体
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, maxRequestBodySize+1))
			if err == nil {
				if len(bodyBytes) > maxRequestBodySize {
					requestBody = string(bodyBytes[:maxRequestBodySize]) + "...(truncated)"
				} else {
					requestBody = string(bodyBytes)
				}
				// 回写 Body 供后续 handler 使用
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 脱敏: 移除密码等敏感字段
		requestBody = sanitizeRequestBody(requestBody)

		// 包装 ResponseWriter 以捕获状态码
		capture := &responseCapture{ResponseWriter: c.Writer, statusCode: 200}
		c.Writer = capture

		startTime := time.Now()

		// 执行后续 handler
		c.Next()

		duration := time.Since(startTime).Milliseconds()

		// 仅在认证成功时记录
		sysAdmin, err := jwt.GetAdmin(c)
		if err != nil || sysAdmin == nil {
			return
		}

		url := c.Request.URL.Path
		module := inferModule(url)
		operType := inferOperType(method)
		description := GetAPIDescription(url, strings.ToLower(method))

		auditLog := model.SysAuditLog{
			AdminId:     sysAdmin.ID,
			Username:    sysAdmin.Username,
			Module:      module,
			OperType:    operType,
			Method:      method,
			Url:         url,
			RequestBody: requestBody,
			StatusCode:  capture.statusCode,
			Duration:    duration,
			Ip:          c.ClientIP(),
			Description: description,
			CreateTime:  util.HTime{Time: time.Now()},
		}

		// 异步写入避免阻塞请求
		go func(entry model.SysAuditLog) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[AuditMiddleware] panic recovered: %v", r)
				}
			}()
			dao.CreateSysAuditLog(entry)
		}(auditLog)
	}
}

// inferModule 从 URL 路径推断操作模块
func inferModule(url string) string {
	parts := strings.Split(strings.TrimPrefix(url, "/api/v1/"), "/")
	if len(parts) == 0 {
		return "other"
	}

	first := strings.ToLower(parts[0])
	moduleMap := map[string]string{
		"admin":          "system",
		"role":           "system",
		"menu":           "system",
		"dept":           "system",
		"post":           "system",
		"sysoperationlog": "system",
		"auditlog":       "system",
		"syslogininfo":   "system",
		"cmdb":           "cmdb",
		"config":         "config",
		"monitor":        "monitor",
		"n9e":            "n9e",
		"k8s":            "k8s",
		"task":           "task",
		"template":       "task",
		"taskjob":        "task",
		"tool":           "tool",
		"apps":           "app",
		"jenkins":        "app",
		"dashboard":      "dashboard",
		"flashduty":      "monitor",
	}

	if module, ok := moduleMap[first]; ok {
		return module
	}
	return "other"
}

// inferOperType 从 HTTP Method 推断操作类型
func inferOperType(method string) string {
	switch strings.ToUpper(method) {
	case "POST":
		return "新增"
	case "PUT":
		return "修改"
	case "DELETE":
		return "删除"
	default:
		return "其他"
	}
}

// sanitizeRequestBody 脱敏请求体中的敏感字段
func sanitizeRequestBody(body string) string {
	// 简单的字符串替换脱敏
	sensitiveKeys := []string{"password", "secret", "token", "accessKey", "secretKey"}
	lower := strings.ToLower(body)
	for _, key := range sensitiveKeys {
		if strings.Contains(lower, key) {
			body = maskSensitiveValue(body, key)
		}
	}
	return body
}

// maskSensitiveValue 遮蔽敏感值（简化实现）
func maskSensitiveValue(body, key string) string {
	// 查找 "key":"value" 或 "key": "value" 模式并替换 value 为 ***
	lower := strings.ToLower(body)
	idx := strings.Index(lower, `"`+key+`"`)
	if idx == -1 {
		return body
	}
	// 查找冒号后的值
	colonIdx := strings.Index(body[idx:], ":")
	if colonIdx == -1 {
		return body
	}
	start := idx + colonIdx + 1
	// 跳过空格
	for start < len(body) && body[start] == ' ' {
		start++
	}
	if start >= len(body) {
		return body
	}
	if body[start] == '"' {
		// 找到结束引号
		end := strings.Index(body[start+1:], `"`)
		if end == -1 {
			return body
		}
		return body[:start+1] + "***" + body[start+1+end:]
	}
	return body
}

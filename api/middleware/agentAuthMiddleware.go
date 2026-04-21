package middleware

import (
	"net/http"
	"strings"

	"dodevops-api/common/config"

	"github.com/gin-gonic/gin"
)

// AgentAuthMiddleware 机器入口鉴权中间件
func AgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expectedToken := ""
		if config.Config != nil {
			expectedToken = config.Config.Integrations.Agent.BearerToken
		}
		if expectedToken == "" {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "agent integration is not configured"})
			c.Abort()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != expectedToken {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "invalid agent token"})
			c.Abort()
			return
		}

		c.Next()
	}
}

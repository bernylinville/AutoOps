package flashduty

import (
	"dodevops-api/api/flashduty/controller"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterFlashDutyRoutes 注册 FlashDuty 路由（需要 JWT）
func RegisterFlashDutyRoutes(r *gin.RouterGroup) {
	ctrl := controller.NewFlashDutyController()

	fdGroup := r.Group("/flashduty")
	{
		// 配置状态 & 连接测试
		fdGroup.GET("/config/status", ctrl.GetConfigStatus)
		fdGroup.POST("/test-connection", ctrl.TestConnection)

		// 告警
		fdGroup.GET("/alerts/active", ctrl.GetActiveAlerts)
		fdGroup.GET("/alerts/summary", ctrl.GetAlertSummary)
		fdGroup.GET("/alerts/host/:ident", ctrl.GetAlertsByHost)

		// 故障
		fdGroup.GET("/incidents/active", ctrl.GetActiveIncidents)
		fdGroup.POST("/incidents/:id/claim", middleware.RbacMiddleware("monitor:flashduty:claim"), ctrl.ClaimIncident)
		fdGroup.POST("/incidents/:id/close", middleware.RbacMiddleware("monitor:flashduty:close"), ctrl.CloseIncident)

		// 值班
		fdGroup.GET("/schedules", ctrl.GetSchedules)
		fdGroup.GET("/oncall/today", ctrl.GetTodayOnCall)

		// SRE 指标
		fdGroup.GET("/insight/metrics", ctrl.GetSREMetrics)
		fdGroup.GET("/insight/trend", ctrl.GetTrendData)
	}
}

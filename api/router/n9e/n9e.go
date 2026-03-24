package n9e

import (
	"dodevops-api/api/n9e/controller"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterN9ERoutes 注册 N9E 路由（需要 JWT）
func RegisterN9ERoutes(r *gin.RouterGroup) {
	ctrl := controller.NewN9EController()
	alertCtrl := controller.NewAlertController()

	n9eGroup := r.Group("/n9e")
	{
		// N9E 配置
		n9eGroup.GET("/config", ctrl.GetConfig)
		n9eGroup.POST("/config", ctrl.SaveConfig)

		// 连接测试
		n9eGroup.POST("/test-connection", ctrl.TestConnection)

		// 数据同步
		n9eGroup.POST("/sync", ctrl.TriggerSync)
		n9eGroup.GET("/sync/status", ctrl.GetSyncStatus)
		n9eGroup.GET("/sync/logs", ctrl.GetSyncLogs)

		// 总览统计
		n9eGroup.GET("/overview", ctrl.GetOverview)

		// 业务组
		n9eGroup.GET("/busi-groups", ctrl.GetBusiGroups)

		// 数据源
		n9eGroup.GET("/datasources", ctrl.GetDatasources)
		n9eGroup.POST("/datasources/:id/check", ctrl.CheckDatasource)

		// PromQL 查询
		n9eGroup.GET("/query", ctrl.QueryPromQL)

		// === 告警规则管理 ===
		n9eGroup.GET("/alert/rules", alertCtrl.GetAlertRules)
		n9eGroup.POST("/alert/rules", middleware.RbacMiddleware("monitor:alert:add"), alertCtrl.CreateAlertRule)
		n9eGroup.PUT("/alert/rules", middleware.RbacMiddleware("monitor:alert:edit"), alertCtrl.UpdateAlertRule)
		n9eGroup.DELETE("/alert/rules/:id", middleware.RbacMiddleware("monitor:alert:delete"), alertCtrl.DeleteAlertRule)

		// === 告警事件 ===
		n9eGroup.GET("/alert/events", alertCtrl.GetAlertEvents)
		n9eGroup.GET("/alert/events/stats", alertCtrl.GetAlertEventStats)

		// === 通知渠道管理 ===
		n9eGroup.GET("/alert/channels", alertCtrl.GetNotifyChannels)
		n9eGroup.POST("/alert/channels", middleware.RbacMiddleware("monitor:channel:add"), alertCtrl.CreateNotifyChannel)
		n9eGroup.PUT("/alert/channels", middleware.RbacMiddleware("monitor:channel:edit"), alertCtrl.UpdateNotifyChannel)
		n9eGroup.DELETE("/alert/channels/:id", middleware.RbacMiddleware("monitor:channel:delete"), alertCtrl.DeleteNotifyChannel)
		n9eGroup.POST("/alert/channels/:id/test", alertCtrl.TestNotifyChannel)
	}
}

// RegisterN9EWebhookRoutes 注册 Webhook 路由（无需 JWT，用 token 校验）
func RegisterN9EWebhookRoutes(r *gin.RouterGroup) {
	alertCtrl := controller.NewAlertController()
	r.POST("/n9e/alert/webhook", alertCtrl.ReceiveWebhook)
}

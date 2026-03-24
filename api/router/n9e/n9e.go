package n9e

import (
	"dodevops-api/api/n9e/controller"

	"github.com/gin-gonic/gin"
)

// RegisterN9ERoutes 注册 N9E 路由
func RegisterN9ERoutes(r *gin.RouterGroup) {
	ctrl := controller.NewN9EController()

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
	}
}

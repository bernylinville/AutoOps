package monitor

import (
	"dodevops-api/api/monitor/controller"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

func InitMonitorRouter(r *gin.RouterGroup) {
	monitorController := controller.NewMonitorController()
	agentController := controller.NewAgentController()

	monitorGroup := r.Group("/monitor")
	monitorGroup.Use(middleware.AuthMiddleware())

	// 主机监控
	monitorGroup.GET("/host/:id", monitorController.GetHostMetrics)                        // 获取主机监控数据
	monitorGroup.GET("/hosts", monitorController.BatchGetHostMetrics)                      // 批量获取主机监控数据(包含在线状态)
	monitorGroup.GET("/hosts/:id/history", monitorController.GetHostMetricHistory)         // 获取主机指定指标的历史数据
	monitorGroup.GET("/hosts/:id/all-metrics", monitorController.GetHostAllMetricsHistory) // 获取主机所有指标的历史数据
	monitorGroup.GET("/hosts/:id/top-processes", monitorController.GetTopProcesses)        // 获取主机TOP进程使用率
	monitorGroup.GET("/hosts/:id/ports", monitorController.GetHostPorts)                   // 获取主机端口信息

	// Agent管理 — H2-P1-1: 补 RBAC
	monitorGroup.POST("/agent/deploy", middleware.RbacMiddleware("monitor:agent:deploy"), agentController.DeployAgent)
	monitorGroup.DELETE("/agent/uninstall", middleware.RbacMiddleware("monitor:agent:uninstall"), agentController.UninstallAgent)
	monitorGroup.GET("/agent/status/:id", agentController.GetAgentStatus)
	monitorGroup.POST("/agent/restart/:id", middleware.RbacMiddleware("monitor:agent:restart"), agentController.RestartAgent)
	monitorGroup.GET("/agent/list", agentController.GetAgentList)
	monitorGroup.GET("/agent/statistics", agentController.GetAgentStatistics)
	monitorGroup.DELETE("/agent/delete/:id", middleware.RbacMiddleware("monitor:agent:delete"), agentController.DeleteAgent)
}

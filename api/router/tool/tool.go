// router/tool/tool.go
package tool

import (
	"dodevops-api/api/tool/controller"
	"dodevops-api/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterToolRoutes 注册导航工具相关路由
func RegisterToolRoutes(router *gin.RouterGroup) {
	// 导航工具管理
	router.POST("/tool", middleware.RbacMiddleware("tool:manage:add"), controller.CreateTool)
	router.GET("/tool/:id", controller.GetToolByID)
	router.PUT("/tool", middleware.RbacMiddleware("tool:manage:edit"), controller.UpdateTool)
	router.DELETE("/tool/:id", middleware.RbacMiddleware("tool:manage:delete"), controller.DeleteTool)
	router.GET("/tool/list", controller.GetToolList)
	router.GET("/tool/all", controller.GetAllTools)

	// 服务部署管理 — H2-P0-4: 部署/卸载加 RBAC
	router.GET("/tool/services", controller.GetServicesList)
	router.GET("/tool/services/:serviceId", controller.GetServiceDetail)
	router.POST("/tool/deploy", middleware.RbacMiddleware("tool:deploy:create"), controller.CreateDeploy)
	router.GET("/tool/deploy/list", controller.GetDeployList)
	router.GET("/tool/deploy/:id/status", controller.GetDeployStatus)
	router.DELETE("/tool/deploy/:id", middleware.RbacMiddleware("tool:deploy:delete"), controller.DeleteDeploy)
}

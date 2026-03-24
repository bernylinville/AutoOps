// configCenter路由注册系统

package configCenter

import (
	"dodevops-api/api/configcenter/controller"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterConfigCenterRoutes 注册配置中心相关路由
func RegisterConfigCenterRoutes(router *gin.RouterGroup) {
	ecsAuthCtrl := controller.NewEcsAuthController()
	accountAuthCtrl := controller.NewAccountAuthController()
	keyManageCtrl := controller.NewKeyManageController()
	syncScheduleCtrl := controller.NewSyncScheduleController()

	// ECS认证凭据管理
	router.GET("/config/ecsauthlist", ecsAuthCtrl.GetEcsAuthList)     // 获取所有凭据
	router.GET("/config/ecsauthinfo", ecsAuthCtrl.GetEcsAuthByName)   // 根据名称查找凭据
	router.GET("/config/ecsauthdetail", ecsAuthCtrl.GetEcsAuthById)   // 根据ID查找凭据详情
	router.POST("/config/ecsauthadd", middleware.RbacMiddleware("config:ecsauth:add"), ecsAuthCtrl.CreateEcsAuth)      // 创建凭据
	router.PUT("/config/ecsauthupdate", middleware.RbacMiddleware("config:ecsauth:edit"), ecsAuthCtrl.UpdateEcsAuth)    // 更新凭据
	router.DELETE("/config/ecsauthdelete", middleware.RbacMiddleware("config:ecsauth:delete"), ecsAuthCtrl.DeleteEcsAuth) // 删除凭据
	// 账号认证管理 — H2-P0-2: 补 RBAC
	router.POST("/config/accountauth", middleware.RbacMiddleware("config:account:add"), accountAuthCtrl.Create)
	router.PUT("/config/accountauth", middleware.RbacMiddleware("config:account:edit"), accountAuthCtrl.Update)
	router.DELETE("/config/accountauth", middleware.RbacMiddleware("config:account:delete"), accountAuthCtrl.Delete)
	router.GET("/config/accountauth", middleware.RbacMiddleware("config:account:view"), accountAuthCtrl.GetByID)
	router.GET("/config/accountauth/list", middleware.RbacMiddleware("config:account:view"), accountAuthCtrl.List)
	router.POST("/config/accountauth/decrypt", middleware.RbacMiddleware("config:account:decrypt"), accountAuthCtrl.DecryptPassword) // 解密密码
	router.GET("/config/accountauth/type", middleware.RbacMiddleware("config:account:view"), accountAuthCtrl.GetByType)
	router.GET("/config/accountauth/alias", middleware.RbacMiddleware("config:account:view"), accountAuthCtrl.GetByAlias)

	// 密钥管理 — H2-P0-2: 补 RBAC
	router.POST("/config/keymanage", middleware.RbacMiddleware("config:key:add"), keyManageCtrl.Create)
	router.PUT("/config/keymanage", middleware.RbacMiddleware("config:key:edit"), keyManageCtrl.Update)
	router.DELETE("/config/keymanage", middleware.RbacMiddleware("config:key:delete"), keyManageCtrl.Delete)
	router.GET("/config/keymanage", middleware.RbacMiddleware("config:key:view"), keyManageCtrl.GetByID)
	router.GET("/config/keymanage/list", middleware.RbacMiddleware("config:key:view"), keyManageCtrl.List)
	router.POST("/config/keymanage/decrypt", middleware.RbacMiddleware("config:key:decrypt"), keyManageCtrl.DecryptKeys) // 解密密钥
	router.GET("/config/keymanage/type", middleware.RbacMiddleware("config:key:view"), keyManageCtrl.GetByType)
	router.POST("/config/keymanage/sync", middleware.RbacMiddleware("config:key:sync"), keyManageCtrl.SyncCloudHosts)   // 同步云主机

	// 定时同步配置管理 — H2-P0-2: 补 RBAC
	router.POST("/config/sync-schedule", middleware.RbacMiddleware("config:sync:add"), syncScheduleCtrl.Create)
	router.PUT("/config/sync-schedule", middleware.RbacMiddleware("config:sync:edit"), syncScheduleCtrl.Update)
	router.DELETE("/config/sync-schedule", middleware.RbacMiddleware("config:sync:delete"), syncScheduleCtrl.Delete)
	router.GET("/config/sync-schedule", middleware.RbacMiddleware("config:sync:view"), syncScheduleCtrl.GetByID)
	router.GET("/config/sync-schedule/list", middleware.RbacMiddleware("config:sync:view"), syncScheduleCtrl.List)
	router.POST("/config/sync-schedule/toggle-status", middleware.RbacMiddleware("config:sync:edit"), syncScheduleCtrl.ToggleStatus)
	router.GET("/config/sync-schedule/active", middleware.RbacMiddleware("config:sync:view"), syncScheduleCtrl.GetActiveSchedules)
	router.POST("/config/sync-schedule/trigger", middleware.RbacMiddleware("config:sync:trigger"), syncScheduleCtrl.TriggerManualSync)
	router.GET("/config/sync-schedule/scheduler-stats", middleware.RbacMiddleware("config:sync:view"), syncScheduleCtrl.GetSchedulerStats)
	router.GET("/config/sync-schedule/log", middleware.RbacMiddleware("config:sync:view"), syncScheduleCtrl.GetSyncLog)
}

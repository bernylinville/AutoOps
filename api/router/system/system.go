// utils/system/system.go

package system

import (
	"dodevops-api/api/system/controller"
	"dodevops-api/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterSystemRoutes 注册系统相关路由
func RegisterSystemRoutes(router *gin.RouterGroup) {
	// 岗位
	router.POST("/post/add", controller.CreateSysPost)
	router.GET("/post/list", controller.GetSysPostList)
	router.GET("/post/info", controller.GetSysPostById)
	router.PUT("/post/update", controller.UpdateSysPost)
	router.DELETE("/post/delete", controller.DeleteSysPostById)
	router.DELETE("/post/batch/delete", controller.BatchDeleteSysPost)
	router.PUT("/post/updateStatus", controller.UpdateSysPostStatus)
	router.GET("/post/vo/list", controller.QuerySysPostVoList)
	//  部门
	router.GET("/dept/list", controller.GetSysDeptList)         // 部门列表
	router.POST("/dept/add", controller.CreateSysDept)          // 添加部门
	router.GET("/dept/info", controller.GetSysDeptById)         // 根据id查询部门
	router.PUT("/dept/update", controller.UpdateSysDept)        // 修改部门
	router.DELETE("/dept/delete", controller.DeleteSysDeptById) // 删除部门
	router.GET("/dept/vo/list", controller.QuerySysDeptVoList)  // 查询部门树形结构
	router.GET("/dept/users", controller.GetDeptUsers)          // 查询部门下的用户
	// 菜单
	router.POST("/menu/add", middleware.RbacMiddleware("base:menu:add"), controller.CreateSysMenu)
	router.GET("/menu/vo/list", controller.QuerySysMenuVoList)
	router.GET("/menu/info", controller.GetSysMenu)
	router.PUT("/menu/update", middleware.RbacMiddleware("base:menu:edit"), controller.UpdateSysMenu)
	router.DELETE("/menu/delete", middleware.RbacMiddleware("base:menu:delete"), controller.DeleteSysMenu)
	router.GET("/menu/list", controller.GetSysMenuList)
	// 角色
	router.POST("/role/add", middleware.RbacMiddleware("base:role:add"), controller.CreateSysRole)
	router.GET("/role/info", controller.GetSysRoleById)
	router.PUT("/role/update", middleware.RbacMiddleware("base:role:edit"), controller.UpdateSysRole)
	router.DELETE("/role/delete", middleware.RbacMiddleware("base:role:delete"), controller.DeleteSysRoleById)
	router.PUT("/role/updateStatus", middleware.RbacMiddleware("base:role:edit"), controller.UpdateSysRoleStatus)
	router.GET("/role/list", controller.GetSysRoleList)
	router.GET("/role/vo/list", controller.QuerySysRoleVoList)
	router.GET("/role/vo/idList", controller.QueryRoleMenuIdList)
	router.PUT("/role/assignPermissions", middleware.RbacMiddleware("base:role:assign"), controller.AssignPermissions)
	// 用户
	router.POST("/admin/add", middleware.RbacMiddleware("base:admin:add"), controller.CreateSysAdmin)
	router.GET("/admin/info", controller.GetSysAdminInfo)
	router.PUT("/admin/update", middleware.RbacMiddleware("base:admin:edit"), controller.UpdateSysAdmin)
	router.DELETE("/admin/delete", middleware.RbacMiddleware("base:admin:delete"), controller.DeleteSysAdminById)
	router.PUT("/admin/updateStatus", middleware.RbacMiddleware("base:admin:edit"), controller.UpdateSysAdminStatus)
	router.PUT("/admin/updatePassword", middleware.RbacMiddleware("base:admin:reset"), controller.ResetSysAdminPassword)
	router.GET("/admin/list", controller.GetSysAdminList)
	router.POST("/upload", controller.Upload)
	router.PUT("/admin/updatePersonal", controller.UpdatePersonal)
	router.PUT("/admin/updatePersonalPassword", controller.UpdatePersonalPassword)
	// 日志 — H2-P0-1: 审计日志必须加 RBAC，禁止普通用户清空
	router.GET("/sysLoginInfo/list", middleware.RbacMiddleware("base:log:view"), controller.GetSysLoginInfoList)
	router.DELETE("/sysLoginInfo/batch/delete", middleware.RbacMiddleware("base:log:delete"), controller.BatchDeleteSysLoginInfo)
	router.DELETE("/sysLoginInfo/delete", middleware.RbacMiddleware("base:log:delete"), controller.DeleteSysLoginInfoById)
	router.DELETE("/sysLoginInfo/clean", middleware.RbacMiddleware("base:log:clean"), controller.CleanSysLoginInfo)
	router.GET("/sysOperationLog/list", middleware.RbacMiddleware("base:log:view"), controller.GetSysOperationLogList)
	router.DELETE("/sysOperationLog/delete", middleware.RbacMiddleware("base:log:delete"), controller.DeleteSysOperationLogById)
	router.DELETE("/sysOperationLog/batch/delete", middleware.RbacMiddleware("base:log:delete"), controller.BatchDeleteSysOperationLog)
	router.DELETE("/sysOperationLog/clean", middleware.RbacMiddleware("base:log:clean"), controller.CleanSysOperationLog)
}

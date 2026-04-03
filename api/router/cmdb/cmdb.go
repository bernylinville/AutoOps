package cmdb

import (
	"dodevops-api/api/cmdb/controller"
	"dodevops-api/api/cmdb/service"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterCmdbRoutes 注册系统相关路由
func RegisterCmdbRoutes(router *gin.RouterGroup) {
	// 资产分组
	router.POST("/cmdb/groupadd", controller.CreateCmdbGroup)                    // 添加资产分组
	router.GET("/cmdb/grouplist", controller.GetAllCmdbGroups)                   // 获取所有资产分组树
	router.GET("/cmdb/grouplistwithhosts", controller.GetAllCmdbGroupsWithHosts) // 获取所有资产分组树及关联主机
	router.PUT("/cmdb/groupupdate", controller.UpdateCmdbGroup)                  // 更新资产分组
	router.DELETE("/cmdb/groupdelete", controller.DeleteCmdbGroup)               // 删除资产分组
	router.GET("/cmdb/groupbyname", controller.GetCmdbGroupByName)               // 根据名称查询分组
	// 主机管理
	router.POST("/cmdb/hostcreate", controller.NewCmdbHostController().CreateCmdbHost)            // 创建主机
	router.PUT("/cmdb/hostupdate", controller.NewCmdbHostController().UpdateCmdbHost)             // 更新主机
	router.DELETE("/cmdb/hostdelete", controller.NewCmdbHostController().DeleteCmdbHost)          // 删除主机
	router.GET("/cmdb/hostlist", controller.NewCmdbHostController().GetCmdbHostListWithPage)      // 获取主机列表(分页)
	router.GET("/cmdb/hostinfo", controller.NewCmdbHostController().GetCmdbHostById)              // 根据ID获取主机
	router.GET("/cmdb/hostgroup", controller.NewCmdbHostController().GetCmdbHostsByGroupId)       // 根据分组ID获取主机列表
	router.GET("/cmdb/hostbyname", controller.NewCmdbHostController().GetCmdbHostsByHostNameLike) // 根据主机名称模糊查询
	router.GET("/cmdb/hostbyip", controller.NewCmdbHostController().GetCmdbHostsByIP)             // 根据IP查询主机
	router.GET("/cmdb/hostbystatus", controller.NewCmdbHostController().GetCmdbHostsByStatus)     // 根据状态查询主机
	router.POST("/cmdb/hostimport", controller.NewCmdbHostController().ImportHostsFromExcel)      // 从Excel导入主机
	router.GET("/cmdb/hosttemplate", controller.NewCmdbHostController().DownloadHostTemplate)     // 下载主机导入模板
	router.POST("/cmdb/hostsync", controller.NewCmdbHostController().SyncHostInfo)                // 同步主机基本信息
	router.PUT("/cmdb/host/lifecycle", controller.NewCmdbHostController().UpdateHostLifecycle)             // 手动变更生命周期状态
	router.PUT("/cmdb/host/lifecycle/batch", controller.NewCmdbHostController().BatchUpdateHostLifecycle) // 批量变更生命周期状态
	// 云主机管理
	router.POST("/cmdb/hostcloudcreatealiyun", controller.NewCmdbHostCloudController().CreateAliyunHost)                          // 创建阿里云主机
	router.POST("/cmdb/hostcloudcreatetencent", controller.NewCmdbHostCloudController().CreateTencentHost)                        // 创建腾讯云主机
	router.POST("/cmdb/hostcloudcreatebaidu", controller.NewCmdbHostCloudController().CreateBaiduHost)                            // 创建百度云主机
	router.GET("/cmdb/hostssh/connect/:id", middleware.RbacMiddleware("cmdb:ecs:terminal"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).ConnectTerminal) // SSH终端连接
	router.GET("/cmdb/hostssh/command/:id", middleware.RbacMiddleware("cmdb:ecs:shell"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).ExecuteCommand)     // SSH执行命令
	router.POST("/cmdb/hostssh/upload/:id", middleware.RbacMiddleware("cmdb:ecs:upload"), controller.NewCmdbHostSSHController(service.GetCmdbHostSSHService()).UploadFile)         // SSH文件上传
	// SQL执行 — H9: TODO: 这些路由需要 RBAC 权限控制，仅 DBA 角色可访问
	// 当前所有登录用户都可以执行任意 SQL，是严重安全隐患
	router.POST("/cmdb/sql/select", middleware.RbacMiddleware("cmdb:sql:select"), controller.GetCmdbSQLRecordController().ExecuteSelect)       // 执行查询语句
	router.POST("/cmdb/sql", middleware.RbacMiddleware("cmdb:sql:execute"), controller.GetCmdbSQLRecordController().ExecuteInsert)              // 执行插入语句
	router.PUT("/cmdb/sql", middleware.RbacMiddleware("cmdb:sql:execute"), controller.GetCmdbSQLRecordController().ExecuteUpdate)               // 执行更新语句
	router.DELETE("/cmdb/sql", middleware.RbacMiddleware("cmdb:sql:execute"), controller.GetCmdbSQLRecordController().ExecuteDelete)            // 执行删除语句
	router.POST("/cmdb/sql/execute", middleware.RbacMiddleware("cmdb:sql:execute"), controller.GetCmdbSQLRecordController().ExecuteSQL)         // 执行原生SQL
	router.POST("/cmdb/sql/databaselist", middleware.RbacMiddleware("cmdb:sql:list"), controller.GetCmdbSQLRecordController().ListDatabases)                   // 获取数据库列表
	// SQL日志管理 — 需要 DBA 权限
	router.GET("/cmdb/sqlLog/list", middleware.RbacMiddleware("cmdb:sqllog:manage"), controller.GetCmdbSqlLogList)         // 分页获取SQL操作日志列表
	router.DELETE("/cmdb/sqlLog/delete", middleware.RbacMiddleware("cmdb:sqllog:manage"), controller.DeleteCmdbSqlLogById) // 根据id删除SQL操作日志
	router.DELETE("/cmdb/sqlLog/clean", middleware.RbacMiddleware("cmdb:sqllog:manage"), controller.CleanCmdbSqlLog)       // 清空SQL操作日志
	// 数据库管理
	router.POST("/cmdb/database", controller.NewCmdbSQLController().CreateDatabase)           // 创建数据库
	router.PUT("/cmdb/database", controller.NewCmdbSQLController().UpdateDatabase)            // 更新数据库
	router.DELETE("/cmdb/database", controller.NewCmdbSQLController().DeleteDatabase)         // 删除数据库
	router.GET("/cmdb/database/info", controller.NewCmdbSQLController().GetDatabase)          // 根据ID获取数据库详情
	router.GET("/cmdb/databaselist", controller.NewCmdbSQLController().ListDatabases)         // 获取数据库列表
	router.GET("/cmdb/database/byname", controller.NewCmdbSQLController().GetDatabasesByName) // 根据名称查询数据库
	router.GET("/cmdb/database/bytype", controller.NewCmdbSQLController().GetDatabasesByType) // 根据类型查询数据库

	// ========================================
	// CI 配置管理（动态CI模型）
	// ========================================
	ciCtrl := controller.NewCITypeController()

	// CI 类型管理
	router.GET("/cmdb/ci/type/list", ciCtrl.GetCITypeList)           // 获取CI类型列表
	router.GET("/cmdb/ci/type/detail", ciCtrl.GetCITypeDetail)       // 获取CI类型详情(含属性)
	router.POST("/cmdb/ci/type", ciCtrl.CreateCIType)                // 创建CI类型
	router.PUT("/cmdb/ci/type", ciCtrl.UpdateCIType)                 // 更新CI类型
	router.DELETE("/cmdb/ci/type", ciCtrl.DeleteCIType)              // 删除CI类型

	// CI 属性管理
	router.GET("/cmdb/ci/attribute/list", ciCtrl.GetCITypeAttributes)      // 获取类型属性列表
	router.POST("/cmdb/ci/attribute", ciCtrl.CreateCITypeAttribute)        // 创建属性
	router.PUT("/cmdb/ci/attribute", ciCtrl.UpdateCITypeAttribute)         // 更新属性
	router.DELETE("/cmdb/ci/attribute", ciCtrl.DeleteCITypeAttribute)      // 删除属性

	// CI 实例管理
	router.GET("/cmdb/ci/instance/list", ciCtrl.GetCIInstanceList)         // 获取CI实例列表(分页)
	router.GET("/cmdb/ci/instance/detail", ciCtrl.GetCIInstanceDetail)     // 获取CI实例详情
	router.POST("/cmdb/ci/instance", ciCtrl.CreateCIInstance)              // 创建CI实例
	router.PUT("/cmdb/ci/instance", ciCtrl.UpdateCIInstance)               // 更新CI实例
	router.DELETE("/cmdb/ci/instance", ciCtrl.DeleteCIInstance)            // 删除CI实例

	// CI 关系管理
	router.GET("/cmdb/ci/relation/list", ciCtrl.GetCIRelations)            // 获取CI关系列表
	router.POST("/cmdb/ci/relation", ciCtrl.CreateCIRelation)              // 创建CI关系
	router.DELETE("/cmdb/ci/relation", ciCtrl.DeleteCIRelation)            // 删除CI关系

	// CI 拓扑图（Phase 3）
	router.GET("/cmdb/ci/instance/all", ciCtrl.GetAllCIInstances)          // 全量CI实例（供拓扑图根节点搜索）
	router.GET("/cmdb/ci/topology", ciCtrl.GetCITopology)                  // CI 拓扑图数据（WITH RECURSIVE）

	// ========================================
	// 变更日志（Phase 4: 资产生命周期）
	// ========================================
	changeLogCtrl := controller.NewChangeLogController()
	router.GET("/cmdb/changelog/list", changeLogCtrl.GetChangeLogs)        // 分页查询变更日志

	// ========================================
	// 网络设备管理（Phase 5）
	// ========================================
	ndCtrl := controller.NewNetworkDeviceController()
	router.GET("/cmdb/network/list", ndCtrl.GetNetworkDevices)             // 网络设备列表（含最新巡检结果）
	router.POST("/cmdb/network/inspect", ndCtrl.InspectDevice)             // 发起 TCP 连通性巡检
	router.GET("/cmdb/network/inspect/history", ndCtrl.GetInspectionHistory) // 巡检历史

	// ========================================
	// 项目管理（Phase 2: Project Dimension）
	// ========================================
	projectCtrl := controller.NewProjectController()

	router.GET("/cmdb/project/list", projectCtrl.GetProjectList)           // 分页项目列表（含资产计数）
	router.GET("/cmdb/project/detail", projectCtrl.GetProjectDetail)       // 项目详情
	router.POST("/cmdb/project", projectCtrl.CreateProject)                // 创建项目
	router.PUT("/cmdb/project", projectCtrl.UpdateProject)                 // 更新项目
	router.DELETE("/cmdb/project", projectCtrl.DeleteProject)              // 删除项目（含关联检查）
	router.GET("/cmdb/project/stats", projectCtrl.GetProjectStats)         // 项目资产统计
	router.GET("/cmdb/project/hosts", projectCtrl.GetProjectHosts)         // 项目关联主机（分页）
	router.GET("/cmdb/project/databases", projectCtrl.GetProjectDatabases) // 项目关联数据库（分页）
	router.GET("/cmdb/project/apps", projectCtrl.GetProjectApps)           // 项目关联应用
}

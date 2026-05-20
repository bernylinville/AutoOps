package inspection

import (
	"dodevops-api/api/inspection/controller"
	"dodevops-api/api/inspection/service"
	"dodevops-api/middleware"
	dbR "dodevops-api/pkg/db"

	"github.com/gin-gonic/gin"
)

// RegisterInspectionRoutes 注册巡检系统路由
func RegisterInspectionRoutes(router *gin.RouterGroup) {
	taskSvc := service.NewTaskService(dbR.Db)
	taskCtrl := controller.NewTaskController(taskSvc)
	inspSvc := service.NewInspectionService(dbR.Db)
	inspCtrl := controller.NewInspectionController(inspSvc)
	reportCtrl := controller.NewReportController(dbR.Db)

	router.GET("/inspection/tasks", taskCtrl.ListTasks)
	router.GET("/inspection/tasks/:id", taskCtrl.GetTask)
	router.PUT("/inspection/tasks/:id", middleware.RbacMiddleware("inspection:task:edit"), taskCtrl.UpdateTask)
	router.POST("/inspection/tasks/:id/trigger", middleware.RbacMiddleware("inspection:task:trigger"), inspCtrl.TriggerInspection)

	router.GET("/inspection/runs", inspCtrl.ListRuns)
	router.GET("/inspection/runs/:id", inspCtrl.GetRun)
	router.GET("/inspection/runs/:id/report", reportCtrl.DownloadReport)
	router.GET("/inspection/runs/:id/results", inspCtrl.ListRunResults)
	router.GET("/inspection/runs/:id/alerts", inspCtrl.ListRunAlerts)
	router.GET("/inspection/overview", inspCtrl.GetTodayOverview)
}

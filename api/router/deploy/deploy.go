package deploy

import (
	"dodevops-api/api/deploy/controller"
	"dodevops-api/common"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterDeployRoutes(router *gin.RouterGroup) {
	deployCtrl := controller.NewDeployController(common.GetDB())
	pipelineCtrl := controller.NewPipelineController(common.GetDB())

	router.GET("/deploy/cluster-targets", middleware.RbacMiddleware("deploy:target:view"), deployCtrl.ListClusterTargets)
	router.GET("/deploy/cluster-targets/:id", middleware.RbacMiddleware("deploy:target:view"), deployCtrl.GetClusterTarget)
	router.POST("/deploy/cluster-targets", middleware.RbacMiddleware("deploy:target:create"), deployCtrl.CreateClusterTarget)
	router.PUT("/deploy/cluster-targets/:id", middleware.RbacMiddleware("deploy:target:edit"), deployCtrl.UpdateClusterTarget)
	router.POST("/deploy/cluster-targets/:id/validate-direct-credential", middleware.RbacMiddleware("deploy:target:edit"), deployCtrl.ValidateDirectCredential)
	router.POST("/deploy/cluster-targets/:id/validate-gitops-repo", middleware.RbacMiddleware("deploy:target:edit"), deployCtrl.ValidateGitOpsRepo)
	router.GET("/deploy/gitops/validate-working-tree", middleware.RbacMiddleware("deploy:target:view"), deployCtrl.ValidateGitOpsWorkingTree)
	router.POST("/deploy/requests", middleware.RbacMiddleware("deploy:request:create"), deployCtrl.CreateDeployRequest)
	router.GET("/deploy/requests", middleware.RbacMiddleware("deploy:request:view"), deployCtrl.ListDeployRequests)
	router.GET("/deploy/requests/:id", middleware.RbacMiddleware("deploy:request:view"), deployCtrl.GetDeployRequest)
	router.GET("/deploy/requests/:id/executions", middleware.RbacMiddleware("deploy:request:view"), deployCtrl.ListExecutionRecords)
	router.GET("/deploy/requests/:id/notifications", middleware.RbacMiddleware("deploy:request:view"), deployCtrl.ListNotifications)
	router.POST("/deploy/requests/:id/dispatch-approval", middleware.RbacMiddleware("deploy:request:approve"), deployCtrl.RetryApprovalDispatch)
	router.POST("/deploy/requests/:id/sync-approval", middleware.RbacMiddleware("deploy:request:approve"), deployCtrl.SyncApprovalStatus)
	router.POST("/deploy/requests/:id/approve", middleware.RbacMiddleware("deploy:request:approve"), deployCtrl.ApproveDeployRequest)
	router.POST("/deploy/requests/:id/reject", middleware.RbacMiddleware("deploy:request:approve"), deployCtrl.RejectDeployRequest)
	router.POST("/deploy/requests/:id/execute", middleware.RbacMiddleware("deploy:request:execute"), deployCtrl.ExecuteDeployRequest)
	router.POST("/deploy/requests/:id/rollback", middleware.RbacMiddleware("deploy:request:execute"), deployCtrl.RollbackDeployRequest)
	router.POST("/deploy/requests/:id/cleanup", middleware.RbacMiddleware("deploy:request:execute"), deployCtrl.CleanupDirectRequest)

	router.GET("/pipeline-runs/:id", middleware.RbacMiddleware("deploy:request:view"), pipelineCtrl.GetPipelineRun)
	router.GET("/pipeline-runs/by-request/:requestId", middleware.RbacMiddleware("deploy:request:view"), pipelineCtrl.GetPipelineRunByRequestID)
}

func RegisterAgentDeployRoutes(router *gin.RouterGroup) {
	deployCtrl := controller.NewDeployController(common.GetDB())

	agentGroup := router.Group("/integrations/agent")
	agentGroup.Use(middleware.AgentAuthMiddleware(), middleware.AuditMiddleware("agent"))
	{
		agentGroup.POST("/deploy-requests", deployCtrl.CreateAgentDeployRequest)
		agentGroup.GET("/deploy-requests/:requestNo", deployCtrl.GetDeployRequestByRequestNo)
		agentGroup.GET("/deploy-requests/:requestNo/status", deployCtrl.GetAgentStatus)
		agentGroup.POST("/deploy-requests/:requestNo/dispatch-approval", deployCtrl.RetryApprovalDispatchByRequestNo)
		agentGroup.POST("/deploy-requests/:requestNo/sync-approval", deployCtrl.SyncApprovalStatusByRequestNo)
	}
}

package controller

import (
	"strconv"

	"dodevops-api/api/deploy/model"
	"dodevops-api/api/deploy/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeployController struct {
	service service.IDeployService
}

func NewDeployController(db *gorm.DB) *DeployController {
	return &DeployController{
		service: service.NewDeployService(db),
	}
}

func (dc *DeployController) ListClusterTargets(c *gin.Context) {
	dc.service.ListClusterTargets(c)
}

func (dc *DeployController) GetClusterTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的部署目标ID")
		return
	}
	dc.service.GetClusterTarget(c, uint(id))
}

func (dc *DeployController) CreateClusterTarget(c *gin.Context) {
	var req model.CreateClusterTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.CreateClusterTarget(c, &req)
}

func (dc *DeployController) UpdateClusterTarget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的部署目标ID")
		return
	}
	var req model.UpdateClusterTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.UpdateClusterTarget(c, uint(id), &req)
}

func (dc *DeployController) ValidateDirectCredential(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的部署目标ID")
		return
	}
	dc.service.ValidateDirectCredential(c, uint(id))
}

func (dc *DeployController) ValidateGitOpsWorkingTree(c *gin.Context) {
	dc.service.ValidateGitOpsWorkingTree(c)
}

func (dc *DeployController) ValidateGitOpsRepo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的部署目标ID")
		return
	}
	dc.service.ValidateGitOpsRepo(c, uint(id))
}

func (dc *DeployController) CleanupDirectRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.CleanupDirectRequest(c, uint(id))
}

func (dc *DeployController) CreateDeployRequest(c *gin.Context) {
	var req model.CreateDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.CreateDeployRequest(c, &req)
}

func (dc *DeployController) CreateAgentDeployRequest(c *gin.Context) {
	var req model.CreateAgentDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.CreateAgentDeployRequest(c, &req)
}

func (dc *DeployController) CreateAgentBuildDeployRequest(c *gin.Context) {
	var req model.CreateAgentBuildDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.CreateAgentBuildDeployRequest(c, &req)
}

func (dc *DeployController) CreateAgentProjectOnboardBuildDeployRequest(c *gin.Context) {
	var req model.CreateAgentProjectOnboardBuildDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.CreateAgentProjectOnboardBuildDeployRequest(c, &req)
}

func (dc *DeployController) ListDeployRequests(c *gin.Context) {
	var query model.DeployRequestListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		result.Failed(c, 400, "查询参数错误: "+err.Error())
		return
	}
	dc.service.ListDeployRequests(c, &query)
}

func (dc *DeployController) GetDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.GetDeployRequest(c, uint(id))
}

func (dc *DeployController) GetDeployRequestByRequestNo(c *gin.Context) {
	requestNo := c.Param("requestNo")
	if requestNo == "" {
		result.Failed(c, 400, "无效的申请单号")
		return
	}
	dc.service.GetDeployRequestByRequestNo(c, requestNo)
}

func (dc *DeployController) RetryApprovalDispatch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.RetryApprovalDispatch(c, uint(id))
}

func (dc *DeployController) RetryApprovalDispatchByRequestNo(c *gin.Context) {
	requestNo := c.Param("requestNo")
	if requestNo == "" {
		result.Failed(c, 400, "无效的申请单号")
		return
	}
	dc.service.RetryApprovalDispatchByRequestNo(c, requestNo)
}

func (dc *DeployController) SyncApprovalStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.SyncApprovalStatus(c, uint(id))
}

func (dc *DeployController) SyncApprovalStatusByRequestNo(c *gin.Context) {
	requestNo := c.Param("requestNo")
	if requestNo == "" {
		result.Failed(c, 400, "无效的申请单号")
		return
	}
	dc.service.SyncApprovalStatusByRequestNo(c, requestNo)
}

func (dc *DeployController) ExecuteDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	var req model.ExecuteDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.ExecuteDeployRequest(c, uint(id), &req)
}

func (dc *DeployController) ListExecutionRecords(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.ListExecutionRecords(c, uint(id))
}

func (dc *DeployController) ListNotifications(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.ListNotifications(c, uint(id))
}

func (dc *DeployController) ApproveDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	var req model.ApproveDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.ApproveDeployRequest(c, uint(id), &req)
}

func (dc *DeployController) RejectDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	var req model.RejectDeployRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	dc.service.RejectDeployRequest(c, uint(id), &req)
}

func (dc *DeployController) GetAgentStatus(c *gin.Context) {
	requestNo := c.Param("requestNo")
	if requestNo == "" {
		result.Failed(c, 400, "无效的申请单号")
		return
	}
	dc.service.GetAgentStatusByRequestNo(c, requestNo)
}

func (dc *DeployController) RollbackDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的申请ID")
		return
	}
	dc.service.RollbackDeployRequest(c, uint(id))
}

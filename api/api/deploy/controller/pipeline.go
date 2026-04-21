package controller

import (
	"strconv"

	"dodevops-api/api/deploy/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PipelineController struct {
	service service.IPipelineService
}

func NewPipelineController(db *gorm.DB) *PipelineController {
	return &PipelineController{
		service: service.NewPipelineService(db),
	}
}

func (pc *PipelineController) GetPipelineRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的流水线运行ID")
		return
	}
	resp, err := pc.service.GetPipelineRunWithStages(uint(id))
	if err != nil {
		result.Failed(c, 404, "流水线运行不存在")
		return
	}
	result.Success(c, resp)
}

func (pc *PipelineController) GetPipelineRunByRequestID(c *gin.Context) {
	requestID, err := strconv.ParseUint(c.Param("requestId"), 10, 32)
	if err != nil {
		result.Failed(c, 400, "无效的部署申请ID")
		return
	}
	resp, err := pc.service.GetPipelineRunByRequestID(uint(requestID))
	if err != nil {
		result.Failed(c, 404, "流水线运行不存在")
		return
	}
	result.Success(c, resp)
}

func (pc *PipelineController) GetPipelineRunByRequestNo(c *gin.Context) {
	requestNo := c.Param("requestNo")
	if requestNo == "" {
		result.Failed(c, 400, "无效的申请单号")
		return
	}
	result.Failed(c, 501, "通过申请单号查询流水线运行尚未实现")
}

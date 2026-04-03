package controller

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ChangeLogController struct {
	dao dao.ChangeLogDao
}

func NewChangeLogController() *ChangeLogController {
	return &ChangeLogController{dao: dao.NewChangeLogDao()}
}

// GetChangeLogs 分页查询变更日志
// GET /cmdb/changelog/list?entityType=ci_instance&entityId=1&page=1&pageSize=20
func (c *ChangeLogController) GetChangeLogs(ctx *gin.Context) {
	entityType := ctx.Query("entityType")

	var entityID uint
	if idStr := ctx.Query("entityId"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			result.Failed(ctx, constant.INVALID_PARAMS, "entityId 参数格式错误")
			return
		}
		entityID = uint(id)
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetLogs(entityType, entityID, page, pageSize)
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}

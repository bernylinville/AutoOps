// Phase 2: 项目维度管理 — Controller 层
package controller

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/common/util"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ProjectController struct {
	dao dao.ProjectDao
}

func NewProjectController() *ProjectController {
	return &ProjectController{dao: dao.NewProjectDao()}
}

// ========================================
// 项目 CRUD
// ========================================

// GetProjectList 分页获取项目列表
func (c *ProjectController) GetProjectList(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	keyword := ctx.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	vos, total := c.dao.GetProjectList(page, pageSize, keyword)
	result.SuccessWithPage(ctx, vos, total, page, pageSize)
}

// GetProjectDetail 获取项目详情
func (c *ProjectController) GetProjectDetail(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	p, err := c.dao.GetProjectByID(uint(id))
	if err != nil {
		result.Failed(ctx, constant.PROJECT_NOT_FOUND, "项目不存在")
		return
	}
	result.Success(ctx, p)
}

// CreateProject 创建项目
func (c *ProjectController) CreateProject(ctx *gin.Context) {
	var dto model.CreateProjectDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}
	if dto.Name == "" || dto.Code == "" {
		result.Failed(ctx, constant.INVALID_PARAMS, "项目名称和代码不能为空")
		return
	}

	if c.dao.CheckCodeExists(dto.Code) {
		result.Failed(ctx, constant.PROJECT_CODE_EXISTS, "项目代码已存在")
		return
	}

	now := util.HTime{Time: time.Now()}
	p := model.Project{
		Name:        dto.Name,
		Code:        dto.Code,
		Description: dto.Description,
		OwnerID:     dto.OwnerID,
		Status:      1,
		CreateTime:  now,
		UpdateTime:  now,
	}

	if err := c.dao.CreateProject(&p); err != nil {
		result.Failed(ctx, constant.PROJECT_CREATE_FAILED, "创建项目失败: "+err.Error())
		return
	}
	result.Success(ctx, p)
}

// UpdateProject 更新项目
func (c *ProjectController) UpdateProject(ctx *gin.Context) {
	var dto model.UpdateProjectDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}
	if dto.ID == 0 {
		result.Failed(ctx, constant.INVALID_PARAMS, "项目ID不能为空")
		return
	}

	if _, err := c.dao.GetProjectByID(dto.ID); err != nil {
		result.Failed(ctx, constant.PROJECT_NOT_FOUND, "项目不存在")
		return
	}

	data := map[string]interface{}{
		"update_time": time.Now(),
	}
	if dto.Name != "" {
		data["name"] = dto.Name
	}
	if dto.Description != "" {
		data["description"] = dto.Description
	}
	if dto.OwnerID != nil {
		data["owner_id"] = dto.OwnerID
	}
	if dto.Status != nil {
		data["status"] = *dto.Status
	}

	if err := c.dao.UpdateProject(dto.ID, data); err != nil {
		result.Failed(ctx, constant.PROJECT_UPDATE_FAILED, "更新项目失败")
		return
	}
	result.Success(ctx, true)
}

// DeleteProject 删除项目
func (c *ProjectController) DeleteProject(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	if _, err := c.dao.GetProjectByID(uint(id)); err != nil {
		result.Failed(ctx, constant.PROJECT_NOT_FOUND, "项目不存在")
		return
	}

	hasResources, err := c.dao.DeleteProject(uint(id))
	if hasResources {
		result.Failed(ctx, constant.PROJECT_HAS_RESOURCES, "项目下存在关联资源（主机/数据库/应用），请先解除关联")
		return
	}
	if err != nil {
		result.Failed(ctx, constant.PROJECT_DELETE_FAILED, "删除项目失败")
		return
	}
	result.Success(ctx, true)
}

// ========================================
// 资产统计 & 关联资产查询
// ========================================

// GetProjectStats 获取项目资产统计
func (c *ProjectController) GetProjectStats(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	stats := c.dao.GetProjectStats(uint(id))
	result.Success(ctx, stats)
}

// GetProjectHosts 获取项目关联主机（分页）
func (c *ProjectController) GetProjectHosts(ctx *gin.Context) {
	idStr := ctx.Query("projectId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetProjectHosts(uint(id), page, pageSize)
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}

// GetProjectDatabases 获取项目关联数据库（分页）
func (c *ProjectController) GetProjectDatabases(ctx *gin.Context) {
	idStr := ctx.Query("projectId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetProjectDatabases(uint(id), page, pageSize)
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}

// GetProjectApps 获取项目关联应用
func (c *ProjectController) GetProjectApps(ctx *gin.Context) {
	idStr := ctx.Query("projectId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	apps, err := c.dao.GetProjectApps(uint(id))
	if err != nil {
		result.Failed(ctx, constant.PROJECT_NOT_FOUND, "获取关联应用失败")
		return
	}
	result.Success(ctx, apps)
}

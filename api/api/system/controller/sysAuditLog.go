// 审计日志 控制层

package controller

import (
	"dodevops-api/api/system/model"
	"dodevops-api/api/system/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetSysAuditLogList 分页获取审计日志列表
// @Tags System系统管理
// @Summary 分页获取审计日志列表接口
// @Produce json
// @Description 分页获取审计日志列表接口
// @Param pageSize query int false "每页数"
// @Param pageNum query int false "分页数"
// @Param username query string false "用户名"
// @Param module query string false "操作模块"
// @Param operType query string false "操作类型"
// @Param beginTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {object} result.Result
// @router /api/v1/auditLog/list [get]
// @Security ApiKeyAuth
func GetSysAuditLogList(c *gin.Context) {
	Username := c.Query("username")
	Module := c.Query("module")
	OperType := c.Query("operType")
	BeginTime := c.Query("beginTime")
	EndTime := c.Query("endTime")
	PageSize, _ := strconv.Atoi(c.Query("pageSize"))
	PageNum, _ := strconv.Atoi(c.Query("pageNum"))
	service.SysAuditLogService().GetSysAuditLogList(c, Username, Module, OperType, BeginTime, EndTime, PageSize, PageNum)
}

// DeleteSysAuditLogById 根据id删除审计日志
// @Tags System系统管理
// @Summary 根据id删除审计日志
// @Produce json
// @Description 根据id删除审计日志
// @Param data body model.SysAuditLogIdDto true "data"
// @Success 200 {object} result.Result
// @router /api/v1/auditLog/delete [delete]
// @Security ApiKeyAuth
func DeleteSysAuditLogById(c *gin.Context) {
	var dto model.SysAuditLogIdDto
	_ = c.BindJSON(&dto)
	service.SysAuditLogService().DeleteSysAuditLogById(c, dto)
}

// BatchDeleteSysAuditLog 批量删除审计日志
// @Tags System系统管理
// @Summary 批量删除审计日志接口
// @Produce json
// @Description 批量删除审计日志接口
// @Param data body model.BatchDeleteSysAuditLogDto true "data"
// @Success 200 {object} result.Result
// @router /api/v1/auditLog/batch/delete [delete]
// @Security ApiKeyAuth
func BatchDeleteSysAuditLog(c *gin.Context) {
	var dto model.BatchDeleteSysAuditLogDto
	_ = c.BindJSON(&dto)
	service.SysAuditLogService().BatchDeleteSysAuditLog(c, dto)
}

// CleanSysAuditLog 清空审计日志
// @Tags System系统管理
// @Summary 清空审计日志接口
// @Produce json
// @Description 清空审计日志接口
// @Success 200 {object} result.Result
// @router /api/v1/auditLog/clean [delete]
// @Security ApiKeyAuth
func CleanSysAuditLog(c *gin.Context) {
	service.SysAuditLogService().CleanSysAuditLog(c)
}

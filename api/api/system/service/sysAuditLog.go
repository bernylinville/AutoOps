// 审计日志 服务层

package service

import (
	"dodevops-api/api/system/dao"
	"dodevops-api/api/system/model"
	"dodevops-api/common/result"
	"github.com/gin-gonic/gin"
)

type ISysAuditLogService interface {
	GetSysAuditLogList(c *gin.Context, Username, Module, OperType, BeginTime, EndTime string, PageSize, PageNum int)
	DeleteSysAuditLogById(c *gin.Context, dto model.SysAuditLogIdDto)
	BatchDeleteSysAuditLog(c *gin.Context, dto model.BatchDeleteSysAuditLogDto)
	CleanSysAuditLog(c *gin.Context)
}

type SysAuditLogServiceImpl struct{}

// CleanSysAuditLog 清空审计日志
func (s SysAuditLogServiceImpl) CleanSysAuditLog(c *gin.Context) {
	dao.CleanSysAuditLog()
	result.Success(c, true)
}

// BatchDeleteSysAuditLog 批量删除审计日志
func (s SysAuditLogServiceImpl) BatchDeleteSysAuditLog(c *gin.Context, dto model.BatchDeleteSysAuditLogDto) {
	dao.BatchDeleteSysAuditLog(dto)
	result.Success(c, true)
}

// DeleteSysAuditLogById 根据id删除审计日志
func (s SysAuditLogServiceImpl) DeleteSysAuditLogById(c *gin.Context, dto model.SysAuditLogIdDto) {
	dao.DeleteSysAuditLogById(dto)
	result.Success(c, true)
}

// GetSysAuditLogList 分页查询审计日志列表
func (s SysAuditLogServiceImpl) GetSysAuditLogList(c *gin.Context, Username, Module, OperType, BeginTime, EndTime string, PageSize, PageNum int) {
	if PageSize < 1 {
		PageSize = 10
	}
	if PageNum < 1 {
		PageNum = 1
	}
	logs, count := dao.GetSysAuditLogList(Username, Module, OperType, BeginTime, EndTime, PageSize, PageNum)
	result.Success(c, map[string]interface{}{"total": count, "pageSize": PageSize, "pageNum": PageNum, "list": logs})
}

var sysAuditLogService = SysAuditLogServiceImpl{}

func SysAuditLogService() ISysAuditLogService {
	return &sysAuditLogService
}

// 审计日志 数据层

package dao

import (
	"dodevops-api/api/system/model"
	. "dodevops-api/pkg/db"
)

// CreateSysAuditLog 新增审计日志
func CreateSysAuditLog(log model.SysAuditLog) {
	Db.Create(&log)
}

// GetSysAuditLogList 分页查询审计日志列表
func GetSysAuditLogList(Username, Module, OperType, BeginTime, EndTime string, PageSize, PageNum int) (logs []model.SysAuditLog, count int64) {
	curDb := Db.Table("sys_audit_log")
	if Username != "" {
		curDb = curDb.Where("username = ?", Username)
	}
	if Module != "" {
		curDb = curDb.Where("module = ?", Module)
	}
	if OperType != "" {
		curDb = curDb.Where("oper_type = ?", OperType)
	}
	if BeginTime != "" && EndTime != "" {
		curDb = curDb.Where("`create_time` BETWEEN ? AND ?", BeginTime, EndTime)
	}
	curDb.Count(&count)
	curDb.Limit(PageSize).Offset((PageNum - 1) * PageSize).Order("create_time desc").Find(&logs)
	return logs, count
}

// DeleteSysAuditLogById 根据id删除审计日志
func DeleteSysAuditLogById(dto model.SysAuditLogIdDto) {
	Db.Delete(&model.SysAuditLog{}, dto)
}

// BatchDeleteSysAuditLog 批量删除审计日志
func BatchDeleteSysAuditLog(dto model.BatchDeleteSysAuditLogDto) {
	Db.Where("id in (?)", dto.Ids).Delete(&model.SysAuditLog{})
}

// CleanSysAuditLog 清空审计日志
func CleanSysAuditLog() {
	Db.Exec("truncate table sys_audit_log")
}

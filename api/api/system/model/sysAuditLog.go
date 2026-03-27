// 审计日志模型

package model

import "dodevops-api/common/util"

// SysAuditLog 审计日志
type SysAuditLog struct {
	ID          uint       `gorm:"column:id;comment:'主键';primaryKey;NOT NULL" json:"id"`
	AdminId     uint       `gorm:"column:admin_id;comment:'用户ID';NOT NULL" json:"adminId"`
	Username    string     `gorm:"column:username;type:varchar(64);comment:'用户名';NOT NULL" json:"username"`
	Module      string     `gorm:"column:module;type:varchar(32);comment:'操作模块'" json:"module"`
	OperType    string     `gorm:"column:oper_type;type:varchar(16);comment:'操作类型'" json:"operType"`
	Method      string     `gorm:"column:method;type:varchar(16);comment:'HTTP方法';NOT NULL" json:"method"`
	Url         string     `gorm:"column:url;type:varchar(500);comment:'请求URL'" json:"url"`
	RequestBody string     `gorm:"column:request_body;type:text;comment:'请求体'" json:"requestBody"`
	StatusCode  int        `gorm:"column:status_code;comment:'响应状态码'" json:"statusCode"`
	Duration    int64      `gorm:"column:duration;comment:'耗时(ms)'" json:"duration"`
	Ip          string     `gorm:"column:ip;type:varchar(64);comment:'客户端IP'" json:"ip"`
	Description string     `gorm:"column:description;type:varchar(255);comment:'操作描述'" json:"description"`
	CreateTime  util.HTime `gorm:"column:create_time;comment:'创建时间';NOT NULL" json:"createTime"`
}

func (SysAuditLog) TableName() string {
	return "sys_audit_log"
}

// SysAuditLogIdDto 单条删除参数
type SysAuditLogIdDto struct {
	Id uint `json:"id"`
}

// BatchDeleteSysAuditLogDto 批量删除参数
type BatchDeleteSysAuditLogDto struct {
	Ids []uint `json:"ids"`
}

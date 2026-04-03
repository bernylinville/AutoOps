// Phase 2: 项目维度管理 — 核心模型
package model

import "dodevops-api/common/util"

// ========================================
// Project 项目主表
// ========================================

// Project 项目（业务维度）
type Project struct {
	ID          uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	Name        string     `gorm:"column:name;type:varchar(100);NOT NULL;uniqueIndex;comment:'项目名称'" json:"name"`
	Code        string     `gorm:"column:code;type:varchar(50);uniqueIndex;NOT NULL;comment:'项目代码(英文)'" json:"code"`
	Description string     `gorm:"column:description;type:varchar(500);comment:'项目描述'" json:"description"`
	OwnerID     *uint      `gorm:"column:owner_id;comment:'负责人ID(关联sys_admin)'" json:"ownerId"`
	Status      int        `gorm:"column:status;default:1;comment:'状态:1-活跃,2-归档,3-已废弃'" json:"status"`
	CreateTime  util.HTime `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime  util.HTime `gorm:"column:update_time;comment:'更新时间'" json:"updateTime"`
}

func (Project) TableName() string {
	return "cmdb_project"
}

// ========================================
// DTOs（接口请求参数）
// ========================================

// CreateProjectDto 创建项目
type CreateProjectDto struct {
	Name        string `json:"name" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Description string `json:"description"`
	OwnerID     *uint  `json:"ownerId"`
}

// UpdateProjectDto 更新项目
type UpdateProjectDto struct {
	ID          uint   `json:"id" validate:"required"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerID     *uint  `json:"ownerId"`
	Status      *int   `json:"status"`
}

// ========================================
// VOs（接口响应视图）
// ========================================

// ProjectVo 项目列表视图
type ProjectVo struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Code        string     `json:"code"`
	Description string     `json:"description"`
	OwnerID     *uint      `json:"ownerId"`
	OwnerName   string     `json:"ownerName"`
	Status      int        `json:"status"`
	HostCount   int64      `json:"hostCount"`
	DBCount     int64      `json:"dbCount"`
	AppCount    int64      `json:"appCount"`
	CreateTime  util.HTime `json:"createTime"`
	UpdateTime  util.HTime `json:"updateTime"`
}

// ProjectStatsVo 项目资产统计（详情页）
type ProjectStatsVo struct {
	ProjectID      uint           `json:"projectId"`
	TotalHosts     int64          `json:"totalHosts"`
	OnlineHosts    int64          `json:"onlineHosts"`
	OfflineHosts   int64          `json:"offlineHosts"`
	TotalDatabases int64          `json:"totalDatabases"`
	TotalApps      int64          `json:"totalApps"`
	HostsByGroup   []GroupCountVo `json:"hostsByGroup"`
	DBsByType      []TypeCountVo  `json:"dbsByType"`
}

// GroupCountVo 按分组统计
type GroupCountVo struct {
	GroupID   uint   `gorm:"column:group_id" json:"groupId"`
	GroupName string `gorm:"column:group_name" json:"groupName"`
	Count     int64  `gorm:"column:count" json:"count"`
}

// TypeCountVo 按类型统计
type TypeCountVo struct {
	Type  int   `gorm:"column:type" json:"type"`
	Count int64 `gorm:"column:count" json:"count"`
}

// ProjectAppVo 项目关联应用简要信息（用于跨包 Scan）
type ProjectAppVo struct {
	ID              uint   `gorm:"column:id" json:"id"`
	Name            string `gorm:"column:name" json:"name"`
	Code            string `gorm:"column:code" json:"code"`
	Status          int    `gorm:"column:status" json:"status"`
	BusinessGroupID uint   `gorm:"column:business_group_id" json:"businessGroupId"`
	ProgrammingLang string `gorm:"column:programming_lang" json:"programmingLang"`
}

// CI 变更日志模型 — Phase 4: 资产生命周期
package model

import "dodevops-api/common/util"

// CIChangeLog 记录 CI 实例、主机、数据库的字段级变更
type CIChangeLog struct {
	ID         uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EntityType string     `gorm:"column:entity_type;type:varchar(50);NOT NULL;index;comment:'实体类型: ci_instance/cmdb_host/cmdb_sql'" json:"entityType"`
	EntityID   uint       `gorm:"column:entity_id;NOT NULL;index;comment:'实体ID'" json:"entityId"`
	EntityName string     `gorm:"column:entity_name;type:varchar(200);comment:'实体名称（记录时快照）'" json:"entityName"`
	Field      string     `gorm:"column:field;type:varchar(100);NOT NULL;comment:'变更字段名'" json:"field"`
	OldValue   string     `gorm:"column:old_value;type:varchar(2000);comment:'变更前值'" json:"oldValue"`
	NewValue   string     `gorm:"column:new_value;type:varchar(2000);comment:'变更后值'" json:"newValue"`
	OperatorID uint       `gorm:"column:operator_id;comment:'操作人ID'" json:"operatorId"`
	Operator   string     `gorm:"column:operator;type:varchar(100);comment:'操作人名称'" json:"operator"`
	CreateTime util.HTime `gorm:"column:create_time;NOT NULL;index;comment:'变更时间'" json:"createTime"`
}

func (CIChangeLog) TableName() string {
	return "ci_change_log"
}

// CIChangeLogQueryDto 查询参数
type CIChangeLogQueryDto struct {
	EntityType string `form:"entityType"`
	EntityID   uint   `form:"entityId"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

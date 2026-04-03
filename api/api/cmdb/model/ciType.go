// CI 类型模型 — 动态 CMDB 核心
package model

import (
	"dodevops-api/common/util"

	"gorm.io/datatypes"
)

// ========================================
// CI 类型（模板定义）
// ========================================

// CIType 定义一种 CI 类别（如：服务器、数据库、中间件等）
type CIType struct {
	ID          uint            `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	Name        string          `gorm:"column:name;type:varchar(100);NOT NULL;comment:'类型名称'" json:"name"`
	Code        string          `gorm:"column:code;type:varchar(50);uniqueIndex;NOT NULL;comment:'类型代码(英文)'" json:"code"`
	Icon        string          `gorm:"column:icon;type:varchar(100);comment:'图标名称'" json:"icon"`
	Category    string          `gorm:"column:category;type:varchar(50);NOT NULL;comment:'分类:server/database/network/middleware/storage/cloud/custom'" json:"category"`
	Description string          `gorm:"column:description;type:varchar(500);comment:'类型描述'" json:"description"`
	BuiltIn     bool            `gorm:"column:built_in;default:false;comment:'是否内置类型'" json:"builtIn"`
	Enabled     bool            `gorm:"column:enabled;default:true;comment:'是否启用'" json:"enabled"`
	SortOrder   int             `gorm:"column:sort_order;default:0;comment:'排序'" json:"sortOrder"`
	CreateTime  util.HTime      `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime  util.HTime      `gorm:"column:update_time;comment:'更新时间'" json:"updateTime"`
	Attributes  []CITypeAttribute `gorm:"foreignKey:CITypeID" json:"attributes,omitempty"`
}

func (CIType) TableName() string {
	return "ci_type"
}

// ========================================
// CI 类型属性（字段定义）
// ========================================

// CITypeAttribute 定义 CI 类型的一个属性字段
type CITypeAttribute struct {
	ID           uint           `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	CITypeID     uint           `gorm:"column:ci_type_id;NOT NULL;index;comment:'所属CI类型ID'" json:"ciTypeId"`
	Name         string         `gorm:"column:name;type:varchar(100);NOT NULL;comment:'属性名称'" json:"name"`
	Code         string         `gorm:"column:code;type:varchar(50);NOT NULL;comment:'属性代码(英文)'" json:"code"`
	DataType     string         `gorm:"column:data_type;type:varchar(20);NOT NULL;comment:'数据类型:string/integer/float/boolean/enum/date/ip/url'" json:"dataType"`
	Required     bool           `gorm:"column:required;default:false;comment:'是否必填'" json:"required"`
	DefaultValue string         `gorm:"column:default_value;type:varchar(500);comment:'默认值'" json:"defaultValue"`
	EnumOptions  datatypes.JSON `gorm:"column:enum_options;type:jsonb;comment:'枚举选项(JSON数组)'" json:"enumOptions"`
	Placeholder  string         `gorm:"column:placeholder;type:varchar(200);comment:'输入提示'" json:"placeholder"`
	SortOrder    int            `gorm:"column:sort_order;default:0;comment:'排序'" json:"sortOrder"`
	ShowInList   bool           `gorm:"column:show_in_list;default:true;comment:'是否在列表中显示'" json:"showInList"`
	Searchable   bool           `gorm:"column:searchable;default:false;comment:'是否可搜索'" json:"searchable"`
	CreateTime   util.HTime     `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
}

func (CITypeAttribute) TableName() string {
	return "ci_type_attribute"
}

// ========================================
// CI 实例（具体的配置项）
// ========================================

// CIInstance 一条具体的 CI 记录
type CIInstance struct {
	ID         uint           `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	CITypeID   uint           `gorm:"column:ci_type_id;NOT NULL;index;comment:'CI类型ID'" json:"ciTypeId"`
	CIType     CIType         `gorm:"foreignKey:CITypeID" json:"ciType,omitempty"`
	Name       string         `gorm:"column:name;type:varchar(200);NOT NULL;comment:'实例名称'" json:"name"`
	Status     int            `gorm:"column:status;default:1;comment:'状态:1-运行中,2-已停机,3-维护中,4-已下线'" json:"status"`
	Attributes datatypes.JSON `gorm:"column:attributes;type:jsonb;comment:'动态属性(JSONB)'" json:"attributes"`
	GroupID    *uint          `gorm:"column:group_id;comment:'关联业务组ID'" json:"groupId"`
	Remark     string         `gorm:"column:remark;type:varchar(500);comment:'备注'" json:"remark"`
	CreateTime util.HTime     `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime util.HTime     `gorm:"column:update_time;comment:'更新时间'" json:"updateTime"`
}

func (CIInstance) TableName() string {
	return "ci_instance"
}

// ========================================
// CI 关系（实例间的依赖关系）
// ========================================

// CIRelation 两个 CI 实例之间的关系
type CIRelation struct {
	ID           uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	FromCIID     uint       `gorm:"column:from_ci_id;NOT NULL;index;comment:'源CI实例ID'" json:"fromCiId"`
	ToCIID       uint       `gorm:"column:to_ci_id;NOT NULL;index;comment:'目标CI实例ID'" json:"toCiId"`
	RelationType string     `gorm:"column:relation_type;type:varchar(50);NOT NULL;comment:'关系类型:depends_on/runs_on/connects_to/contains'" json:"relationType"`
	CreateTime   util.HTime `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
}

func (CIRelation) TableName() string {
	return "ci_relation"
}

// ========================================
// DTOs（接口参数）
// ========================================

// CreateCITypeDto 创建 CI 类型
type CreateCITypeDto struct {
	Name        string `json:"name" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Icon        string `json:"icon"`
	Category    string `json:"category" validate:"required"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// UpdateCITypeDto 更新 CI 类型
type UpdateCITypeDto struct {
	ID          uint   `json:"id" validate:"required"`
	Name        string `json:"name" validate:"required"`
	Icon        string `json:"icon"`
	Category    string `json:"category"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
	Enabled     *bool  `json:"enabled"`
}

// CreateCITypeAttributeDto 创建属性
type CreateCITypeAttributeDto struct {
	CITypeID     uint   `json:"ciTypeId" validate:"required"`
	Name         string `json:"name" validate:"required"`
	Code         string `json:"code" validate:"required"`
	DataType     string `json:"dataType" validate:"required"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue"`
	EnumOptions  string `json:"enumOptions"` // JSON string
	Placeholder  string `json:"placeholder"`
	SortOrder    int    `json:"sortOrder"`
	ShowInList   bool   `json:"showInList"`
	Searchable   bool   `json:"searchable"`
}

// UpdateCITypeAttributeDto 更新属性
type UpdateCITypeAttributeDto struct {
	ID           uint   `json:"id" validate:"required"`
	Name         string `json:"name"`
	DataType     string `json:"dataType"`
	Required     *bool  `json:"required"`
	DefaultValue string `json:"defaultValue"`
	EnumOptions  string `json:"enumOptions"`
	Placeholder  string `json:"placeholder"`
	SortOrder    int    `json:"sortOrder"`
	ShowInList   *bool  `json:"showInList"`
	Searchable   *bool  `json:"searchable"`
}

// CreateCIInstanceDto 创建 CI 实例
type CreateCIInstanceDto struct {
	CITypeID   uint                   `json:"ciTypeId" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Status     int                    `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	GroupID    *uint                  `json:"groupId"`
	Remark     string                 `json:"remark"`
}

// UpdateCIInstanceDto 更新 CI 实例
type UpdateCIInstanceDto struct {
	ID         uint                   `json:"id" validate:"required"`
	Name       string                 `json:"name"`
	Status     int                    `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	GroupID    *uint                  `json:"groupId"`
	Remark     string                 `json:"remark"`
}

// CreateCIRelationDto 创建关系
type CreateCIRelationDto struct {
	FromCIID     uint   `json:"fromCiId" validate:"required"`
	ToCIID       uint   `json:"toCiId" validate:"required"`
	RelationType string `json:"relationType" validate:"required"`
}

// ========================================
// VOs（视图对象）
// ========================================

// CITypeVo CI 类型列表视图
type CITypeVo struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	Icon           string `json:"icon"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	BuiltIn        bool   `json:"builtIn"`
	Enabled        bool   `json:"enabled"`
	SortOrder      int    `json:"sortOrder"`
	AttributeCount int    `json:"attributeCount"`
	InstanceCount  int64  `json:"instanceCount"`
}

// CIInstanceVo CI 实例列表视图
type CIInstanceVo struct {
	ID         uint                   `json:"id"`
	CITypeID   uint                   `json:"ciTypeId"`
	TypeName   string                 `json:"typeName"`
	TypeCode   string                 `json:"typeCode"`
	TypeIcon   string                 `json:"typeIcon"`
	Name       string                 `json:"name"`
	Status     int                    `json:"status"`
	Attributes map[string]interface{} `json:"attributes"`
	GroupID    *uint                  `json:"groupId"`
	GroupName  string                 `json:"groupName"`
	Remark     string                 `json:"remark"`
	CreateTime util.HTime             `json:"createTime"`
	UpdateTime util.HTime             `json:"updateTime"`
}

// CIRelationVo CI 关系视图
type CIRelationVo struct {
	ID             uint       `json:"id"`
	FromCIID       uint       `json:"fromCiId"`
	FromCIName     string     `json:"fromCiName"`
	FromCITypeName string     `json:"fromCiTypeName"`
	ToCIID         uint       `json:"toCiId"`
	ToCIName       string     `json:"toCiName"`
	ToCITypeName   string     `json:"toCiTypeName"`
	RelationType   string     `json:"relationType"`
	CreateTime     util.HTime `json:"createTime"`
}

// ========================================
// CI 拓扑图（Phase 3）
// ========================================

// TopologyNode 拓扑图中的一个节点
type TopologyNode struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	CITypeID uint   `json:"ciTypeId"`
	TypeName string `json:"typeName"`
	TypeIcon string `json:"typeIcon"`
	TypeCode string `json:"typeCode"`
	Status   int    `json:"status"`
	IsRoot   bool   `json:"isRoot"`
}

// TopologyEdge 拓扑图中的一条连线
type TopologyEdge struct {
	ID           uint   `json:"id"`
	FromCIID     uint   `json:"fromCiId"`
	ToCIID       uint   `json:"toCiId"`
	RelationType string `json:"relationType"`
}

// CITopologyVo CI 拓扑图数据（节点 + 连线）
type CITopologyVo struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

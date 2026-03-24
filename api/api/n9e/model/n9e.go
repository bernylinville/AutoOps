package model

import (
	"dodevops-api/common/util"
)

// N9EConfig N9E 连接配置（数据库存储，仅一条记录）
type N9EConfig struct {
	ID             uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	Endpoint       string     `gorm:"column:endpoint;type:varchar(500);NOT NULL;comment:'N9E API 地址'" json:"endpoint"`
	Token          string     `gorm:"column:token;type:varchar(500);NOT NULL;comment:'X-User-Token'" json:"token"`
	Timeout        int        `gorm:"column:timeout;default:30;comment:'请求超时(秒)'" json:"timeout"`
	SyncCron       string     `gorm:"column:sync_cron;type:varchar(50);comment:'自动同步 Cron 表达式'" json:"syncCron"`
	Enabled        bool       `gorm:"column:enabled;default:false;comment:'是否启用'" json:"enabled"`
	LastSyncTime   util.HTime `gorm:"column:last_sync_time;comment:'最后同步时间'" json:"lastSyncTime"`
	LastSyncResult string     `gorm:"column:last_sync_result;type:text;comment:'最后同步结果 JSON'" json:"lastSyncResult"`
	CreateTime     util.HTime `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime     util.HTime `gorm:"column:update_time;NOT NULL;comment:'更新时间'" json:"updateTime"`
}

func (N9EConfig) TableName() string {
	return "n9e_config"
}

// N9EBusiGroup N9E 业务组（同步自 N9E）
type N9EBusiGroup struct {
	ID         uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	N9EGroupID int64      `gorm:"column:n9e_group_id;uniqueIndex;NOT NULL;comment:'N9E 业务组 ID'" json:"n9eGroupId"`
	Name       string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:'业务组名称'" json:"name"`
	CreateTime util.HTime `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime util.HTime `gorm:"column:update_time;NOT NULL;comment:'更新时间'" json:"updateTime"`
}

func (N9EBusiGroup) TableName() string {
	return "n9e_busi_group"
}

// N9EDataSource N9E 数据源（同步自 N9E）
type N9EDataSource struct {
	ID           uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
	N9ESourceID  int64      `gorm:"column:n9e_source_id;uniqueIndex;NOT NULL;comment:'N9E 数据源 ID'" json:"n9eSourceId"`
	Name         string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:'数据源名称'" json:"name"`
	PluginType   string     `gorm:"column:plugin_type;type:varchar(50);comment:'插件类型'" json:"pluginType"`
	Category     string     `gorm:"column:category;type:varchar(50);comment:'分类'" json:"category"`
	URL          string     `gorm:"column:url;type:varchar(500);comment:'HTTP URL'" json:"url"`
	Status       string     `gorm:"column:status;type:varchar(20);comment:'状态'" json:"status"`
	CreateTime   util.HTime `gorm:"column:create_time;NOT NULL;comment:'创建时间'" json:"createTime"`
	UpdateTime   util.HTime `gorm:"column:update_time;NOT NULL;comment:'更新时间'" json:"updateTime"`
}

func (N9EDataSource) TableName() string {
	return "n9e_datasource"
}

// SaveN9EConfigDto 保存 N9E 配置 DTO
type SaveN9EConfigDto struct {
	Endpoint string `json:"endpoint" validate:"required"`
	Token    string `json:"token" validate:"required"`
	Timeout  int    `json:"timeout"`
	SyncCron string `json:"syncCron"`
	Enabled  bool   `json:"enabled"`
}

// TestConnectionDto 测试 N9E 连接 DTO
type TestConnectionDto struct {
	Endpoint string `json:"endpoint" validate:"required"`
	Token    string `json:"token" validate:"required"`
	Timeout  int    `json:"timeout"`
}

// N9ESyncLog 同步日志记录
type N9ESyncLog struct {
	ID         uint   `gorm:"column:id;primaryKey" json:"id"`
	SyncType   string `gorm:"column:sync_type;type:varchar(20);default:'full'" json:"syncType"`
	Status     string `gorm:"column:status;type:varchar(20);default:'success'" json:"status"`
	ResultJSON string `gorm:"column:result_json;type:text" json:"resultJson"`
	ErrorMsg   string `gorm:"column:error_msg;type:text" json:"errorMsg"`
	DurationMs int    `gorm:"column:duration_ms" json:"durationMs"`
	TriggerBy  string `gorm:"column:trigger_by;type:varchar(20);default:'manual'" json:"triggerBy"`
	CreatedAt  util.HTime `gorm:"column:created_at" json:"createdAt"`
}

func (N9ESyncLog) TableName() string {
	return "n9e_sync_log"
}

// PromQLQueryDto PromQL 查询参数
type PromQLQueryDto struct {
	DatasourceID uint   `form:"datasourceId"`
	Query        string `form:"query" binding:"required"`
	Start        string `form:"start"`
	End          string `form:"end"`
	Step         string `form:"step"`
}

package dao

import (
	"dodevops-api/api/n9e/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GetN9EConfig 获取 N9E 配置（仅取第一条记录）
func GetN9EConfig() (*model.N9EConfig, error) {
	var config model.N9EConfig
	err := common.GetDB().First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveN9EConfig 保存 N9E 配置（存在则更新，不存在则创建）
func SaveN9EConfig(dto model.SaveN9EConfigDto) (*model.N9EConfig, error) {
	db := common.GetDB()
	now := util.HTime{Time: time.Now()}

	var existing model.N9EConfig
	err := db.First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 新建
		config := model.N9EConfig{
			Endpoint:   dto.Endpoint,
			Token:      dto.Token,
			Timeout:    dto.Timeout,
			SyncCron:   dto.SyncCron,
			Enabled:    dto.Enabled,
			CreateTime: now,
			UpdateTime: now,
		}
		if config.Timeout <= 0 {
			config.Timeout = 30
		}
		if err := db.Create(&config).Error; err != nil {
			return nil, err
		}
		return &config, nil
	}

	if err != nil {
		return nil, err
	}

	// 更新
	timeout := dto.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	updates := map[string]interface{}{
		"endpoint":    dto.Endpoint,
		"timeout":     timeout,
		"sync_cron":   dto.SyncCron,
		"enabled":     dto.Enabled,
		"update_time": now,
	}

	// 仅当 Token 不包含掩码字符时才更新（防止脱敏值回写覆盖真实凭据）
	if !strings.Contains(dto.Token, "****") {
		updates["token"] = dto.Token
	}

	err = db.Model(&existing).Updates(updates).Error
	if err != nil {
		return nil, err
	}

	return &existing, nil
}

// UpdateN9ESyncResult 更新最后同步结果
func UpdateN9ESyncResult(resultJSON string) error {
	now := util.HTime{Time: time.Now()}
	return common.GetDB().Model(&model.N9EConfig{}).
		Where("id > 0").
		Updates(map[string]interface{}{
			"last_sync_time":   now,
			"last_sync_result": resultJSON,
			"update_time":      now,
		}).Error
}

// GetN9EBusiGroups 获取所有 N9E 业务组
func GetN9EBusiGroups() ([]model.N9EBusiGroup, error) {
	var groups []model.N9EBusiGroup
	err := common.GetDB().Order("name ASC").Find(&groups).Error
	return groups, err
}

// UpsertN9EBusiGroup 创建或更新 N9E 业务组
func UpsertN9EBusiGroup(n9eGroupID int64, name string) (created bool, err error) {
	db := common.GetDB()
	now := util.HTime{Time: time.Now()}

	var existing model.N9EBusiGroup
	err = db.Where("n9e_group_id = ?", n9eGroupID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		group := model.N9EBusiGroup{
			N9EGroupID: n9eGroupID,
			Name:       name,
			CreateTime: now,
			UpdateTime: now,
		}
		return true, db.Create(&group).Error
	}

	if err != nil {
		return false, err
	}

	if existing.Name != name {
		return false, db.Model(&existing).Updates(map[string]interface{}{
			"name":        name,
			"update_time": now,
		}).Error
	}

	// 仅更新 update_time
	return false, db.Model(&existing).Update("update_time", now).Error
}

// GetN9EDataSources 获取所有 N9E 数据源
func GetN9EDataSources() ([]model.N9EDataSource, error) {
	var datasources []model.N9EDataSource
	err := common.GetDB().Order("name ASC").Find(&datasources).Error
	return datasources, err
}

// GetN9EDataSourceByID 按 ID 获取数据源
func GetN9EDataSourceByID(id uint) (*model.N9EDataSource, error) {
	var ds model.N9EDataSource
	err := common.GetDB().First(&ds, id).Error
	if err != nil {
		return nil, err
	}
	return &ds, nil
}

// UpsertN9EDataSource 创建或更新 N9E 数据源
func UpsertN9EDataSource(n9eSourceID int64, name, pluginType, category, url, status string) (created bool, err error) {
	db := common.GetDB()
	now := util.HTime{Time: time.Now()}

	var existing model.N9EDataSource
	err = db.Where("n9e_source_id = ?", n9eSourceID).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		ds := model.N9EDataSource{
			N9ESourceID: n9eSourceID,
			Name:        name,
			PluginType:  pluginType,
			Category:    category,
			URL:         url,
			Status:      status,
			CreateTime:  now,
			UpdateTime:  now,
		}
		return true, db.Create(&ds).Error
	}

	if err != nil {
		return false, err
	}

	return false, db.Model(&existing).Updates(map[string]interface{}{
		"name":        name,
		"plugin_type": pluginType,
		"category":    category,
		"url":         url,
		"status":      status,
		"update_time": now,
	}).Error
}

// FindOrCreateCmdbGroup 根据名称查找 CMDB 分组，不存在则自动创建
// 使用事务保证并发安全，避免竞态条件导致重复创建
func FindOrCreateCmdbGroup(name string) (uint, error) {
	db := common.GetDB()
	var groupID uint

	err := db.Transaction(func(tx *gorm.DB) error {
		// 事务内查找同名分组
		var group struct {
			ID uint `gorm:"column:id"`
		}
		err := tx.Table("cmdb_group").Where("name = ?", name).First(&group).Error
		if err == nil {
			groupID = group.ID
			return nil
		}

		if err != gorm.ErrRecordNotFound {
			return err
		}

		// 不存在则创建（parent_id=0 作为根分组）
		now := util.HTime{Time: time.Now()}
		newGroup := map[string]interface{}{
			"parent_id":   0,
			"name":        name,
			"create_time": now,
		}
		result := tx.Table("cmdb_group").Create(&newGroup)
		if result.Error != nil {
			return result.Error
		}

		// 获取新创建的 ID
		var created struct {
			ID uint `gorm:"column:id"`
		}
		if err := tx.Table("cmdb_group").Where("name = ?", name).First(&created).Error; err != nil {
			return err
		}
		groupID = created.ID
		return nil
	})

	return groupID, err
}

// GetN9EHostIdents 获取所有 N9E 来源主机的 ident 列表
func GetN9EHostIdents() ([]string, error) {
	var idents []string
	err := common.GetDB().Table("cmdb_host").
		Where("source_type = ?", "n9e").
		Pluck("n9e_ident", &idents).Error
	return idents, err
}

// MarkHostsStale 将指定 ident 的主机标记为 stale（status=4）
func MarkHostsStale(idents []string) (int64, error) {
	if len(idents) == 0 {
		return 0, nil
	}
	now := util.HTime{Time: time.Now()}
	result := common.GetDB().Table("cmdb_host").
		Where("n9e_ident IN ? AND source_type = ?", idents, "n9e").
		Updates(map[string]interface{}{
			"status":      4,
			"update_time": now,
		})
	return result.RowsAffected, result.Error
}

// CreateSyncLog 创建同步日志记录
func CreateSyncLog(syncType, status, resultJSON, errorMsg, triggerBy string, durationMs int) error {
	log := model.N9ESyncLog{
		SyncType:   syncType,
		Status:     status,
		ResultJSON: resultJSON,
		ErrorMsg:   errorMsg,
		DurationMs: durationMs,
		TriggerBy:  triggerBy,
	}
	return common.GetDB().Create(&log).Error
}

// GetSyncLogs 获取同步日志列表（按时间倒序）
func GetSyncLogs(limit int) ([]model.N9ESyncLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var logs []model.N9ESyncLog
	err := common.GetDB().Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

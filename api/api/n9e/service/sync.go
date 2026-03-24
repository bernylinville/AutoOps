package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/api/n9e/model"
	"dodevops-api/common"
	cmdbModel "dodevops-api/api/cmdb/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/log"
	"time"
)

// SyncService N9E 数据同步服务
type SyncService struct {
	mu sync.Mutex
}

var syncServiceInstance = &SyncService{}

func GetSyncService() *SyncService {
	return syncServiceInstance
}

// FullSync 执行全量同步（业务组 + 主机 + 数据源）
// triggerBy: "manual" 或 "cron"
func (s *SyncService) FullSync(ctx context.Context, triggerBy string) (*model.SyncResult, error) {
	// 防重入：同一时间只允许一个同步任务运行
	if !s.mu.TryLock() {
		return nil, fmt.Errorf("同步任务正在执行中，请稍后重试")
	}
	defer s.mu.Unlock()

	startTime := time.Now()

	// 获取 N9E 配置
	cfg, err := dao.GetN9EConfig()
	if err != nil {
		return nil, fmt.Errorf("get N9E config: %w", err)
	}

	if !cfg.Enabled {
		return nil, fmt.Errorf("N9E integration is not enabled")
	}

	client := NewN9EClient(cfg.Endpoint, cfg.Token, cfg.Timeout)
	result := &model.SyncResult{}

	// 1. 同步业务组
	groups, err := client.GetBusiGroups(ctx)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("fetch N9E business groups: %w", err)
	}
	result.BusiGroups, err = s.syncBusiGroups(groups)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("sync business groups: %w", err)
	}
	log.Log().Infof("[N9E Sync] business groups: created=%d, updated=%d, skipped=%d",
		result.BusiGroups.Created, result.BusiGroups.Updated, result.BusiGroups.Skipped)

	// 2. 同步主机
	targets, err := client.GetTargets(ctx)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("fetch N9E targets: %w", err)
	}
	result.Hosts, err = s.syncHosts(targets)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("sync hosts: %w", err)
	}
	log.Log().Infof("[N9E Sync] hosts: created=%d, updated=%d, skipped=%d, stale=%d",
		result.Hosts.Created, result.Hosts.Updated, result.Hosts.Skipped, result.Hosts.Stale)

	// 3. 同步数据源
	datasources, err := client.GetDatasourceBriefs(ctx)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("fetch N9E datasources: %w", err)
	}
	result.Datasources, err = s.syncDatasources(datasources)
	if err != nil {
		s.recordSyncLog("full", "failed", nil, err.Error(), triggerBy, startTime)
		return nil, fmt.Errorf("sync datasources: %w", err)
	}
	log.Log().Infof("[N9E Sync] datasources: created=%d, updated=%d, skipped=%d",
		result.Datasources.Created, result.Datasources.Updated, result.Datasources.Skipped)

	// 保存同步结果
	resultJSON, _ := json.Marshal(result)
	if err := dao.UpdateN9ESyncResult(string(resultJSON)); err != nil {
		log.Log().Warnf("[N9E Sync] failed to save sync result: %v", err)
	}

	// 记录同步日志
	s.recordSyncLog("full", "success", result, "", triggerBy, startTime)

	return result, nil
}

// recordSyncLog 记录同步日志
func (s *SyncService) recordSyncLog(syncType, status string, result *model.SyncResult, errorMsg, triggerBy string, startTime time.Time) {
	durationMs := int(time.Since(startTime).Milliseconds())
	var resultJSON string
	if result != nil {
		b, _ := json.Marshal(result)
		resultJSON = string(b)
	}
	if triggerBy == "" {
		triggerBy = "manual"
	}
	if err := dao.CreateSyncLog(syncType, status, resultJSON, errorMsg, triggerBy, durationMs); err != nil {
		log.Log().Warnf("[N9E Sync] failed to create sync log: %v", err)
	}
}

// syncBusiGroups 同步业务组
func (s *SyncService) syncBusiGroups(groups []model.BusiGroupData) (model.SyncStats, error) {
	var stats model.SyncStats
	for _, group := range groups {
		name := strings.TrimSpace(group.Name)
		created, err := dao.UpsertN9EBusiGroup(group.ID, name)
		if err != nil {
			return stats, fmt.Errorf("upsert business group %d: %w", group.ID, err)
		}
		if created {
			stats.Created++
		} else {
			stats.Updated++
		}
	}
	return stats, nil
}

// syncHosts 同步主机到 cmdb_host 表
func (s *SyncService) syncHosts(targets []model.TargetData) (model.SyncStats, error) {
	var stats model.SyncStats
	db := common.GetDB()

	// 收集本次同步到的所有 ident，用于后续 stale 检测
	syncedIdents := make(map[string]bool)

	for _, target := range targets {
		ident := strings.TrimSpace(target.Ident)
		if ident == "" {
			stats.Skipped++
			continue
		}

		syncedIdents[ident] = true
		hostname, cpuModel, memoryTotal, _ := ExtractTargetMetadata(target)
		ip := strings.TrimSpace(target.HostIP)
		if ip == "" {
			ip = strings.TrimSpace(target.RemoteAddr)
		}

		// 解析业务组 → 映射到 CMDB 分组
		groupID := resolveGroupID(target)

		// 查找是否已存在（按 n9e_ident 匹配）
		var existing cmdbModel.CmdbHost
		err := db.Where("n9e_ident = ?", ident).First(&existing).Error

		if err != nil {
			// 记录不存在，也尝试按 IP 匹配
			if ip != "" {
				err2 := db.Where("(public_ip = ? OR private_ip = ? OR ssh_ip = ?) AND source_type = 'manual'",
					ip, ip, ip).First(&existing).Error
				if err2 == nil {
					// 根据 target_up 设置状态: 1=在线, 0=离线(3)
					hostStatus := 3
					if target.TargetUp == 1 {
						hostStatus = 1
					}
					updates := map[string]interface{}{
						"source_type": "n9e",
						"n9e_id":      target.ID,
						"n9e_ident":   ident,
						"os":          strings.TrimSpace(target.OS),
						"status":      hostStatus,
						"update_time": util.HTime{Time: time.Now()},
					}
					if groupID != 0 {
						updates["group_id"] = groupID
					}
					if err := db.Model(&existing).Updates(updates).Error; err != nil {
						log.Log().Warnf("[N9E Sync] failed to update host %s: %v", ident, err)
						stats.Skipped++
						continue
					}
					stats.Updated++
					continue
				}
			}

			// 完全新建
			if groupID == 0 {
				groupID = 1 // 默认分组
			}
			// 根据 target_up 设置状态: 1=在线, 0=离线(3)
			hostStatus := 3
			if target.TargetUp == 1 {
				hostStatus = 1
			}
			newHost := cmdbModel.CmdbHost{
				HostName:   hostname,
				Name:       ident,
				GroupID:    groupID,
				PublicIP:   ip,
				PrivateIP:  ip,
				SSHIP:      ip,
				SSHName:    "root",
				SSHPort:    22,
				OS:         strings.TrimSpace(target.OS),
				CPU:        cpuModel,
				Memory:     memoryTotal,
				Status:     hostStatus, // 根据 target_up: 1=在线, 3=离线
				Vendor:     1, // 自建
				SourceType: "n9e",
				N9EID:      target.ID,
				N9EIdent:   ident,
				CreateTime: util.HTime{Time: time.Now()},
				UpdateTime: util.HTime{Time: time.Now()},
			}
			if err := db.Create(&newHost).Error; err != nil {
				log.Log().Warnf("[N9E Sync] failed to create host %s: %v", ident, err)
				stats.Skipped++
				continue
			}
			stats.Created++
			continue
		}

		// 已存在，更新
		updates := map[string]interface{}{
			"n9e_id":      target.ID,
			"n9e_ident":   ident,
			"source_type": "n9e",
			"os":          strings.TrimSpace(target.OS),
			"update_time": util.HTime{Time: time.Now()},
		}
		// 如果之前是 stale，根据 target_up 恢复状态
		if existing.Status == 4 {
			if target.TargetUp == 1 {
				updates["status"] = 1
			} else {
				updates["status"] = 3
			}
		} else {
			// 正常更新也同步最新 target_up 状态
			if target.TargetUp == 1 {
				updates["status"] = 1
			} else {
				updates["status"] = 3
			}
		}
		if hostname != "" && existing.HostName == "" {
			updates["host_name"] = hostname
		}
		if ip != "" && existing.PublicIP == "" {
			updates["public_ip"] = ip
		}
		if cpuModel != "" {
			updates["cpu"] = cpuModel
		}
		if memoryTotal != "" {
			updates["memory"] = memoryTotal
		}
		if groupID != 0 {
			updates["group_id"] = groupID
		}

		if err := db.Model(&existing).Updates(updates).Error; err != nil {
			log.Log().Warnf("[N9E Sync] failed to update host %s: %v", ident, err)
			stats.Skipped++
			continue
		}
		stats.Updated++
	}

	// Stale 检测：找出数据库中 source_type='n9e' 但本次同步未出现的主机
	existingIdents, err := dao.GetN9EHostIdents()
	if err != nil {
		log.Log().Warnf("[N9E Sync] failed to get existing N9E idents for stale check: %v", err)
	} else {
		var staleIdents []string
		for _, ident := range existingIdents {
			if !syncedIdents[ident] {
				staleIdents = append(staleIdents, ident)
			}
		}
		if len(staleIdents) > 0 {
			affected, err := dao.MarkHostsStale(staleIdents)
			if err != nil {
				log.Log().Warnf("[N9E Sync] failed to mark stale hosts: %v", err)
			} else {
				stats.Stale = int(affected)
				log.Log().Infof("[N9E Sync] marked %d hosts as stale", affected)
			}
		}
	}

	return stats, nil
}

// resolveGroupID 根据 N9E target 的 GroupObjs 解析 CMDB 分组 ID
func resolveGroupID(target model.TargetData) uint {
	if len(target.GroupObjs) == 0 {
		return 0
	}

	// 使用第一个业务组名称映射到 CMDB 分组
	groupName := strings.TrimSpace(target.GroupObjs[0].Name)
	if groupName == "" {
		return 0
	}

	groupID, err := dao.FindOrCreateCmdbGroup(groupName)
	if err != nil {
		log.Log().Warnf("[N9E Sync] failed to resolve group '%s': %v", groupName, err)
		return 0
	}

	return groupID
}

// syncDatasources 同步数据源
func (s *SyncService) syncDatasources(datasources []model.DatasourceData) (model.SyncStats, error) {
	var stats model.SyncStats
	for _, ds := range datasources {
		created, err := dao.UpsertN9EDataSource(
			ds.ID,
			strings.TrimSpace(ds.Name),
			strings.TrimSpace(ds.PluginType),
			strings.TrimSpace(ds.Category),
			strings.TrimSpace(ds.HTTP.URL),
			strings.TrimSpace(ds.Status),
		)
		if err != nil {
			return stats, fmt.Errorf("upsert datasource %d: %w", ds.ID, err)
		}
		if created {
			stats.Created++
		} else {
			stats.Updated++
		}
	}
	return stats, nil
}

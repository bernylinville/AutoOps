package controller

import (
	"dodevops-api/common"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetOverview 获取 N9E / CMDB 总览统计
// @Summary 获取 CMDB 总览统计
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/overview [get]
func (ctrl *N9EController) GetOverview(c *gin.Context) {
	db := common.GetDB()

	// 主机统计
	var hostStats struct {
		Total    int64 `json:"total"`
		N9E      int64 `json:"n9e"`
		Manual   int64 `json:"manual"`
		Cloud    int64 `json:"cloud"`
		Online   int64 `json:"online"`
		Stale    int64 `json:"stale"`
		Degraded int64 `json:"degraded"`
	}

	db.Table("cmdb_host").Count(&hostStats.Total)
	db.Table("cmdb_host").Where("source_type = ?", "n9e").Count(&hostStats.N9E)
	db.Table("cmdb_host").Where("source_type = ? OR source_type = ''", "manual").Count(&hostStats.Manual)
	db.Table("cmdb_host").Where("source_type IN ?", []string{"aliyun", "tencent"}).Count(&hostStats.Cloud)
	db.Table("cmdb_host").Where("status = ?", 1).Count(&hostStats.Online)
	db.Table("cmdb_host").Where("status = ?", 4).Count(&hostStats.Stale)
	db.Table("cmdb_host").Where("status = ?", 5).Count(&hostStats.Degraded)

	// 业务组统计
	var groupCount int64
	db.Table("n9e_busi_group").Count(&groupCount)

	// 数据源统计
	var dsCount int64
	db.Table("n9e_datasource").Count(&dsCount)

	// CMDB 分组统计
	var cmdbGroupCount int64
	db.Table("cmdb_group").Count(&cmdbGroupCount)

	// 最后同步信息
	var syncInfo struct {
		LastSyncTime   *string `gorm:"column:last_sync_time" json:"lastSyncTime"`
		LastSyncResult *string `gorm:"column:last_sync_result" json:"lastSyncResult"`
		Enabled        bool    `gorm:"column:enabled" json:"enabled"`
	}
	db.Table("n9e_config").Select("last_sync_time, last_sync_result, enabled").First(&syncInfo)

	result.Success(c, gin.H{
		"hosts": gin.H{
			"total":    hostStats.Total,
			"n9e":      hostStats.N9E,
			"manual":   hostStats.Manual,
			"cloud":    hostStats.Cloud,
			"online":   hostStats.Online,
			"offline":  hostStats.Total - hostStats.Online - hostStats.Stale - hostStats.Degraded,
			"stale":    hostStats.Stale,
			"degraded": hostStats.Degraded,
		},
		"n9eBusiGroups":  groupCount,
		"datasources":    dsCount,
		"cmdbGroups":     cmdbGroupCount,
		"lastSyncTime":   syncInfo.LastSyncTime,
		"lastSyncResult": syncInfo.LastSyncResult,
		"n9eEnabled":     syncInfo.Enabled,
		"busiGroupStats": getBusiGroupHostCounts(db),
		"healthScore":    calcHealthScore(hostStats.Total, hostStats.Online),
	})
}

// getBusiGroupHostCounts 获取每个业务组关联的主机数
func getBusiGroupHostCounts(db *gorm.DB) []gin.H {
	type groupStat struct {
		ID        uint   `gorm:"column:id"`
		Name      string `gorm:"column:name"`
		HostCount int64  `gorm:"column:host_count"`
	}
	var stats []groupStat
	db.Table("n9e_busi_group AS g").
		Select("g.id, g.name, COUNT(h.id) AS host_count").
		Joins("LEFT JOIN cmdb_host h ON h.group_name = g.name AND h.source_type = 'n9e'").
		Group("g.id, g.name").
		Order("host_count DESC, g.name ASC").
		Find(&stats)

	result := make([]gin.H, len(stats))
	for i, s := range stats {
		result[i] = gin.H{"id": s.ID, "name": s.Name, "hostCount": s.HostCount}
	}
	return result
}

// calcHealthScore 计算健康分数（在线率 0-100）
func calcHealthScore(total, online int64) int {
	if total == 0 {
		return 100
	}
	return int(float64(online) / float64(total) * 100)
}

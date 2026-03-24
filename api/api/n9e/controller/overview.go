package controller

import (
	"dodevops-api/common"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
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
		Total  int64 `json:"total"`
		N9E    int64 `json:"n9e"`
		Manual int64 `json:"manual"`
		Cloud  int64 `json:"cloud"`
		Online int64 `json:"online"`
		Stale  int64 `json:"stale"`
	}

	db.Table("cmdb_host").Count(&hostStats.Total)
	db.Table("cmdb_host").Where("source_type = ?", "n9e").Count(&hostStats.N9E)
	db.Table("cmdb_host").Where("source_type = ? OR source_type = ''", "manual").Count(&hostStats.Manual)
	db.Table("cmdb_host").Where("source_type IN ?", []string{"aliyun", "tencent"}).Count(&hostStats.Cloud)
	db.Table("cmdb_host").Where("status = ?", 1).Count(&hostStats.Online)
	db.Table("cmdb_host").Where("status = ?", 4).Count(&hostStats.Stale)

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
			"total":   hostStats.Total,
			"n9e":     hostStats.N9E,
			"manual":  hostStats.Manual,
			"cloud":   hostStats.Cloud,
			"online":  hostStats.Online,
			"offline": hostStats.Total - hostStats.Online - hostStats.Stale,
			"stale":   hostStats.Stale,
		},
		"n9eBusiGroups":  groupCount,
		"datasources":    dsCount,
		"cmdbGroups":     cmdbGroupCount,
		"lastSyncTime":   syncInfo.LastSyncTime,
		"lastSyncResult": syncInfo.LastSyncResult,
		"n9eEnabled":     syncInfo.Enabled,
	})
}

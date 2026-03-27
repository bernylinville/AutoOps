package controller

import (
	"context"
	"fmt"
	"strconv"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/api/n9e/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

// QueryVMMetrics 通用 VictoriaMetrics PromQL 查询
// @Summary 通用 VM PromQL 查询
// @Tags N9E-VM
// @Param datasourceId query int false "数据源ID(默认使用第一个)"
// @Param query query string true "PromQL 表达式"
// @Param start query string false "开始时间(unix timestamp)"
// @Param end query string false "结束时间(unix timestamp)"
// @Param step query string false "步长(如15s,1m)"
// @Router /api/v1/n9e/vm/query [get]
func (ctrl *N9EController) QueryVMMetrics(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		result.Failed(c, 400, "query 参数不能为空")
		return
	}

	dsID := parseUintParam(c, "datasourceId")
	dsURL, err := resolveDatasourceURL(dsID)
	if err != nil {
		result.Failed(c, 400, err.Error())
		return
	}

	vmSvc := service.GetVMService()
	start := c.Query("start")
	end := c.Query("end")
	step := c.DefaultQuery("step", "60s")

	if start != "" && end != "" {
		data, err := vmSvc.QueryRange(context.Background(), dsURL, query, start, end, step)
		if err != nil {
			result.Failed(c, 500, "查询失败: "+err.Error())
			return
		}
		result.Success(c, data)
		return
	}

	data, err := vmSvc.QueryInstant(context.Background(), dsURL, query)
	if err != nil {
		result.Failed(c, 500, "查询失败: "+err.Error())
		return
	}
	result.Success(c, data)
}

// GetHostVMMetrics 获取指定主机的实时 VM 监控数据
// @Summary 获取主机实时监控数据(VictoriaMetrics)
// @Tags N9E-VM
// @Param ident path string true "主机标识(N9E ident)"
// @Param datasourceId query int false "数据源ID"
// @Router /api/v1/n9e/vm/host/{ident} [get]
func (ctrl *N9EController) GetHostVMMetrics(c *gin.Context) {
	ident := c.Param("ident")
	if ident == "" {
		result.Failed(c, 400, "主机标识不能为空")
		return
	}

	dsID := parseUintParam(c, "datasourceId")
	vmSvc := service.GetVMService()

	metrics, err := vmSvc.GetHostMetrics(context.Background(), dsID, ident)
	if err != nil {
		result.Failed(c, 500, "获取监控数据失败: "+err.Error())
		return
	}

	result.Success(c, metrics)
}

// GetHostVMHistory 获取主机历史 VM 监控数据
// @Summary 获取主机历史监控数据(VictoriaMetrics)
// @Tags N9E-VM
// @Param ident path string true "主机标识(N9E ident)"
// @Param datasourceId query int false "数据源ID"
// @Param start query string true "开始时间(unix timestamp)"
// @Param end query string true "结束时间(unix timestamp)"
// @Param step query string false "步长(默认60s)"
// @Router /api/v1/n9e/vm/host/{ident}/history [get]
func (ctrl *N9EController) GetHostVMHistory(c *gin.Context) {
	ident := c.Param("ident")
	if ident == "" {
		result.Failed(c, 400, "主机标识不能为空")
		return
	}

	start := c.Query("start")
	end := c.Query("end")
	if start == "" || end == "" {
		result.Failed(c, 400, "start 和 end 参数不能为空")
		return
	}

	step := c.DefaultQuery("step", "60s")
	dsID := parseUintParam(c, "datasourceId")
	vmSvc := service.GetVMService()

	data, err := vmSvc.GetHostMetricsHistory(context.Background(), dsID, ident, start, end, step)
	if err != nil {
		result.Failed(c, 500, "获取历史监控数据失败: "+err.Error())
		return
	}

	result.Success(c, data)
}

// GetClusterVMOverview 获取集群 VM 监控总览
// @Summary 获取集群监控总览(VictoriaMetrics)
// @Tags N9E-VM
// @Param datasourceId query int false "数据源ID"
// @Router /api/v1/n9e/vm/overview [get]
func (ctrl *N9EController) GetClusterVMOverview(c *gin.Context) {
	dsID := parseUintParam(c, "datasourceId")
	vmSvc := service.GetVMService()

	overview, err := vmSvc.GetClusterOverview(context.Background(), dsID)
	if err != nil {
		result.Failed(c, 500, "获取集群总览失败: "+err.Error())
		return
	}

	result.Success(c, overview)
}

// parseUintParam 解析 uint 查询参数
func parseUintParam(c *gin.Context, key string) uint {
	val := c.Query(key)
	if val == "" {
		return 0
	}
	n, _ := strconv.ParseUint(val, 10, 64)
	return uint(n)
}

// resolveDatasourceURL 解析数据源 URL
func resolveDatasourceURL(dsID uint) (string, error) {
	if dsID == 0 {
		dsList, err := dao.GetN9EDataSources()
		if err != nil || len(dsList) == 0 {
			return "", fmt.Errorf("没有可用的数据源")
		}
		return dsList[0].URL, nil
	}
	ds, err := dao.GetN9EDataSourceByID(dsID)
	if err != nil {
		return "", err
	}
	return ds.URL, nil
}

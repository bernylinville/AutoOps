package controller

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/api/n9e/model"
	"dodevops-api/api/n9e/service"
	"dodevops-api/common/result"
	"dodevops-api/scheduler"

	"github.com/gin-gonic/gin"
)

// N9EController N9E 控制器
type N9EController struct {
	syncService *service.SyncService
}

// NewN9EController 创建 N9E 控制器
func NewN9EController() *N9EController {
	return &N9EController{
		syncService: service.GetSyncService(),
	}
}

// GetConfig 获取 N9E 配置
// @Summary 获取 N9E 配置
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/config [get]
func (ctrl *N9EController) GetConfig(c *gin.Context) {
	config, err := dao.GetN9EConfig()
	if err != nil {
		// 没有配置时返回空对象
		result.Success(c, gin.H{
			"endpoint": "",
			"token":    "",
			"timeout":  30,
			"syncCron": "",
			"enabled":  false,
		})
		return
	}

	// 隐藏 Token 中间部分
	maskedToken := maskToken(config.Token)
	result.Success(c, gin.H{
		"id":             config.ID,
		"endpoint":       config.Endpoint,
		"token":          maskedToken,
		"timeout":        config.Timeout,
		"syncCron":       config.SyncCron,
		"enabled":        config.Enabled,
		"lastSyncTime":   config.LastSyncTime,
		"lastSyncResult": config.LastSyncResult,
	})
}

// SaveConfig 保存 N9E 配置
// @Summary 保存 N9E 配置
// @Tags N9E
// @Accept json
// @Produce json
// @Param data body model.SaveN9EConfigDto true "N9E配置"
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/config [post]
func (ctrl *N9EController) SaveConfig(c *gin.Context) {
	var dto model.SaveN9EConfigDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}

	if dto.Endpoint == "" {
		result.Failed(c, 400, "Endpoint 不能为空")
		return
	}

	// Token 包含 **** 表示是脱敏回显值，不作为必填校验
	if dto.Token == "" {
		result.Failed(c, 400, "Token 不能为空")
		return
	}

	config, err := dao.SaveN9EConfig(dto)
	if err != nil {
		result.Failed(c, 500, "保存配置失败: "+err.Error())
		return
	}

	// 重新加载 N9E 定时同步配置
	if err := scheduler.GetManager().ReloadN9ECron(); err != nil {
		// 只记录警告，不影响保存结果
		result.Success(c, gin.H{
			"config":      config,
			"cronWarning": err.Error(),
		})
		return
	}

	result.Success(c, config)
}

// TestConnection 测试 N9E 连接
// @Summary 测试 N9E 连接
// @Tags N9E
// @Accept json
// @Produce json
// @Param data body model.TestConnectionDto true "连接信息"
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/test-connection [post]
func (ctrl *N9EController) TestConnection(c *gin.Context) {
	var dto model.TestConnectionDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}

	if dto.Endpoint == "" || dto.Token == "" {
		result.Failed(c, 400, "Endpoint 和 Token 不能为空")
		return
	}

	// 如果 Token 是脱敏值，从数据库读取真实 Token
	token := dto.Token
	if strings.Contains(token, "****") {
		config, err := dao.GetN9EConfig()
		if err != nil || config.Token == "" {
			result.Failed(c, 400, "Token 为脱敏值且数据库中无有效 Token，请重新输入完整 Token")
			return
		}
		token = config.Token
	}

	timeout := dto.Timeout
	if timeout <= 0 {
		timeout = 30
	}

	client := service.NewN9EClient(dto.Endpoint, token, timeout)
	err := client.TestConnection(context.Background())
	if err != nil {
		result.Failed(c, 500, "连接失败: "+err.Error())
		return
	}

	result.Success(c, gin.H{"message": "连接成功"})
}

// TriggerSync 手动触发同步
// @Summary 手动触发 N9E 数据同步
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/sync [post]
func (ctrl *N9EController) TriggerSync(c *gin.Context) {
	syncResult, err := ctrl.syncService.FullSync(context.Background(), "manual")
	if err != nil {
		result.Failed(c, 500, "同步失败: "+err.Error())
		return
	}

	result.Success(c, syncResult)
}

// GetSyncStatus 获取同步状态
// @Summary 获取 N9E 同步状态
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/sync/status [get]
func (ctrl *N9EController) GetSyncStatus(c *gin.Context) {
	config, err := dao.GetN9EConfig()
	if err != nil {
		result.Success(c, gin.H{
			"lastSyncTime":   nil,
			"lastSyncResult": nil,
		})
		return
	}

	result.Success(c, gin.H{
		"lastSyncTime":   config.LastSyncTime,
		"lastSyncResult": config.LastSyncResult,
	})
}

// GetBusiGroups 获取业务组列表
// @Summary 获取 N9E 业务组列表
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/busi-groups [get]
func (ctrl *N9EController) GetBusiGroups(c *gin.Context) {
	groups, err := dao.GetN9EBusiGroups()
	if err != nil {
		result.Failed(c, 500, "获取业务组失败: "+err.Error())
		return
	}

	result.Success(c, groups)
}

// GetDatasources 获取数据源列表
// @Summary 获取 N9E 数据源列表
// @Tags N9E
// @Accept json
// @Produce json
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/datasources [get]
func (ctrl *N9EController) GetDatasources(c *gin.Context) {
	datasources, err := dao.GetN9EDataSources()
	if err != nil {
		result.Failed(c, 500, "获取数据源失败: "+err.Error())
		return
	}

	result.Success(c, datasources)
}

// QueryPromQL PromQL 查询
// @Summary PromQL 查询（透传给 VictoriaMetrics）
// @Tags N9E
// @Accept json
// @Produce json
// @Param datasourceId query int true "数据源 ID"
// @Param query query string true "PromQL 表达式"
// @Param start query string false "开始时间"
// @Param end query string false "结束时间"
// @Param step query string false "步长"
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/query [get]
func (ctrl *N9EController) QueryPromQL(c *gin.Context) {
	var dto model.PromQLQueryDto
	if err := c.ShouldBindQuery(&dto); err != nil {
		result.Failed(c, 400, "参数错误: "+err.Error())
		return
	}

	// 获取数据源 URL
	ds, err := dao.GetN9EDataSourceByID(dto.DatasourceID)
	if err != nil {
		result.Failed(c, 404, "数据源不存在")
		return
	}

	if ds.URL == "" {
		result.Failed(c, 400, "数据源未配置 URL")
		return
	}

	// 获取 N9E 配置中的 token
	cfg, err := dao.GetN9EConfig()
	if err != nil {
		result.Failed(c, 500, "获取 N9E 配置失败")
		return
	}

	client := service.NewN9EClient(cfg.Endpoint, cfg.Token, cfg.Timeout)
	data, err := client.QueryPromQL(context.Background(), ds.URL, dto.Query, dto.Start, dto.End, dto.Step)
	if err != nil {
		result.Failed(c, 500, "查询失败: "+err.Error())
		return
	}

	c.Data(200, "application/json", data)
}

// maskToken 隐藏 Token 中间部分
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// GetSyncLogs 获取同步日志列表
// @Summary 获取同步日志列表
// @Tags N9E
// @Accept json
// @Produce json
// @Param limit query int false "返回条数(默认20,最大100)"
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/sync/logs [get]
func (ctrl *N9EController) GetSyncLogs(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}

	logs, err := dao.GetSyncLogs(limit)
	if err != nil {
		result.Failed(c, 500, "获取同步日志失败: "+err.Error())
		return
	}

	result.Success(c, logs)
}

// CheckDatasource 检测数据源连通性
// @Summary 检测数据源连通性
// @Tags N9E
// @Accept json
// @Produce json
// @Param id path int true "数据源ID"
// @Success 200 {object} result.Result
// @Router /api/v1/n9e/datasources/{id}/check [post]
func (ctrl *N9EController) CheckDatasource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		result.Failed(c, 400, "无效的数据源 ID")
		return
	}

	ds, err := dao.GetN9EDataSourceByID(uint(id))
	if err != nil {
		result.Failed(c, 404, "数据源不存在")
		return
	}

	if ds.URL == "" {
		result.Failed(c, 400, "数据源未配置 URL")
		return
	}

	// Ping 数据源
	start := time.Now()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimRight(ds.URL, "/") + "/-/healthy")
	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		// 尝试直接 GET /
		start = time.Now()
		resp, err = client.Get(strings.TrimRight(ds.URL, "/") + "/")
		latencyMs = time.Since(start).Milliseconds()

		if err != nil {
			result.Success(c, gin.H{
				"status":    "error",
				"latencyMs": latencyMs,
				"error":     err.Error(),
			})
			return
		}
	}
	defer resp.Body.Close()

	result.Success(c, gin.H{
		"status":     "ok",
		"latencyMs":  latencyMs,
		"httpStatus": resp.StatusCode,
	})
}

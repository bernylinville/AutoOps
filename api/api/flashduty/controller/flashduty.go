package controller

import (
	"net/http"
	"strconv"

	"dodevops-api/api/flashduty/service"

	"github.com/gin-gonic/gin"
)

// FlashDutyController FlashDuty 控制器
type FlashDutyController struct {
	alertSvc    *service.AlertService
	incidentSvc *service.IncidentService
	scheduleSvc *service.ScheduleService
	insightSvc  *service.InsightService
}

// NewFlashDutyController 创建控制器
func NewFlashDutyController() *FlashDutyController {
	return &FlashDutyController{
		alertSvc:    service.NewAlertService(),
		incidentSvc: service.NewIncidentService(),
		scheduleSvc: service.NewScheduleService(),
		insightSvc:  service.NewInsightService(),
	}
}

// TestConnection 测试 FlashDuty 连接
func (ctrl *FlashDutyController) TestConnection(c *gin.Context) {
	client := service.GetClient()
	if !client.IsConfigured() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "FlashDuty 未配置 app_key，请在 config.yaml 中配置 flashduty.app_key"})
		return
	}
	if err := client.TestConnection(c.Request.Context()); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "FlashDuty 连接成功"})
}

// GetConfigStatus 获取 FlashDuty 配置状态
func (ctrl *FlashDutyController) GetConfigStatus(c *gin.Context) {
	client := service.GetClient()
	c.JSON(http.StatusOK, gin.H{
		"configured": client.IsConfigured(),
	})
}

// ========== 告警 ==========

// GetActiveAlerts 获取活跃告警列表
func (ctrl *FlashDutyController) GetActiveAlerts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	resp, err := ctrl.alertSvc.GetActiveAlerts(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Data)
}

// GetAlertsByHost 获取指定主机告警
func (ctrl *FlashDutyController) GetAlertsByHost(c *gin.Context) {
	ident := c.Param("ident")
	if ident == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ident 不能为空"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	resp, err := ctrl.alertSvc.GetAlertsByHost(c.Request.Context(), ident, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Data)
}

// GetAlertSummary 获取告警概况
func (ctrl *FlashDutyController) GetAlertSummary(c *gin.Context) {
	summary, err := ctrl.alertSvc.GetAlertSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 补充故障统计
	triggered, processing, err := ctrl.incidentSvc.GetIncidentSummary(c.Request.Context())
	if err == nil {
		summary.ActiveIncidents = triggered + processing
		summary.TriggeredCount = triggered
		summary.ProcessingCount = processing
	}

	c.JSON(http.StatusOK, summary)
}

// ========== 故障 ==========

// GetActiveIncidents 获取活跃故障列表
func (ctrl *FlashDutyController) GetActiveIncidents(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	resp, err := ctrl.incidentSvc.GetActiveIncidents(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Data)
}

// ClaimIncident 认领故障
func (ctrl *FlashDutyController) ClaimIncident(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "incident_id 不能为空"})
		return
	}
	if err := ctrl.incidentSvc.ClaimIncident(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "故障认领成功"})
}

// CloseIncident 关闭故障
func (ctrl *FlashDutyController) CloseIncident(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "incident_id 不能为空"})
		return
	}
	var body struct {
		Desc string `json:"desc"`
	}
	c.ShouldBindJSON(&body)
	if err := ctrl.incidentSvc.CloseIncident(c.Request.Context(), id, body.Desc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "故障关闭成功"})
}

// ========== 值班 ==========

// GetTodayOnCall 获取今日值班人
func (ctrl *FlashDutyController) GetTodayOnCall(c *gin.Context) {
	// team_ids 可选
	var teamIDs []int
	if teamIDStr := c.QueryArray("team_ids"); len(teamIDStr) > 0 {
		for _, s := range teamIDStr {
			if id, err := strconv.Atoi(s); err == nil {
				teamIDs = append(teamIDs, id)
			}
		}
	}

	onCallList, err := ctrl.scheduleSvc.GetTodayOnCall(c.Request.Context(), teamIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": onCallList})
}

// GetSchedules 获取值班表列表
func (ctrl *FlashDutyController) GetSchedules(c *gin.Context) {
	var teamIDs []int
	if teamIDStr := c.QueryArray("team_ids"); len(teamIDStr) > 0 {
		for _, s := range teamIDStr {
			if id, err := strconv.Atoi(s); err == nil {
				teamIDs = append(teamIDs, id)
			}
		}
	}

	resp, err := ctrl.scheduleSvc.GetSchedules(c.Request.Context(), teamIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp.Data)
}

// ========== 分析指标 ==========

// GetSREMetrics 获取 SRE 指标
func (ctrl *FlashDutyController) GetSREMetrics(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	metrics, err := ctrl.insightSvc.GetAccountMetrics(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// GetTrendData 获取趋势数据
func (ctrl *FlashDutyController) GetTrendData(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	metrics, err := ctrl.insightSvc.GetTrendData(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

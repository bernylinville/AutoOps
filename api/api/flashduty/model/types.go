package model

// ========== FlashDuty API 通用结构 ==========

// Response FlashDuty API 统一响应结构
type Response struct {
	Error *ErrorInfo   `json:"error,omitempty"`
	Data  interface{} `json:"data,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ========== 告警模型 ==========

// Alert FlashDuty 告警
type Alert struct {
	AlertID         string            `json:"alert_id"`
	ChannelID       int               `json:"channel_id"`
	ChannelName     string            `json:"channel_name"`
	IntegrationID   int               `json:"integration_id"`
	IntegrationName string            `json:"integration_name"`
	IntegrationType string            `json:"integration_type"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	AlertKey        string            `json:"alert_key"`
	AlertSeverity   string            `json:"alert_severity"` // Critical, Warning, Info
	AlertStatus     string            `json:"alert_status"`   // Critical, Warning, Info, Ok
	StartTime       int64             `json:"start_time"`
	LastTime        int64             `json:"last_time"`
	EndTime         int64             `json:"end_time,omitempty"`
	CreatedAt       int64             `json:"created_at"`
	UpdatedAt       int64             `json:"updated_at"`
	Labels          map[string]string `json:"labels"`
	Incident        *AlertIncident    `json:"incident,omitempty"`
	EventCnt        int               `json:"event_cnt"`
	EverMuted       bool              `json:"ever_muted"`
}

// AlertIncident 告警关联的故障
type AlertIncident struct {
	IncidentID string `json:"incident_id"`
	Title      string `json:"title"`
	Progress   string `json:"progress"` // Triggered, Processing, Closed
}

// AlertListRequest 告警列表请求
type AlertListRequest struct {
	P             int               `json:"p,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Asc           *bool             `json:"asc,omitempty"`
	Query         string            `json:"query,omitempty"`
	Title         string            `json:"title,omitempty"`
	AlertSeverity string            `json:"alert_severity,omitempty"`
	IsActive      *bool             `json:"is_active,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	ChannelIDs    []int             `json:"channel_ids,omitempty"`
	StartTime     int64             `json:"start_time"`
	EndTime       int64             `json:"end_time"`
}

// AlertListResponse 告警列表响应
type AlertListResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Items       []Alert `json:"items"`
		Total       int     `json:"total"`
		HasNextPage bool    `json:"has_next_page"`
	} `json:"data"`
}

// ========== 故障模型 ==========

// Incident FlashDuty 故障
type Incident struct {
	IncidentID      string            `json:"incident_id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	IncidentSeverity string           `json:"incident_severity"` // Critical, Warning, Info
	Progress        string            `json:"progress"`          // Triggered, Processing, Closed
	ChannelID       int               `json:"channel_id"`
	ChannelName     string            `json:"channel_name"`
	StartTime       int64             `json:"start_time"`
	LastTime        int64             `json:"last_time"`
	EndTime         int64             `json:"end_time,omitempty"`
	AckTime         int64             `json:"ack_time,omitempty"`
	CloseTime       int64             `json:"close_time,omitempty"`
	CreatedAt       int64             `json:"created_at"`
	UpdatedAt       int64             `json:"updated_at"`
	Labels          map[string]string `json:"labels"`
	AlertCnt        int               `json:"alert_cnt"`
	Assignees       []Assignee        `json:"assignees,omitempty"`
	Creator         string            `json:"creator,omitempty"`
}

// Assignee 故障负责人
type Assignee struct {
	PersonID   int    `json:"person_id"`
	PersonName string `json:"person_name"`
	Email      string `json:"email"`
}

// IncidentListRequest 故障列表请求
type IncidentListRequest struct {
	P             int      `json:"p,omitempty"`
	Limit         int      `json:"limit,omitempty"`
	Query         string   `json:"query,omitempty"`
	Progresses    []string `json:"progresses,omitempty"` // Triggered, Processing, Closed
	Severities    []string `json:"severities,omitempty"`
	ChannelIDs    []int    `json:"channel_ids,omitempty"`
	StartTime     int64    `json:"start_time"`
	EndTime       int64    `json:"end_time"`
}

// IncidentListResponse 故障列表响应
type IncidentListResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Items       []Incident `json:"items"`
		Total       int        `json:"total"`
		HasNextPage bool       `json:"has_next_page"`
	} `json:"data"`
}

// IncidentActionRequest 故障操作请求（认领/关闭等）
type IncidentActionRequest struct {
	IncidentID string `json:"incident_id"`
	Desc       string `json:"desc,omitempty"` // 备注
}

// ========== 值班模型 ==========

// Schedule 值班表
type Schedule struct {
	ScheduleID   int    `json:"schedule_id"`
	ScheduleName string `json:"schedule_name"`
	Desc         string `json:"desc"`
	GroupID      int    `json:"group_id"`
	Enabled      bool   `json:"endabled"` // 注意: FlashDuty API 拼写为 endabled
	CreateAt     int64  `json:"create_at"`
	UpdateAt     int64  `json:"update_at"`
}

// ScheduleListRequest 值班列表请求
type ScheduleListRequest struct {
	Query   string `json:"query,omitempty"`
	P       int    `json:"p,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	TeamIDs []int  `json:"team_ids"`
	Start   int64  `json:"start,omitempty"`
	End     int64  `json:"end,omitempty"`
}

// ScheduleListResponse 值班列表响应
type ScheduleListResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Items []Schedule `json:"items"`
		Total int        `json:"total"`
	} `json:"data"`
}

// SchedulePreviewRequest 值班预览请求
type SchedulePreviewRequest struct {
	ScheduleID int   `json:"schedule_id"`
	Start      int64 `json:"start"`
	End        int64 `json:"end"`
}

// ScheduleShift 班次信息
type ScheduleShift struct {
	Start      int64  `json:"start"`
	End        int64  `json:"end"`
	PersonID   int    `json:"person_id"`
	PersonName string `json:"person_name"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
}

// SchedulePreviewResponse 值班预览响应
type SchedulePreviewResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Shifts []ScheduleShift `json:"shifts"`
	} `json:"data"`
}

// ========== 分析看板模型 ==========

// InsightRequest 分析看板请求
type InsightRequest struct {
	StartTime     int64    `json:"start_time"`
	EndTime       int64    `json:"end_time"`
	TeamIDs       []int    `json:"team_ids,omitempty"`
	ChannelIDs    []int    `json:"channel_ids,omitempty"`
	Severities    []string `json:"severities,omitempty"`
	TimeZone      string   `json:"time_zone,omitempty"`
	Query         string   `json:"query"`
	Labels        map[string]string `json:"labels"`
	Fields        map[string]string `json:"fields"`
	AggregateUnit string   `json:"aggregate_unit,omitempty"` // day, week, month
}

// InsightMetrics 分析指标
type InsightMetrics struct {
	TotalIncidentCnt             int     `json:"total_incident_cnt"`
	TotalIncidentsAcknowledged   int     `json:"total_incidents_acknowledged"`
	TotalIncidentsClosed         int     `json:"total_incidents_closed"`
	TotalIncidentsAutoClosed     int     `json:"total_incidents_auto_closed"`
	TotalIncidentsManuallyClosed int     `json:"total_incidents_manually_closed"`
	TotalIncidentsEscalated      int     `json:"total_incidents_escalated"`
	TotalNotifications           int     `json:"total_notifications"`
	TotalInterruptions           int     `json:"total_interruptions"`
	MeanSecondsToAck             float64 `json:"mean_seconds_to_ack"`   // MTTA
	MeanSecondsToClose           float64 `json:"mean_seconds_to_close"` // MTTR
	NoiseReductionPct            float64 `json:"noise_reduction_pct"`
	AcknowledgementPct           float64 `json:"acknowlegement_pct"` // 注意: FlashDuty 拼写
	TotalAlertCnt                int     `json:"total_alert_cnt"`
	TotalAlertEventCnt           int     `json:"total_alert_event_cnt"`
	Ts                           *int64  `json:"ts,omitempty"`
	Hours                        string  `json:"hours,omitempty"`
	ChannelID                    *int    `json:"channel_id,omitempty"`
	ChannelName                  string  `json:"channel_name,omitempty"`
}

// InsightResponse 分析看板响应
type InsightResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Items []InsightMetrics `json:"items"`
	} `json:"data"`
}

// ========== 协作空间模型 ==========

// Channel 协作空间
type Channel struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Desc        string `json:"desc"`
	Enabled     bool   `json:"enabled"`
}

// ChannelListResponse 协作空间列表响应
type ChannelListResponse struct {
	Error *ErrorInfo `json:"error,omitempty"`
	Data  struct {
		Items []Channel `json:"items"`
		Total int       `json:"total"`
	} `json:"data"`
}

// ========== AutoOps 前端聚合模型 ==========

// DashboardAlertSummary 仪表盘告警概况
type DashboardAlertSummary struct {
	ActiveAlerts   int `json:"activeAlerts"`
	CriticalCount  int `json:"criticalCount"`
	WarningCount   int `json:"warningCount"`
	InfoCount      int `json:"infoCount"`
	ActiveIncidents int `json:"activeIncidents"`
	TriggeredCount int `json:"triggeredCount"`
	ProcessingCount int `json:"processingCount"`
}

// OnCallInfo 今日值班人信息
type OnCallInfo struct {
	ScheduleName string `json:"scheduleName"`
	PersonName   string `json:"personName"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	ShiftStart   int64  `json:"shiftStart"`
	ShiftEnd     int64  `json:"shiftEnd"`
}

// SREMetrics SRE 指标汇总
type SREMetrics struct {
	MTTA               float64          `json:"mtta"`              // 平均认领耗时(秒)
	MTTR               float64          `json:"mttr"`              // 平均关闭耗时(秒)
	NoiseReductionPct  float64          `json:"noiseReductionPct"` // 降噪率
	AckPct             float64          `json:"ackPct"`            // 响应率
	TotalIncidents     int              `json:"totalIncidents"`
	TotalAlerts        int              `json:"totalAlerts"`
	TotalAlertEvents   int              `json:"totalAlertEvents"`
	TrendData          []TrendDataPoint `json:"trendData,omitempty"`
}

// TrendDataPoint 趋势数据点
type TrendDataPoint struct {
	Timestamp      int64   `json:"timestamp"`
	IncidentCount  int     `json:"incidentCount"`
	AlertCount     int     `json:"alertCount"`
	MTTA           float64 `json:"mtta"`
	MTTR           float64 `json:"mttr"`
}

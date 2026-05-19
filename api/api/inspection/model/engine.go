// Package model provides engine data models for the inspection feature.
// Ported from inspection-tool/internal/model, adapted for AutoOps.
package model

import (
	"strings"
	"time"
)

// =============================================================================
// Alert Types
// =============================================================================

// AlertLevel represents the severity level of an alert.
type AlertLevel string

const (
	AlertLevelNormal   AlertLevel = "normal"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert represents a threshold violation alert for a host metric.
type Alert struct {
	Hostname          string            `json:"hostname"`
	MetricName        string            `json:"metric_name"`
	MetricDisplayName string            `json:"metric_display_name"`
	CurrentValue      float64           `json:"current_value"`
	FormattedValue    string            `json:"formatted_value"`
	WarningThreshold  float64           `json:"warning_threshold"`
	CriticalThreshold float64           `json:"critical_threshold"`
	Level             AlertLevel        `json:"level"`
	Message           string            `json:"message"`
	Labels            map[string]string `json:"labels,omitempty"`
}

// NewAlert creates a new Alert.
func NewAlert(hostname, metricName string, currentValue float64, level AlertLevel) *Alert {
	return &Alert{
		Hostname:     hostname,
		MetricName:   metricName,
		CurrentValue: currentValue,
		Level:        level,
	}
}

// IsWarning returns true if warning level.
func (a *Alert) IsWarning() bool { return a.Level == AlertLevelWarning }

// IsCritical returns true if critical level.
func (a *Alert) IsCritical() bool { return a.Level == AlertLevelCritical }

// AlertSummary provides aggregated alert statistics.
type AlertSummary struct {
	TotalAlerts   int `json:"total_alerts"`
	WarningCount  int `json:"warning_count"`
	CriticalCount int `json:"critical_count"`
}

// NewAlertSummary creates AlertSummary from alerts.
func NewAlertSummary(alerts []*Alert) *AlertSummary {
	summary := &AlertSummary{}
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		summary.TotalAlerts++
		switch alert.Level {
		case AlertLevelWarning:
			summary.WarningCount++
		case AlertLevelCritical:
			summary.CriticalCount++
		}
	}
	return summary
}

// =============================================================================
// Host Types
// =============================================================================

// HostStatus represents the overall health status of a host.
type HostStatus string

const (
	HostStatusNormal   HostStatus = "normal"
	HostStatusWarning  HostStatus = "warning"
	HostStatusCritical HostStatus = "critical"
	HostStatusFailed   HostStatus = "failed"
)

// DiskMountInfo represents disk usage for a single mount point.
type DiskMountInfo struct {
	Path        string  `json:"path"`
	Total       int64   `json:"total"`
	Free        int64   `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// HostMeta contains basic metadata about a host from N9E API.
type HostMeta struct {
	Ident         string          `json:"ident"`
	Hostname      string          `json:"hostname"`
	IP            string          `json:"ip"`
	OS            string          `json:"os"`
	OSVersion     string          `json:"os_version"`
	KernelVersion string          `json:"kernel_version"`
	CPUCores      int             `json:"cpu_cores"`
	CPUModel      string          `json:"cpu_model"`
	MemoryTotal   int64           `json:"memory_total"`
	DiskMounts    []DiskMountInfo `json:"disk_mounts"`
}

// CleanIdent extracts hostname from ident (handles "hostname@IP" format).
func CleanIdent(ident string) string {
	if idx := strings.Index(ident, "@"); idx > 0 {
		return ident[:idx]
	}
	return ident
}

// HostResult represents inspection result for a single host.
type HostResult struct {
	Hostname      string                  `json:"hostname"`
	IP            string                  `json:"ip"`
	OS            string                  `json:"os"`
	OSVersion     string                  `json:"os_version"`
	KernelVersion string                  `json:"kernel_version"`
	CPUCores      int                     `json:"cpu_cores"`
	CPUModel      string                  `json:"cpu_model"`
	MemoryTotal   int64                   `json:"memory_total"`
	BootTime      string                  `json:"boot_time"`
	Status        HostStatus              `json:"status"`
	Metrics       map[string]*MetricValue `json:"metrics"`
	Alerts        []*Alert                `json:"alerts,omitempty"`
	CollectedAt   time.Time               `json:"collected_at"`
	Error         string                  `json:"error,omitempty"`
}

// NewHostResult creates HostResult from HostMeta.
func NewHostResult(meta *HostMeta) *HostResult {
	if meta == nil {
		return &HostResult{
			Status:  HostStatusFailed,
			Metrics: make(map[string]*MetricValue),
		}
	}
	return &HostResult{
		Hostname:      meta.Hostname,
		IP:            meta.IP,
		OS:            meta.OS,
		OSVersion:     meta.OSVersion,
		KernelVersion: meta.KernelVersion,
		CPUCores:      meta.CPUCores,
		CPUModel:      meta.CPUModel,
		MemoryTotal:   meta.MemoryTotal,
		Status:        HostStatusNormal,
		Metrics:       make(map[string]*MetricValue),
		Alerts:        make([]*Alert, 0),
	}
}

// SetMetric adds or updates a metric.
func (r *HostResult) SetMetric(value *MetricValue) {
	if r.Metrics == nil {
		r.Metrics = make(map[string]*MetricValue)
	}
	if value != nil {
		r.Metrics[value.Name] = value
	}
}

// GetMetric retrieves metric by name.
func (r *HostResult) GetMetric(name string) *MetricValue {
	if r.Metrics == nil {
		return nil
	}
	return r.Metrics[name]
}

// AddAlert adds alert and updates status to most severe level.
func (r *HostResult) AddAlert(alert *Alert) {
	if alert == nil {
		return
	}
	r.Alerts = append(r.Alerts, alert)
	if alert.Level == AlertLevelCritical {
		r.Status = HostStatusCritical
	} else if alert.Level == AlertLevelWarning && r.Status != HostStatusCritical {
		r.Status = HostStatusWarning
	}
}

// =============================================================================
// Inspection Result Types
// =============================================================================

// InspectionSummary provides aggregated statistics.
type InspectionSummary struct {
	TotalHosts    int `json:"total_hosts"`
	NormalHosts   int `json:"normal_hosts"`
	WarningHosts  int `json:"warning_hosts"`
	CriticalHosts int `json:"critical_hosts"`
	FailedHosts   int `json:"failed_hosts"`
}

// NewInspectionSummary creates summary from host results.
func NewInspectionSummary(hosts []*HostResult) *InspectionSummary {
	summary := &InspectionSummary{}
	for _, host := range hosts {
		if host == nil {
			continue
		}
		summary.TotalHosts++
		switch host.Status {
		case HostStatusNormal:
			summary.NormalHosts++
		case HostStatusWarning:
			summary.WarningHosts++
		case HostStatusCritical:
			summary.CriticalHosts++
		case HostStatusFailed:
			summary.FailedHosts++
		}
	}
	return summary
}

// InspectionResult represents the complete inspection result.
type InspectionResult struct {
	InspectionTime time.Time          `json:"inspection_time"`
	Duration       time.Duration      `json:"duration"`
	Summary        *InspectionSummary `json:"summary"`
	Hosts          []*HostResult      `json:"hosts"`
	Alerts         []*Alert           `json:"alerts"`
	AlertSummary   *AlertSummary      `json:"alert_summary"`
	Version        string             `json:"version,omitempty"`
}

// NewInspectionResult creates new result.
func NewInspectionResult(inspectionTime time.Time) *InspectionResult {
	return &InspectionResult{
		InspectionTime: inspectionTime,
		Hosts:          make([]*HostResult, 0),
		Alerts:         make([]*Alert, 0),
	}
}

// AddHost adds host result and collects its alerts.
func (r *InspectionResult) AddHost(host *HostResult) {
	if host == nil {
		return
	}
	r.Hosts = append(r.Hosts, host)
	r.Alerts = append(r.Alerts, host.Alerts...)
}

// Finalize calculates summaries.
func (r *InspectionResult) Finalize(endTime time.Time) {
	r.Duration = endTime.Sub(r.InspectionTime)
	r.Summary = NewInspectionSummary(r.Hosts)
	r.AlertSummary = NewAlertSummary(r.Alerts)
}

// =============================================================================
// Metric Types
// =============================================================================

// MetricStatus represents evaluation status of a metric value.
type MetricStatus string

const (
	MetricStatusNormal   MetricStatus = "normal"
	MetricStatusWarning  MetricStatus = "warning"
	MetricStatusCritical MetricStatus = "critical"
	MetricStatusPending  MetricStatus = "pending"
)

// MetricCategory represents metric category.
type MetricCategory string

const (
	MetricCategoryCPU     MetricCategory = "cpu"
	MetricCategoryMemory  MetricCategory = "memory"
	MetricCategoryDisk    MetricCategory = "disk"
	MetricCategorySystem  MetricCategory = "system"
	MetricCategoryProcess MetricCategory = "process"
)

// MetricFormat represents display formatting.
type MetricFormat string

const (
	MetricFormatPercent   MetricFormat = "percent"
	MetricFormatSize      MetricFormat = "size"
	MetricFormatDuration  MetricFormat = "duration"
	MetricFormatNumber    MetricFormat = "number"
	MetricFormatNTPOffset MetricFormat = "ntp_offset"
)

// AggregateType represents aggregation across multiple values.
type AggregateType string

const (
	AggregateMax AggregateType = "max"
	AggregateMin AggregateType = "min"
	AggregateAvg AggregateType = "avg"
)

// MetricDefinition defines metadata for a metric.
type MetricDefinition struct {
	Name          string         `yaml:"name" json:"name"`
	DisplayName   string         `yaml:"display_name" json:"display_name"`
	Query         string         `yaml:"query" json:"query"`
	Unit          string         `yaml:"unit" json:"unit"`
	Category      MetricCategory `yaml:"category" json:"category"`
	Format        MetricFormat   `yaml:"format,omitempty" json:"format,omitempty"`
	Aggregate     AggregateType  `yaml:"aggregate,omitempty" json:"aggregate,omitempty"`
	ExpandByLabel string         `yaml:"expand_by_label,omitempty" json:"expand_by_label,omitempty"`
	Status        string         `yaml:"status,omitempty" json:"status,omitempty"`
	Note          string         `yaml:"note,omitempty" json:"note,omitempty"`
}

// IsPending returns true if metric is not yet implemented.
func (d *MetricDefinition) IsPending() bool {
	return d.Status == "pending" || d.Query == ""
}

// HasExpandLabel returns true if metric expands by label (e.g., disk by path).
func (d *MetricDefinition) HasExpandLabel() bool {
	return d.ExpandByLabel != ""
}

// MetricValue represents a collected metric value.
type MetricValue struct {
	Name           string            `json:"name"`
	RawValue       float64           `json:"raw_value"`
	FormattedValue string            `json:"formatted_value"`
	Status         MetricStatus      `json:"status"`
	Labels         map[string]string `json:"labels,omitempty"`
	IsNA           bool              `json:"is_na"`
	Timestamp      int64             `json:"timestamp,omitempty"`
}

// NewNAMetricValue creates N/A MetricValue for pending metrics.
func NewNAMetricValue(name string) *MetricValue {
	return &MetricValue{
		Name:           name,
		RawValue:       0,
		FormattedValue: "N/A",
		Status:         MetricStatusPending,
		IsNA:           true,
	}
}

// NewMetricValue creates MetricValue with raw value.
func NewMetricValue(name string, rawValue float64) *MetricValue {
	return &MetricValue{
		Name:     name,
		RawValue: rawValue,
		Status:   MetricStatusNormal,
		IsNA:     false,
	}
}

// HostMetrics contains all metrics for a single host.
type HostMetrics struct {
	Hostname string                  `json:"hostname"`
	Metrics  map[string]*MetricValue `json:"metrics"`
}

// NewHostMetrics creates new HostMetrics.
func NewHostMetrics(hostname string) *HostMetrics {
	return &HostMetrics{
		Hostname: hostname,
		Metrics:  make(map[string]*MetricValue),
	}
}

// SetMetric adds or updates metric.
func (h *HostMetrics) SetMetric(value *MetricValue) {
	if h.Metrics == nil {
		h.Metrics = make(map[string]*MetricValue)
	}
	h.Metrics[value.Name] = value
}

// GetMetric retrieves metric by name.
func (h *HostMetrics) GetMetric(name string) *MetricValue {
	if h.Metrics == nil {
		return nil
	}
	return h.Metrics[name]
}

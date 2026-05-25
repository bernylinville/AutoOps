// Package engine provides the inspection engine core.
// Ported from inspection-tool/internal/service/evaluator.go.
package engine

import (
	"fmt"
	"math"
	"strings"

	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/log"
)

// metricThresholdMap maps metric names to their threshold config fields.
var metricThresholdMap = map[string]string{
	"cpu_usage":         "cpu_usage",
	"memory_usage":      "memory_usage",
	"disk_usage_max":    "disk_usage",
	"processes_zombies": "zombie_processes",
	"load_per_core":     "load_per_core",
	"ntp_offset":        "ntp_offset",
}

// Evaluator performs threshold evaluation on collected metrics.
type Evaluator struct {
	thresholds *ThresholdsConfig
	metricDefs map[string]*model.MetricDefinition
}

// NewEvaluator creates a new Evaluator with the given threshold configuration.
func NewEvaluator(thresholds *ThresholdsConfig, metrics []*model.MetricDefinition) *Evaluator {
	metricDefs := make(map[string]*model.MetricDefinition)
	for _, m := range metrics {
		metricDefs[m.Name] = m
	}

	if thresholds == nil {
		thresholds = DefaultThresholds()
	}

	return &Evaluator{
		thresholds: thresholds,
		metricDefs: metricDefs,
	}
}

// EvaluateAll evaluates all hosts and returns the complete evaluation result.
func (e *Evaluator) EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult {
	result := &EvaluationResult{
		HostResults: make([]*HostEvaluationResult, 0, len(hostMetrics)),
		Alerts:      make([]*model.Alert, 0),
	}

	for hostname, metrics := range hostMetrics {
		hostResult := e.EvaluateHost(hostname, metrics)
		result.HostResults = append(result.HostResults, hostResult)
		result.Alerts = append(result.Alerts, hostResult.Alerts...)
	}

	result.Summary = model.NewAlertSummary(result.Alerts)

	log.Log().Infof("[Evaluator] evaluation completed: hosts=%d alerts=%d warnings=%d criticals=%d",
		len(result.HostResults), result.Summary.TotalAlerts,
		result.Summary.WarningCount, result.Summary.CriticalCount)

	return result
}

// EvaluateHost evaluates all metrics for a single host.
func (e *Evaluator) EvaluateHost(hostname string, hostMetrics *model.HostMetrics) *HostEvaluationResult {
	result := &HostEvaluationResult{
		Hostname: hostname,
		Status:   model.HostStatusNormal,
		Metrics:  make(map[string]*model.MetricValue),
		Alerts:   make([]*model.Alert, 0),
	}

	if hostMetrics == nil || hostMetrics.Metrics == nil {
		log.Log().Warnf("[Evaluator] no metrics for host %s", hostname)
		return result
	}

	result.Metrics = hostMetrics.Metrics

	for metricName, metricValue := range hostMetrics.Metrics {
		if metricValue == nil {
			continue
		}

		alert := e.evaluateMetric(hostname, metricName, metricValue)
		if alert != nil {
			result.Alerts = append(result.Alerts, alert)
		}
	}

	result.Status = e.determineHostStatus(result.Alerts)
	return result
}

// evaluateMetric evaluates a single metric and returns an Alert if threshold exceeded.
func (e *Evaluator) evaluateMetric(hostname, metricName string, value *model.MetricValue) *model.Alert {
	if value.IsNA {
		value.Status = model.MetricStatusPending
		return nil
	}

	value.FormattedValue = e.formatMetricValue(metricName, value.RawValue)

	// Expanded metrics (e.g., disk_usage:/home) are display-only — don't alert.
	if strings.Contains(metricName, ":") {
		baseName := strings.Split(metricName, ":")[0]
		threshold := e.getThreshold(baseName + "_max")
		if threshold == nil {
			threshold = e.getThreshold(baseName)
		}
		if threshold != nil {
			e.setMetricStatus(value, threshold)
		}
		return nil
	}

	threshold := e.getThreshold(metricName)
	if threshold == nil {
		value.Status = model.MetricStatusNormal
		return nil
	}

	// NTP special handling: stratum=0 means not synchronized.
	if metricName == "ntp_offset" {
		if stratum, ok := value.Labels["stratum"]; ok && stratum == "0" {
			value.Status = model.MetricStatusCritical
			value.FormattedValue = "N/A (未同步)"
			return &model.Alert{
				Hostname:          hostname,
				MetricName:        metricName,
				MetricDisplayName: e.getMetricDisplayName(metricName),
				CurrentValue:      value.RawValue,
				FormattedValue:    value.FormattedValue,
				WarningThreshold:  threshold.Warning,
				CriticalThreshold: threshold.Critical,
				Level:             model.AlertLevelCritical,
				Message:           "NTP 时间未同步 (stratum=0)",
				Labels:            value.Labels,
			}
		}
	}

	// NTP offset uses absolute value for evaluation.
	evalValue := value.RawValue
	if metricName == "ntp_offset" {
		evalValue = math.Abs(value.RawValue)
	}

	level := e.evaluateThreshold(evalValue, threshold)
	e.setMetricStatusByValue(value, evalValue, threshold)

	if level == model.AlertLevelNormal {
		return nil
	}

	return &model.Alert{
		Hostname:          hostname,
		MetricName:        metricName,
		MetricDisplayName: e.getMetricDisplayName(metricName),
		CurrentValue:      value.RawValue,
		FormattedValue:    value.FormattedValue,
		WarningThreshold:  threshold.Warning,
		CriticalThreshold: threshold.Critical,
		Level:             level,
		Message:           e.buildAlertMessage(metricName, value.RawValue, level, threshold),
		Labels:            value.Labels,
	}
}

func (e *Evaluator) setMetricStatus(value *model.MetricValue, threshold *ThresholdPair) {
	e.setMetricStatusByValue(value, value.RawValue, threshold)
}

func (e *Evaluator) setMetricStatusByValue(value *model.MetricValue, evalValue float64, threshold *ThresholdPair) {
	if evalValue >= threshold.Critical {
		value.Status = model.MetricStatusCritical
	} else if evalValue >= threshold.Warning {
		value.Status = model.MetricStatusWarning
	} else {
		value.Status = model.MetricStatusNormal
	}
}

func (e *Evaluator) evaluateThreshold(value float64, threshold *ThresholdPair) model.AlertLevel {
	if value >= threshold.Critical {
		return model.AlertLevelCritical
	}
	if value >= threshold.Warning {
		return model.AlertLevelWarning
	}
	return model.AlertLevelNormal
}

func (e *Evaluator) getThreshold(metricName string) *ThresholdPair {
	if e.thresholds == nil {
		return nil
	}

	thresholdKey, ok := metricThresholdMap[metricName]
	if !ok {
		return nil
	}

	switch thresholdKey {
	case "cpu_usage":
		return &e.thresholds.CPUUsage
	case "memory_usage":
		return &e.thresholds.MemoryUsage
	case "disk_usage":
		return &e.thresholds.DiskUsage
	case "zombie_processes":
		return &e.thresholds.ZombieProcesses
	case "load_per_core":
		return &e.thresholds.LoadPerCore
	case "ntp_offset":
		return &e.thresholds.NTPOffset
	default:
		return nil
	}
}

func (e *Evaluator) getMetricDisplayName(metricName string) string {
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	if def, ok := e.metricDefs[baseName]; ok {
		return def.DisplayName
	}
	return metricName
}

func (e *Evaluator) buildAlertMessage(metricName string, value float64, level model.AlertLevel, threshold *ThresholdPair) string {
	displayName := e.getMetricDisplayName(metricName)
	levelStr := "警告"
	thresholdValue := threshold.Warning
	if level == model.AlertLevelCritical {
		levelStr = "严重"
		thresholdValue = threshold.Critical
	}

	unit := e.getMetricUnit(metricName)
	if unit == "%" {
		return fmt.Sprintf("%s %s: %.1f%% (阈值: %.1f%%)", displayName, levelStr, value, thresholdValue)
	}
	return fmt.Sprintf("%s %s: %.2f (阈值: %.2f)", displayName, levelStr, value, thresholdValue)
}

func (e *Evaluator) getMetricUnit(metricName string) string {
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	if def, ok := e.metricDefs[baseName]; ok {
		return def.Unit
	}
	return ""
}

func (e *Evaluator) determineHostStatus(alerts []*model.Alert) model.HostStatus {
	if len(alerts) == 0 {
		return model.HostStatusNormal
	}

	hasCritical := false
	hasWarning := false

	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		switch alert.Level {
		case model.AlertLevelCritical:
			hasCritical = true
		case model.AlertLevelWarning:
			hasWarning = true
		}
	}

	if hasCritical {
		return model.HostStatusCritical
	}
	if hasWarning {
		return model.HostStatusWarning
	}
	return model.HostStatusNormal
}

func (e *Evaluator) formatMetricValue(metricName string, value float64) string {
	baseName := metricName
	if idx := strings.Index(metricName, ":"); idx > 0 {
		baseName = metricName[:idx]
	}
	if strings.HasSuffix(baseName, "_max") {
		baseName = strings.TrimSuffix(baseName, "_max")
	}

	def, ok := e.metricDefs[baseName]
	if !ok {
		return fmt.Sprintf("%.2f", value)
	}

	switch def.Format {
	case model.MetricFormatPercent:
		return fmt.Sprintf("%.1f%%", value)
	case model.MetricFormatSize:
		return formatBytes(int64(value))
	case model.MetricFormatDuration:
		return formatUptime(value)
	case model.MetricFormatNumber:
		if value == float64(int64(value)) {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.2f", value)
	case model.MetricFormatNTPOffset:
		return formatNTPOffset(value)
	case model.MetricFormatBoolean:
		if value == 1 {
			return "是"
		}
		return "否"
	default:
		if def.Unit == "%" {
			return fmt.Sprintf("%.1f%%", value)
		}
		if def.Unit == "个" || def.Unit == "core" {
			return fmt.Sprintf("%.0f", value)
		}
		return fmt.Sprintf("%.2f", value)
	}
}

// formatBytes formats bytes to human-readable size.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatUptime formats seconds to human-readable uptime.
func formatUptime(seconds float64) string {
	days := int(seconds / 86400)
	hours := int((seconds - float64(days*86400)) / 3600)
	minutes := int((seconds - float64(days*86400) - float64(hours*3600)) / 60)

	if days > 0 {
		return fmt.Sprintf("%d天%d时%d分", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%d时%d分", hours, minutes)
	}
	return fmt.Sprintf("%d分钟", minutes)
}

func formatNTPOffset(seconds float64) string {
	msValue := seconds * 1000
	if msValue >= 0 {
		return fmt.Sprintf("+%.1fms", msValue)
	}
	return fmt.Sprintf("%.1fms", msValue)
}

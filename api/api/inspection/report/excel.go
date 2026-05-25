// Package report provides Excel report generation for inspection results.
// Host report semantics are aligned with inspection-tool/internal/report/excel/writer.go.
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"dodevops-api/api/inspection/model"

	"github.com/xuri/excelize/v2"
)

const (
	// Sheet names from inspection-tool.
	sheetSummary       = "巡检概览"
	sheetBaselineCheck = "基线检查"
	sheetDetail        = "详细数据"
	sheetAlerts        = "异常汇总"
	defaultSheet       = "Sheet1"

	colorWarningBg  = "FFEB9C"
	colorWarningFg  = "9C6500"
	colorCriticalBg = "FFC7CE"
	colorCriticalFg = "9C0006"
	colorHeaderBg   = "4472C4"
	colorHeaderFg   = "FFFFFF"
	colorNormalBg   = "C6EFCE"
	colorNormalFg   = "006100"
)

var sysctlParamNames = []string{
	"net.ipv4.ip_local_port_range_min",
	"net.ipv4.ip_local_port_range_max",
	"net.netfilter.nf_conntrack_max",
	"net.ipv4.tcp_max_tw_buckets",
	"net.netfilter.nf_conntrack_tcp_timeout_fin_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_time_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_close_wait",
	"net.netfilter.nf_conntrack_tcp_timeout_established",
	"net.ipv4.tcp_tw_reuse",
	"net.ipv4.tcp_timestamps",
}

var sysctlDisplayNames = []string{
	"端口范围(最小)",
	"端口范围(最大)",
	"连接跟踪最大",
	"TIME_WAIT桶数",
	"FIN_WAIT超时",
	"TIME_WAIT超时",
	"CLOSE_WAIT超时",
	"ESTABLISHED超时",
	"tcp_tw_reuse",
	"tcp_timestamps",
}

type hostExcelWriter struct {
	timezone *time.Location
}

func newHostExcelWriter(timezone *time.Location) *hostExcelWriter {
	if timezone == nil {
		var err error
		timezone, err = time.LoadLocation("Asia/Shanghai")
		if err != nil {
			timezone = time.UTC
		}
	}
	return &hostExcelWriter{timezone: timezone}
}

// WriteHostReport generates a Host inspection Excel report.
func WriteHostReport(result *model.InspectionResult, outputPath string) error {
	return newHostExcelWriter(nil).Write(result, outputPath)
}

func (w *hostExcelWriter) Write(result *model.InspectionResult, outputPath string) error {
	if result == nil {
		return fmt.Errorf("inspection result is nil")
	}
	if !strings.HasSuffix(strings.ToLower(outputPath), ".xlsx") {
		outputPath += ".xlsx"
	}

	f := excelize.NewFile()
	defer f.Close()

	if err := w.createSummarySheet(f, result); err != nil {
		return fmt.Errorf("failed to create summary sheet: %w", err)
	}
	if err := w.createBaselineCheckSheet(f, result); err != nil {
		return fmt.Errorf("failed to create baseline check sheet: %w", err)
	}
	if err := w.createDetailSheet(f, result); err != nil {
		return fmt.Errorf("failed to create detail sheet: %w", err)
	}

	alerts := collectReportAlerts(result)
	if len(alerts) > 0 {
		if err := w.createUnifiedHostAlertsSheet(f, alerts); err != nil {
			return fmt.Errorf("failed to create alerts sheet: %w", err)
		}
	}

	_ = f.DeleteSheet(defaultSheet)
	idx, _ := f.GetSheetIndex(sheetSummary)
	if idx >= 0 {
		f.SetActiveSheet(idx)
	}

	if err := ensureDir(filepath.Dir(outputPath)); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save excel: %w", err)
	}
	return nil
}

func (w *hostExcelWriter) createSummarySheet(f *excelize.File, result *model.InspectionResult) error {
	idx, err := f.NewSheet(sheetSummary)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetSummary, err)
	}
	f.SetActiveSheet(idx)

	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 18},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}
	valueStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	_ = f.SetColWidth(sheetSummary, "A", "A", 20)
	_ = f.SetColWidth(sheetSummary, "B", "B", 30)
	_ = f.MergeCell(sheetSummary, "A1", "B1")
	_ = f.SetCellValue(sheetSummary, "A1", "系统巡检报告")
	_ = f.SetCellStyle(sheetSummary, "A1", "B1", titleStyle)
	_ = f.SetRowHeight(sheetSummary, 1, 30)

	summary := getSummary(result)
	alertSummary := getAlertSummary(result)
	summaryData := []struct {
		label string
		value interface{}
	}{
		{"巡检时间", result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")},
		{"巡检耗时", formatDuration(result.Duration)},
		{"主机总数", summary.TotalHosts},
		{"正常主机", summary.NormalHosts},
		{"警告主机", summary.WarningHosts},
		{"严重主机", summary.CriticalHosts},
		{"失败主机", summary.FailedHosts},
		{"告警总数", alertSummary.TotalAlerts},
		{"警告告警", alertSummary.WarningCount},
		{"严重告警", alertSummary.CriticalCount},
	}
	if result.Version != "" {
		summaryData = append(summaryData, struct {
			label string
			value interface{}
		}{"工具版本", result.Version})
	}

	for i, item := range summaryData {
		row := i + 3
		labelCell := fmt.Sprintf("A%d", row)
		valueCell := fmt.Sprintf("B%d", row)
		_ = f.SetCellValue(sheetSummary, labelCell, item.label)
		_ = f.SetCellValue(sheetSummary, valueCell, item.value)
		_ = f.SetCellStyle(sheetSummary, labelCell, labelCell, headerStyle)
		_ = f.SetCellStyle(sheetSummary, valueCell, valueCell, valueStyle)
		_ = f.SetRowHeight(sheetSummary, row, 22)
	}
	return nil
}

func (w *hostExcelWriter) createBaselineCheckSheet(f *excelize.File, result *model.InspectionResult) error {
	_, err := f.NewSheet(sheetBaselineCheck)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetBaselineCheck, err)
	}

	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}
	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}
	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}
	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	headers := []string{
		"巡检时间", "主机名", "IP地址", "操作系统", "内核版本", "运行时间",
		"密码过期", "密码策略", "文件句柄", "公网访问",
	}
	headers = append(headers, sysctlDisplayNames...)
	colWidths := []float64{20, 20, 15, 25, 30, 15, 30, 40, 15, 10, 12, 12, 14, 14, 12, 12, 12, 12, 12, 12}
	for i, width := range colWidths {
		col := columnName(i + 1)
		_ = f.SetColWidth(sheetBaselineCheck, col, col, width)
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		_ = f.SetCellValue(sheetBaselineCheck, cell, header)
		_ = f.SetCellStyle(sheetBaselineCheck, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheetBaselineCheck, 1, 25)
	_ = f.SetPanes(sheetBaselineCheck, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	inspectionTimeStr := result.InspectionTime.In(w.timezone).Format("2006-01-02 15:04:05")
	for i, host := range result.Hosts {
		if host == nil {
			continue
		}
		rowStr := fmt.Sprintf("%d", i+2)
		_ = f.SetCellValue(sheetBaselineCheck, "A"+rowStr, inspectionTimeStr)
		_ = f.SetCellValue(sheetBaselineCheck, "B"+rowStr, host.Hostname)
		_ = f.SetCellValue(sheetBaselineCheck, "C"+rowStr, host.IP)
		_ = f.SetCellValue(sheetBaselineCheck, "D"+rowStr, fmt.Sprintf("%s %s", host.OS, host.OSVersion))
		_ = f.SetCellValue(sheetBaselineCheck, "E"+rowStr, host.KernelVersion)
		w.setMetricCell(f, sheetBaselineCheck, "F"+rowStr, host.Metrics["uptime"], 0, 0, 0)
		w.setExpandedMetricCell(f, sheetBaselineCheck, "G"+rowStr, "password_expiry", host.Metrics)
		w.setExpandedMetricCell(f, sheetBaselineCheck, "H"+rowStr, "password_policy", host.Metrics)
		w.setFileHandleCell(f, sheetBaselineCheck, "I"+rowStr, host.Metrics["open_files"], host.Metrics["max_files"], warningStyle, criticalStyle)
		w.setPublicNetworkCell(f, sheetBaselineCheck, "J"+rowStr, host.Metrics["public_network"], normalStyle, criticalStyle)
		for j, paramName := range sysctlParamNames {
			col := columnName(11 + j)
			metricName := fmt.Sprintf("sysctl_params:%s", paramName)
			w.setSysctlCell(f, sheetBaselineCheck, col+rowStr, host.Metrics[metricName])
		}
	}
	return nil
}

func (w *hostExcelWriter) createDetailSheet(f *excelize.File, result *model.InspectionResult) error {
	_, err := f.NewSheet(sheetDetail)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetDetail, err)
	}

	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}
	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}
	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}
	normalStyle, err := w.createNormalStyle(f)
	if err != nil {
		return err
	}

	headers := []string{
		"主机名", "IP地址", "状态", "内核版本",
		"CPU利用率", "内存利用率", "内存空闲", "磁盘最大利用率",
		"运行时间", "NTP时间偏差", "僵尸进程", "打开句柄数", "句柄最大值",
	}
	diskPaths := w.collectDiskPaths(result.Hosts)
	for _, path := range diskPaths {
		headers = append(headers, fmt.Sprintf("磁盘:%s", path))
	}

	colWidths := map[string]float64{"A": 20, "B": 15, "C": 10, "D": 25, "E": 12, "F": 12, "G": 12, "H": 14, "I": 15, "J": 14, "K": 10, "L": 12, "M": 12}
	for col, width := range colWidths {
		_ = f.SetColWidth(sheetDetail, col, col, width)
	}
	for i := range diskPaths {
		col := columnName(14 + i)
		_ = f.SetColWidth(sheetDetail, col, col, 15)
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		_ = f.SetCellValue(sheetDetail, cell, header)
		_ = f.SetCellStyle(sheetDetail, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheetDetail, 1, 25)
	_ = f.SetPanes(sheetDetail, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	for i, host := range result.Hosts {
		if host == nil {
			continue
		}
		rowStr := fmt.Sprintf("%d", i+2)
		_ = f.SetCellValue(sheetDetail, "A"+rowStr, host.Hostname)
		_ = f.SetCellValue(sheetDetail, "B"+rowStr, host.IP)
		_ = f.SetCellValue(sheetDetail, "C"+rowStr, statusText(host.Status))
		_ = f.SetCellValue(sheetDetail, "D"+rowStr, host.KernelVersion)
		w.setMetricCell(f, sheetDetail, "E"+rowStr, host.Metrics["cpu_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "F"+rowStr, host.Metrics["memory_usage"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "G"+rowStr, host.Metrics["memory_available"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "H"+rowStr, host.Metrics["disk_usage_max"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "I"+rowStr, host.Metrics["uptime"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "J"+rowStr, host.Metrics["ntp_offset"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "K"+rowStr, host.Metrics["processes_zombies"], warningStyle, criticalStyle, normalStyle)
		w.setMetricCell(f, sheetDetail, "L"+rowStr, host.Metrics["open_files"], 0, 0, 0)
		w.setMetricCell(f, sheetDetail, "M"+rowStr, host.Metrics["max_files"], 0, 0, 0)
		for j, path := range diskPaths {
			col := columnName(14 + j)
			metricName := fmt.Sprintf("disk_usage:%s", path)
			w.setMetricCell(f, sheetDetail, col+rowStr, host.Metrics[metricName], warningStyle, criticalStyle, normalStyle)
		}
		if style := w.getStatusStyle(host.Status, normalStyle, warningStyle, criticalStyle); style > 0 {
			_ = f.SetCellStyle(sheetDetail, "C"+rowStr, "C"+rowStr, style)
		}
	}
	return nil
}

func (w *hostExcelWriter) createUnifiedHostAlertsSheet(f *excelize.File, alerts []*model.Alert) error {
	_, err := f.NewSheet(sheetAlerts)
	if err != nil {
		return fmt.Errorf("create sheet %s: %w", sheetAlerts, err)
	}
	headerStyle, err := w.createHeaderStyle(f)
	if err != nil {
		return err
	}
	warningStyle, err := w.createWarningStyle(f)
	if err != nil {
		return err
	}
	criticalStyle, err := w.createCriticalStyle(f)
	if err != nil {
		return err
	}
	sourceStyle, err := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D6E3F8"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return err
	}

	headers := []string{"来源类型", "实例标识", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}
	colWidths := []float64{12, 25, 10, 18, 15, 12, 12, 45}
	for i, width := range colWidths {
		col := columnName(i + 1)
		_ = f.SetColWidth(sheetAlerts, col, col, width)
	}
	for i, header := range headers {
		cell := fmt.Sprintf("%s1", columnName(i+1))
		_ = f.SetCellValue(sheetAlerts, cell, header)
		_ = f.SetCellStyle(sheetAlerts, cell, cell, headerStyle)
	}
	_ = f.SetRowHeight(sheetAlerts, 1, 25)
	_ = f.SetPanes(sheetAlerts, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})

	sortedAlerts := make([]*model.Alert, 0, len(alerts))
	for _, alert := range alerts {
		if alert != nil {
			sortedAlerts = append(sortedAlerts, alert)
		}
	}
	sort.Slice(sortedAlerts, func(i, j int) bool {
		if sortedAlerts[i].Level != sortedAlerts[j].Level {
			return alertLevelPriority(sortedAlerts[i].Level) > alertLevelPriority(sortedAlerts[j].Level)
		}
		return sortedAlerts[i].Hostname < sortedAlerts[j].Hostname
	})

	for i, alert := range sortedAlerts {
		rowStr := fmt.Sprintf("%d", i+2)
		_ = f.SetCellValue(sheetAlerts, "A"+rowStr, "Host")
		_ = f.SetCellValue(sheetAlerts, "B"+rowStr, alert.Hostname)
		_ = f.SetCellValue(sheetAlerts, "C"+rowStr, alertLevelText(alert.Level))
		_ = f.SetCellValue(sheetAlerts, "D"+rowStr, alert.MetricDisplayName)
		_ = f.SetCellValue(sheetAlerts, "E"+rowStr, alert.FormattedValue)
		_ = f.SetCellValue(sheetAlerts, "F"+rowStr, formatThreshold(alert.WarningThreshold, alert.MetricName))
		_ = f.SetCellValue(sheetAlerts, "G"+rowStr, formatThreshold(alert.CriticalThreshold, alert.MetricName))
		_ = f.SetCellValue(sheetAlerts, "H"+rowStr, alert.Message)
		_ = f.SetCellStyle(sheetAlerts, "A"+rowStr, "A"+rowStr, sourceStyle)
		var levelStyle int
		switch alert.Level {
		case model.AlertLevelCritical:
			levelStyle = criticalStyle
		case model.AlertLevelWarning:
			levelStyle = warningStyle
		}
		if levelStyle > 0 {
			_ = f.SetCellStyle(sheetAlerts, "C"+rowStr, "C"+rowStr, levelStyle)
		}
	}
	return nil
}

func (w *hostExcelWriter) createHeaderStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: colorHeaderFg},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{colorHeaderBg}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
}

func (w *hostExcelWriter) createWarningStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: colorWarningFg},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{colorWarningBg}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
}

func (w *hostExcelWriter) createCriticalStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: colorCriticalFg},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{colorCriticalBg}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
}

func (w *hostExcelWriter) createNormalStyle(f *excelize.File) (int, error) {
	return f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: colorNormalFg},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{colorNormalBg}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
}

func (w *hostExcelWriter) setMetricCell(f *excelize.File, sheet, cell string, metric *model.MetricValue, warningStyle, criticalStyle, normalStyle int) {
	if metric == nil || metric.IsNA {
		_ = f.SetCellValue(sheet, cell, "N/A")
		return
	}
	value := metric.FormattedValue
	if value == "" {
		value = fmt.Sprintf("%.2f", metric.RawValue)
	}
	_ = f.SetCellValue(sheet, cell, value)
	var style int
	switch metric.Status {
	case model.MetricStatusCritical:
		style = criticalStyle
	case model.MetricStatusWarning:
		style = warningStyle
	case model.MetricStatusNormal:
		// Match inspection-tool: normal metric values stay unstyled; status cells
		// and explicit baseline pass/fail cells own green styling.
	}
	if style > 0 {
		_ = f.SetCellStyle(sheet, cell, cell, style)
	}
}

func (w *hostExcelWriter) setExpandedMetricCell(f *excelize.File, sheet, cell, metricPrefix string, metrics map[string]*model.MetricValue) {
	var parts []string
	for name, mv := range metrics {
		if !strings.HasPrefix(name, metricPrefix+":") || mv == nil || mv.IsNA {
			continue
		}
		labelValue := strings.TrimPrefix(name, metricPrefix+":")
		switch metricPrefix {
		case "password_expiry":
			switch mv.RawValue {
			case -1:
				parts = append(parts, fmt.Sprintf("%s:永不过期", labelValue))
			case -2:
				parts = append(parts, fmt.Sprintf("%s:无法获取", labelValue))
			default:
				parts = append(parts, fmt.Sprintf("%s:%.0f天", labelValue, mv.RawValue))
			}
		case "password_policy", "sysctl_params":
			parts = append(parts, fmt.Sprintf("%s=%.0f", labelValue, mv.RawValue))
		default:
			parts = append(parts, fmt.Sprintf("%s:%.2f", labelValue, mv.RawValue))
		}
	}
	if len(parts) == 0 {
		_ = f.SetCellValue(sheet, cell, "N/A")
		return
	}
	sort.Strings(parts)
	_ = f.SetCellValue(sheet, cell, strings.Join(parts, ", "))
}

func (w *hostExcelWriter) setFileHandleCell(f *excelize.File, sheet, cell string, openFiles, maxFiles *model.MetricValue, warningStyle, criticalStyle int) {
	if openFiles == nil || openFiles.IsNA || maxFiles == nil || maxFiles.IsNA {
		_ = f.SetCellValue(sheet, cell, "N/A")
		return
	}
	_ = f.SetCellValue(sheet, cell, fmt.Sprintf("%.0f / %.0f", openFiles.RawValue, maxFiles.RawValue))
	if maxFiles.RawValue <= 0 {
		return
	}
	usagePercent := openFiles.RawValue / maxFiles.RawValue * 100
	if usagePercent >= 90 {
		_ = f.SetCellStyle(sheet, cell, cell, criticalStyle)
	} else if usagePercent >= 70 {
		_ = f.SetCellStyle(sheet, cell, cell, warningStyle)
	}
}

func (w *hostExcelWriter) setPublicNetworkCell(f *excelize.File, sheet, cell string, metric *model.MetricValue, normalStyle, criticalStyle int) {
	if metric == nil || metric.IsNA {
		_ = f.SetCellValue(sheet, cell, "N/A")
		return
	}
	if metric.RawValue == 1 {
		_ = f.SetCellValue(sheet, cell, "成功")
		_ = f.SetCellStyle(sheet, cell, cell, criticalStyle)
	} else {
		_ = f.SetCellValue(sheet, cell, "失败")
		_ = f.SetCellStyle(sheet, cell, cell, normalStyle)
	}
}

func (w *hostExcelWriter) setSysctlCell(f *excelize.File, sheet, cell string, metric *model.MetricValue) {
	if metric == nil || metric.IsNA {
		_ = f.SetCellValue(sheet, cell, "N/A")
		return
	}
	_ = f.SetCellValue(sheet, cell, fmt.Sprintf("%.0f", metric.RawValue))
}

func (w *hostExcelWriter) collectDiskPaths(hosts []*model.HostResult) []string {
	pathSet := make(map[string]bool)
	for _, host := range hosts {
		if host == nil {
			continue
		}
		for name := range host.Metrics {
			if strings.HasPrefix(name, "disk_usage:") {
				pathSet[strings.TrimPrefix(name, "disk_usage:")] = true
			}
		}
	}
	paths := make([]string, 0, len(pathSet))
	for path := range pathSet {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (w *hostExcelWriter) getStatusStyle(status model.HostStatus, normalStyle, warningStyle, criticalStyle int) int {
	switch status {
	case model.HostStatusCritical:
		return criticalStyle
	case model.HostStatusWarning:
		return warningStyle
	case model.HostStatusNormal:
		return normalStyle
	default:
		return 0
	}
}

func collectReportAlerts(result *model.InspectionResult) []*model.Alert {
	if result == nil {
		return nil
	}
	if len(result.Alerts) > 0 {
		return result.Alerts
	}
	var alerts []*model.Alert
	for _, host := range result.Hosts {
		if host != nil {
			alerts = append(alerts, host.Alerts...)
		}
	}
	return alerts
}

// columnName converts a 1-based column index to Excel column name (A, B, ..., Z, AA, AB, ...).
func columnName(index int) string {
	result := ""
	for index > 0 {
		index--
		result = string(rune('A'+index%26)) + result
		index /= 26
	}
	return result
}

// ensureDir creates the directory for the output path if it does not exist.
func ensureDir(path string) error {
	if path == "" {
		return nil
	}
	return os.MkdirAll(path, 0755)
}

// GenerateReportPath generates the output path for a report file.
func GenerateReportPath(outputDir string, runID uint, dateFormat string) string {
	t := time.Now().Format("20060102")
	if dateFormat != "" {
		t = dateFormat
	}
	filename := fmt.Sprintf("inspection_report_%d_%s.xlsx", runID, t)
	return filepath.Join(outputDir, fmt.Sprintf("%d", runID), filename)
}

func getSummary(r *model.InspectionResult) model.InspectionSummary {
	if r != nil && r.Summary != nil {
		return *r.Summary
	}
	return model.InspectionSummary{}
}

func getAlertSummary(r *model.InspectionResult) model.AlertSummary {
	if r != nil && r.AlertSummary != nil {
		return *r.AlertSummary
	}
	alerts := collectReportAlerts(r)
	if len(alerts) > 0 {
		return *model.NewAlertSummary(alerts)
	}
	return model.AlertSummary{}
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1f秒", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1f分钟", d.Minutes())
	}
	return fmt.Sprintf("%.1f小时", d.Hours())
}

// statusText converts host status to Chinese text.
func statusText(status model.HostStatus) string {
	switch status {
	case model.HostStatusNormal:
		return "正常"
	case model.HostStatusWarning:
		return "警告"
	case model.HostStatusCritical:
		return "严重"
	case model.HostStatusFailed:
		return "失败"
	default:
		return "未知"
	}
}

// alertLevelText converts alert level to Chinese text.
func alertLevelText(level model.AlertLevel) string {
	switch level {
	case model.AlertLevelNormal:
		return "正常"
	case model.AlertLevelWarning:
		return "警告"
	case model.AlertLevelCritical:
		return "严重"
	default:
		return "未知"
	}
}

// alertLevelPriority returns a numeric priority for sorting (higher = more severe).
func alertLevelPriority(level model.AlertLevel) int {
	switch level {
	case model.AlertLevelCritical:
		return 2
	case model.AlertLevelWarning:
		return 1
	default:
		return 0
	}
}

// formatThreshold formats a threshold value based on metric type.
func formatThreshold(value float64, metricName string) string {
	switch metricName {
	case "cpu_usage", "memory_usage", "disk_usage", "disk_usage_max":
		return fmt.Sprintf("%.1f%%", value)
	case "load_per_core":
		return fmt.Sprintf("%.2f", value)
	case "processes_zombies":
		return fmt.Sprintf("%.0f", value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

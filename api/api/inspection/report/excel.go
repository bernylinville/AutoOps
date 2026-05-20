// Package report provides Excel report generation for inspection results.
// Ported from inspection-tool/internal/report/excel/writer.go.
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
	sheetSummary  = "巡检概览"
	sheetBaseline = "基线检查"
	sheetDetail   = "详细数据"
	sheetAlerts   = "异常汇总"
)

// WriteHostReport generates a Host inspection Excel report.
func WriteHostReport(result *model.InspectionResult, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	// Pre-create styles.
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	warningStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "#9C6500"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFEB9C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	criticalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "#9C0006"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FFC7CE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	normalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "#006100"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#C6EFCE"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 18},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	centerStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// --- Sheet 1: 巡检概览 ---
	writeSummarySheet(f, result, titleStyle, headerStyle)

	// --- Sheet 2: 基线检查 ---
	writeBaselineSheet(f, result, headerStyle, normalStyle, warningStyle, criticalStyle, centerStyle)

	// --- Sheet 3: 详细数据 ---
	writeDetailSheet(f, result, headerStyle, centerStyle)

	// --- Sheet 4: 异常汇总 ---
	writeAlertsSheet(f, result, headerStyle, warningStyle, criticalStyle)

	// Remove default sheet.
	f.DeleteSheet("Sheet1")

	// Ensure output directory exists.
	dir := filepath.Dir(outputPath)
	if err := ensureDir(dir); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := f.SaveAs(outputPath); err != nil {
		return fmt.Errorf("save excel: %w", err)
	}

	return nil
}

func writeSummarySheet(f *excelize.File, r *model.InspectionResult, titleStyle, headerStyle int) {
	idx, _ := f.NewSheet(sheetSummary)
	f.SetActiveSheet(idx)

	// Title
	f.SetCellValue(sheetSummary, "A1", "主机巡检报告")
	f.MergeCell(sheetSummary, "A1", "F1")
	f.SetCellStyle(sheetSummary, "A1", "F1", titleStyle)
	f.SetRowHeight(sheetSummary, 1, 30)

	// Report info.
	info := [][]string{
		{"巡检时间", r.InspectionTime.Format("2006-01-02 15:04:05")},
		{"巡检耗时", r.Duration.String()},
		{"巡检版本", r.Version},
		{"主机总数", fmt.Sprintf("%d 台", getSummary(r).TotalHosts)},
		{"正常主机", fmt.Sprintf("%d 台", getSummary(r).NormalHosts)},
		{"警告主机", fmt.Sprintf("%d 台", getSummary(r).WarningHosts)},
		{"严重主机", fmt.Sprintf("%d 台", getSummary(r).CriticalHosts)},
		{"失败主机", fmt.Sprintf("%d 台", getSummary(r).FailedHosts)},
		{"异常总数", fmt.Sprintf("%d 条（警告 %d, 严重 %d）", r.AlertSummary.TotalAlerts, r.AlertSummary.WarningCount, r.AlertSummary.CriticalCount)},
	}

	for i, row := range info {
		rowNum := i + 3
		f.SetCellValue(sheetSummary, fmt.Sprintf("A%d", rowNum), row[0]+":")
		f.SetCellValue(sheetSummary, fmt.Sprintf("B%d", rowNum), row[1])
	}

	f.SetColWidth(sheetSummary, "A", "A", 18)
	f.SetColWidth(sheetSummary, "B", "B", 50)
}

func writeBaselineSheet(f *excelize.File, r *model.InspectionResult, headerStyle, normalStyle, warningStyle, criticalStyle, centerStyle int) {
	idx, _ := f.NewSheet(sheetBaseline)
	f.SetActiveSheet(idx)

	headers := []string{"主机名", "IP 地址", "操作系统", "CPU 利用率", "内存利用率", "磁盘利用率(最大)", "僵尸进程", "单核负载", "NTP 偏移", "运行时间", "状态"}

	for i, h := range headers {
		cell := fmt.Sprintf("%s1", colName(i))
		f.SetCellValue(sheetBaseline, cell, h)
		f.SetCellStyle(sheetBaseline, cell, cell, headerStyle)
	}

	for i, host := range r.Hosts {
		row := i + 2
		if host == nil {
			continue
		}

		f.SetCellValue(sheetBaseline, fmt.Sprintf("A%d", row), host.Hostname)
		f.SetCellValue(sheetBaseline, fmt.Sprintf("B%d", row), host.IP)
		f.SetCellValue(sheetBaseline, fmt.Sprintf("C%d", row), host.OS)

		metrics := []string{"cpu_usage", "memory_usage", "disk_usage_max", "processes_zombies", "load_per_core", "ntp_offset"}
		for j, name := range metrics {
			cell := fmt.Sprintf("%s%d", colName(j+3), row)
			if mv := host.GetMetric(name); mv != nil {
				f.SetCellValue(sheetBaseline, cell, mv.FormattedValue)
			} else {
				f.SetCellValue(sheetBaseline, cell, "-")
			}
			f.SetCellStyle(sheetBaseline, cell, cell, centerStyle)
		}

		// Uptime.
		if mv := host.GetMetric("uptime"); mv != nil {
			f.SetCellValue(sheetBaseline, fmt.Sprintf("J%d", row), mv.FormattedValue)
		} else {
			f.SetCellValue(sheetBaseline, fmt.Sprintf("J%d", row), "-")
		}

		// Status with color.
		statusCell := fmt.Sprintf("K%d", row)
		f.SetCellValue(sheetBaseline, statusCell, string(host.Status))
		switch host.Status {
		case model.HostStatusCritical:
			f.SetCellStyle(sheetBaseline, statusCell, statusCell, criticalStyle)
		case model.HostStatusWarning:
			f.SetCellStyle(sheetBaseline, statusCell, statusCell, warningStyle)
		case model.HostStatusNormal:
			f.SetCellStyle(sheetBaseline, statusCell, statusCell, normalStyle)
		}
	}

	// Set column widths.
	for col := 0; col < len(headers); col++ {
		f.SetColWidth(sheetBaseline, colName(col), colName(col), 18)
	}

	// Freeze header row.
	f.SetPanes(sheetBaseline, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})
}

func writeDetailSheet(f *excelize.File, r *model.InspectionResult, headerStyle, centerStyle int) {
	idx, _ := f.NewSheet(sheetDetail)
	f.SetActiveSheet(idx)

	headers := []string{"主机名", "IP 地址", "指标", "当前值", "状态"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", colName(i))
		f.SetCellValue(sheetDetail, cell, h)
		f.SetCellStyle(sheetDetail, cell, cell, headerStyle)
	}

	metricOrder := []string{"cpu_usage", "memory_usage", "memory_total", "memory_free", "memory_available",
		"disk_usage_max", "disk_total", "disk_free", "load_1m", "load_5m", "load_15m",
		"load_per_core", "processes_total", "processes_zombies", "uptime", "cpu_cores", "ntp_offset"}

	row := 2
	for _, host := range r.Hosts {
		if host == nil {
			continue
		}
		for _, name := range metricOrder {
			mv := host.GetMetric(name)
			if mv == nil {
				// Try expanded metrics (e.g., disk_usage:/home).
				found := false
				for mn, mvv := range host.Metrics {
					if strings.HasPrefix(mn, name+":") {
						f.SetCellValue(sheetDetail, fmt.Sprintf("A%d", row), host.Hostname)
						f.SetCellValue(sheetDetail, fmt.Sprintf("B%d", row), host.IP)
						f.SetCellValue(sheetDetail, fmt.Sprintf("C%d", row), mn)
						f.SetCellValue(sheetDetail, fmt.Sprintf("D%d", row), mvv.FormattedValue)
						f.SetCellValue(sheetDetail, fmt.Sprintf("E%d", row), string(mvv.Status))
						row++
						found = true
					}
				}
				if found {
					continue
				}
				f.SetCellValue(sheetDetail, fmt.Sprintf("A%d", row), host.Hostname)
				f.SetCellValue(sheetDetail, fmt.Sprintf("B%d", row), host.IP)
				f.SetCellValue(sheetDetail, fmt.Sprintf("C%d", row), name)
				f.SetCellValue(sheetDetail, fmt.Sprintf("D%d", row), "-")
				f.SetCellValue(sheetDetail, fmt.Sprintf("E%d", row), "-")
				row++
			} else {
				f.SetCellValue(sheetDetail, fmt.Sprintf("A%d", row), host.Hostname)
				f.SetCellValue(sheetDetail, fmt.Sprintf("B%d", row), host.IP)
				f.SetCellValue(sheetDetail, fmt.Sprintf("C%d", row), name)
				f.SetCellValue(sheetDetail, fmt.Sprintf("D%d", row), mv.FormattedValue)
				f.SetCellValue(sheetDetail, fmt.Sprintf("E%d", row), string(mv.Status))
				row++
			}
		}
	}

	for col := 0; col < len(headers); col++ {
		f.SetColWidth(sheetDetail, colName(col), colName(col), 18)
	}
	f.SetPanes(sheetDetail, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})
}

func writeAlertsSheet(f *excelize.File, r *model.InspectionResult, headerStyle, warningStyle, criticalStyle int) {
	idx, _ := f.NewSheet(sheetAlerts)
	f.SetActiveSheet(idx)

	headers := []string{"主机名", "指标", "当前值", "告警阈值(警告)", "告警阈值(严重)", "级别", "告警详情"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s1", colName(i))
		f.SetCellValue(sheetAlerts, cell, h)
		f.SetCellStyle(sheetAlerts, cell, cell, headerStyle)
	}

	// Sort alerts: critical first, then by hostname.
	sorted := make([]*model.Alert, len(r.Alerts))
	copy(sorted, r.Alerts)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Level != sorted[j].Level {
			return sorted[i].Level == model.AlertLevelCritical
		}
		return sorted[i].Hostname < sorted[j].Hostname
	})

	for i, alert := range sorted {
		row := i + 2
		f.SetCellValue(sheetAlerts, fmt.Sprintf("A%d", row), alert.Hostname)
		f.SetCellValue(sheetAlerts, fmt.Sprintf("B%d", row), alert.MetricDisplayName)
		f.SetCellValue(sheetAlerts, fmt.Sprintf("C%d", row), alert.FormattedValue)
		f.SetCellValue(sheetAlerts, fmt.Sprintf("D%d", row), alert.WarningThreshold)
		f.SetCellValue(sheetAlerts, fmt.Sprintf("E%d", row), alert.CriticalThreshold)
		f.SetCellValue(sheetAlerts, fmt.Sprintf("F%d", row), string(alert.Level))
		f.SetCellValue(sheetAlerts, fmt.Sprintf("G%d", row), alert.Message)

		// Color the level cell.
		levelCell := fmt.Sprintf("F%d", row)
		switch alert.Level {
		case model.AlertLevelCritical:
			f.SetCellStyle(sheetAlerts, levelCell, levelCell, criticalStyle)
		case model.AlertLevelWarning:
			f.SetCellStyle(sheetAlerts, levelCell, levelCell, warningStyle)
		}
	}

	for col := 0; col < len(headers); col++ {
		f.SetColWidth(sheetAlerts, colName(col), colName(col), 20)
	}
	f.SetColWidth(sheetAlerts, "G", "G", 50)
	f.SetPanes(sheetAlerts, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2"})
}

// colName converts 0-based index to Excel column letters.
func colName(n int) string {
	if n < 26 {
		return string(rune('A' + n))
	}
	return string(rune('A'+n/26-1)) + string(rune('A'+n%26))
}

// ensureDir creates the directory for the output path if it does not exist.
func ensureDir(path string) error {
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
	if r.Summary != nil {
		return *r.Summary
	}
	return model.InspectionSummary{}
}

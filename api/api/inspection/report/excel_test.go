package report

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"dodevops-api/api/inspection/model"

	"github.com/xuri/excelize/v2"
)

func TestWriteHostReport_MatchesInspectionToolWorkbookShape(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "host_report.xlsx")
	result := sampleInspectionResult()

	if err := WriteHostReport(result, outputPath); err != nil {
		t.Fatalf("WriteHostReport() error = %v", err)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("report file not created: %v", err)
	}

	f, err := excelize.OpenFile(outputPath)
	if err != nil {
		t.Fatalf("open report: %v", err)
	}
	defer f.Close()

	wantSheets := []string{sheetSummary, sheetBaselineCheck, sheetDetail, sheetAlerts}
	if got := f.GetSheetList(); !reflect.DeepEqual(got, wantSheets) {
		t.Fatalf("sheets = %#v, want %#v", got, wantSheets)
	}

	assertCell(t, f, sheetSummary, "A1", "系统巡检报告")
	assertCell(t, f, sheetSummary, "A5", "主机总数")
	assertCell(t, f, sheetSummary, "B5", "2")
	assertCell(t, f, sheetSummary, "A13", "工具版本")

	wantBaselineHeaders := []string{
		"巡检时间", "主机名", "IP地址", "操作系统", "内核版本", "运行时间",
		"密码过期", "密码策略", "文件句柄", "公网访问",
		"端口范围(最小)", "端口范围(最大)", "连接跟踪最大", "TIME_WAIT桶数",
		"FIN_WAIT超时", "TIME_WAIT超时", "CLOSE_WAIT超时", "ESTABLISHED超时", "tcp_tw_reuse", "tcp_timestamps",
	}
	assertRowPrefix(t, f, sheetBaselineCheck, 1, wantBaselineHeaders)
	assertCell(t, f, sheetBaselineCheck, "B2", "host-1")
	assertCell(t, f, sheetBaselineCheck, "G2", "root:永不过期, zabbix:无法获取")
	assertCell(t, f, sheetBaselineCheck, "H2", "PASS_MAX_DAYS=90, PASS_MIN_LEN=15")
	assertCell(t, f, sheetBaselineCheck, "I2", "1200 / 4000")
	assertCell(t, f, sheetBaselineCheck, "J2", "成功")
	assertCell(t, f, sheetBaselineCheck, "K2", "1024")
	assertCell(t, f, sheetBaselineCheck, "T2", "1")

	wantDetailHeaders := []string{
		"主机名", "IP地址", "状态", "内核版本", "CPU利用率", "内存利用率", "内存空闲",
		"磁盘最大利用率", "运行时间", "NTP时间偏差", "僵尸进程", "打开句柄数", "句柄最大值", "磁盘:/", "磁盘:/data",
	}
	assertRowPrefix(t, f, sheetDetail, 1, wantDetailHeaders)
	assertCell(t, f, sheetDetail, "A2", "host-1")
	assertCell(t, f, sheetDetail, "C2", "正常")
	assertCell(t, f, sheetDetail, "N2", "45.0%")
	assertCell(t, f, sheetDetail, "A3", "host-2")
	assertCell(t, f, sheetDetail, "C3", "警告")
	assertCell(t, f, sheetDetail, "A4", "") // one row per host, not one row per metric

	wantAlertHeaders := []string{"来源类型", "实例标识", "告警级别", "指标名称", "当前值", "警告阈值", "严重阈值", "告警消息"}
	assertRowPrefix(t, f, sheetAlerts, 1, wantAlertHeaders)
	assertCell(t, f, sheetAlerts, "A2", "Host")
	assertCell(t, f, sheetAlerts, "B2", "host-2")
	assertCell(t, f, sheetAlerts, "C2", "警告")
	assertCell(t, f, sheetAlerts, "D2", "CPU 利用率")
	assertCell(t, f, sheetAlerts, "F2", "70.0%")
	assertCell(t, f, sheetAlerts, "G2", "90.0%")
}

func TestWriteHostReport_EmptyResultOmitsAlertsSheetLikeInspectionTool(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "empty")
	result := &model.InspectionResult{
		InspectionTime: time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC),
		Duration:       time.Second,
		Summary:        &model.InspectionSummary{},
		AlertSummary:   &model.AlertSummary{},
	}

	if err := WriteHostReport(result, outputPath); err != nil {
		t.Fatalf("WriteHostReport() error = %v", err)
	}
	f, err := excelize.OpenFile(outputPath + ".xlsx")
	if err != nil {
		t.Fatalf("open report: %v", err)
	}
	defer f.Close()

	wantSheets := []string{sheetSummary, sheetBaselineCheck, sheetDetail}
	if got := f.GetSheetList(); !reflect.DeepEqual(got, wantSheets) {
		t.Fatalf("sheets = %#v, want %#v", got, wantSheets)
	}
}

func sampleInspectionResult() *model.InspectionResult {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	inspectionTime := time.Date(2026, 5, 15, 17, 10, 59, 0, loc)

	host1 := &model.HostResult{
		Hostname:      "host-1",
		IP:            "192.168.1.10",
		OS:            "linux",
		OSVersion:     "",
		KernelVersion: "5.15.0",
		Status:        model.HostStatusNormal,
		Metrics: map[string]*model.MetricValue{
			"cpu_usage":                     metric("cpu_usage", 10, "10.0%", model.MetricStatusNormal),
			"memory_usage":                  metric("memory_usage", 30, "30.0%", model.MetricStatusNormal),
			"memory_available":              metric("memory_available", 8*1024*1024*1024, "8.00 GB", model.MetricStatusNormal),
			"disk_usage_max":                metric("disk_usage_max", 45, "45.0%", model.MetricStatusNormal),
			"disk_usage:/":                  metric("disk_usage:/", 45, "45.0%", model.MetricStatusNormal),
			"disk_usage:/data":              metric("disk_usage:/data", 12, "12.0%", model.MetricStatusNormal),
			"uptime":                        metric("uptime", 7200, "2时0分", model.MetricStatusNormal),
			"ntp_offset":                    metric("ntp_offset", 0.001, "+1.0ms", model.MetricStatusNormal),
			"processes_zombies":             metric("processes_zombies", 0, "0", model.MetricStatusNormal),
			"open_files":                    metric("open_files", 1200, "1200", model.MetricStatusNormal),
			"max_files":                     metric("max_files", 4000, "4000", model.MetricStatusNormal),
			"public_network":                metric("public_network", 1, "是", model.MetricStatusNormal),
			"password_expiry:root":          metric("password_expiry:root", -1, "", model.MetricStatusNormal),
			"password_expiry:zabbix":        metric("password_expiry:zabbix", -2, "", model.MetricStatusNormal),
			"password_policy:PASS_MAX_DAYS": metric("password_policy:PASS_MAX_DAYS", 90, "", model.MetricStatusNormal),
			"password_policy:PASS_MIN_LEN":  metric("password_policy:PASS_MIN_LEN", 15, "", model.MetricStatusNormal),
			"sysctl_params:net.ipv4.ip_local_port_range_min": metric("sysctl_params:net.ipv4.ip_local_port_range_min", 1024, "", model.MetricStatusNormal),
			"sysctl_params:net.ipv4.ip_local_port_range_max": metric("sysctl_params:net.ipv4.ip_local_port_range_max", 65000, "", model.MetricStatusNormal),
			"sysctl_params:net.netfilter.nf_conntrack_max":   metric("sysctl_params:net.netfilter.nf_conntrack_max", 262144, "", model.MetricStatusNormal),
			"sysctl_params:net.ipv4.tcp_max_tw_buckets":      metric("sysctl_params:net.ipv4.tcp_max_tw_buckets", 5000, "", model.MetricStatusNormal),
			"sysctl_params:net.ipv4.tcp_tw_reuse":            metric("sysctl_params:net.ipv4.tcp_tw_reuse", 0, "", model.MetricStatusNormal),
			"sysctl_params:net.ipv4.tcp_timestamps":          metric("sysctl_params:net.ipv4.tcp_timestamps", 1, "", model.MetricStatusNormal),
		},
	}

	alert := &model.Alert{
		Hostname:          "host-2",
		MetricName:        "cpu_usage",
		MetricDisplayName: "CPU 利用率",
		CurrentValue:      75,
		FormattedValue:    "75.0%",
		WarningThreshold:  70,
		CriticalThreshold: 90,
		Level:             model.AlertLevelWarning,
		Message:           "CPU 利用率 警告: 75.0% (阈值: 70.0%)",
	}
	host2 := &model.HostResult{
		Hostname:      "host-2",
		IP:            "192.168.1.11",
		OS:            "linux",
		KernelVersion: "5.15.0",
		Status:        model.HostStatusWarning,
		Metrics: map[string]*model.MetricValue{
			"cpu_usage":         metric("cpu_usage", 75, "75.0%", model.MetricStatusWarning),
			"memory_usage":      metric("memory_usage", 25, "25.0%", model.MetricStatusNormal),
			"memory_available":  metric("memory_available", 4*1024*1024*1024, "4.00 GB", model.MetricStatusNormal),
			"disk_usage_max":    metric("disk_usage_max", 20, "20.0%", model.MetricStatusNormal),
			"uptime":            metric("uptime", 60, "1分钟", model.MetricStatusNormal),
			"ntp_offset":        metric("ntp_offset", 0.001, "+1.0ms", model.MetricStatusNormal),
			"processes_zombies": metric("processes_zombies", 0, "0", model.MetricStatusNormal),
			"open_files":        metric("open_files", 10, "10", model.MetricStatusNormal),
			"max_files":         metric("max_files", 1000, "1000", model.MetricStatusNormal),
			"public_network":    metric("public_network", 0, "否", model.MetricStatusNormal),
		},
		Alerts: []*model.Alert{alert},
	}

	return &model.InspectionResult{
		InspectionTime: inspectionTime,
		Duration:       5 * time.Second,
		Summary:        &model.InspectionSummary{TotalHosts: 2, NormalHosts: 1, WarningHosts: 1},
		Hosts:          []*model.HostResult{host1, host2},
		Alerts:         []*model.Alert{alert},
		AlertSummary:   &model.AlertSummary{TotalAlerts: 1, WarningCount: 1},
		Version:        "1.0.0-test",
	}
}

func metric(name string, raw float64, formatted string, status model.MetricStatus) *model.MetricValue {
	return &model.MetricValue{Name: name, RawValue: raw, FormattedValue: formatted, Status: status}
}

func assertCell(t *testing.T, f *excelize.File, sheet, cell, want string) {
	t.Helper()
	got, err := f.GetCellValue(sheet, cell)
	if err != nil {
		t.Fatalf("GetCellValue(%s!%s): %v", sheet, cell, err)
	}
	if got != want {
		t.Fatalf("%s!%s = %q, want %q", sheet, cell, got, want)
	}
}

func assertRowPrefix(t *testing.T, f *excelize.File, sheet string, row int, want []string) {
	t.Helper()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatalf("GetRows(%s): %v", sheet, err)
	}
	if len(rows) < row {
		t.Fatalf("sheet %s has %d rows, want at least %d", sheet, len(rows), row)
	}
	got := rows[row-1]
	if len(got) < len(want) {
		t.Fatalf("%s row %d has %d cells, want at least %d: %#v", sheet, row, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s row %d col %d = %q, want %q", sheet, row, i+1, got[i], want[i])
		}
	}
}

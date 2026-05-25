// Package engine provides Host inspection metric definitions.
// Ported from inspection-tool/configs/metrics.yaml.
package engine

import (
	"dodevops-api/api/inspection/model"
)

// HostMetricDefinitions returns the standard Host inspection metric definitions.
// These match the PromQL metrics collected by telegraf/categraf agents.
func HostMetricDefinitions() []*model.MetricDefinition {
	return []*model.MetricDefinition{
		// --- CPU ---
		{
			Name:        "cpu_usage",
			DisplayName: "CPU 利用率",
			Query:       `cpu_usage_active{cpu="cpu-total"}`,
			Unit:        "%",
			Category:    model.MetricCategoryCPU,
			Format:      model.MetricFormatPercent,
			Note:        "整机 CPU 利用率，标签 cpu=cpu-total 表示所有核心的聚合值",
		},

		// --- Memory ---
		{
			Name:        "memory_usage",
			DisplayName: "内存利用率",
			Query:       "100 - mem_available_percent",
			Unit:        "%",
			Category:    model.MetricCategoryMemory,
			Format:      model.MetricFormatPercent,
			Note:        "基于 MemAvailable 计算，比传统的 used/total 更准确",
		},
		{
			Name:        "memory_total",
			DisplayName: "内存总量",
			Query:       "mem_total",
			Unit:        "bytes",
			Category:    model.MetricCategoryMemory,
			Format:      model.MetricFormatSize,
		},
		{
			Name:        "memory_free",
			DisplayName: "内存空闲",
			Query:       "mem_free",
			Unit:        "bytes",
			Category:    model.MetricCategoryMemory,
			Format:      model.MetricFormatSize,
		},
		{
			Name:        "memory_available",
			DisplayName: "可分配内存",
			Query:       "mem_available",
			Unit:        "bytes",
			Category:    model.MetricCategoryMemory,
			Format:      model.MetricFormatSize,
		},

		// --- Disk ---
		{
			Name:          "disk_usage",
			DisplayName:   "磁盘利用率",
			Query:         "disk_used_percent",
			Unit:          "%",
			Category:      model.MetricCategoryDisk,
			Format:        model.MetricFormatPercent,
			Aggregate:     model.AggregateMax,
			ExpandByLabel: "path",
			Note:          "各挂载点的磁盘使用率，告警基于最大值判断",
		},
		{
			Name:          "disk_total",
			DisplayName:   "磁盘总量",
			Query:         "disk_total",
			Unit:          "bytes",
			Category:      model.MetricCategoryDisk,
			Format:        model.MetricFormatSize,
			ExpandByLabel: "path",
		},
		{
			Name:          "disk_free",
			DisplayName:   "磁盘剩余",
			Query:         "disk_free",
			Unit:          "bytes",
			Category:      model.MetricCategoryDisk,
			Format:        model.MetricFormatSize,
			ExpandByLabel: "path",
		},

		// --- System ---
		{
			Name:        "uptime",
			DisplayName: "运行时间",
			Query:       "system_uptime",
			Unit:        "seconds",
			Category:    model.MetricCategorySystem,
			Format:      model.MetricFormatDuration,
		},
		{
			Name:        "cpu_cores",
			DisplayName: "CPU 核心数",
			Query:       "system_n_cpus",
			Unit:        "个",
			Category:    model.MetricCategorySystem,
		},
		{
			Name:        "load_1m",
			DisplayName: "1分钟负载",
			Query:       "system_load1",
			Unit:        "",
			Category:    model.MetricCategorySystem,
		},
		{
			Name:        "load_5m",
			DisplayName: "5分钟负载",
			Query:       "system_load5",
			Unit:        "",
			Category:    model.MetricCategorySystem,
		},
		{
			Name:        "load_15m",
			DisplayName: "15分钟负载",
			Query:       "system_load15",
			Unit:        "",
			Category:    model.MetricCategorySystem,
		},
		{
			Name:        "load_per_core",
			DisplayName: "单核负载",
			Query:       "system_load_norm_1",
			Unit:        "",
			Category:    model.MetricCategorySystem,
			Note:        "1分钟负载除以CPU核心数，大于1.0表示过载",
		},

		// --- Process ---
		{
			Name:        "processes_total",
			DisplayName: "总进程数",
			Query:       "processes_total",
			Unit:        "个",
			Category:    model.MetricCategoryProcess,
		},
		{
			Name:        "processes_zombies",
			DisplayName: "僵尸进程数",
			Query:       "processes_zombies",
			Unit:        "个",
			Category:    model.MetricCategoryProcess,
		},

		// --- NTP ---
		{
			Name:        "ntp_offset",
			DisplayName: "NTP 时间偏差",
			Query:       "chrony_system_time",
			Unit:        "ms",
			Category:    model.MetricCategorySystem,
			Format:      model.MetricFormatNTPOffset,
			Note:        "系统时钟与 NTP 源的偏差，stratum=0 表示未同步",
		},

		// --- Security / baseline checks collected by collect_host_security.sh.j2 ---
		{
			Name:        "public_network",
			DisplayName: "公网访问检查",
			Query:       "host_public_network_access",
			Unit:        "",
			Category:    model.MetricCategorySecurity,
			Format:      model.MetricFormatBoolean,
			Note:        "检查服务器是否可访问公网 (1=可访问, 0=不可访问)",
		},
		{
			Name:          "password_expiry",
			DisplayName:   "密码过期天数",
			Query:         "host_password_expiry_days",
			Unit:          "天",
			Category:      model.MetricCategorySecurity,
			ExpandByLabel: "user",
			Note:          "系统用户密码过期天数 (-1=永不过期, -2=无法获取)",
		},
		{
			Name:          "password_policy",
			DisplayName:   "密码策略",
			Query:         "host_password_policy",
			Unit:          "",
			Category:      model.MetricCategorySecurity,
			ExpandByLabel: "param",
			Note:          "密码策略参数值 (PASS_MAX_DAYS, PASS_MIN_DAYS, PASS_MIN_LEN, PASS_WARN_AGE)",
		},
		{
			Name:        "open_files",
			DisplayName: "打开文件句柄数",
			Query:       "host_open_files",
			Unit:        "个",
			Category:    model.MetricCategorySystem,
			Note:        "当前打开的文件句柄数，来自 /proc/sys/fs/file-nr",
		},
		{
			Name:        "max_files",
			DisplayName: "句柄数最大值",
			Query:       "host_max_files",
			Unit:        "个",
			Category:    model.MetricCategorySystem,
			Note:        "系统文件句柄数上限，来自 /proc/sys/fs/file-max",
		},
		{
			Name:          "sysctl_params",
			DisplayName:   "系统参数检查",
			Query:         "host_sysctl",
			Unit:          "",
			Category:      model.MetricCategorySystem,
			ExpandByLabel: "param",
			Note:          "sysctl 内核参数值 (10个关键网络参数)",
		},
	}
}

// AlertableMetricNames returns the names of metrics that trigger alerts.
func AlertableMetricNames() []string {
	return []string{"cpu_usage", "memory_usage", "disk_usage_max", "processes_zombies", "load_per_core", "ntp_offset"}
}

// BuildMetricNameDisplayMap returns a map from metric name to display name.
func BuildMetricNameDisplayMap() map[string]string {
	m := make(map[string]string)
	for _, def := range HostMetricDefinitions() {
		m[def.Name] = def.DisplayName
	}
	// Include aggregated/derived names.
	m["disk_usage_max"] = "磁盘利用率(最大)"
	return m
}

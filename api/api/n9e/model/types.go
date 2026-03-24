package model

// N9E API 响应类型定义
// 移植自 CMDB internal/connector/n9e/types.go

// N9EResponse N9E API 通用响应结构
type N9EResponse[T any] struct {
	Dat T      `json:"dat"`
	Err string `json:"err"`
}

// TargetListData N9E 主机列表分页数据
type TargetListData struct {
	List  []TargetData `json:"list"`
	Total int          `json:"total"`
}

// TargetData N9E 主机目标数据
type TargetData struct {
	ID           int64             `json:"id"`
	Ident        string            `json:"ident"`
	Hostname     string            `json:"hostname"`
	HostIP       string            `json:"host_ip"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	CPUNum       int               `json:"cpu_num"`
	TargetUp     int               `json:"target_up"`
	AgentVersion string            `json:"agent_version"`
	TagsMaps     map[string]string `json:"tags_maps"`
	GroupIDs     []int64           `json:"group_ids"`
	GroupObjs    []GroupObj        `json:"group_objs"`
	UpdateAt     int64             `json:"update_at"`
	MemUtil      float64           `json:"mem_util"`
	CPUUtil      float64           `json:"cpu_util"`
	RemoteAddr   string            `json:"remote_addr"`
	ExtendInfo   string            `json:"extend_info"`
}

// TargetExtendInfo N9E 主机扩展信息
type TargetExtendInfo struct {
	CPU      TargetCPUInfo      `json:"cpu"`
	Memory   TargetMemoryInfo   `json:"memory"`
	Platform TargetPlatformInfo `json:"platform"`
}

// TargetCPUInfo CPU 信息
type TargetCPUInfo struct {
	ModelName string `json:"model_name"`
}

// TargetMemoryInfo 内存信息
type TargetMemoryInfo struct {
	Total string `json:"total"`
}

// TargetPlatformInfo 平台信息
type TargetPlatformInfo struct {
	Hostname      string `json:"hostname"`
	KernelRelease string `json:"kernel_release"`
}

// GroupObj 业务组对象（内嵌在 TargetData 中）
type GroupObj struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// BusiGroupData N9E 业务组数据
type BusiGroupData struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	UpdateAt int64  `json:"update_at"`
}

// DatasourceData N9E 数据源数据
type DatasourceData struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PluginType string `json:"plugin_type"`
	Category   string `json:"category"`
	Status     string `json:"status"`
	HTTP       struct {
		URL string `json:"url"`
	} `json:"http"`
}

// SyncResult 同步结果统计
type SyncResult struct {
	BusiGroups  SyncStats `json:"busiGroups"`
	Hosts       SyncStats `json:"hosts"`
	Datasources SyncStats `json:"datasources"`
}

// SyncStats 同步统计数据
type SyncStats struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
	Stale   int `json:"stale,omitempty"`
}

// 网络设备管理模型 — Phase 5
package model

import "dodevops-api/common/util"

// NetworkInspection 网络设备巡检记录
type NetworkInspection struct {
	ID           uint       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CIInstanceID uint       `gorm:"column:ci_instance_id;NOT NULL;index;comment:'关联CI实例ID'" json:"ciInstanceId"`
	DeviceName   string     `gorm:"column:device_name;type:varchar(200);comment:'设备名称（快照）'" json:"deviceName"`
	MgmtIP       string     `gorm:"column:mgmt_ip;type:varchar(64);NOT NULL;comment:'被检测管理IP'" json:"mgmtIp"`
	Reachable    bool       `gorm:"column:reachable;NOT NULL;comment:'是否可达'" json:"reachable"`
	LatencyMs    int        `gorm:"column:latency_ms;default:0;comment:'延迟(ms)，不可达时为-1'" json:"latencyMs"`
	Port         int        `gorm:"column:port;comment:'检测端口，SNMP时为161'" json:"port"`
	Method       string     `gorm:"column:method;type:varchar(20);default:'tcp';comment:'巡检方式: tcp/snmp'" json:"method"`
	SysDescr     string     `gorm:"column:sys_descr;type:varchar(500);comment:'SNMP sysDescr'" json:"sysDescr"`
	SysUpTime    string     `gorm:"column:sys_uptime;type:varchar(100);comment:'SNMP sysUpTime'" json:"sysUptime"`
	ErrorMsg     string     `gorm:"column:error_msg;type:varchar(500);comment:'不可达原因'" json:"errorMsg"`
	Operator     string     `gorm:"column:operator;type:varchar(100);comment:'操作人'" json:"operator"`
	CreateTime   util.HTime `gorm:"column:create_time;NOT NULL;index;comment:'巡检时间'" json:"createTime"`
}

func (NetworkInspection) TableName() string {
	return "network_inspection"
}

// NetworkInspectDto 发起巡检请求
type NetworkInspectDto struct {
	CIInstanceID uint `json:"ciInstanceId" validate:"required"`
}

// NetworkInspectionVo 巡检结果视图
type NetworkInspectionVo struct {
	ID           uint       `json:"id"`
	CIInstanceID uint       `json:"ciInstanceId"`
	DeviceName   string     `json:"deviceName"`
	MgmtIP       string     `json:"mgmtIp"`
	Reachable    bool       `json:"reachable"`
	LatencyMs    int        `json:"latencyMs"`
	Port         int        `json:"port"`
	Method       string     `json:"method"`
	SysDescr     string     `json:"sysDescr"`
	SysUpTime    string     `json:"sysUptime"`
	ErrorMsg     string     `json:"errorMsg"`
	Operator     string     `json:"operator"`
	CreateTime   util.HTime `json:"createTime"`
}

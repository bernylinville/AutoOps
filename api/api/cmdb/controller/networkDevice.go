package controller

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	n9emodel "dodevops-api/api/n9e/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/common/util"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/gin-gonic/gin"
)

type NetworkDeviceController struct {
	dao dao.NetworkDeviceDao
	ci  dao.CITypeDao
}

func NewNetworkDeviceController() *NetworkDeviceController {
	return &NetworkDeviceController{
		dao: dao.NewNetworkDeviceDao(),
		ci:  dao.NewCITypeDao(),
	}
}

// GetNetworkDevices 分页获取网络设备列表（自动过滤 network_device 类型）
func (c *NetworkDeviceController) GetNetworkDevices(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	keyword := ctx.Query("keyword")
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetNetworkDeviceInstances(page, pageSize, keyword)

	// 为每个设备附加最新巡检结果
	type deviceRow struct {
		model.CIInstanceVo
		LastInspection *model.NetworkInspectionVo `json:"lastInspection"`
	}

	rows := make([]deviceRow, 0, len(list))
	for _, inst := range list {
		vo := model.CIInstanceVo{
			ID:         inst.ID,
			CITypeID:   inst.CITypeID,
			TypeName:   inst.CIType.Name,
			TypeCode:   inst.CIType.Code,
			TypeIcon:   inst.CIType.Icon,
			Name:       inst.Name,
			Status:     inst.Status,
			Remark:     inst.Remark,
			CreateTime: inst.CreateTime,
			UpdateTime: inst.UpdateTime,
		}
		if inst.Attributes != nil {
			var attrs map[string]interface{}
			if err := json.Unmarshal(inst.Attributes, &attrs); err == nil {
				vo.Attributes = attrs
			}
		}

		row := deviceRow{CIInstanceVo: vo}
		if latest, err := c.dao.GetLatestInspection(inst.ID); err == nil {
			row.LastInspection = &model.NetworkInspectionVo{
				ID:           latest.ID,
				CIInstanceID: latest.CIInstanceID,
				DeviceName:   latest.DeviceName,
				MgmtIP:       latest.MgmtIP,
				Reachable:    latest.Reachable,
				LatencyMs:    latest.LatencyMs,
				Port:         latest.Port,
				ErrorMsg:     latest.ErrorMsg,
				Operator:     latest.Operator,
				CreateTime:   latest.CreateTime,
			}
		}
		rows = append(rows, row)
	}

	result.SuccessWithPage(ctx, rows, total, page, pageSize)
}

// InspectDevice 对指定网络设备发起巡检：优先 SNMP，不可用时 fallback TCP
func (c *NetworkDeviceController) InspectDevice(ctx *gin.Context) {
	var dto model.NetworkInspectDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 验证 CI 实例存在
	inst, err := c.ci.GetCIInstanceByID(dto.CIInstanceID)
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI实例不存在")
		return
	}

	// 从 JSONB attributes 提取管理 IP 和 SNMP 配置
	var attrs map[string]interface{}
	if inst.Attributes != nil {
		_ = json.Unmarshal(inst.Attributes, &attrs)
	}
	mgmtIP, _ := attrs["mgmt_ip"].(string)
	if mgmtIP == "" {
		result.Failed(ctx, constant.INVALID_PARAMS, "该设备未配置管理IP（mgmt_ip属性）")
		return
	}
	snmpCommunity, _ := attrs["snmp_community"].(string)

	// 获取操作人
	var operatorName string
	if obj, ok := ctx.Get(constant.ContextKeyUserObj); ok {
		if admin, ok2 := obj.(*systemmodel.JwtAdmin); ok2 {
			operatorName = admin.Username
		}
	}

	var (
		reachable   bool
		latencyMs   = -1
		successPort int
		method      string
		sysDescr    string
		sysUpTime   string
		errMsg      string
	)

	// 优先 SNMP（需配置 snmp_community 属性）
	if snmpCommunity != "" {
		reachable, latencyMs, sysDescr, sysUpTime, errMsg = probeSNMP(mgmtIP, snmpCommunity)
		method = "snmp"
		successPort = 161
	}

	// SNMP 未配置或失败，fallback TCP（端口 22 → 23 → 80）
	if !reachable && method != "snmp" {
		method = "tcp"
		checkPorts := []int{22, 23, 80}
		for _, port := range checkPorts {
			addr := fmt.Sprintf("%s:%d", mgmtIP, port)
			start := time.Now()
			conn, dialErr := net.DialTimeout("tcp", addr, 3*time.Second)
			elapsed := int(time.Since(start).Milliseconds())
			if dialErr == nil {
				conn.Close()
				reachable = true
				latencyMs = elapsed
				successPort = port
				errMsg = ""
				break
			}
			errMsg = dialErr.Error()
		}
	}

	insp := &model.NetworkInspection{
		CIInstanceID: dto.CIInstanceID,
		DeviceName:   inst.Name,
		MgmtIP:       mgmtIP,
		Reachable:    reachable,
		LatencyMs:    latencyMs,
		Port:         successPort,
		Method:       method,
		SysDescr:     sysDescr,
		SysUpTime:    sysUpTime,
		ErrorMsg:     errMsg,
		Operator:     operatorName,
		CreateTime:   util.HTime{Time: time.Now()},
	}

	if err := c.dao.CreateInspection(insp); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "保存巡检结果失败")
		return
	}

	// 连续 3 次不可达 → 写 critical 告警事件
	checkConsecutiveFailures(c.dao, dto.CIInstanceID, inst.Name, mgmtIP)

	result.Success(ctx, model.NetworkInspectionVo{
		ID:           insp.ID,
		CIInstanceID: insp.CIInstanceID,
		DeviceName:   insp.DeviceName,
		MgmtIP:       insp.MgmtIP,
		Reachable:    insp.Reachable,
		LatencyMs:    insp.LatencyMs,
		Port:         insp.Port,
		Method:       insp.Method,
		SysDescr:     insp.SysDescr,
		SysUpTime:    insp.SysUpTime,
		ErrorMsg:     insp.ErrorMsg,
		Operator:     insp.Operator,
		CreateTime:   insp.CreateTime,
	})
}

// probeSNMP 尝试 SNMP v2c 采集 sysDescr + sysUpTime，返回可达性、延迟、OID值
func probeSNMP(host, community string) (reachable bool, latencyMs int, sysDescr, sysUpTime, errMsg string) {
	gs := &gosnmp.GoSNMP{
		Target:    host,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   3 * time.Second,
		Retries:   1,
	}
	if err := gs.Connect(); err != nil {
		return false, -1, "", "", err.Error()
	}
	defer gs.Conn.Close()

	oids := []string{
		"1.3.6.1.2.1.1.1.0", // sysDescr
		"1.3.6.1.2.1.1.3.0", // sysUpTime
	}
	start := time.Now()
	resp, err := gs.Get(oids)
	elapsed := int(time.Since(start).Milliseconds())
	if err != nil {
		return false, -1, "", "", err.Error()
	}

	for _, pdu := range resp.Variables {
		switch pdu.Name {
		case ".1.3.6.1.2.1.1.1.0":
			sysDescr = gosnmp.ToBigInt(pdu.Value).String()
			if bs, ok := pdu.Value.([]byte); ok {
				sysDescr = string(bs)
			}
		case ".1.3.6.1.2.1.1.3.0":
			sysUpTime = fmt.Sprintf("%v", pdu.Value)
		}
	}

	return true, elapsed, sysDescr, sysUpTime, ""
}

// checkConsecutiveFailures 检查最近 3 次巡检是否全部不可达，是则写 critical 告警事件
func checkConsecutiveFailures(d dao.NetworkDeviceDao, ciInstanceID uint, deviceName, mgmtIP string) {
	const threshold = 3
	recent := d.GetLastNInspections(ciInstanceID, threshold)
	if len(recent) < threshold {
		return
	}
	for _, r := range recent {
		if r.Reachable {
			return // 有可达记录，不触发
		}
	}

	db := common.GetDB()
	if db == nil {
		return
	}
	db.Create(&n9emodel.AlertEvent{
		RuleName:  "网络设备连续不可达",
		AlertName: fmt.Sprintf("网络设备 [%s](%s) 连续 %d 次巡检不可达", deviceName, mgmtIP, threshold),
		Severity:  "critical",
		Status:    "firing",
		Labels: fmt.Sprintf(
			`{"ci_instance_id":"%d","device_name":"%s","mgmt_ip":"%s"}`,
			ciInstanceID, deviceName, mgmtIP,
		),
		Annotations: fmt.Sprintf(
			`{"summary":"设备 %s 最近 %d 次巡检均不可达，请检查网络或设备状态","mgmt_ip":"%s"}`,
			deviceName, threshold, mgmtIP,
		),
		StartsAt:     time.Now(),
		NotifyStatus: "pending",
	})
}

// GetInspectionHistory 获取指定设备的巡检历史
func (c *NetworkDeviceController) GetInspectionHistory(ctx *gin.Context) {
	ciIDStr := ctx.Query("ciId")
	ciID, err := strconv.ParseUint(ciIDStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetInspectionHistory(uint(ciID), page, pageSize)
	result.SuccessWithPage(ctx, list, total, page, pageSize)
}

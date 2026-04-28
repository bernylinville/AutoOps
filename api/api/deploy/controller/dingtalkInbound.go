package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"dodevops-api/api/system/dao"
	"dodevops-api/api/system/model"
	"dodevops-api/common/config"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/common/util"

	"github.com/gin-gonic/gin"
)

// DingtalkInbound dingtalk inbound webhook handler (feature-flag skeleton)
func DingtalkInbound(c *gin.Context) {
	if !config.Config.Integrations.Dingtalk.Inbound.Enabled {
		body, _ := io.ReadAll(c.Request.Body)
		size := len(body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		dao.CreateSysAuditLog(model.SysAuditLog{
			AdminId:     0,
			Username:    "dingtalk_robot",
			Module:      "dingtalk_inbound",
			OperType:    "其他",
			Method:      c.Request.Method,
			Url:         c.Request.URL.Path,
			RequestBody: "",
			StatusCode:  http.StatusServiceUnavailable,
			Duration:    0,
			Ip:          c.ClientIP(),
			Description: fmt.Sprintf("dingtalk_inbound_attempt size=%d ip=%s", size, c.ClientIP()),
			CreateTime:  util.HTime{Time: time.Now()},
		})

		result.FailedWithStatus(c, http.StatusServiceUnavailable,
			constant.INBOUND_DISABLED, "inbound-disabled",
			"dingtalk inbound integration is not enabled")
		return
	}

	// v2: HMAC verify + parse + delegate to CreateAgentDeployRequest
	result.FailedWithStatus(c, http.StatusNotImplemented, 500, "not-implemented", "dingtalk inbound v2 not yet implemented")
}

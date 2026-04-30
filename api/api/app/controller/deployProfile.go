package controller

import (
	"strconv"

	"dodevops-api/api/app/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

func (ac *ApplicationController) ListDeployProfiles(c *gin.Context) {
	appID, ok := parseUintParam(c, "id", "无效的应用ID")
	if !ok {
		return
	}
	ac.appService.ListDeployProfiles(c, appID)
}

func (ac *ApplicationController) CreateDeployProfile(c *gin.Context) {
	appID, ok := parseUintParam(c, "id", "无效的应用ID")
	if !ok {
		return
	}
	var req model.CreateAppDeployProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	ac.appService.CreateDeployProfile(c, appID, &req)
}

func (ac *ApplicationController) UpdateDeployProfile(c *gin.Context) {
	appID, ok := parseUintParam(c, "id", "无效的应用ID")
	if !ok {
		return
	}
	profileID, ok := parseUintParam(c, "profile_id", "无效的部署配置ID")
	if !ok {
		return
	}
	var req model.UpdateAppDeployProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		result.Failed(c, 400, "请求参数错误: "+err.Error())
		return
	}
	ac.appService.UpdateDeployProfile(c, appID, profileID, &req)
}

func (ac *ApplicationController) DeleteDeployProfile(c *gin.Context) {
	appID, ok := parseUintParam(c, "id", "无效的应用ID")
	if !ok {
		return
	}
	profileID, ok := parseUintParam(c, "profile_id", "无效的部署配置ID")
	if !ok {
		return
	}
	ac.appService.DeleteDeployProfile(c, appID, profileID)
}

func (ac *ApplicationController) ValidateDeployProfile(c *gin.Context) {
	appID, ok := parseUintParam(c, "id", "无效的应用ID")
	if !ok {
		return
	}
	profileID, ok := parseUintParam(c, "profile_id", "无效的部署配置ID")
	if !ok {
		return
	}
	ac.appService.ValidateDeployProfile(c, appID, profileID)
}

func parseUintParam(c *gin.Context, name, message string) (uint, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil {
		result.Failed(c, 400, message)
		return 0, false
	}
	return uint(id), true
}

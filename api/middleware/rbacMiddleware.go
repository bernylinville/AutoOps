package middleware

import (
	"dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/constant"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RbacMiddleware 返回 RBAC 权限校验中间件
// requiredPerm 为所需权限码，如 "cmdb:ecs:terminal"
func RbacMiddleware(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取当前用户
		userObj, exists := c.Get(constant.ContextKeyUserObj)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无法获取用户信息"})
			c.Abort()
			return
		}

		admin, ok := userObj.(*model.JwtAdmin)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "用户信息格式异常"})
			c.Abort()
			return
		}

		// 2. 查询/缓存用户权限
		perms := GetCachedPermissions(admin.ID)
		if perms == nil {
			// 直接查询数据库（避免 import dao 导致循环依赖）
			permList := queryUserPermissions(admin.ID)
			perms = make(map[string]bool, len(permList))
			for _, p := range permList {
				perms[p] = true
			}
			SetCachedPermissions(admin.ID, perms)
		}

		// 3. 校验权限
		if !perms[requiredPerm] {
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "权限不足: " + requiredPerm,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// queryUserPermissions 查询用户所有权限码（内部使用，避免跨包依赖）
func queryUserPermissions(adminID uint) []string {
	var perms []string
	common.GetDB().Table("sys_menu sm").
		Select("DISTINCT sm.value").
		Joins("JOIN sys_role_menu srm ON sm.id = srm.menu_id").
		Joins("JOIN sys_role sr ON sr.id = srm.role_id").
		Joins("JOIN sys_admin_role sar ON sar.role_id = sr.id").
		Where("sar.admin_id = ?", adminID).
		Where("sr.status = ?", 1).
		Where("sm.value != ''").
		Scan(&perms)
	return perms
}

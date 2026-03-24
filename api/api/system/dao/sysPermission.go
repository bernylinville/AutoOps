package dao

import (
	. "dodevops-api/pkg/db"
)

// QueryUserPermissions 查询用户所有权限码
// 通过 sys_admin_role → sys_role → sys_role_menu → sys_menu 链路查询
func QueryUserPermissions(adminID uint) []string {
	var perms []string
	Db.Table("sys_menu sm").
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

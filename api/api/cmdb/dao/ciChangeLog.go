package dao

import (
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ChangeLogDao struct {
	db *gorm.DB
}

func NewChangeLogDao() ChangeLogDao {
	return ChangeLogDao{db: common.GetDB()}
}

// CreateLogs 批量写入变更日志（空切片直接返回）
func (d *ChangeLogDao) CreateLogs(logs []model.CIChangeLog) error {
	if len(logs) == 0 {
		return nil
	}
	return d.db.Create(&logs).Error
}

// GetLogs 分页查询变更日志，entityType/entityID 为空时不过滤
func (d *ChangeLogDao) GetLogs(entityType string, entityID uint, page, pageSize int) ([]model.CIChangeLog, int64) {
	var list []model.CIChangeLog
	var total int64

	query := d.db.Model(&model.CIChangeLog{})
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if entityID > 0 {
		query = query.Where("entity_id = ?", entityID)
	}

	query.Count(&total)
	query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	return list, total
}

// SeedChangeLogMenu 初始化变更日志菜单（幂等，首次启动时插入 sys_menu）
func (d *ChangeLogDao) SeedChangeLogMenu() error {
	var existing systemmodel.SysMenu
	if d.db.Where("url = ?", "/cmdb/changelog").First(&existing).Error == nil {
		return nil
	}

	var parentMenu systemmodel.SysMenu
	if err := d.db.Where("menu_name = ? AND menu_type = ?", "资产管理", 1).First(&parentMenu).Error; err != nil {
		fmt.Println("SeedChangeLogMenu: 未找到'资产管理'父菜单，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}
	menu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "变更日志",
		Icon:       "Tickets",
		Value:      "cmdb:changelog:list",
		MenuType:   2,
		Url:        "/cmdb/changelog",
		MenuStatus: 2,
		Sort:       101,
		CreateTime: now,
	}
	if err := d.db.Create(&menu).Error; err != nil {
		return err
	}

	d.db.Create(&systemmodel.SysMenu{
		ParentId: menu.ID, MenuName: "查看日志", MenuType: 3,
		Value: "cmdb:changelog:list", MenuStatus: 2, Sort: 1, CreateTime: now,
	})

	fmt.Printf("SeedChangeLogMenu: 变更日志菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, menu.ID)
	return nil
}

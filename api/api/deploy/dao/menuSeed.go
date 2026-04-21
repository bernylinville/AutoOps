package dao

import (
	"fmt"
	"time"

	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/util"

	"gorm.io/gorm"
)

type MenuSeedDao struct {
	db *gorm.DB
}

func NewMenuSeedDao(db *gorm.DB) *MenuSeedDao {
	return &MenuSeedDao{db: db}
}

// SeedDeployMenu 初始化部署中心菜单与按钮权限（幂等）
func (d *MenuSeedDao) SeedDeployMenu() error {
	var existing systemmodel.SysMenu
	if err := d.db.Where("url = ?", "/k8s/release-center").First(&existing).Error; err == nil {
		return nil
	}

	parentNames := []string{"容器管理", "服务管理", "任务中心"}
	var parentMenu systemmodel.SysMenu
	found := false
	for _, name := range parentNames {
		if err := d.db.Where("menu_name = ? AND menu_type = ?", name, 1).First(&parentMenu).Error; err == nil {
			found = true
			break
		}
	}
	if !found {
		fmt.Println("SeedDeployMenu: 未找到合适的父菜单(容器管理/服务管理/任务中心)，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}
	deployMenu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "部署中心",
		Icon:       "Promotion",
		Value:      "deploy:request:view",
		MenuType:   2,
		Url:        "/k8s/release-center",
		MenuStatus: 2,
		Sort:       98,
		CreateTime: now,
	}
	if err := d.db.Create(&deployMenu).Error; err != nil {
		fmt.Printf("SeedDeployMenu: 创建菜单失败: %v\n", err)
		return err
	}

	buttons := []systemmodel.SysMenu{
		{ParentId: deployMenu.ID, MenuName: "查看部署申请", MenuType: 3, Value: "deploy:request:view", MenuStatus: 2, Sort: 1, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "创建部署申请", MenuType: 3, Value: "deploy:request:create", MenuStatus: 2, Sort: 2, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "审批部署申请", MenuType: 3, Value: "deploy:request:approve", MenuStatus: 2, Sort: 3, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "重发审批投递", MenuType: 3, Value: "deploy:request:approve", MenuStatus: 2, Sort: 4, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "同步审批状态", MenuType: 3, Value: "deploy:request:approve", MenuStatus: 2, Sort: 5, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "执行部署申请", MenuType: 3, Value: "deploy:request:execute", MenuStatus: 2, Sort: 6, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "清理Direct资源", MenuType: 3, Value: "deploy:request:execute", MenuStatus: 2, Sort: 7, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "查看部署目标", MenuType: 3, Value: "deploy:target:view", MenuStatus: 2, Sort: 8, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "创建部署目标", MenuType: 3, Value: "deploy:target:create", MenuStatus: 2, Sort: 9, CreateTime: now},
		{ParentId: deployMenu.ID, MenuName: "更新部署目标", MenuType: 3, Value: "deploy:target:edit", MenuStatus: 2, Sort: 10, CreateTime: now},
	}
	for _, btn := range buttons {
		if err := d.db.Create(&btn).Error; err != nil {
			fmt.Printf("SeedDeployMenu: 创建按钮权限 '%s' 失败: %v\n", btn.Value, err)
		}
	}

	fmt.Printf("SeedDeployMenu: 部署中心菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, deployMenu.ID)
	return nil
}

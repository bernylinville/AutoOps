package dao

import (
	"fmt"
	"time"

	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/util"

	"gorm.io/gorm"
)

// MenuSeedDAO 巡检菜单与按钮权限种子数据
type MenuSeedDAO struct {
	db *gorm.DB
}

// NewMenuSeedDAO creates a MenuSeedDAO.
func NewMenuSeedDAO(db *gorm.DB) *MenuSeedDAO {
	return &MenuSeedDAO{db: db}
}

// SeedInspectionMenu 初始化巡检中心菜单与按钮权限（幂等）
func (d *MenuSeedDAO) SeedInspectionMenu() error {
	var existing systemmodel.SysMenu
	if err := d.db.Where("url = ?", "/inspection/overview").First(&existing).Error; err == nil {
		return nil
	}

	// 寻找合适父菜单（容器管理 / 服务管理 / 任务中心 / 监控管理）
	parentNames := []string{"监控管理", "容器管理", "服务管理", "任务中心"}
	var parentMenu systemmodel.SysMenu
	found := false
	for _, name := range parentNames {
		if err := d.db.Where("menu_name = ? AND menu_type = ?", name, 1).First(&parentMenu).Error; err == nil {
			found = true
			break
		}
	}
	if !found {
		fmt.Println("SeedInspectionMenu: 未找到合适父菜单，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}
	inspectionMenu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "巡检中心",
		Icon:       "Search",
		Value:      "inspection:overview:view",
		MenuType:   2,
		Url:        "/inspection/overview",
		MenuStatus: 2,
		Sort:       99,
		CreateTime: now,
	}
	if err := d.db.Create(&inspectionMenu).Error; err != nil {
		fmt.Printf("SeedInspectionMenu: 创建菜单「巡检中心」失败: %v\n", err)
		return err
	}

	// 页面权限
	pages := []systemmodel.SysMenu{
		{ParentId: inspectionMenu.ID, MenuName: "巡检概览", MenuType: 2, Value: "inspection:overview:view", Url: "/inspection/overview", MenuStatus: 2, Sort: 1, CreateTime: now},
		{ParentId: inspectionMenu.ID, MenuName: "任务管理", MenuType: 2, Value: "inspection:task:view", Url: "/inspection/tasks", MenuStatus: 2, Sort: 2, CreateTime: now},
		{ParentId: inspectionMenu.ID, MenuName: "运行历史", MenuType: 2, Value: "inspection:run:view", Url: "/inspection/runs", MenuStatus: 2, Sort: 3, CreateTime: now},
	}
	for _, page := range pages {
		if err := d.db.Create(&page).Error; err != nil {
			fmt.Printf("SeedInspectionMenu: 创建页面菜单「%s」失败: %v\n", page.MenuName, err)
			continue
		}
	}

	// 按钮权限
	buttons := []systemmodel.SysMenu{
		{ParentId: inspectionMenu.ID, MenuName: "编辑巡检任务", MenuType: 3, Value: "inspection:task:edit", MenuStatus: 2, Sort: 10, CreateTime: now},
		{ParentId: inspectionMenu.ID, MenuName: "手动触发巡检", MenuType: 3, Value: "inspection:task:trigger", MenuStatus: 2, Sort: 11, CreateTime: now},
		{ParentId: inspectionMenu.ID, MenuName: "下载巡检报告", MenuType: 3, Value: "inspection:report:view", MenuStatus: 2, Sort: 12, CreateTime: now},
	}
	for _, btn := range buttons {
		if err := d.db.Create(&btn).Error; err != nil {
			fmt.Printf("SeedInspectionMenu: 创建按钮权限「%s」失败: %v\n", btn.Value, err)
			continue
		}
	}

	fmt.Printf("SeedInspectionMenu: 巡检中心菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, inspectionMenu.ID)
	return nil
}

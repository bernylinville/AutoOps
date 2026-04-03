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

type NetworkDeviceDao struct {
	db *gorm.DB
}

func NewNetworkDeviceDao() NetworkDeviceDao {
	return NetworkDeviceDao{db: common.GetDB()}
}

// GetNetworkDeviceInstances 分页获取 network_device 类型的 CI 实例
func (d *NetworkDeviceDao) GetNetworkDeviceInstances(page, pageSize int, keyword string) ([]model.CIInstance, int64) {
	var list []model.CIInstance
	var total int64

	// 先取 network_device 类型 ID
	var ciType model.CIType
	if err := d.db.Where("code = ?", "network_device").First(&ciType).Error; err != nil {
		return list, 0
	}

	query := d.db.Model(&model.CIInstance{}).Where("ci_type_id = ?", ciType.ID)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR attributes::text ILIKE ?", like, like)
	}

	query.Count(&total)
	query.Preload("CIType").Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list)

	return list, total
}

// GetLatestInspection 获取指定 CI 实例最近一次巡检记录
func (d *NetworkDeviceDao) GetLatestInspection(ciInstanceID uint) (*model.NetworkInspection, error) {
	var insp model.NetworkInspection
	err := d.db.Where("ci_instance_id = ?", ciInstanceID).Order("id DESC").First(&insp).Error
	if err != nil {
		return nil, err
	}
	return &insp, nil
}

// GetInspectionHistory 分页获取巡检历史
func (d *NetworkDeviceDao) GetInspectionHistory(ciInstanceID uint, page, pageSize int) ([]model.NetworkInspection, int64) {
	var list []model.NetworkInspection
	var total int64

	d.db.Model(&model.NetworkInspection{}).Where("ci_instance_id = ?", ciInstanceID).Count(&total)
	d.db.Where("ci_instance_id = ?", ciInstanceID).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)

	return list, total
}

// CreateInspection 写入巡检记录
func (d *NetworkDeviceDao) CreateInspection(insp *model.NetworkInspection) error {
	return d.db.Create(insp).Error
}

// GetLastNInspections 获取指定设备最近 n 条巡检记录（按 id DESC）
func (d *NetworkDeviceDao) GetLastNInspections(ciInstanceID uint, n int) []model.NetworkInspection {
	var list []model.NetworkInspection
	d.db.Where("ci_instance_id = ?", ciInstanceID).Order("id DESC").Limit(n).Find(&list)
	return list
}

// SeedNetworkMenu 初始化网络设备管理菜单（幂等，首次启动时插入 sys_menu）
func (d *NetworkDeviceDao) SeedNetworkMenu() error {
	var existing systemmodel.SysMenu
	if d.db.Where("url = ?", "/cmdb/network").First(&existing).Error == nil {
		return nil
	}

	var parentMenu systemmodel.SysMenu
	if err := d.db.Where("menu_name = ? AND menu_type = ?", "资产管理", 1).First(&parentMenu).Error; err != nil {
		fmt.Println("SeedNetworkMenu: 未找到'资产管理'父菜单，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}
	menu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "网络设备管理",
		Icon:       "Connection",
		Value:      "cmdb:network:list",
		MenuType:   2,
		Url:        "/cmdb/network",
		MenuStatus: 2,
		Sort:       102,
		CreateTime: now,
	}
	if err := d.db.Create(&menu).Error; err != nil {
		return err
	}

	buttons := []systemmodel.SysMenu{
		{ParentId: menu.ID, MenuName: "设备列表", MenuType: 3, Value: "cmdb:network:list", MenuStatus: 2, Sort: 1, CreateTime: now},
		{ParentId: menu.ID, MenuName: "巡检设备", MenuType: 3, Value: "cmdb:network:inspect", MenuStatus: 2, Sort: 2, CreateTime: now},
	}
	for _, btn := range buttons {
		d.db.Create(&btn)
	}

	fmt.Printf("SeedNetworkMenu: 网络设备管理菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, menu.ID)
	return nil
}

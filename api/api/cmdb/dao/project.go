// Phase 2: 项目维度管理 — DAO 层
package dao

import (
	"dodevops-api/api/cmdb/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type ProjectDao struct {
	db *gorm.DB
}

func NewProjectDao() ProjectDao {
	return ProjectDao{db: common.GetDB()}
}

// ========================================
// Project CRUD
// ========================================

// GetProjectList 分页获取项目列表（含资产计数）
func (d *ProjectDao) GetProjectList(page, pageSize int, keyword string) ([]model.ProjectVo, int64) {
	var projects []model.Project
	var total int64

	query := d.db.Model(&model.Project{})
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ?", like, like)
	}

	query.Count(&total)
	query.Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&projects)

	var vos []model.ProjectVo
	for _, p := range projects {
		vo := model.ProjectVo{
			ID:          p.ID,
			Name:        p.Name,
			Code:        p.Code,
			Description: p.Description,
			OwnerID:     p.OwnerID,
			Status:      p.Status,
			CreateTime:  p.CreateTime,
			UpdateTime:  p.UpdateTime,
		}
		d.db.Table("cmdb_host").Where("project_id = ?", p.ID).Count(&vo.HostCount)
		d.db.Table("cmdb_sql").Where("project_id = ?", p.ID).Count(&vo.DBCount)
		d.db.Table("app_application").Where("project_id = ?", p.ID).Count(&vo.AppCount)
		vos = append(vos, vo)
	}
	return vos, total
}

// GetProjectByID 根据ID获取项目
func (d *ProjectDao) GetProjectByID(id uint) (model.Project, error) {
	var p model.Project
	err := d.db.Where("id = ?", id).First(&p).Error
	return p, err
}

// CreateProject 创建项目
func (d *ProjectDao) CreateProject(p *model.Project) error {
	return d.db.Create(p).Error
}

// UpdateProject 更新项目
func (d *ProjectDao) UpdateProject(id uint, data map[string]interface{}) error {
	return d.db.Model(&model.Project{}).Where("id = ?", id).Updates(data).Error
}

// DeleteProject 删除项目（含关联资源检查）
func (d *ProjectDao) DeleteProject(id uint) (bool, error) {
	var hostCount, dbCount, appCount int64
	d.db.Table("cmdb_host").Where("project_id = ?", id).Count(&hostCount)
	d.db.Table("cmdb_sql").Where("project_id = ?", id).Count(&dbCount)
	d.db.Table("app_application").Where("project_id = ?", id).Count(&appCount)
	if hostCount+dbCount+appCount > 0 {
		return true, nil // hasResources = true
	}
	return false, d.db.Delete(&model.Project{}, id).Error
}

// CheckCodeExists 检查项目代码是否已存在
func (d *ProjectDao) CheckCodeExists(code string) bool {
	var count int64
	d.db.Model(&model.Project{}).Where("code = ?", code).Count(&count)
	return count > 0
}

// CheckCodeExistsExcept 检查代码是否被其他项目占用（更新时用）
func (d *ProjectDao) CheckCodeExistsExcept(code string, excludeID uint) bool {
	var count int64
	d.db.Model(&model.Project{}).Where("code = ? AND id != ?", code, excludeID).Count(&count)
	return count > 0
}

// ========================================
// 资产统计
// ========================================

// GetProjectStats 获取项目资产统计
func (d *ProjectDao) GetProjectStats(projectID uint) model.ProjectStatsVo {
	stats := model.ProjectStatsVo{ProjectID: projectID}

	d.db.Table("cmdb_host").Where("project_id = ?", projectID).Count(&stats.TotalHosts)
	d.db.Table("cmdb_host").Where("project_id = ? AND status = 1", projectID).Count(&stats.OnlineHosts)
	d.db.Table("cmdb_host").Where("project_id = ? AND status NOT IN (1,2)", projectID).Count(&stats.OfflineHosts)
	d.db.Table("cmdb_sql").Where("project_id = ?", projectID).Count(&stats.TotalDatabases)
	d.db.Table("app_application").Where("project_id = ?", projectID).Count(&stats.TotalApps)

	// 主机按分组分布
	d.db.Table("cmdb_host h").
		Select("h.group_id, COALESCE(g.name,'未分组') as group_name, COUNT(h.id) as count").
		Joins("LEFT JOIN cmdb_group g ON h.group_id = g.id").
		Where("h.project_id = ?", projectID).
		Group("h.group_id, g.name").
		Scan(&stats.HostsByGroup)

	// 数据库按类型分布
	d.db.Table("cmdb_sql").
		Select("type, COUNT(*) as count").
		Where("project_id = ?", projectID).
		Group("type").
		Scan(&stats.DBsByType)

	return stats
}

// ========================================
// 项目关联资产查询
// ========================================

// GetProjectHosts 获取项目关联主机列表（分页）
func (d *ProjectDao) GetProjectHosts(projectID uint, page, pageSize int) ([]model.CmdbHost, int64) {
	var list []model.CmdbHost
	var total int64
	d.db.Model(&model.CmdbHost{}).Where("project_id = ?", projectID).Count(&total)
	d.db.Model(&model.CmdbHost{}).Preload("Group").
		Where("project_id = ?", projectID).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)
	return list, total
}

// GetProjectDatabases 获取项目关联数据库列表（分页）
func (d *ProjectDao) GetProjectDatabases(projectID uint, page, pageSize int) ([]model.CmdbSQL, int64) {
	var list []model.CmdbSQL
	var total int64
	d.db.Model(&model.CmdbSQL{}).Where("project_id = ?", projectID).Count(&total)
	d.db.Model(&model.CmdbSQL{}).
		Where("project_id = ?", projectID).
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)
	return list, total
}

// GetProjectApps 获取项目关联应用列表
func (d *ProjectDao) GetProjectApps(projectID uint) ([]model.ProjectAppVo, error) {
	var apps []model.ProjectAppVo
	err := d.db.Table("app_application").
		Select("id, name, code, status, business_group_id, programming_lang").
		Where("project_id = ?", projectID).
		Scan(&apps).Error
	return apps, err
}

// ========================================
// 菜单初始化 Seed
// ========================================

// SeedProjectMenu 首次启动时自动初始化"项目管理"菜单项和按钮权限
func (d *ProjectDao) SeedProjectMenu() error {
	// 检查是否已经 seed 过（以 url = '/cmdb/project' 判断）
	var existing systemmodel.SysMenu
	err := d.db.Where("url = ?", "/cmdb/project").First(&existing).Error
	if err == nil {
		// 已存在，跳过
		return nil
	}

	// 查找"资产管理"目录菜单（MenuType = 1）
	var parentMenu systemmodel.SysMenu
	err = d.db.Where("menu_name = ? AND menu_type = ?", "资产管理", 1).First(&parentMenu).Error
	if err != nil {
		fmt.Println("SeedProjectMenu: 未找到'资产管理'父菜单，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}

	// 插入菜单项
	projectMenu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "项目管理",
		Icon:       "FolderOpened",
		Value:      "cmdb:project:list",
		MenuType:   2,
		Url:        "/cmdb/project",
		MenuStatus: 2,
		Sort:       99,
		CreateTime: now,
	}
	if err := d.db.Create(&projectMenu).Error; err != nil {
		fmt.Printf("SeedProjectMenu: 创建菜单失败: %v\n", err)
		return err
	}

	// 插入按钮权限
	buttons := []systemmodel.SysMenu{
		{ParentId: projectMenu.ID, MenuName: "项目列表", MenuType: 3, Value: "cmdb:project:list", MenuStatus: 2, Sort: 1, CreateTime: now},
		{ParentId: projectMenu.ID, MenuName: "创建项目", MenuType: 3, Value: "cmdb:project:create", MenuStatus: 2, Sort: 2, CreateTime: now},
		{ParentId: projectMenu.ID, MenuName: "更新项目", MenuType: 3, Value: "cmdb:project:update", MenuStatus: 2, Sort: 3, CreateTime: now},
		{ParentId: projectMenu.ID, MenuName: "删除项目", MenuType: 3, Value: "cmdb:project:delete", MenuStatus: 2, Sort: 4, CreateTime: now},
	}
	for _, btn := range buttons {
		if err := d.db.Create(&btn).Error; err != nil {
			fmt.Printf("SeedProjectMenu: 创建按钮权限 '%s' 失败: %v\n", btn.Value, err)
		}
	}

	fmt.Printf("SeedProjectMenu: 项目管理菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, projectMenu.ID)
	return nil
}

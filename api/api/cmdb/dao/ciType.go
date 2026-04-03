package dao

import (
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common"
	"dodevops-api/common/util"
	systemmodel "dodevops-api/api/system/model"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type CITypeDao struct {
	db *gorm.DB
}

func NewCITypeDao() CITypeDao {
	return CITypeDao{db: common.GetDB()}
}

// ========================================
// CI Type CRUD
// ========================================

func (d *CITypeDao) GetCITypeList() ([]model.CIType, error) {
	var list []model.CIType
	err := d.db.Order("sort_order ASC, id ASC").Find(&list).Error
	return list, err
}

func (d *CITypeDao) GetCITypeByID(id uint) (model.CIType, error) {
	var t model.CIType
	err := d.db.Preload("Attributes", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Where("id = ?", id).First(&t).Error
	return t, err
}

func (d *CITypeDao) GetCITypeByCode(code string) (model.CIType, error) {
	var t model.CIType
	err := d.db.Where("code = ?", code).First(&t).Error
	return t, err
}

func (d *CITypeDao) CreateCIType(t *model.CIType) error {
	return d.db.Create(t).Error
}

func (d *CITypeDao) UpdateCIType(id uint, data map[string]interface{}) error {
	return d.db.Model(&model.CIType{}).Where("id = ?", id).Updates(data).Error
}

func (d *CITypeDao) DeleteCIType(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 删除所有属性
		if err := tx.Where("ci_type_id = ?", id).Delete(&model.CITypeAttribute{}).Error; err != nil {
			return err
		}
		// 删除类型
		return tx.Delete(&model.CIType{}, id).Error
	})
}

func (d *CITypeDao) GetInstanceCountByTypeID(typeID uint) int64 {
	var count int64
	d.db.Model(&model.CIInstance{}).Where("ci_type_id = ?", typeID).Count(&count)
	return count
}

func (d *CITypeDao) CheckCodeExists(code string) bool {
	var count int64
	d.db.Model(&model.CIType{}).Where("code = ?", code).Count(&count)
	return count > 0
}

// ========================================
// CI Type Attribute CRUD
// ========================================

func (d *CITypeDao) GetAttributesByTypeID(typeID uint) ([]model.CITypeAttribute, error) {
	var list []model.CITypeAttribute
	err := d.db.Where("ci_type_id = ?", typeID).Order("sort_order ASC").Find(&list).Error
	return list, err
}

func (d *CITypeDao) GetAttributeByID(id uint) (model.CITypeAttribute, error) {
	var attr model.CITypeAttribute
	err := d.db.Where("id = ?", id).First(&attr).Error
	return attr, err
}

func (d *CITypeDao) CreateAttribute(attr *model.CITypeAttribute) error {
	return d.db.Create(attr).Error
}

func (d *CITypeDao) UpdateAttribute(id uint, data map[string]interface{}) error {
	return d.db.Model(&model.CITypeAttribute{}).Where("id = ?", id).Updates(data).Error
}

func (d *CITypeDao) DeleteAttribute(id uint) error {
	return d.db.Delete(&model.CITypeAttribute{}, id).Error
}

func (d *CITypeDao) CheckAttrCodeExists(typeID uint, code string) bool {
	var count int64
	d.db.Model(&model.CITypeAttribute{}).Where("ci_type_id = ? AND code = ?", typeID, code).Count(&count)
	return count > 0
}

// ========================================
// CI Instance CRUD
// ========================================

func (d *CITypeDao) GetCIInstanceList(typeID uint, page, pageSize int, keyword string) ([]model.CIInstance, int64) {
	var list []model.CIInstance
	var total int64

	query := d.db.Model(&model.CIInstance{}).Where("ci_type_id = ?", typeID)
	if keyword != "" {
		like := "%" + keyword + "%"
		// 搜索名称 + JSONB 属性值
		query = query.Where("name ILIKE ? OR attributes::text ILIKE ?", like, like)
	}

	query.Count(&total)
	query.Preload("CIType").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list)

	return list, total
}

func (d *CITypeDao) GetCIInstanceByID(id uint) (model.CIInstance, error) {
	var inst model.CIInstance
	err := d.db.Preload("CIType").Where("id = ?", id).First(&inst).Error
	return inst, err
}

func (d *CITypeDao) CreateCIInstance(inst *model.CIInstance) error {
	return d.db.Create(inst).Error
}

func (d *CITypeDao) UpdateCIInstance(id uint, data map[string]interface{}) error {
	return d.db.Model(&model.CIInstance{}).Where("id = ?", id).Updates(data).Error
}

func (d *CITypeDao) DeleteCIInstance(id uint) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		// 删除相关关系
		if err := tx.Where("from_ci_id = ? OR to_ci_id = ?", id, id).Delete(&model.CIRelation{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.CIInstance{}, id).Error
	})
}

func (d *CITypeDao) GetAllCIInstances() ([]model.CIInstance, error) {
	var list []model.CIInstance
	err := d.db.Preload("CIType").Find(&list).Error
	return list, err
}

// ========================================
// CI Instance JSONB 查询
// ========================================

// QueryByAttribute 按动态属性查询：WHERE attributes->>'key' = 'value'
func (d *CITypeDao) QueryByAttribute(typeID uint, attrCode string, value string) ([]model.CIInstance, error) {
	var list []model.CIInstance
	err := d.db.Where("ci_type_id = ? AND attributes->>? = ?", typeID, attrCode, value).Find(&list).Error
	return list, err
}

// ========================================
// CI Relation CRUD
// ========================================

func (d *CITypeDao) GetRelationsByCIID(ciID uint) ([]model.CIRelation, error) {
	var list []model.CIRelation
	err := d.db.Where("from_ci_id = ? OR to_ci_id = ?", ciID, ciID).Find(&list).Error
	return list, err
}

func (d *CITypeDao) CreateRelation(rel *model.CIRelation) error {
	return d.db.Create(rel).Error
}

func (d *CITypeDao) DeleteRelation(id uint) error {
	return d.db.Delete(&model.CIRelation{}, id).Error
}

func (d *CITypeDao) CheckRelationExists(fromID, toID uint, relType string) bool {
	var count int64
	d.db.Model(&model.CIRelation{}).Where("from_ci_id = ? AND to_ci_id = ? AND relation_type = ?", fromID, toID, relType).Count(&count)
	return count > 0
}

// ========================================
// CI 拓扑图（Phase 3）
// ========================================

type nodeIDRow struct {
	ID uint `gorm:"column:id"`
}

// GetCITopology 用 PostgreSQL WITH RECURSIVE 递归查询 CI 拓扑
// direction: "down"（向下追依赖）| "up"（向上追来源）| "all"（双向）
func (d *CITypeDao) GetCITopology(rootCIID uint, direction string) (model.CITopologyVo, error) {
	// WITH RECURSIVE：从根节点出发，沿 from_ci_id→to_ci_id 方向展开
	downCTE := `
WITH RECURSIVE topology AS (
    SELECT id FROM ci_instance WHERE id = ?
    UNION
    SELECT ci.id
    FROM ci_instance ci
    JOIN ci_relation r ON r.to_ci_id = ci.id
    JOIN topology t ON t.id = r.from_ci_id
)
SELECT id FROM topology`

	// WITH RECURSIVE：从根节点出发，沿 to_ci_id→from_ci_id 方向回溯
	upCTE := `
WITH RECURSIVE topology AS (
    SELECT id FROM ci_instance WHERE id = ?
    UNION
    SELECT ci.id
    FROM ci_instance ci
    JOIN ci_relation r ON r.from_ci_id = ci.id
    JOIN topology t ON t.id = r.to_ci_id
)
SELECT id FROM topology`

	idSet := make(map[uint]struct{})

	collectIDs := func(sql string) error {
		var rows []nodeIDRow
		if err := d.db.Raw(sql, rootCIID).Scan(&rows).Error; err != nil {
			return err
		}
		for _, r := range rows {
			idSet[r.ID] = struct{}{}
		}
		return nil
	}

	switch direction {
	case "down":
		if err := collectIDs(downCTE); err != nil {
			return model.CITopologyVo{}, err
		}
	case "up":
		if err := collectIDs(upCTE); err != nil {
			return model.CITopologyVo{}, err
		}
	default: // "all"
		if err := collectIDs(downCTE); err != nil {
			return model.CITopologyVo{}, err
		}
		if err := collectIDs(upCTE); err != nil {
			return model.CITopologyVo{}, err
		}
	}

	// 至少根节点本身
	idSet[rootCIID] = struct{}{}

	nodeIDs := make([]uint, 0, len(idSet))
	for id := range idSet {
		nodeIDs = append(nodeIDs, id)
	}

	// 批量获取实例详情（含 CI 类型）
	var instances []model.CIInstance
	if err := d.db.Preload("CIType").Where("id IN ?", nodeIDs).Find(&instances).Error; err != nil {
		return model.CITopologyVo{}, err
	}

	nodes := make([]model.TopologyNode, 0, len(instances))
	for _, inst := range instances {
		nodes = append(nodes, model.TopologyNode{
			ID:       inst.ID,
			Name:     inst.Name,
			CITypeID: inst.CITypeID,
			TypeName: inst.CIType.Name,
			TypeIcon: inst.CIType.Icon,
			TypeCode: inst.CIType.Code,
			Status:   inst.Status,
			IsRoot:   inst.ID == rootCIID,
		})
	}

	// 获取节点集合内部的所有关系
	var relations []model.CIRelation
	if err := d.db.Where("from_ci_id IN ? AND to_ci_id IN ?", nodeIDs, nodeIDs).Find(&relations).Error; err != nil {
		return model.CITopologyVo{}, err
	}

	edges := make([]model.TopologyEdge, 0, len(relations))
	for _, rel := range relations {
		edges = append(edges, model.TopologyEdge{
			ID:           rel.ID,
			FromCIID:     rel.FromCIID,
			ToCIID:       rel.ToCIID,
			RelationType: rel.RelationType,
		})
	}

	return model.CITopologyVo{Nodes: nodes, Edges: edges}, nil
}

// ========================================
// CI 预置类型初始化
// ========================================

func (d *CITypeDao) SeedBuiltinTypes() error {
	builtinTypes := []struct {
		Name     string
		Code     string
		Icon     string
		Category string
		Desc     string
		Attrs    []struct {
			Name     string
			Code     string
			DataType string
			Required bool
			ShowList bool
			Search   bool
			Sort     int
		}
	}{
		{
			Name: "服务器", Code: "server", Icon: "Monitor", Category: "server",
			Desc: "物理服务器或虚拟机",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "IP地址", Code: "ip_address", DataType: "ip", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "操作系统", Code: "os", DataType: "string", Required: false, ShowList: true, Search: false, Sort: 2},
				{Name: "CPU", Code: "cpu", DataType: "string", Required: false, ShowList: true, Search: false, Sort: 3},
				{Name: "内存(GB)", Code: "memory_gb", DataType: "integer", Required: false, ShowList: true, Search: false, Sort: 4},
				{Name: "磁盘(GB)", Code: "disk_gb", DataType: "integer", Required: false, ShowList: true, Search: false, Sort: 5},
				{Name: "机房位置", Code: "location", DataType: "string", Required: false, ShowList: false, Search: true, Sort: 6},
				{Name: "序列号", Code: "serial_number", DataType: "string", Required: false, ShowList: false, Search: true, Sort: 7},
			},
		},
		{
			Name: "数据库", Code: "database", Icon: "Coin", Category: "database",
			Desc: "关系型或NoSQL数据库实例",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "数据库类型", Code: "db_type", DataType: "enum", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "版本", Code: "version", DataType: "string", Required: false, ShowList: true, Search: false, Sort: 2},
				{Name: "端口", Code: "port", DataType: "integer", Required: true, ShowList: true, Search: false, Sort: 3},
				{Name: "连接地址", Code: "host", DataType: "string", Required: true, ShowList: true, Search: true, Sort: 4},
				{Name: "字符集", Code: "charset", DataType: "string", Required: false, ShowList: false, Search: false, Sort: 5},
			},
		},
		{
			Name: "网络设备", Code: "network_device", Icon: "Connection", Category: "network",
			Desc: "交换机、路由器、防火墙等",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "管理IP", Code: "mgmt_ip", DataType: "ip", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "设备类型", Code: "device_type", DataType: "enum", Required: true, ShowList: true, Search: true, Sort: 2},
				{Name: "品牌", Code: "brand", DataType: "string", Required: false, ShowList: true, Search: true, Sort: 3},
				{Name: "型号", Code: "model", DataType: "string", Required: false, ShowList: true, Search: false, Sort: 4},
				{Name: "固件版本", Code: "firmware", DataType: "string", Required: false, ShowList: false, Search: false, Sort: 5},
				{Name: "机柜位置", Code: "rack_location", DataType: "string", Required: false, ShowList: false, Search: true, Sort: 6},
			},
		},
		{
			Name: "中间件", Code: "middleware", Icon: "SetUp", Category: "middleware",
			Desc: "消息队列、缓存、Web服务器等",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "类型", Code: "mw_type", DataType: "enum", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "版本", Code: "version", DataType: "string", Required: false, ShowList: true, Search: false, Sort: 2},
				{Name: "端口", Code: "port", DataType: "integer", Required: false, ShowList: true, Search: false, Sort: 3},
				{Name: "部署地址", Code: "deploy_host", DataType: "string", Required: true, ShowList: true, Search: true, Sort: 4},
				{Name: "集群模式", Code: "cluster_mode", DataType: "enum", Required: false, ShowList: true, Search: false, Sort: 5},
			},
		},
		{
			Name: "存储", Code: "storage", Icon: "Files", Category: "storage",
			Desc: "NAS、SAN、对象存储等",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "存储类型", Code: "storage_type", DataType: "enum", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "容量(TB)", Code: "capacity_tb", DataType: "float", Required: false, ShowList: true, Search: false, Sort: 2},
				{Name: "管理地址", Code: "mgmt_url", DataType: "url", Required: false, ShowList: true, Search: true, Sort: 3},
				{Name: "品牌型号", Code: "brand_model", DataType: "string", Required: false, ShowList: true, Search: true, Sort: 4},
			},
		},
		{
			Name: "负载均衡", Code: "load_balancer", Icon: "Share", Category: "network",
			Desc: "F5/Nginx/HAProxy等负载均衡设备",
			Attrs: []struct {
				Name     string
				Code     string
				DataType string
				Required bool
				ShowList bool
				Search   bool
				Sort     int
			}{
				{Name: "VIP地址", Code: "vip", DataType: "ip", Required: true, ShowList: true, Search: true, Sort: 1},
				{Name: "类型", Code: "lb_type", DataType: "enum", Required: true, ShowList: true, Search: true, Sort: 2},
				{Name: "后端节点数", Code: "backend_count", DataType: "integer", Required: false, ShowList: true, Search: false, Sort: 3},
				{Name: "管理地址", Code: "mgmt_url", DataType: "url", Required: false, ShowList: false, Search: false, Sort: 4},
			},
		},
	}

	for i, bt := range builtinTypes {
		// 检查是否已存在
		if d.CheckCodeExists(bt.Code) {
			continue
		}

		// 设置枚举选项
		var attrs []model.CITypeAttribute
		for _, a := range bt.Attrs {
			attr := model.CITypeAttribute{
				Name:       a.Name,
				Code:       a.Code,
				DataType:   a.DataType,
				Required:   a.Required,
				ShowInList: a.ShowList,
				Searchable: a.Search,
				SortOrder:  a.Sort,
				CreateTime: util.HTime{Time: time.Now()},
			}

			// 为 enum 类型设置默认选项
			if a.DataType == "enum" {
				var options []string
				switch a.Code {
				case "db_type":
					options = []string{"MySQL", "PostgreSQL", "MongoDB", "Redis", "Oracle", "SQL Server", "MariaDB"}
				case "device_type":
					options = []string{"交换机", "路由器", "防火墙", "AP", "其他"}
				case "mw_type":
					options = []string{"Nginx", "Tomcat", "RabbitMQ", "Kafka", "Redis", "Elasticsearch", "Zookeeper"}
				case "cluster_mode":
					options = []string{"单机", "主从", "集群", "哨兵"}
				case "storage_type":
					options = []string{"NAS", "SAN", "对象存储", "块存储", "分布式存储"}
				case "lb_type":
					options = []string{"F5", "Nginx", "HAProxy", "LVS", "云SLB"}
				}
				if len(options) > 0 {
					optJSON, _ := json.Marshal(options)
					attr.EnumOptions = optJSON
				}
			}
			attrs = append(attrs, attr)
		}

		now := util.HTime{Time: time.Now()}
		ciType := model.CIType{
			Name:        bt.Name,
			Code:        bt.Code,
			Icon:        bt.Icon,
			Category:    bt.Category,
			Description: bt.Desc,
			BuiltIn:     true,
			Enabled:     true,
			SortOrder:   i + 1,
			CreateTime:  now,
			UpdateTime:  now,
			Attributes:  attrs,
		}

		if err := d.db.Create(&ciType).Error; err != nil {
			fmt.Printf("Seed CI type '%s' failed: %v\n", bt.Code, err)
			continue
		}
		fmt.Printf("Seeded CI type: %s (%d attributes)\n", bt.Name, len(attrs))
	}

	return nil
}

// SeedSNMPCommunityAttr 幂等地为 network_device CI 类型补充 snmp_community 属性
func (d *CITypeDao) SeedSNMPCommunityAttr() error {
	var ciType model.CIType
	if err := d.db.Where("code = ?", "network_device").First(&ciType).Error; err != nil {
		// 类型不存在，等下次 SeedBuiltinTypes 处理
		return nil
	}

	var existing model.CITypeAttribute
	if d.db.Where("ci_type_id = ? AND code = ?", ciType.ID, "snmp_community").First(&existing).Error == nil {
		return nil // already seeded
	}

	attr := model.CITypeAttribute{
		CITypeID:   ciType.ID,
		Name:       "SNMP社区字符串",
		Code:       "snmp_community",
		DataType:   "string",
		Required:   false,
		ShowInList: false,
		Searchable: false,
		SortOrder:  7,
		CreateTime: util.HTime{Time: time.Now()},
	}
	if err := d.db.Create(&attr).Error; err != nil {
		return err
	}
	fmt.Printf("SeedSNMPCommunityAttr: network_device 新增 snmp_community 属性 (ciTypeID=%d)\n", ciType.ID)
	return nil
}

// SeedTopologyMenu 初始化 CI 拓扑图菜单（幂等，首次启动时插入 sys_menu）
func (d *CITypeDao) SeedTopologyMenu() error {
	var existing systemmodel.SysMenu
	if d.db.Where("url = ?", "/cmdb/ci/topology").First(&existing).Error == nil {
		return nil
	}

	var parentMenu systemmodel.SysMenu
	if err := d.db.Where("menu_name = ? AND menu_type = ?", "资产管理", 1).First(&parentMenu).Error; err != nil {
		fmt.Println("SeedTopologyMenu: 未找到'资产管理'父菜单，跳过")
		return nil
	}

	now := util.HTime{Time: time.Now()}
	menu := systemmodel.SysMenu{
		ParentId:   parentMenu.ID,
		MenuName:   "CI拓扑图",
		Icon:       "Share",
		Value:      "cmdb:ci:topology:list",
		MenuType:   2,
		Url:        "/cmdb/ci/topology",
		MenuStatus: 2,
		Sort:       100,
		CreateTime: now,
	}
	if err := d.db.Create(&menu).Error; err != nil {
		return err
	}

	d.db.Create(&systemmodel.SysMenu{
		ParentId: menu.ID, MenuName: "查看拓扑", MenuType: 3,
		Value: "cmdb:ci:topology:list", MenuStatus: 2, Sort: 1, CreateTime: now,
	})

	fmt.Printf("SeedTopologyMenu: CI拓扑图菜单初始化完成 (parentID=%d, menuID=%d)\n", parentMenu.ID, menu.ID)
	return nil
}

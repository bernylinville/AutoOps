package controller

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"dodevops-api/common/result"
	"dodevops-api/common/util"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type CITypeController struct {
	dao       dao.CITypeDao
	changeLog dao.ChangeLogDao
}

func NewCITypeController() *CITypeController {
	return &CITypeController{
		dao:       dao.NewCITypeDao(),
		changeLog: dao.NewChangeLogDao(),
	}
}

// ========================================
// CI Type APIs
// ========================================

// GetCITypeList 获取所有 CI 类型
func (c *CITypeController) GetCITypeList(ctx *gin.Context) {
	list, err := c.dao.GetCITypeList()
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "获取CI类型列表失败")
		return
	}

	var vos []model.CITypeVo
	for _, t := range list {
		vos = append(vos, model.CITypeVo{
			ID:             t.ID,
			Name:           t.Name,
			Code:           t.Code,
			Icon:           t.Icon,
			Category:       t.Category,
			Description:    t.Description,
			BuiltIn:        t.BuiltIn,
			Enabled:        t.Enabled,
			SortOrder:      t.SortOrder,
			InstanceCount:  c.dao.GetInstanceCountByTypeID(t.ID),
		})
	}
	result.Success(ctx, vos)
}

// GetCITypeDetail 获取 CI 类型详情（含属性）
func (c *CITypeController) GetCITypeDetail(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	t, err := c.dao.GetCITypeByID(uint(id))
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI类型不存在")
		return
	}
	result.Success(ctx, t)
}

// CreateCIType 创建 CI 类型
func (c *CITypeController) CreateCIType(ctx *gin.Context) {
	var dto model.CreateCITypeDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	if c.dao.CheckCodeExists(dto.Code) {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "类型代码已存在")
		return
	}

	ciType := model.CIType{
		Name:        dto.Name,
		Code:        dto.Code,
		Icon:        dto.Icon,
		Category:    dto.Category,
		Description: dto.Description,
		SortOrder:   dto.SortOrder,
		Enabled:     true,
		CreateTime:  util.HTime{Time: time.Now()},
		UpdateTime:  util.HTime{Time: time.Now()},
	}

	if err := c.dao.CreateCIType(&ciType); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "创建CI类型失败: "+err.Error())
		return
	}
	result.Success(ctx, ciType)
}

// UpdateCIType 更新 CI 类型
func (c *CITypeController) UpdateCIType(ctx *gin.Context) {
	var dto model.UpdateCITypeDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 检查内置类型不允许修改 code
	existing, err := c.dao.GetCITypeByID(dto.ID)
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI类型不存在")
		return
	}

	data := map[string]interface{}{
		"name":        dto.Name,
		"icon":        dto.Icon,
		"category":    dto.Category,
		"description": dto.Description,
		"sort_order":  dto.SortOrder,
		"update_time": time.Now(),
	}

	if dto.Enabled != nil {
		data["enabled"] = *dto.Enabled
	}

	_ = existing // 保留引用

	if err := c.dao.UpdateCIType(dto.ID, data); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "更新CI类型失败")
		return
	}
	result.Success(ctx, true)
}

// DeleteCIType 删除 CI 类型
func (c *CITypeController) DeleteCIType(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 检查内置类型
	t, err := c.dao.GetCITypeByID(uint(id))
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI类型不存在")
		return
	}
	if t.BuiltIn {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "内置类型不允许删除")
		return
	}

	// 检查是否有实例
	count := c.dao.GetInstanceCountByTypeID(uint(id))
	if count > 0 {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "该类型下存在CI实例，无法删除")
		return
	}

	if err := c.dao.DeleteCIType(uint(id)); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "删除CI类型失败")
		return
	}
	result.Success(ctx, true)
}

// ========================================
// CI Type Attribute APIs
// ========================================

// GetCITypeAttributes 获取类型的属性列表
func (c *CITypeController) GetCITypeAttributes(ctx *gin.Context) {
	typeIDStr := ctx.Query("ciTypeId")
	typeID, err := strconv.ParseUint(typeIDStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	list, err := c.dao.GetAttributesByTypeID(uint(typeID))
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "获取属性列表失败")
		return
	}
	result.Success(ctx, list)
}

// CreateCITypeAttribute 创建属性
func (c *CITypeController) CreateCITypeAttribute(ctx *gin.Context) {
	var dto model.CreateCITypeAttributeDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 检查 CI 类型是否存在
	if _, err := c.dao.GetCITypeByID(dto.CITypeID); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI类型不存在")
		return
	}

	// 检查属性代码是否重复
	if c.dao.CheckAttrCodeExists(dto.CITypeID, dto.Code) {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "属性代码已存在")
		return
	}

	attr := model.CITypeAttribute{
		CITypeID:     dto.CITypeID,
		Name:         dto.Name,
		Code:         dto.Code,
		DataType:     dto.DataType,
		Required:     dto.Required,
		DefaultValue: dto.DefaultValue,
		Placeholder:  dto.Placeholder,
		SortOrder:    dto.SortOrder,
		ShowInList:   dto.ShowInList,
		Searchable:   dto.Searchable,
		CreateTime:   util.HTime{Time: time.Now()},
	}

	// 处理枚举选项
	if dto.EnumOptions != "" {
		attr.EnumOptions = datatypes.JSON(dto.EnumOptions)
	}

	if err := c.dao.CreateAttribute(&attr); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "创建属性失败: "+err.Error())
		return
	}
	result.Success(ctx, attr)
}

// UpdateCITypeAttribute 更新属性
func (c *CITypeController) UpdateCITypeAttribute(ctx *gin.Context) {
	var dto model.UpdateCITypeAttributeDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	data := map[string]interface{}{
		"name":       dto.Name,
		"data_type":  dto.DataType,
		"default_value": dto.DefaultValue,
		"placeholder": dto.Placeholder,
		"sort_order": dto.SortOrder,
	}
	if dto.Required != nil {
		data["required"] = *dto.Required
	}
	if dto.ShowInList != nil {
		data["show_in_list"] = *dto.ShowInList
	}
	if dto.Searchable != nil {
		data["searchable"] = *dto.Searchable
	}
	if dto.EnumOptions != "" {
		data["enum_options"] = datatypes.JSON(dto.EnumOptions)
	}

	if err := c.dao.UpdateAttribute(dto.ID, data); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "更新属性失败")
		return
	}
	result.Success(ctx, true)
}

// DeleteCITypeAttribute 删除属性
func (c *CITypeController) DeleteCITypeAttribute(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	if err := c.dao.DeleteAttribute(uint(id)); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "删除属性失败")
		return
	}
	result.Success(ctx, true)
}

// ========================================
// CI Instance APIs
// ========================================

// GetCIInstanceList 获取 CI 实例列表（分页）
func (c *CITypeController) GetCIInstanceList(ctx *gin.Context) {
	typeIDStr := ctx.Query("ciTypeId")
	typeID, err := strconv.ParseUint(typeIDStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "CI类型ID参数错误")
		return
	}

	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	keyword := ctx.Query("keyword")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	list, total := c.dao.GetCIInstanceList(uint(typeID), page, pageSize, keyword)

	var vos []model.CIInstanceVo
	for _, inst := range list {
		vo := model.CIInstanceVo{
			ID:         inst.ID,
			CITypeID:   inst.CITypeID,
			Name:       inst.Name,
			Status:     inst.Status,
			GroupID:    inst.GroupID,
			Remark:     inst.Remark,
			CreateTime: inst.CreateTime,
			UpdateTime: inst.UpdateTime,
		}

		if inst.CIType.ID > 0 {
			vo.TypeName = inst.CIType.Name
			vo.TypeCode = inst.CIType.Code
			vo.TypeIcon = inst.CIType.Icon
		}

		// 解析 JSONB 属性
		if inst.Attributes != nil {
			var attrs map[string]interface{}
			if err := json.Unmarshal(inst.Attributes, &attrs); err == nil {
				vo.Attributes = attrs
			}
		}

		vos = append(vos, vo)
	}

	result.SuccessWithPage(ctx, vos, total, page, pageSize)
}

// GetCIInstanceDetail 获取 CI 实例详情
func (c *CITypeController) GetCIInstanceDetail(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	inst, err := c.dao.GetCIInstanceByID(uint(id))
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI实例不存在")
		return
	}

	vo := model.CIInstanceVo{
		ID:         inst.ID,
		CITypeID:   inst.CITypeID,
		TypeName:   inst.CIType.Name,
		TypeCode:   inst.CIType.Code,
		TypeIcon:   inst.CIType.Icon,
		Name:       inst.Name,
		Status:     inst.Status,
		GroupID:    inst.GroupID,
		Remark:     inst.Remark,
		CreateTime: inst.CreateTime,
		UpdateTime: inst.UpdateTime,
	}

	if inst.Attributes != nil {
		var attrs map[string]interface{}
		if err := json.Unmarshal(inst.Attributes, &attrs); err == nil {
			vo.Attributes = attrs
		}
	}

	result.Success(ctx, vo)
}

// CreateCIInstance 创建 CI 实例
func (c *CITypeController) CreateCIInstance(ctx *gin.Context) {
	var dto model.CreateCIInstanceDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 验证 CI 类型存在
	ciType, err := c.dao.GetCITypeByID(dto.CITypeID)
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI类型不存在")
		return
	}

	// 验证必填属性
	attrs, _ := c.dao.GetAttributesByTypeID(dto.CITypeID)
	for _, attr := range attrs {
		if attr.Required {
			if dto.Attributes == nil {
				result.Failed(ctx, constant.INVALID_PARAMS, "缺少必填属性: "+attr.Name)
				return
			}
			if _, ok := dto.Attributes[attr.Code]; !ok {
				result.Failed(ctx, constant.INVALID_PARAMS, "缺少必填属性: "+attr.Name)
				return
			}
		}
	}

	// 序列化属性
	var attrJSON datatypes.JSON
	if dto.Attributes != nil {
		j, err := json.Marshal(dto.Attributes)
		if err != nil {
			result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "属性数据格式错误")
			return
		}
		attrJSON = j
	}

	status := dto.Status
	if status == 0 {
		status = 1 // 默认运行中
	}

	inst := model.CIInstance{
		CITypeID:   dto.CITypeID,
		Name:       dto.Name,
		Status:     status,
		Attributes: attrJSON,
		GroupID:    dto.GroupID,
		Remark:     dto.Remark,
		CreateTime: util.HTime{Time: time.Now()},
		UpdateTime: util.HTime{Time: time.Now()},
	}

	if err := c.dao.CreateCIInstance(&inst); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "创建CI实例失败: "+err.Error())
		return
	}

	_ = ciType // 保留引用
	result.Success(ctx, gin.H{"id": inst.ID, "name": inst.Name})
}

// UpdateCIInstance 更新 CI 实例
func (c *CITypeController) UpdateCIInstance(ctx *gin.Context) {
	var dto model.UpdateCIInstanceDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 更新前抓取旧值（用于变更日志）
	oldInst, _ := c.dao.GetCIInstanceByID(dto.ID)

	data := map[string]interface{}{
		"update_time": time.Now(),
	}

	if dto.Name != "" {
		data["name"] = dto.Name
	}
	if dto.Status > 0 {
		data["status"] = dto.Status
	}
	if dto.GroupID != nil {
		data["group_id"] = dto.GroupID
	}
	if dto.Remark != "" {
		data["remark"] = dto.Remark
	}
	if dto.Attributes != nil {
		j, err := json.Marshal(dto.Attributes)
		if err != nil {
			result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "属性数据格式错误")
			return
		}
		data["attributes"] = datatypes.JSON(j)
	}

	if err := c.dao.UpdateCIInstance(dto.ID, data); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "更新CI实例失败")
		return
	}

	// 写入字段级变更日志
	c.recordInstanceChanges(ctx, oldInst, dto)

	result.Success(ctx, true)
}

// recordInstanceChanges 对比新旧值，写入 ci_change_log
func (c *CITypeController) recordInstanceChanges(ctx *gin.Context, old model.CIInstance, dto model.UpdateCIInstanceDto) {
	var operatorID uint
	var operatorName string
	if obj, ok := ctx.Get(constant.ContextKeyUserObj); ok {
		if admin, ok2 := obj.(*systemmodel.JwtAdmin); ok2 {
			operatorID = admin.ID
			operatorName = admin.Username
		}
	}

	entityName := old.Name
	if dto.Name != "" {
		entityName = dto.Name
	}
	now := util.HTime{Time: time.Now()}

	newLog := func(field, oldVal, newVal string) model.CIChangeLog {
		return model.CIChangeLog{
			EntityType: "ci_instance",
			EntityID:   dto.ID,
			EntityName: entityName,
			Field:      field,
			OldValue:   oldVal,
			NewValue:   newVal,
			OperatorID: operatorID,
			Operator:   operatorName,
			CreateTime: now,
		}
	}

	var logs []model.CIChangeLog

	if dto.Name != "" && dto.Name != old.Name {
		logs = append(logs, newLog("name", old.Name, dto.Name))
	}
	if dto.Status > 0 && dto.Status != old.Status {
		logs = append(logs, newLog("status", fmt.Sprintf("%d", old.Status), fmt.Sprintf("%d", dto.Status)))
	}
	if dto.Remark != "" && dto.Remark != old.Remark {
		logs = append(logs, newLog("remark", old.Remark, dto.Remark))
	}
	if dto.Attributes != nil {
		newAttrs, _ := json.Marshal(dto.Attributes)
		oldAttrs := []byte("{}")
		if old.Attributes != nil {
			oldAttrs = old.Attributes
		}
		if string(newAttrs) != string(oldAttrs) {
			logs = append(logs, newLog("attributes", string(oldAttrs), string(newAttrs)))
		}
	}

	_ = c.changeLog.CreateLogs(logs)
}

// DeleteCIInstance 删除 CI 实例
func (c *CITypeController) DeleteCIInstance(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	if err := c.dao.DeleteCIInstance(uint(id)); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "删除CI实例失败")
		return
	}
	result.Success(ctx, true)
}

// ========================================
// CI Relation APIs
// ========================================

// GetCIRelations 获取 CI 实例的关系
func (c *CITypeController) GetCIRelations(ctx *gin.Context) {
	ciIDStr := ctx.Query("ciId")
	ciID, err := strconv.ParseUint(ciIDStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	rels, err := c.dao.GetRelationsByCIID(uint(ciID))
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "获取关系失败")
		return
	}

	var vos []model.CIRelationVo
	for _, rel := range rels {
		vo := model.CIRelationVo{
			ID:           rel.ID,
			FromCIID:     rel.FromCIID,
			ToCIID:       rel.ToCIID,
			RelationType: rel.RelationType,
			CreateTime:   rel.CreateTime,
		}

		// 填充名称
		if from, err := c.dao.GetCIInstanceByID(rel.FromCIID); err == nil {
			vo.FromCIName = from.Name
			vo.FromCITypeName = from.CIType.Name
		}
		if to, err := c.dao.GetCIInstanceByID(rel.ToCIID); err == nil {
			vo.ToCIName = to.Name
			vo.ToCITypeName = to.CIType.Name
		}

		vos = append(vos, vo)
	}
	result.Success(ctx, vos)
}

// CreateCIRelation 创建关系
func (c *CITypeController) CreateCIRelation(ctx *gin.Context) {
	var dto model.CreateCIRelationDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	// 验证不能自关联
	if dto.FromCIID == dto.ToCIID {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "不能创建自关联关系")
		return
	}

	// 验证两个实例都存在
	if _, err := c.dao.GetCIInstanceByID(dto.FromCIID); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "源CI实例不存在")
		return
	}
	if _, err := c.dao.GetCIInstanceByID(dto.ToCIID); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "目标CI实例不存在")
		return
	}

	// 检查重复
	if c.dao.CheckRelationExists(dto.FromCIID, dto.ToCIID, dto.RelationType) {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "该关系已存在")
		return
	}

	rel := model.CIRelation{
		FromCIID:     dto.FromCIID,
		ToCIID:       dto.ToCIID,
		RelationType: dto.RelationType,
		CreateTime:   util.HTime{Time: time.Now()},
	}

	if err := c.dao.CreateRelation(&rel); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "创建关系失败")
		return
	}
	result.Success(ctx, gin.H{"id": rel.ID})
}

// DeleteCIRelation 删除关系
func (c *CITypeController) DeleteCIRelation(ctx *gin.Context) {
	idStr := ctx.Query("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误")
		return
	}

	if err := c.dao.DeleteRelation(uint(id)); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "删除关系失败")
		return
	}
	result.Success(ctx, true)
}

// ========================================
// CI 拓扑图 APIs（Phase 3）
// ========================================

// GetAllCIInstances 获取所有 CI 实例（用于拓扑图根节点搜索）
func (c *CITypeController) GetAllCIInstances(ctx *gin.Context) {
	keyword := ctx.Query("keyword")
	instances, err := c.dao.GetAllCIInstances()
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "获取CI实例列表失败")
		return
	}

	var vos []model.CIInstanceVo
	for _, inst := range instances {
		if keyword != "" {
			lower := strings.ToLower(keyword)
			if !strings.Contains(strings.ToLower(inst.Name), lower) &&
				!strings.Contains(strings.ToLower(inst.CIType.Name), lower) {
				continue
			}
		}
		vos = append(vos, model.CIInstanceVo{
			ID:       inst.ID,
			CITypeID: inst.CITypeID,
			TypeName: inst.CIType.Name,
			TypeCode: inst.CIType.Code,
			TypeIcon: inst.CIType.Icon,
			Name:     inst.Name,
			Status:   inst.Status,
		})
	}
	result.Success(ctx, vos)
}

// GetCITopology 获取 CI 拓扑图数据（WITH RECURSIVE）
func (c *CITypeController) GetCITopology(ctx *gin.Context) {
	ciIDStr := ctx.Query("ciId")
	ciID, err := strconv.ParseUint(ciIDStr, 10, 32)
	if err != nil {
		result.Failed(ctx, constant.INVALID_PARAMS, "参数错误：ciId 必须为正整数")
		return
	}

	direction := ctx.DefaultQuery("direction", "all")
	if direction != "up" && direction != "down" && direction != "all" {
		direction = "all"
	}

	if _, err := c.dao.GetCIInstanceByID(uint(ciID)); err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "CI实例不存在")
		return
	}

	topo, err := c.dao.GetCITopology(uint(ciID), direction)
	if err != nil {
		result.Failed(ctx, constant.CI_TYPE_NOT_FOUND, "获取拓扑数据失败: "+err.Error())
		return
	}
	result.Success(ctx, topo)
}

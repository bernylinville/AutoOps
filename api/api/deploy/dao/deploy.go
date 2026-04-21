package dao

import (
	"dodevops-api/api/deploy/model"
	"time"

	"gorm.io/gorm"
)

type IDeployDao interface {
	ListClusterTargets() ([]model.ClusterTarget, error)
	GetClusterTargetByID(id uint) (*model.ClusterTarget, error)
	CreateClusterTarget(target *model.ClusterTarget) error
	UpdateClusterTarget(id uint, updates map[string]interface{}) error
	CreateDeployRequest(req *model.DeployRequest) error
	GetDeployRequestByID(id uint) (*model.DeployRequest, error)
	GetDeployRequestByRequestNo(requestNo string) (*model.DeployRequest, error)
	UpdateDeployRequest(id uint, updates map[string]interface{}) error
	TryStartExecution(id uint, startedAt time.Time) (bool, error)
	ListDeployRequests(query *model.DeployRequestListQuery) ([]model.DeployRequest, int64, error)
	CreateApprovalRecord(record *model.ApprovalRecord) error
	CreateExecutionRecord(record *model.ExecutionRecord) error
	ListExecutionRecordsByRequestID(requestID uint) ([]model.ExecutionRecord, error)
	GetLatestExecutionRecordByRequestID(requestID uint) (*model.ExecutionRecord, error)
	CreateDeployNotification(notification *model.DeployNotification) error
	UpdateDeployNotification(id uint, updates map[string]interface{}) error
	ListNotificationsByRequestID(requestID uint) ([]model.DeployNotification, error)
	ListPendingApprovalSyncRequests(limit int) ([]model.DeployRequest, error)
	GetActiveResourceOwner(clusterTargetID uint, namespace, kind, name string) (*model.ResourceOwner, error)
	CreateResourceOwner(owner *model.ResourceOwner) error
	DeactivateResourceOwnersByRequestID(requestID uint) error
	ListExpiredDirectRequests(now time.Time, limit int) ([]model.DeployRequest, error)
}

type DeployDao struct {
	db *gorm.DB
}

func NewDeployDao(db *gorm.DB) IDeployDao {
	return &DeployDao{db: db}
}

func (d *DeployDao) ListClusterTargets() ([]model.ClusterTarget, error) {
	var targets []model.ClusterTarget
	err := d.db.Order("id DESC").Find(&targets).Error
	return targets, err
}

func (d *DeployDao) GetClusterTargetByID(id uint) (*model.ClusterTarget, error) {
	var target model.ClusterTarget
	if err := d.db.First(&target, id).Error; err != nil {
		return nil, err
	}
	return &target, nil
}

func (d *DeployDao) CreateClusterTarget(target *model.ClusterTarget) error {
	return d.db.Create(target).Error
}

func (d *DeployDao) UpdateClusterTarget(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.ClusterTarget{}).Where("id = ?", id).Updates(updates).Error
}

func (d *DeployDao) CreateDeployRequest(req *model.DeployRequest) error {
	return d.db.Create(req).Error
}

func (d *DeployDao) GetDeployRequestByID(id uint) (*model.DeployRequest, error) {
	var req model.DeployRequest
	if err := d.db.First(&req, id).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (d *DeployDao) GetDeployRequestByRequestNo(requestNo string) (*model.DeployRequest, error) {
	var req model.DeployRequest
	if err := d.db.Where("request_no = ?", requestNo).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func (d *DeployDao) UpdateDeployRequest(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.DeployRequest{}).Where("id = ?", id).Updates(updates).Error
}

func (d *DeployDao) TryStartExecution(id uint, startedAt time.Time) (bool, error) {
	result := d.db.Model(&model.DeployRequest{}).
		Where("id = ?", id).
		Where("approval_status = ?", model.ApprovalStatusApproved).
		Where("execution_status NOT IN ?", []string{model.ExecutionStatusRunning, model.ExecutionStatusSucceeded}).
		Updates(map[string]interface{}{
			"request_status":   model.DeployRequestStatusExecuting,
			"execution_status": model.ExecutionStatusRunning,
			"started_at":       startedAt,
			"updated_at":       startedAt,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (d *DeployDao) ListDeployRequests(query *model.DeployRequestListQuery) ([]model.DeployRequest, int64, error) {
	var requests []model.DeployRequest
	var total int64

	cur := d.db.Model(&model.DeployRequest{})
	if query.RequestStatus != "" {
		cur = cur.Where("request_status = ?", query.RequestStatus)
	}
	if query.ApprovalStatus != "" {
		cur = cur.Where("approval_status = ?", query.ApprovalStatus)
	}
	if query.ExecutionStatus != "" {
		cur = cur.Where("execution_status = ?", query.ExecutionStatus)
	}
	if query.Mode != "" {
		cur = cur.Where("mode = ?", query.Mode)
	}
	if query.ClusterTargetID != nil {
		cur = cur.Where("cluster_target_id = ?", *query.ClusterTargetID)
	}
	if query.RequestedBy != nil {
		cur = cur.Where("requested_by = ?", *query.RequestedBy)
	}

	if err := cur.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	err := cur.Offset(offset).Limit(pageSize).Order("id DESC").Find(&requests).Error
	return requests, total, err
}

func (d *DeployDao) CreateApprovalRecord(record *model.ApprovalRecord) error {
	return d.db.Create(record).Error
}

func (d *DeployDao) CreateExecutionRecord(record *model.ExecutionRecord) error {
	return d.db.Create(record).Error
}

func (d *DeployDao) ListExecutionRecordsByRequestID(requestID uint) ([]model.ExecutionRecord, error) {
	var records []model.ExecutionRecord
	err := d.db.Where("request_id = ?", requestID).Order("id DESC").Find(&records).Error
	return records, err
}

func (d *DeployDao) GetLatestExecutionRecordByRequestID(requestID uint) (*model.ExecutionRecord, error) {
	var record model.ExecutionRecord
	if err := d.db.Where("request_id = ?", requestID).Order("id DESC").First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (d *DeployDao) CreateDeployNotification(notification *model.DeployNotification) error {
	return d.db.Create(notification).Error
}

func (d *DeployDao) UpdateDeployNotification(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.DeployNotification{}).Where("id = ?", id).Updates(updates).Error
}

func (d *DeployDao) ListNotificationsByRequestID(requestID uint) ([]model.DeployNotification, error) {
	var notifications []model.DeployNotification
	err := d.db.Where("request_id = ?", requestID).Order("id DESC").Find(&notifications).Error
	return notifications, err
}

func (d *DeployDao) ListPendingApprovalSyncRequests(limit int) ([]model.DeployRequest, error) {
	var requests []model.DeployRequest
	query := d.db.
		Where("approval_status = ?", model.ApprovalStatusPending).
		Where("approval_dispatch_status = ?", model.ApprovalDispatchStatusDispatched).
		Where("dingtalk_process_instance_id != ''")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("updated_at ASC").Find(&requests).Error
	return requests, err
}

func (d *DeployDao) GetActiveResourceOwner(clusterTargetID uint, namespace, kind, name string) (*model.ResourceOwner, error) {
	var owner model.ResourceOwner
	err := d.db.
		Where("cluster_target_id = ?", clusterTargetID).
		Where("namespace = ?", namespace).
		Where("kind = ?", kind).
		Where("name = ?", name).
		Where("active = ?", true).
		First(&owner).Error
	if err != nil {
		return nil, err
	}
	return &owner, nil
}

func (d *DeployDao) CreateResourceOwner(owner *model.ResourceOwner) error {
	return d.db.Create(owner).Error
}

func (d *DeployDao) DeactivateResourceOwnersByRequestID(requestID uint) error {
	return d.db.Model(&model.ResourceOwner{}).Where("request_id = ?", requestID).Updates(map[string]interface{}{
		"active":     false,
		"updated_at": time.Now(),
	}).Error
}

func (d *DeployDao) ListExpiredDirectRequests(now time.Time, limit int) ([]model.DeployRequest, error) {
	var requests []model.DeployRequest
	query := d.db.
		Where("mode = ?", model.DeployModeDirect).
		Where("execution_status = ?", model.ExecutionStatusSucceeded).
		Where("ttl_hours is not null").
		Where("finished_at is not null")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("finished_at ASC").Find(&requests).Error
	if err != nil {
		return nil, err
	}

	expired := make([]model.DeployRequest, 0)
	for _, req := range requests {
		if req.TTLHours == nil || req.FinishedAt == nil {
			continue
		}
		expireAt := req.FinishedAt.Add(time.Duration(*req.TTLHours) * time.Hour)
		if !expireAt.After(now) {
			expired = append(expired, req)
		}
	}
	return expired, nil
}

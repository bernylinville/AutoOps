package dao

import (
	"dodevops-api/api/deploy/model"
	"time"

	"gorm.io/gorm"
)

type IPipelineDao interface {
	CreatePipelineRun(run *model.PipelineRun) error
	GetPipelineRunByID(id uint) (*model.PipelineRun, error)
	GetPipelineRunByRequestID(requestID uint) (*model.PipelineRun, error)
	UpdatePipelineRun(id uint, updates map[string]interface{}) error
	ListPipelineRunsByStatus(status string, limit int) ([]model.PipelineRun, error)
	// ListPendingApprovedPipelineRuns returns pipeline runs with status=pending whose deploy request is approved
	ListPendingApprovedPipelineRuns(limit int) ([]model.PipelineRun, error)
	// ClaimPipelineRun atomically claims a pending pipeline run for processing by updating status to building
	ClaimPipelineRun(id uint, claimToken string) (bool, error)
	// ListStalePipelineRuns returns pipeline runs stuck in building/scanning/deploying for longer than timeout
	ListStalePipelineRuns(timeout time.Duration, limit int) ([]model.PipelineRun, error)
	CreatePipelineStageRecord(record *model.PipelineStageRecord) error
	GetPipelineStageRecordsByPipelineRunID(pipelineRunID uint) ([]model.PipelineStageRecord, error)
	UpdatePipelineStageRecord(id uint, updates map[string]interface{}) error
	GetPipelineStageRecordByStage(pipelineRunID uint, stage string) (*model.PipelineStageRecord, error)
}

type PipelineDao struct {
	db *gorm.DB
}

func NewPipelineDao(db *gorm.DB) IPipelineDao {
	return &PipelineDao{db: db}
}

func (d *PipelineDao) CreatePipelineRun(run *model.PipelineRun) error {
	return d.db.Create(run).Error
}

func (d *PipelineDao) GetPipelineRunByID(id uint) (*model.PipelineRun, error) {
	var run model.PipelineRun
	if err := d.db.First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (d *PipelineDao) GetPipelineRunByRequestID(requestID uint) (*model.PipelineRun, error) {
	var run model.PipelineRun
	if err := d.db.Where("request_id = ?", requestID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (d *PipelineDao) UpdatePipelineRun(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.PipelineRun{}).Where("id = ?", id).Updates(updates).Error
}

func (d *PipelineDao) ListPipelineRunsByStatus(status string, limit int) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	query := d.db.Where("status = ?", status)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("created_at ASC").Find(&runs).Error
	return runs, err
}

func (d *PipelineDao) ListPendingApprovedPipelineRuns(limit int) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	query := d.db.Table("deploy_pipeline_run").
		Select("deploy_pipeline_run.*").
		Joins("JOIN deploy_request ON deploy_request.id = deploy_pipeline_run.request_id").
		Where("deploy_pipeline_run.status = ?", model.PipelineStatusPending).
		Where("deploy_request.approval_status = ?", model.ApprovalStatusApproved).
		Where("deploy_request.execution_status = ?", model.ExecutionStatusPending)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("deploy_pipeline_run.created_at ASC").Find(&runs).Error
	return runs, err
}

func (d *PipelineDao) ClaimPipelineRun(id uint, claimToken string) (bool, error) {
	result := d.db.Model(&model.PipelineRun{}).
		Where("id = ? AND status = ?", id, model.PipelineStatusPending).
		Updates(map[string]interface{}{
			"status":        model.PipelineStatusBuilding,
			"current_stage": model.PipelineStageBuild,
			"updated_at":    time.Now(),
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (d *PipelineDao) ListStalePipelineRuns(timeout time.Duration, limit int) ([]model.PipelineRun, error) {
	var runs []model.PipelineRun
	cutoff := time.Now().Add(-timeout)
	query := d.db.Where("status IN ?", []string{model.PipelineStatusBuilding, model.PipelineStatusScanning, model.PipelineStatusDeploying}).
		Where("updated_at < ?", cutoff)
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Order("updated_at ASC").Find(&runs).Error
	return runs, err
}

func (d *PipelineDao) CreatePipelineStageRecord(record *model.PipelineStageRecord) error {
	return d.db.Create(record).Error
}

func (d *PipelineDao) GetPipelineStageRecordsByPipelineRunID(pipelineRunID uint) ([]model.PipelineStageRecord, error) {
	var records []model.PipelineStageRecord
	err := d.db.Where("pipeline_run_id = ?", pipelineRunID).Order("id DESC").Find(&records).Error
	return records, err
}

func (d *PipelineDao) UpdatePipelineStageRecord(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.PipelineStageRecord{}).Where("id = ?", id).Updates(updates).Error
}

func (d *PipelineDao) GetPipelineStageRecordByStage(pipelineRunID uint, stage string) (*model.PipelineStageRecord, error) {
	var record model.PipelineStageRecord
	if err := d.db.Where("pipeline_run_id = ?", pipelineRunID).Where("stage = ?", stage).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

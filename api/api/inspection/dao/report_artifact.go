package dao

import (
	"dodevops-api/api/inspection/model"

	"gorm.io/gorm"
)

// ReportArtifactDAO 巡检报告文件数据访问
type ReportArtifactDAO struct {
	db *gorm.DB
}

// NewReportArtifactDAO creates a ReportArtifactDAO.
func NewReportArtifactDAO(db *gorm.DB) *ReportArtifactDAO {
	return &ReportArtifactDAO{db: db}
}

// Create 创建报告文件记录
func (d *ReportArtifactDAO) Create(artifact *model.InspectionReportArtifact) error {
	return d.db.Create(artifact).Error
}

// GetByRunID 按运行记录 ID 查询报告
func (d *ReportArtifactDAO) GetByRunID(runID uint) (*model.InspectionReportArtifact, error) {
	var artifact model.InspectionReportArtifact
	if err := d.db.Where("run_id = ?", runID).First(&artifact).Error; err != nil {
		return nil, err
	}
	return &artifact, nil
}

// Update 更新报告文件记录
func (d *ReportArtifactDAO) Update(id uint, updates map[string]interface{}) error {
	return d.db.Model(&model.InspectionReportArtifact{}).Where("id = ?", id).Updates(updates).Error
}

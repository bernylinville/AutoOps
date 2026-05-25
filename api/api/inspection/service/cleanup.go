package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dodevops-api/api/inspection/dao"
	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/log"

	"gorm.io/gorm"
)

// CleanupService handles periodic cleanup of old inspection data.
type CleanupService struct {
	db        *gorm.DB
	runDAO    *dao.RunDAO
	reportDAO *dao.ReportArtifactDAO
}

// NewCleanupService creates a CleanupService.
func NewCleanupService(db *gorm.DB) *CleanupService {
	return &CleanupService{
		db:        db,
		runDAO:    dao.NewRunDAO(db),
		reportDAO: dao.NewReportArtifactDAO(db),
	}
}

const retentionDays = 30

// CleanupExpired removes inspection data older than 30 days.
func (s *CleanupService) CleanupExpired() error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays).Format("2006-01-02")
	log.Log().Infof("[Cleanup] starting expired data cleanup (cutoff: %s)", cutoff)

	const batchSize = 500
	totalDeleted := 0

	for {
		var batchIDs []uint
		if err := s.db.Model(&model.InspectionRun{}).
			Where("run_date < ?", cutoff).
			Limit(batchSize).
			Pluck("id", &batchIDs).Error; err != nil {
			return fmt.Errorf("query expired runs: %w", err)
		}

		if len(batchIDs) == 0 {
			break
		}

		log.Log().Infof("[Cleanup] deleting batch of %d expired runs", len(batchIDs))

		var artifacts []model.InspectionReportArtifact

		err := s.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("run_id IN ?", batchIDs).Delete(&model.InspectionNotification{}).Error; err != nil {
				return fmt.Errorf("delete notifications: %w", err)
			}
			if err := tx.Where("run_id IN ?", batchIDs).Delete(&model.InspectionAlert{}).Error; err != nil {
				return fmt.Errorf("delete alerts: %w", err)
			}
			if err := tx.Where("run_id IN ?", batchIDs).Delete(&model.InspectionTargetResult{}).Error; err != nil {
				return fmt.Errorf("delete target results: %w", err)
			}

			if err := tx.Where("run_id IN ?", batchIDs).Find(&artifacts).Error; err != nil {
				return fmt.Errorf("query report artifacts: %w", err)
			}
			if err := tx.Where("run_id IN ?", batchIDs).Delete(&model.InspectionReportArtifact{}).Error; err != nil {
				return fmt.Errorf("delete report artifacts: %w", err)
			}

			if err := tx.Where("id IN ?", batchIDs).Delete(&model.InspectionRun{}).Error; err != nil {
				return fmt.Errorf("delete runs: %w", err)
			}

			return nil
		})

		if err != nil {
			log.Log().Errorf("[Cleanup] batch failed: %v", err)
			return err
		}

		for _, artifact := range artifacts {
			s.removeReportFile(artifact.FilePath)
		}

		totalDeleted += len(batchIDs)
	}

	if totalDeleted == 0 {
		log.Log().Info("[Cleanup] no expired runs found")
	} else {
		log.Log().Infof("[Cleanup] completed: %d runs and associated data deleted", totalDeleted)
	}
	return nil
}


// removeReportFile removes a report file from disk, logging but not failing on error.
func (s *CleanupService) removeReportFile(filePath string) {
	if filePath == "" {
		return
	}

	// Path traversal guard.
	cleaned := filepath.Clean(filePath)
	if !strings.HasPrefix(cleaned, "/data/inspection/") {
		log.Log().Warnf("[Cleanup] refusing to delete file outside /data/inspection/: %s", filePath)
		return
	}

	if err := os.Remove(cleaned); err != nil {
		if !os.IsNotExist(err) {
			log.Log().Warnf("[Cleanup] failed to delete report file %s: %v", cleaned, err)
		}
	} else {
		log.Log().Debugf("[Cleanup] deleted report file: %s", cleaned)
	}
}

package controller

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dodevops-api/api/inspection/dao"
	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ReportController handles report download endpoints.
type ReportController struct {
	reportDAO *dao.ReportArtifactDAO
}

// NewReportController creates a ReportController.
func NewReportController(db *gorm.DB) *ReportController {
	return &ReportController{reportDAO: dao.NewReportArtifactDAO(db)}
}

// DownloadReport GET /inspection/runs/:id/report
func (c *ReportController) DownloadReport(ctx *gin.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	artifact, err := c.reportDAO.GetByRunID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Failed(ctx, int(result.ApiCode.INSPECTION_REPORT_NOT_FOUND), service.ErrReportNotFound.Error())
			return
		}
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	// Check artifact generation succeeded.
	if artifact.Status != model.ReportStatusSuccess {
		result.Failed(ctx, int(result.ApiCode.INSPECTION_REPORT_NOT_FOUND), service.ErrReportNotFound.Error())
		return
	}

	// Check artifact has not expired.
	if time.Now().After(artifact.ExpiresAt.Time) {
		result.Failed(ctx, int(result.ApiCode.INSPECTION_REPORT_NOT_FOUND), service.ErrReportNotFound.Error())
		return
	}

	// Validate file path to prevent path traversal.
	filePath := filepath.Clean(artifact.FilePath)
	baseDir := filepath.Clean("/data/inspection")
	rel, err := filepath.Rel(baseDir, filePath)
	if err != nil || strings.HasPrefix(rel, "..") {
		result.Failed(ctx, int(result.ApiCode.FAILED), "报告文件路径无效")
		return
	}

	// Verify file exists.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		result.Failed(ctx, int(result.ApiCode.INSPECTION_REPORT_NOT_FOUND), service.ErrReportNotFound.Error())
		return
	}

	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(filePath)))
	ctx.File(filePath)
}

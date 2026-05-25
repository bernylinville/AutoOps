// Package controller provides HTTP controllers for inspection.
package controller

import (
	"context"
	"errors"
	"time"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service"
	"dodevops-api/common/result"
	"dodevops-api/pkg/jwt"
	"dodevops-api/pkg/log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InspectionController 巡检执行与结果查询控制器
type InspectionController struct {
	inspSvc *service.InspectionService
}

// NewInspectionController creates an InspectionController.
func NewInspectionController(inspSvc *service.InspectionService) *InspectionController {
	return &InspectionController{inspSvc: inspSvc}
}

// TriggerInspection POST /inspection/tasks/:id/trigger
// 手动触发一次巡检执行，返回 run ID 后异步执行。
func (c *InspectionController) TriggerInspection(ctx *gin.Context) {
	taskID, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	// 获取触发用户
	var triggeredBy *uint
	admin, err := jwt.GetAdmin(ctx)
	if err == nil {
		uid := admin.ID
		triggeredBy = &uid
	}

	// 验证任务存在
	if _, err := c.inspSvc.TaskService().GetTaskRaw(taskID); err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			result.Failed(ctx, int(result.ApiCode.INSPECTION_TASK_NOT_FOUND), err.Error())
			return
		}
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	// 创建运行记录并立即返回
	run := &model.InspectionRun{
		TaskID:      taskID,
		TriggerType: model.TriggerTypeManual,
		TriggeredBy: triggeredBy,
		Status:      model.RunStatusPending,
	}
	runDate := time.Now().Format("2006-01-02")
	run.RunDate = runDate

	if err := c.inspSvc.RunDAO().Create(run); err != nil {
		result.Failed(ctx, int(result.ApiCode.INSPECTION_TRIGGER_FAILED), "创建运行记录失败: "+err.Error())
		return
	}

	// 异步执行
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		updatedRun, err := c.inspSvc.ExecuteInspection(bgCtx, taskID, model.TriggerTypeManual, triggeredBy, run)
		if err != nil {
			log.Log().Errorf("[Inspection] manual trigger for task %d failed: %v", taskID, err)
			return
		}

		log.Log().Infof("[Inspection] manual trigger for task %d completed (run %d)", taskID, updatedRun.ID)
	}()

	result.Success(ctx, gin.H{
		"runId":       run.ID,
		"taskId":      taskID,
		"triggerType": model.TriggerTypeManual,
		"status":      model.RunStatusPending,
	})
}

// ListRuns GET /inspection/runs — 查询运行历史列表
func (c *InspectionController) ListRuns(ctx *gin.Context) {
	var q model.RunListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	runs, total, err := c.inspSvc.RunDAO().List(&q)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	// 批量补充业务组名称
	c.enrichRunTaskInfo(runs)

	result.SuccessWithPage(ctx, runs, total, q.Page, q.PageSize)
}

// enrichRunTaskInfo 批量查询任务信息并补充到运行记录中
func (c *InspectionController) enrichRunTaskInfo(runs []*model.InspectionRun) {
	if len(runs) == 0 {
		return
	}
	// 收集唯一 taskID
	taskIDs := make(map[uint]bool)
	for _, r := range runs {
		taskIDs[r.TaskID] = true
	}
	// 批量查询
	for tid := range taskIDs {
		task, err := c.inspSvc.TaskService().GetTaskRaw(tid)
		if err != nil {
			continue
		}
		for _, r := range runs {
			if r.TaskID == tid {
				r.N9EGroupName = task.N9EGroupName
				r.TaskName = task.Name
			}
		}
	}
}

// GetRun GET /inspection/runs/:id — 查询单次运行详情
func (c *InspectionController) GetRun(ctx *gin.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	run, err := c.inspSvc.RunDAO().GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Failed(ctx, int(result.ApiCode.INSPECTION_RUN_NOT_FOUND), "巡检运行记录不存在")
			return
		}
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}
	c.enrichRunTaskInfo([]*model.InspectionRun{run})

	result.Success(ctx, run)
}

// ListRunResults GET /inspection/runs/:id/results — 查询运行的主机级结果
func (c *InspectionController) ListRunResults(ctx *gin.Context) {
	runID, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	var q model.ResultListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	results, total, err := c.inspSvc.ResultDAO().ListByRunIDWithQuery(runID, &q)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	result.SuccessWithPage(ctx, results, total, q.Page, q.PageSize)
}

// ListRunAlerts GET /inspection/runs/:id/alerts — 查询运行的异常明细
func (c *InspectionController) ListRunAlerts(ctx *gin.Context) {
	runID, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	var q model.AlertListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	alerts, total, err := c.inspSvc.AlertDAO().ListByRunID(runID, &q)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	result.SuccessWithPage(ctx, alerts, total, q.Page, q.PageSize)
}

// GetTodayOverview GET /inspection/overview — 今日巡检概览（含统计和最近告警）
func (c *InspectionController) GetTodayOverview(ctx *gin.Context) {
	stats, err := c.inspSvc.RunDAO().GetTodayStats()
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	recentAlerts, _ := c.inspSvc.AlertDAO().GetRecentAlerts(10)

	result.Success(ctx, gin.H{
		"stats":        stats,
		"recentAlerts": recentAlerts,
	})
}

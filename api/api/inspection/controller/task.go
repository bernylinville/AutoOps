// Package controller provides HTTP controllers for inspection.
package controller

import (
	"errors"
	"strconv"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
)

// parseUintParam parses a uint path parameter from the gin context.
func parseUintParam(ctx *gin.Context, name string) (uint, error) {
	val, err := strconv.ParseUint(ctx.Param(name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(val), nil
}

// TaskController 巡检任务控制器
type TaskController struct {
	taskService *service.TaskService
}

// NewTaskController creates a TaskController.
func NewTaskController(taskService *service.TaskService) *TaskController {
	return &TaskController{taskService: taskService}
}

// ListTasks GET /inspection/tasks
func (c *TaskController) ListTasks(ctx *gin.Context) {
	var q model.TaskListQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	vos, total, err := c.taskService.ListTasks(&q)
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	result.SuccessWithPage(ctx, vos, total, q.Page, q.PageSize)
}

// GetTask GET /inspection/tasks/:id
func (c *TaskController) GetTask(ctx *gin.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	vo, err := c.taskService.GetTask(id)
	if err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			result.Failed(ctx, int(result.ApiCode.INSPECTION_TASK_NOT_FOUND), err.Error())
			return
		}
		result.Failed(ctx, int(result.ApiCode.FAILED), err.Error())
		return
	}

	result.Success(ctx, vo)
}

// UpdateTask PUT /inspection/tasks/:id
func (c *TaskController) UpdateTask(ctx *gin.Context) {
	id, err := parseUintParam(ctx, "id")
	if err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	var dto model.UpdateTaskDto
	if err := ctx.ShouldBindJSON(&dto); err != nil {
		result.Failed(ctx, int(result.ApiCode.FAILED), "参数校验失败: "+err.Error())
		return
	}

	if err := c.taskService.UpdateTask(id, &dto); err != nil {
		if errors.Is(err, service.ErrTaskNotFound) {
			result.Failed(ctx, int(result.ApiCode.INSPECTION_TASK_NOT_FOUND), err.Error())
			return
		}
		result.Failed(ctx, int(result.ApiCode.INSPECTION_TASK_UPDATE_FAILED), err.Error())
		return
	}

	result.Success(ctx, nil)
}

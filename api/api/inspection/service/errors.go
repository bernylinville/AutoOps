// Package service provides business logic for inspection.
package service

import "errors"

var (
	ErrTaskNotFound         = errors.New("巡检任务不存在")
	ErrTaskUpdateFailed     = errors.New("巡检任务更新失败")
	ErrRunNotFound          = errors.New("巡检运行记录不存在")
	ErrTriggerFailed        = errors.New("巡检触发失败")
	ErrReportNotFound       = errors.New("巡检报告不存在")
	ErrReportGenerateFailed = errors.New("巡检报告生成失败")
	ErrAlreadyRunning       = errors.New("巡检任务正在运行中")
)

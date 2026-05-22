// Package service provides business logic for inspection.
package service

import (
	"errors"

	"dodevops-api/api/inspection/dao"
	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/log"

	"gorm.io/gorm"
)

// TaskService 巡检任务服务
type TaskService struct {
	taskDAO *dao.TaskDAO
}

// NewTaskService creates a TaskService.
func NewTaskService(db *gorm.DB) *TaskService {
	return &TaskService{taskDAO: dao.NewTaskDAO(db)}
}

// ListTasks 任务列表
func (s *TaskService) ListTasks(q *model.TaskListQuery) ([]*model.TaskVO, int64, error) {
	tasks, total, err := s.taskDAO.List(q)
	if err != nil {
		return nil, 0, err
	}

	vos := make([]*model.TaskVO, len(tasks))
	for i, t := range tasks {
		vos[i] = t.ToVO()
	}

	return vos, total, nil
}

// GetTask 任务详情（脱敏）
func (s *TaskService) GetTask(id uint) (*model.TaskVO, error) {
	task, err := s.taskDAO.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task.ToVO(), nil
}

// GetTaskRaw 获取原始任务（不脱敏，内部使用）
func (s *TaskService) GetTaskRaw(id uint) (*model.InspectionTask, error) {
	task, err := s.taskDAO.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	return task, nil
}

// UpdateTask 更新任务配置
func (s *TaskService) UpdateTask(id uint, dto *model.UpdateTaskDto) error {
	task, err := s.taskDAO.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		return err
	}

	task.ApplyUpdate(dto)
	if err := s.taskDAO.Update(task); err != nil {
		return err
	}

	log.Log().Infof("[Inspection] task %d updated", id)
	return nil
}

// ListAllEnabledTasks 查询所有启用的任务
func (s *TaskService) ListAllEnabledTasks() ([]*model.InspectionTask, error) {
	return s.taskDAO.ListAllEnabled()
}

// ListAllTasks 查询所有任务
func (s *TaskService) ListAllTasks() ([]*model.InspectionTask, error) {
	return s.taskDAO.ListAll()
}

// TaskDAO returns the underlying DAO for service-level use.
func (s *TaskService) TaskDAO() *dao.TaskDAO {
	return s.taskDAO
}

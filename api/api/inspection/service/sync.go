// Package service provides business logic for inspection.
package service

import (
	"context"
	"time"

	"dodevops-api/api/inspection/model"
	n9emodel "dodevops-api/api/n9e/model"
	"dodevops-api/common/util"
	"dodevops-api/pkg/log"
)

// SyncTasksFromN9EGroups syncs inspection tasks with N9E business groups.
// - For groups without a task: creates a default task (enabled=false, name=group name, default cron).
// - For tasks whose n9e_group_id no longer exists in N9E: sets enabled=false.
func (s *TaskService) SyncTasksFromN9EGroups(ctx context.Context, busiGroups []n9emodel.BusiGroupData) {
	if len(busiGroups) == 0 {
		log.Log().Info("[Inspection Sync] no N9E busi groups to sync")
		return
	}

	// 构建 N9E 分组 ID 集合
	busiGroupIDs := make(map[int64]bool, len(busiGroups))
	for _, group := range busiGroups {
		busiGroupIDs[group.ID] = true
	}

	// 获取所有已有任务
	existingTasks, err := s.taskDAO.ListAll()
	if err != nil {
		log.Log().Errorf("[Inspection Sync] failed to list existing tasks: %v", err)
		return
	}

	// 构建已有任务 n9e_group_id 映射
	existingTaskByGroup := make(map[int64]*model.InspectionTask, len(existingTasks))
	for _, task := range existingTasks {
		existingTaskByGroup[task.N9EGroupID] = task
	}

	created := 0
	disabled := 0

	// 为没有任务的分组创建默认任务
	for _, group := range busiGroups {
		if _, exists := existingTaskByGroup[group.ID]; !exists {
			task := &model.InspectionTask{
				N9EGroupID:   group.ID,
				N9EGroupName: group.Name,
				Name:         group.Name,
				Enabled:      false,
				Cron:         "CRON_TZ=Asia/Shanghai 0 10 * * *",
				CreateTime:   util.HTime{Time: time.Now()},
				UpdateTime:   util.HTime{Time: time.Now()},
			}
			if err := s.taskDAO.Create(task); err != nil {
				log.Log().Errorf("[Inspection Sync] failed to create task for group %d (%s): %v", group.ID, group.Name, err)
				continue
			}
			created++
			log.Log().Infof("[Inspection Sync] created task %d for N9E group %d (%s)", task.ID, group.ID, group.Name)
		}
	}

	// 对于 N9E 中不再存在的分组，禁用对应任务
	for _, task := range existingTasks {
		if _, exists := busiGroupIDs[task.N9EGroupID]; !exists {
			if task.Enabled {
				if err := s.taskDAO.UpdateField(task.ID, "enabled", false); err != nil {
					log.Log().Errorf("[Inspection Sync] failed to disable task %d (group %d): %v", task.ID, task.N9EGroupID, err)
					continue
				}
				disabled++
				log.Log().Infof("[Inspection Sync] disabled task %d (N9E group %d no longer exists)", task.ID, task.N9EGroupID)
			}
		}
	}

	log.Log().Infof("[Inspection Sync] completed: created=%d, disabled=%d, total_groups=%d",
		created, disabled, len(busiGroups))

	// Suppress unused context warning — ctx is for future cancellation/timeout use.
	_ = ctx
}

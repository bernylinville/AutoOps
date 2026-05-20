// Package scheduler provides cron-based scheduling for inspection tasks.
package scheduler

import (
	"context"
	"sync"
	"time"

	"dodevops-api/api/inspection/model"
	"dodevops-api/api/inspection/service"
	"dodevops-api/pkg/log"

	"github.com/robfig/cron/v3"
)

// Scheduler manages cron jobs for inspection tasks.
type Scheduler struct {
	c       *cron.Cron
	taskSvc *service.TaskService
	inspSvc *service.InspectionService
	entries map[uint]cron.EntryID
	mu      sync.Mutex
}

// NewScheduler creates a new Scheduler.
func NewScheduler(taskSvc *service.TaskService, inspSvc *service.InspectionService) *Scheduler {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}

	return &Scheduler{
		c:       cron.New(cron.WithLocation(loc)),
		taskSvc: taskSvc,
		inspSvc: inspSvc,
		entries: make(map[uint]cron.EntryID),
	}
}

// Start loads all enabled tasks from DB and registers cron jobs.
func (s *Scheduler) Start() error {
	tasks, err := s.taskSvc.ListAllEnabledTasks()
	if err != nil {
		return err
	}

	for _, task := range tasks {
		s.registerTask(task.ID, task.Cron)
	}

	s.c.Start()
	log.Log().Infof("[Scheduler] started with %d tasks", len(tasks))
	return nil
}

// ReloadTask removes the existing cron entry for a task and re-registers if enabled.
func (s *Scheduler) ReloadTask(taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing entry
	if entryID, ok := s.entries[taskID]; ok {
		s.c.Remove(entryID)
		delete(s.entries, taskID)
	}

	// Re-register if task still enabled
	task, err := s.taskSvc.GetTaskRaw(taskID)
	if err != nil {
		log.Log().Warnf("[Scheduler] reload task %d failed: %v", taskID, err)
		return
	}

	if task.Enabled {
		s.registerTask(task.ID, task.Cron)
		log.Log().Infof("[Scheduler] task %d re-registered", taskID)
	}
}

// registerTask adds a cron job for the given task.
func (s *Scheduler) registerTask(taskID uint, cronExpr string) {
	if cronExpr == "" {
		log.Log().Warnf("[Scheduler] task %d has no cron expression, skipping", taskID)
		return
	}

	tid := taskID // capture for closure
	entryID, err := s.c.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		log.Log().Infof("[Scheduler] executing inspection for task %d", tid)
		_, err := s.inspSvc.ExecuteInspection(ctx, tid, model.TriggerTypeCron, nil, nil)
		if err != nil {
			log.Log().Errorf("[Scheduler] inspection task %d failed: %v", tid, err)
		}
	})
	if err != nil {
		log.Log().Errorf("[Scheduler] failed to register task %d: %v", taskID, err)
		return
	}

	s.entries[taskID] = entryID
	log.Log().Infof("[Scheduler] task %d registered with cron '%s'", taskID, cronExpr)
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() {
	ctx := s.c.Stop()
	<-ctx.Done()
	log.Log().Info("[Scheduler] stopped")
}

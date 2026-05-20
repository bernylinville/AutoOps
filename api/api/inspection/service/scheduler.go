// Package service provides business logic for inspection.
package service

import (
	"dodevops-api/api/inspection/dao"
	"dodevops-api/pkg/log"

	"gorm.io/gorm"
)

// maxConcurrentInspection is the maximum number of concurrent inspection runs.
const maxConcurrentInspection = 5

// Scheduler controls concurrent inspection execution.
type Scheduler struct {
	semaphore chan struct{}
	runDAO    *dao.RunDAO
}

// NewScheduler creates a Scheduler with the given DB connection.
func NewScheduler(db *gorm.DB) *Scheduler {
	return &Scheduler{
		semaphore: make(chan struct{}, maxConcurrentInspection),
		runDAO:    dao.NewRunDAO(db),
	}
}

// acquireSlot blocks until a concurrency slot is available.
func (s *Scheduler) acquireSlot() {
	s.semaphore <- struct{}{}
}

// releaseSlot releases a concurrency slot.
func (s *Scheduler) releaseSlot() {
	<-s.semaphore
}

// SkipIfRunning checks if a task already has an active (pending/running) inspection and should be skipped.
func (s *Scheduler) SkipIfRunning(taskID uint) bool {
	active, err := s.runDAO.HasActiveRunByTaskID(taskID)
	if err != nil {
		log.Log().Errorf("[Scheduler] failed to check active status for task %d: %v", taskID, err)
		return true // Skip on error to be safe
	}
	return active
}

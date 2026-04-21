package scheduler

import (
	"log"
	"sync"
	"time"

	deployservice "dodevops-api/api/deploy/service"
	"dodevops-api/common"
	"dodevops-api/common/config"
	"gorm.io/gorm"
)

type PipelineScheduler struct {
	mu      sync.Mutex
	ticker  *time.Ticker
	stopCh  chan struct{}
	running bool
	db      *gorm.DB
}

func NewPipelineScheduler() *PipelineScheduler {
	return &PipelineScheduler{db: common.GetDB()}
}

func (s *PipelineScheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	interval := 15 * time.Second
	if config.Config != nil && config.Config.DingtalkApproval.PollIntervalSeconds > 0 {
		interval = time.Duration(config.Config.DingtalkApproval.PollIntervalSeconds) * time.Second
	}
	s.ticker = time.NewTicker(interval)
	s.stopCh = make(chan struct{})
	s.running = true

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.syncOnce()
			case <-s.stopCh:
				return
			}
		}
	}()

	log.Printf("流水线调度器已启动（每 %s 执行）", interval.String())
	return nil
}

func (s *PipelineScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	if s.ticker != nil {
		s.ticker.Stop()
	}
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.running = false
	log.Println("流水线调度器已停止")
}

func (s *PipelineScheduler) syncOnce() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PipelineScheduler] panic recovered: %v", r)
		}
	}()

	db := common.GetDB()
	if db == nil {
		return
	}
	s.db = db

	pipelineService := deployservice.NewPipelineService(db)

	if err := pipelineService.ProcessPendingPipelineRuns(5); err != nil {
		log.Printf("[PipelineScheduler] process pending runs error: %v", err)
	}

	if err := pipelineService.RecoverStalePipelineRuns(2*time.Hour, 5); err != nil {
		log.Printf("[PipelineScheduler] recover stale runs error: %v", err)
	}
}

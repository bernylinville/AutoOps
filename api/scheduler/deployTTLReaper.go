package scheduler

import (
	"log"
	"sync"
	"time"

	deploydao "dodevops-api/api/deploy/dao"
	deployservice "dodevops-api/api/deploy/service"
	"dodevops-api/common"
)

type DeployTTLReaper struct {
	mu      sync.Mutex
	ticker  *time.Ticker
	stopCh  chan struct{}
	running bool
}

func NewDeployTTLReaper() *DeployTTLReaper {
	return &DeployTTLReaper{}
}

func (r *DeployTTLReaper) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return nil
	}
	r.ticker = time.NewTicker(1 * time.Minute)
	r.stopCh = make(chan struct{})
	r.running = true

	go func() {
		for {
			select {
			case <-r.ticker.C:
				r.reapOnce()
			case <-r.stopCh:
				return
			}
		}
	}()

	log.Println("部署 TTL 清理调度器已启动（每 1 分钟执行）")
	return nil
}

func (r *DeployTTLReaper) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return
	}
	if r.ticker != nil {
		r.ticker.Stop()
	}
	if r.stopCh != nil {
		close(r.stopCh)
	}
	r.running = false
	log.Println("部署 TTL 清理调度器已停止")
}

func (r *DeployTTLReaper) reapOnce() {
	db := common.GetDB()
	if db == nil {
		return
	}

	dao := deploydao.NewDeployDao(db)
	requests, err := dao.ListExpiredDirectRequests(time.Now(), 50)
	if err != nil {
		log.Printf("[DeployTTL] 查询到期 direct 申请失败: %v", err)
		return
	}
	if len(requests) == 0 {
		return
	}

	service := deployservice.NewDeployService(db)
	for _, req := range requests {
		if err := service.CleanupDirectRequestByID(req.ID); err != nil {
			log.Printf("[DeployTTL] 清理到期申请失败 requestNo=%s err=%v", req.RequestNo, err)
		}
	}
}

package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"dodevops-api/api/n9e/dao"
	"dodevops-api/api/n9e/service"

	"github.com/robfig/cron/v3"
)

// N9ESyncScheduler N9E 数据定时同步调度器
type N9ESyncScheduler struct {
	cron        *cron.Cron
	syncService *service.SyncService
	entryID     cron.EntryID
	cronExpr    string // 当前运行中的 cron 表达式
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewN9ESyncScheduler 创建 N9E 同步调度器
func NewN9ESyncScheduler() *N9ESyncScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &N9ESyncScheduler{
		cron: cron.New(cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
		)),
		syncService: service.GetSyncService(),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start 启动 N9E 同步调度器
func (s *N9ESyncScheduler) Start() error {
	log.Println("启动 N9E 同步调度器...")

	// 从数据库加载 N9E 配置
	if err := s.LoadCron(); err != nil {
		// 配置不存在或未启用不算错误，只是不注册任务
		log.Printf("N9E 同步调度器: %v (将在配置保存后自动启动)", err)
	}

	s.cron.Start()
	log.Println("N9E 同步调度器启动成功")
	return nil
}

// Stop 停止 N9E 同步调度器
func (s *N9ESyncScheduler) Stop() {
	log.Println("停止 N9E 同步调度器...")
	s.cancel()
	s.cron.Stop()
	log.Println("N9E 同步调度器已停止")
}

// LoadCron 从数据库加载 cron 配置并注册定时任务
func (s *N9ESyncScheduler) LoadCron() error {
	config, err := dao.GetN9EConfig()
	if err != nil {
		return fmt.Errorf("未找到 N9E 配置")
	}

	if !config.Enabled {
		s.removeCronJob()
		return fmt.Errorf("N9E 集成未启用")
	}

	if config.SyncCron == "" {
		s.removeCronJob()
		return fmt.Errorf("未设置同步 Cron 表达式")
	}

	return s.setCronJob(config.SyncCron)
}

// ReloadCron 重新加载 cron 配置（配置变更时调用）
func (s *N9ESyncScheduler) ReloadCron() error {
	log.Println("重新加载 N9E 同步 Cron 配置...")
	return s.LoadCron()
}

// setCronJob 设置 cron 定时任务
func (s *N9ESyncScheduler) setCronJob(cronExpr string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 如果 cron 表达式未变化，跳过
	if s.cronExpr == cronExpr && s.entryID != 0 {
		log.Printf("N9E 同步 Cron 表达式未变化: %s", cronExpr)
		return nil
	}

	// 移除旧任务
	if s.entryID != 0 {
		s.cron.Remove(s.entryID)
		s.entryID = 0
		s.cronExpr = ""
	}

	// 注册新任务
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeSyncJob()
	})
	if err != nil {
		return fmt.Errorf("无效的 Cron 表达式 '%s': %w", cronExpr, err)
	}

	s.entryID = entryID
	s.cronExpr = cronExpr
	log.Printf("N9E 同步定时任务已注册: Cron=%s", cronExpr)
	return nil
}

// removeCronJob 移除 cron 定时任务
func (s *N9ESyncScheduler) removeCronJob() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.entryID != 0 {
		s.cron.Remove(s.entryID)
		s.entryID = 0
		s.cronExpr = ""
		log.Println("N9E 同步定时任务已移除")
	}
}

// executeSyncJob 执行同步任务
func (s *N9ESyncScheduler) executeSyncJob() {
	log.Println("[N9E Cron] 开始执行定时同步任务...")
	startTime := time.Now()

	result, err := s.syncService.FullSync(s.ctx, "cron")
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("[N9E Cron] 同步失败 (耗时 %v): %v", duration, err)
		return
	}

	log.Printf("[N9E Cron] 同步完成 (耗时 %v): 业务组[新增=%d,更新=%d] 主机[新增=%d,更新=%d,跳过=%d] 数据源[新增=%d,更新=%d]",
		duration,
		result.BusiGroups.Created, result.BusiGroups.Updated,
		result.Hosts.Created, result.Hosts.Updated, result.Hosts.Skipped,
		result.Datasources.Created, result.Datasources.Updated,
	)
}

// GetStats 获取调度器状态信息
func (s *N9ESyncScheduler) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats := map[string]interface{}{
		"has_job":   s.entryID != 0,
		"cron_expr": s.cronExpr,
	}

	if s.entryID != 0 {
		entry := s.cron.Entry(s.entryID)
		if !entry.Next.IsZero() {
			stats["next_run"] = entry.Next.Format("2006-01-02 15:04:05")
		}
		if !entry.Prev.IsZero() {
			stats["last_run"] = entry.Prev.Format("2006-01-02 15:04:05")
		}
	}

	return stats
}

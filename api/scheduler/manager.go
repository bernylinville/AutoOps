package scheduler

import (
	"log"
	"sync"

	"dodevops-api/api/configcenter/model"
	inspScheduler "dodevops-api/api/inspection/scheduler"
	inspService "dodevops-api/api/inspection/service"

	"gorm.io/gorm"
)

// Manager 调度器管理器
type Manager struct {
	syncScheduler               *SyncScheduler
	n9eSyncScheduler            *N9ESyncScheduler
	expireAlertScheduler        *ExpireAlertScheduler
	deployApprovalSyncScheduler *DeployApprovalSyncScheduler
	deployTTLReaper             *DeployTTLReaper
	pipelineScheduler           *PipelineScheduler
	inspectionScheduler         *inspScheduler.Scheduler
	mutex                       sync.RWMutex
	running                     bool
}

var (
	instance *Manager
	once     sync.Once
)

// GetManager 获取调度器管理器实例（单例模式）
func GetManager(db *gorm.DB) *Manager {
	once.Do(func() {
		var inspSch *inspScheduler.Scheduler
		if db != nil {
			taskSvc := inspService.NewTaskService(db)
			inspSvc := inspService.NewInspectionService(db)
			inspSch = inspScheduler.NewScheduler(taskSvc, inspSvc)
		}
		instance = &Manager{
			syncScheduler:               NewSyncScheduler(),
			n9eSyncScheduler:            NewN9ESyncScheduler(),
			expireAlertScheduler:        NewExpireAlertScheduler(),
			deployApprovalSyncScheduler: NewDeployApprovalSyncScheduler(),
			deployTTLReaper:             NewDeployTTLReaper(),
			pipelineScheduler:           NewPipelineScheduler(),
			inspectionScheduler:         inspSch,
			running:                     false,
		}
	})
	return instance
}

// Start 启动所有调度器
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		log.Println("调度器管理器已经在运行中")
		return nil
	}

	log.Println("启动调度器管理器...")

	// 启动云厂商定时同步调度器
	if err := m.syncScheduler.Start(); err != nil {
		return err
	}

	// 启动 N9E 定时同步调度器
	if err := m.n9eSyncScheduler.Start(); err != nil {
		log.Printf("N9E 同步调度器启动警告: %v", err)
	}

	// 启动主机到期预警调度器（Phase 4）
	if err := m.expireAlertScheduler.Start(); err != nil {
		log.Printf("到期预警调度器启动警告: %v", err)
	}

	if err := m.deployApprovalSyncScheduler.Start(); err != nil {
		log.Printf("部署审批同步调度器启动警告: %v", err)
	}

	if err := m.deployTTLReaper.Start(); err != nil {
		log.Printf("部署 TTL 清理调度器启动警告: %v", err)
	}

	if err := m.pipelineScheduler.Start(); err != nil {
		log.Printf("流水线调度器启动警告: %v", err)
	}

	if m.inspectionScheduler != nil {
		if err := m.inspectionScheduler.Start(); err != nil {
			log.Printf("巡检调度器启动警告: %v", err)
		}
	}

	m.running = true
	log.Println("调度器管理器启动成功")
	return nil
}

// Stop 停止所有调度器
func (m *Manager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		log.Println("调度器管理器未在运行")
		return
	}

	log.Println("停止调度器管理器...")

	// 停止定时同步调度器
	m.syncScheduler.Stop()

	// 停止 N9E 同步调度器
	m.n9eSyncScheduler.Stop()

	// 停止到期预警调度器
	m.expireAlertScheduler.Stop()

	m.deployApprovalSyncScheduler.Stop()
	m.deployTTLReaper.Stop()
	m.pipelineScheduler.Stop()

	if m.inspectionScheduler != nil {
		m.inspectionScheduler.Stop()
	}

	m.running = false
	log.Println("调度器管理器已停止")
}

// ReloadInspectionTask reloads one inspection cron job after task configuration changes.
func (m *Manager) ReloadInspectionTask(taskID uint) {
	if m == nil || !m.running || m.inspectionScheduler == nil {
		return
	}
	m.inspectionScheduler.ReloadTask(taskID)
}

// IsRunning 检查调度器管理器是否在运行
func (m *Manager) IsRunning() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.running
}

// AddSyncSchedule 添加定时同步配置
func (m *Manager) AddSyncSchedule(schedule *model.SyncSchedule) error {
	if !m.running {
		log.Println("调度器管理器未运行，无法添加定时同步配置")
		return nil
	}

	return m.syncScheduler.AddSchedule(schedule)
}

// RemoveSyncSchedule 移除定时同步配置
func (m *Manager) RemoveSyncSchedule(scheduleID uint) {
	if !m.running {
		log.Println("调度器管理器未运行")
		return
	}

	m.syncScheduler.RemoveSchedule(scheduleID)
}

// UpdateSyncSchedule 更新定时同步配置
func (m *Manager) UpdateSyncSchedule(schedule *model.SyncSchedule) error {
	if !m.running {
		log.Println("调度器管理器未运行，无法更新定时同步配置")
		return nil
	}

	return m.syncScheduler.UpdateSchedule(schedule)
}

// ReloadSyncSchedules 重新加载所有定时同步配置
func (m *Manager) ReloadSyncSchedules() error {
	if !m.running {
		log.Println("调度器管理器未运行，无法重新加载配置")
		return nil
	}

	return m.syncScheduler.LoadActiveSchedules()
}

// GetSyncSchedulerStats 获取定时同步调度器状态
func (m *Manager) GetSyncSchedulerStats() map[string]interface{} {
	if !m.running {
		return map[string]interface{}{
			"status": "stopped",
		}
	}

	stats := m.syncScheduler.GetJobStats()
	stats["status"] = "running"
	stats["n9e"] = m.n9eSyncScheduler.GetStats()
	return stats
}

// ReloadN9ECron 重新加载 N9E 同步 Cron 配置
func (m *Manager) ReloadN9ECron() error {
	if !m.running {
		log.Println("调度器管理器未运行，无法重新加载 N9E Cron")
		return nil
	}

	return m.n9eSyncScheduler.ReloadCron()
}

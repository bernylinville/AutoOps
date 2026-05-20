package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dodevops-api/api/inspection/dao"
	"dodevops-api/api/inspection/model"
	inspectReport "dodevops-api/api/inspection/report"
	"dodevops-api/api/inspection/service/engine"
	"dodevops-api/api/inspection/service/engine/vmclient"
	n9edao "dodevops-api/api/n9e/dao"
	n9eservice "dodevops-api/api/n9e/service"
	"dodevops-api/common/config"
	"dodevops-api/common/util"
	"dodevops-api/pkg/log"

	"gorm.io/gorm"
)

// InspectionService orchestrates inspection run persistence, report generation, and notification.
type InspectionService struct {
	db        *gorm.DB
	scheduler *Scheduler
	taskService *TaskService
	runDAO      *dao.RunDAO
	resultDAO   *dao.TargetResultDAO
	alertDAO    *dao.AlertDAO
	reportDAO   *dao.ReportArtifactDAO
	notifier    *InspectionNotifier
}

// NewInspectionService creates an InspectionService.
func NewInspectionService(db *gorm.DB) *InspectionService {
	return &InspectionService{
		db:        db,
		scheduler: NewScheduler(db),
		taskService: NewTaskService(db),
		runDAO:      dao.NewRunDAO(db),
		resultDAO:   dao.NewTargetResultDAO(db),
		alertDAO:    dao.NewAlertDAO(db),
		reportDAO:   dao.NewReportArtifactDAO(db),
		notifier:    NewInspectionNotifier(db),
	}
}

// TaskService returns the underlying TaskService.
func (s *InspectionService) TaskService() *TaskService { return s.taskService }

// RunDAO returns the underlying RunDAO.
func (s *InspectionService) RunDAO() *dao.RunDAO { return s.runDAO }

// ResultDAO returns the underlying TargetResultDAO.
func (s *InspectionService) ResultDAO() *dao.TargetResultDAO { return s.resultDAO }

// AlertDAO returns the underlying AlertDAO.
func (s *InspectionService) AlertDAO() *dao.AlertDAO { return s.alertDAO }

// Scheduler returns the concurrency-control scheduler.
func (s *InspectionService) Scheduler() *Scheduler { return s.scheduler }

// InspectionConfigSnapshot captures the threshold configuration used for a run.
type InspectionConfigSnapshot struct {
	Thresholds interface{} `json:"thresholds,omitempty"`
	Version    string      `json:"version,omitempty"`
}

// ExecuteInspection 执行单次巡检：集成引擎采集、评估、持久化、报告生成、通知
// If preCreatedRun is non-nil, it is reused (the caller already persisted it as pending).
// Otherwise a new run record is created for cron/scheduled invocations.
func (s *InspectionService) ExecuteInspection(ctx context.Context, taskID uint, triggerType string, triggeredBy *uint, preCreatedRun *model.InspectionRun) (*model.InspectionRun, error) {
	// 0. Concurrency control: skip if task already has an active run, then acquire a slot.
	if s.scheduler.SkipIfRunning(taskID) {
		return nil, fmt.Errorf("任务 %d 已有进行中的巡检，跳过本次触发", taskID)
	}
	s.scheduler.acquireSlot()
	defer s.scheduler.releaseSlot()

	// 1. 获取任务配置
	task, err := s.taskService.GetTaskRaw(taskID)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %w", err)
	}

	var run *model.InspectionRun

	if preCreatedRun != nil {
		// Reuse the pending run that the controller already persisted.
		run = preCreatedRun
		run.Status = model.RunStatusRunning
		now := time.Now()
		run.StartedAt = &now
		// Ensure RunDate is set (should already be set via BeforeCreate, but be safe).
		if run.RunDate == "" {
			run.RunDate = now.Format("2006-01-02")
		}
		if err := s.runDAO.Update(run.ID, map[string]interface{}{
			"status":     model.RunStatusRunning,
			"started_at": run.StartedAt,
			"run_date":   run.RunDate,
		}); err != nil {
			return nil, fmt.Errorf("更新运行记录失败: %w", err)
		}
	} else {
		// 2. 创建运行记录
		run = &model.InspectionRun{
			TaskID:      task.ID,
			TriggerType: triggerType,
			TriggeredBy: triggeredBy,
			Status:      model.RunStatusRunning,
		}
		now := time.Now()
		run.StartedAt = &now
		run.RunDate = now.Format("2006-01-02")

		if err := s.runDAO.Create(run); err != nil {
			return nil, fmt.Errorf("创建运行记录失败: %w", err)
		}
	}

	// 3. 获取 N9E 配置并创建客户端
	n9eConfig, err := n9edao.GetN9EConfig()
	if err != nil {
		s.FailRun(run.ID, "获取 N9E 配置失败: "+err.Error())
		return nil, fmt.Errorf("获取 N9E 配置: %w", err)
	}

	n9eClient := n9eservice.NewN9EClient(n9eConfig.Endpoint, n9eConfig.Token, n9eConfig.Timeout)

	// 4. 创建 VM 客户端
	vmEndpoint := s.getVMEndpoint()
	vmClient := vmclient.NewClient(vmEndpoint, 30)

	// 5. 构建主机筛选器
	hostFilter := parseTargetQuery(task.TargetQuery)

	// 6. 创建 Collector
	metrics := engine.HostMetricDefinitions()
	collector := engine.NewCollector(n9eClient, vmClient, metrics, hostFilter, 20)

	// 7. 创建 Evaluator
	thresholds := engine.DefaultThresholds()
	evaluator := engine.NewEvaluator(thresholds, metrics)

	// 8. 创建 Inspector 并运行
	inspector := engine.NewInspector(collector, evaluator)

	log.Log().Infof("[Inspection] starting inspection for task %d (%s)", task.ID, task.Name)

	result, err := inspector.Run(ctx)
	if err != nil {
		s.FailRun(run.ID, "巡检执行失败: "+err.Error())
		return nil, fmt.Errorf("巡检执行失败: %w", err)
	}

	// 9. 持久化结果
	configSnapshot := &InspectionConfigSnapshot{
		Thresholds: thresholds,
		Version:    "1.0.0",
	}
	if err := s.SaveRunResult(run, task, result, configSnapshot, ""); err != nil {
		s.FailRun(run.ID, "持久化结果失败: "+err.Error())
		return nil, fmt.Errorf("持久化结果: %w", err)
	}

	log.Log().Infof("[Inspection] task %d completed: hosts=%d alerts=%d duration=%v",
		task.ID, result.Summary.TotalHosts, result.AlertSummary.TotalAlerts, result.Duration)

	return s.runDAO.GetByID(run.ID)
}

// FailRun 标记运行失败。Exported so callers can mark runs as failed in panic recovery paths.
func (s *InspectionService) FailRun(runID uint, errMsg string) {
	now := time.Now()
	if dbErr := s.runDAO.Update(runID, map[string]interface{}{
		"status":        model.RunStatusFailed,
		"ended_at":      &now,
		"error_message": errMsg,
	}); dbErr != nil {
		log.Log().Errorf("[Inspection] run %d: failed to update status to failed: %v", runID, dbErr)
	}
	log.Log().Errorf("[Inspection] run %d failed: %s", runID, errMsg)
}

// getVMEndpoint 获取 VictoriaMetrics/Prometheus 端点
func (s *InspectionService) getVMEndpoint() string {
	if config.Config != nil && config.Config.Monitor.Prometheus.URL != "" {
		return config.Config.Monitor.Prometheus.URL
	}
	// 兜底：使用 N9E 配置端点
	n9eConfig, err := n9edao.GetN9EConfig()
	if err == nil && n9eConfig.Endpoint != "" {
		return n9eConfig.Endpoint
	}
	return "http://localhost:9090"
}

// parseTargetQuery 解析 TargetQuery 字符串为 HostFilter
func parseTargetQuery(query string) *vmclient.HostFilter {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}

	filter := &vmclient.HostFilter{
		Tags: make(map[string]string),
	}

	parts := strings.Split(query, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		if key == "busigroup" {
			filter.BusinessGroups = append(filter.BusinessGroups, value)
		} else {
			filter.Tags[key] = value
		}
	}

	if len(filter.BusinessGroups) == 0 && len(filter.Tags) == 0 {
		return nil
	}

	return filter
}

// SaveRunResult persists engine results, generates Excel report, and sends notification.
func (s *InspectionService) SaveRunResult(
	run *model.InspectionRun,
	task *model.InspectionTask,
	result *model.InspectionResult,
	configSnapshot *InspectionConfigSnapshot,
	outputDir string,
) error {
	log.Log().Infof("[InspectionService] persisting run %d: total=%d hosts, alerts=%d",
		run.ID, result.Summary.TotalHosts, result.AlertSummary.TotalAlerts)

	// Save config snapshot as JSON (marshal once, reuse in transaction).
	var snapshotJSON model.JSONRaw
	if configSnapshot != nil {
		snapshotBytes, _ := json.Marshal(configSnapshot)
		snapshotJSON = model.JSONRaw(snapshotBytes)
	}

	// Compute target result records for alert foreign-key mapping.
	targetResults := s.buildTargetResults(run.ID, result.Hosts)
	// Step 3: Generate Excel report (outside transaction — filesystem operation).
	reportPath := s.generateReportPath(outputDir, run.ID)
	fileSize := int64(0)
	reportStatus := model.ReportStatusSuccess
	var reportErrMsg string
	if err := inspectReport.WriteHostReport(result, reportPath); err != nil {
		log.Log().Errorf("[InspectionService] run %d: failed to generate Excel: %v", run.ID, err)
		reportStatus = model.ReportStatusFailed
		reportErrMsg = err.Error()
	} else if info, statErr := os.Stat(reportPath); statErr == nil {
		fileSize = info.Size()
	}

	// Steps 1, 2, 4: DB writes in a transaction.
	endTime := time.Now()
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		resultDAO := dao.NewTargetResultDAO(tx)
		alertDAO := dao.NewAlertDAO(tx)
		reportDAO := dao.NewReportArtifactDAO(tx)
		runDAOTx := dao.NewRunDAO(tx)

		// Step 1: Save host results.
		if err := resultDAO.BatchCreate(targetResults); err != nil {
			return fmt.Errorf("保存主机结果失败: %w", err)
		}
			alertRecords := s.buildAlertRecords(run.ID, targetResults, result.Alerts)

		// Step 2: Save alerts.
		if err := alertDAO.BatchCreate(alertRecords); err != nil {
			return fmt.Errorf("保存异常记录失败: %w", err)
		}

		// Save report artifact record.
		artifact := &model.InspectionReportArtifact{
			RunID:        run.ID,
			FilePath:     reportPath,
			FileSize:     fileSize,
			Format:       "excel",
			Status:       reportStatus,
			ErrorMessage: reportErrMsg,
			ExpiresAt:    util.HTime{Time: time.Now().AddDate(0, 0, 30)},
		}
		if err := reportDAO.Create(artifact); err != nil {
			return fmt.Errorf("保存报告记录失败: %w", err)
		}

		// Step 4: Update run record with final stats.
		updates := map[string]interface{}{
			"total_hosts":    result.Summary.TotalHosts,
			"normal_hosts":   result.Summary.NormalHosts,
			"warning_hosts":  result.Summary.WarningHosts,
			"critical_hosts": result.Summary.CriticalHosts,
			"failed_hosts":   result.Summary.FailedHosts,
			"total_alerts":   result.AlertSummary.TotalAlerts,
			"ended_at":       &endTime,
			"duration_ms":    result.Duration.Milliseconds(),
			"status":         determineRunStatus(result),
		}
		if configSnapshot != nil {
			updates["config_snapshot"] = snapshotJSON
		}
		if err := runDAOTx.Update(run.ID, updates); err != nil {
			return fmt.Errorf("更新运行记录失败: %w", err)
		}

		return nil
	})

	if txErr != nil {
		log.Log().Errorf("[InspectionService] run %d: transaction failed: %v", run.ID, txErr)
		// Clean up orphaned Excel file on transaction rollback.
		if reportPath != "" {
			os.Remove(reportPath)
		}
		return txErr
	}

	// Step 5: Send notification asynchronously (non-blocking).
	// Populate run struct from result so the goroutine sees correct summary.
	run.TotalHosts = result.Summary.TotalHosts
	run.NormalHosts = result.Summary.NormalHosts
	run.WarningHosts = result.Summary.WarningHosts
	run.CriticalHosts = result.Summary.CriticalHosts
	run.FailedHosts = result.Summary.FailedHosts
	run.TotalAlerts = result.AlertSummary.TotalAlerts
	run.Status = determineRunStatus(result)
	run.EndedAt = &endTime
	run.DurationMs = result.Duration.Milliseconds()
	go func() {
		if err := s.notifier.NotifyRunResult(run, task); err != nil {
			log.Log().Errorf("[InspectionService] run %d: notification failed: %v", run.ID, err)
		}
	}()

	log.Log().Infof("[InspectionService] run %d: persistence complete", run.ID)
	return nil
}


// generateReportPath generates the output path for a report file.
func (s *InspectionService) generateReportPath(outputDir string, runID uint) string {
	dateStr := time.Now().Format("20060102")
	if outputDir == "" {
		outputDir = "/data/inspection"
	}
	dir := fmt.Sprintf("%s/%d", outputDir, runID)
	os.MkdirAll(dir, 0755)
	filename := fmt.Sprintf("inspection_report_%d_%s.xlsx", runID, dateStr)
	return fmt.Sprintf("%s/%s", dir, filename)
}

// buildTargetResults converts engine host results to DB records.
func (s *InspectionService) buildTargetResults(runID uint, hosts []*model.HostResult) []*model.InspectionTargetResult {
	var results []*model.InspectionTargetResult
	for _, host := range hosts {
		if host == nil {
			continue
		}
		metricsJSON, _ := json.Marshal(host.Metrics)
		tr := &model.InspectionTargetResult{
			RunID:    runID,
			Hostname: host.Hostname,
			IP:       host.IP,
			OS:       host.OS,
			Status:   string(host.Status),
			Error:    host.Error,
			Metrics:  model.JSONRaw(metricsJSON),
			BootTime: host.BootTime,
		}
		if !host.CollectedAt.IsZero() {
			ct := &util.HTime{Time: host.CollectedAt}
			tr.CollectedAt = ct
		}
		results = append(results, tr)
	}
	return results
}

// buildAlertRecords converts engine alerts to DB records, linking to target results.
func (s *InspectionService) buildAlertRecords(runID uint, targets []*model.InspectionTargetResult, alerts []*model.Alert) []*model.InspectionAlert {
	hostTargetMap := make(map[string]uint)
	for _, tr := range targets {
		if tr.Hostname != "" {
			hostTargetMap[tr.Hostname] = tr.ID
		}
	}
	var records []*model.InspectionAlert
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		labelsJSON, _ := json.Marshal(alert.Labels)
		record := &model.InspectionAlert{
			RunID:             runID,
			TargetResultID:    hostTargetMap[alert.Hostname],
			Hostname:          alert.Hostname,
			MetricName:        alert.MetricName,
			MetricDisplayName: alert.MetricDisplayName,
			CurrentValue:      alert.CurrentValue,
			WarningThreshold:  alert.WarningThreshold,
			CriticalThreshold: alert.CriticalThreshold,
			Level:             string(alert.Level),
			Message:           alert.Message,
			Labels:            model.JSONRaw(labelsJSON),
		}
		records = append(records, record)
	}
	return records
}

// determineRunStatus derives the run status from the result.
func determineRunStatus(result *model.InspectionResult) string {
	if result.Summary == nil {
		return model.RunStatusFailed
	}
	if result.Summary.FailedHosts == result.Summary.TotalHosts && result.Summary.TotalHosts > 0 {
		return model.RunStatusFailed
	}
	if result.Summary.CriticalHosts > 0 || result.Summary.WarningHosts > 0 {
		return model.RunStatusPartial
	}
	if result.Summary.FailedHosts > 0 {
		return model.RunStatusPartial
	}
	return model.RunStatusSuccess
}

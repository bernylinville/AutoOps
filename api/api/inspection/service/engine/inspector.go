// Package engine provides the inspection engine core.
// Ported from inspection-tool/internal/service/inspector.go.
package engine

import (
	"context"
	"fmt"
	"time"

	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/log"
)

// Inspector orchestrates the complete inspection workflow.
type Inspector struct {
	collector HostCollector
	evaluator HostEvaluator
	timezone  *time.Location
	version   string
}

// NewInspector creates a new Inspector.
func NewInspector(collector HostCollector, evaluator HostEvaluator) *Inspector {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		log.Log().Warnf("[Inspector] cannot load Asia/Shanghai timezone, falling back to UTC: %v", err)
		loc = time.UTC
	}

	return &Inspector{
		collector: collector,
		evaluator: evaluator,
		timezone:  loc,
		version:   "1.0.0",
	}
}

// Run executes the complete inspection workflow:
// 1. Collects host metadata and metrics
// 2. Evaluates thresholds to generate alerts
// 3. Aggregates results into InspectionResult
func (i *Inspector) Run(ctx context.Context) (*model.InspectionResult, error) {
	startTime := time.Now().In(i.timezone)
	log.Log().Infof("[Inspector] starting inspection (tz=%s)", i.timezone.String())

	result := model.NewInspectionResult(startTime)
	result.Version = i.version

	// Step 1: Collect data.
	log.Log().Info("[Inspector] step 1: collecting data")
	collectionResult, err := i.collector.CollectAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("data collection failed: %w", err)
	}

	if len(collectionResult.Hosts) == 0 {
		log.Log().Warn("[Inspector] no hosts found, completing with empty result")
		result.Finalize(time.Now().In(i.timezone))
		return result, nil
	}

	// Step 2: Evaluate thresholds.
	log.Log().Infof("[Inspector] step 2: evaluating thresholds (hosts_with_metrics=%d)", len(collectionResult.HostMetrics))
	evalResult := i.evaluator.EvaluateAll(collectionResult.HostMetrics)

	// Step 3: Build inspection result.
	log.Log().Info("[Inspector] step 3: building inspection result")
	i.buildInspectionResult(result, collectionResult, evalResult)

	// Step 4: Finalize.
	endTime := time.Now().In(i.timezone)
	result.Finalize(endTime)

	log.Log().Infof("[Inspector] completed: total=%d normal=%d warning=%d critical=%d failed=%d alerts=%d duration=%v",
		result.Summary.TotalHosts,
		result.Summary.NormalHosts,
		result.Summary.WarningHosts,
		result.Summary.CriticalHosts,
		result.Summary.FailedHosts,
		result.AlertSummary.TotalAlerts,
		result.Duration,
	)

	return result, nil
}

// buildInspectionResult merges collection and evaluation results into InspectionResult.
func (i *Inspector) buildInspectionResult(
	result *model.InspectionResult,
	collectionResult *CollectionResult,
	evalResult *EvaluationResult,
) {
	evalByHost := make(map[string]*HostEvaluationResult)
	for _, hostEval := range evalResult.HostResults {
		if hostEval != nil {
			evalByHost[hostEval.Hostname] = hostEval
		}
	}

	failedHosts := make(map[string]string)
	for _, failed := range collectionResult.FailedHosts {
		failedHosts[failed.Hostname] = failed.Error
	}

	for _, hostMeta := range collectionResult.Hosts {
		if hostMeta == nil {
			continue
		}

		hostResult := model.NewHostResult(hostMeta)
		hostResult.CollectedAt = time.Unix(collectionResult.CollectedAt, 0).In(i.timezone)

		// Check if this host failed collection.
		if errMsg, failed := failedHosts[hostMeta.Hostname]; failed {
			hostResult.Status = model.HostStatusFailed
			hostResult.Error = errMsg
			result.AddHost(hostResult)
			continue
		}

		// Merge evaluation results.
		if hostEval, exists := evalByHost[hostMeta.Hostname]; exists {
			hostResult.Metrics = hostEval.Metrics
			hostResult.Alerts = hostEval.Alerts
			hostResult.Status = hostEval.Status

			// Calculate BootTime from uptime metric.
			if uptimeMetric, ok := hostResult.Metrics["uptime"]; ok && !uptimeMetric.IsNA {
				uptimeSeconds := uptimeMetric.RawValue
				bootTime := result.InspectionTime.Add(-time.Duration(uptimeSeconds) * time.Second)
				hostResult.BootTime = bootTime.Format("2006-01-02 15:04:05")
			}
		}

		result.AddHost(hostResult)
	}
}

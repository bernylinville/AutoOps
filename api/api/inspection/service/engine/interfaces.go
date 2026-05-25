// Package engine provides the inspection engine core.
// Ported from inspection-tool/internal/service/interfaces.go.
package engine

import (
	"context"

	"dodevops-api/api/inspection/model"
)

// HostCollector collects host system metrics.
type HostCollector interface {
	CollectAll(ctx context.Context) (*CollectionResult, error)
}

// HostEvaluator evaluates host metrics against thresholds.
type HostEvaluator interface {
	EvaluateAll(hostMetrics map[string]*model.HostMetrics) *EvaluationResult
}

// CollectionResult contains complete collection results.
type CollectionResult struct {
	Hosts       []*model.HostMeta
	HostMetrics map[string]*model.HostMetrics
	FailedHosts []FailedHost
	CollectedAt int64 // Unix timestamp
}

// FailedHost represents a host that failed during collection.
type FailedHost struct {
	Hostname string
	Error    string
}

// HostEvaluationResult contains evaluation for a single host.
type HostEvaluationResult struct {
	Hostname string
	Status   model.HostStatus
	Metrics  map[string]*model.MetricValue
	Alerts   []*model.Alert
}

// EvaluationResult contains complete evaluation results.
type EvaluationResult struct {
	HostResults []*HostEvaluationResult
	Alerts      []*model.Alert
	Summary     *model.AlertSummary
}

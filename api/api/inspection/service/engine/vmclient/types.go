// Package vmclient provides VictoriaMetrics/Prometheus API types for inspection.
// Ported from inspection-tool/internal/client/vm/types.go.
package vmclient

import (
	"fmt"
	"strconv"

	"dodevops-api/pkg/log"
)

// QueryResponse represents the API response from /api/v1/query endpoint.
type QueryResponse struct {
	Status    string    `json:"status"`
	Data      QueryData `json:"data"`
	ErrorType string    `json:"errorType"`
	Error     string    `json:"error"`
	Warnings  []string  `json:"warnings"`
}

// IsSuccess returns true if query was successful.
func (r *QueryResponse) IsSuccess() bool { return r.Status == "success" }

// QueryData contains result data.
type QueryData struct {
	ResultType string   `json:"resultType"`
	Result     []Sample `json:"result"`
}

// IsVector returns true for instant vector.
func (d *QueryData) IsVector() bool { return d.ResultType == "vector" }

// Sample represents a single sample in query result.
type Sample struct {
	Metric Metric        `json:"metric"`
	Value  SampleValue   `json:"value"`
	Values []SampleValue `json:"values"`
}

// GetIdent returns host identifier from metric labels.
func (s *Sample) GetIdent() string {
	if ident, ok := s.Metric["ident"]; ok {
		return ident
	}
	if host, ok := s.Metric["host"]; ok {
		return host
	}
	if instance, ok := s.Metric["instance"]; ok {
		return instance
	}
	return ""
}

// GetLabel returns label value or empty string.
func (s *Sample) GetLabel(name string) string {
	if v, ok := s.Metric[name]; ok {
		return v
	}
	return ""
}

// Metric is a set of label-value pairs.
type Metric map[string]string

// Name returns metric name (__name__ label).
func (m Metric) Name() string {
	if name, ok := m["__name__"]; ok {
		return name
	}
	return ""
}

// SampleValue is [timestamp, value] pair.
type SampleValue [2]interface{}

// Value returns sample value as float64.
func (v SampleValue) Value() (float64, error) {
	if len(v) < 2 {
		return 0, fmt.Errorf("invalid sample value: length %d", len(v))
	}
	switch val := v[1].(type) {
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, fmt.Errorf("parse value %q: %w", val, err)
		}
		return f, nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("unexpected value type: %T", v[1])
	}
}

// MustValue returns value, 0 on error.
func (v SampleValue) MustValue() float64 {
	val, _ := v.Value()
	return val
}

// IsNaN returns true if value is NaN/Inf/invalid.
func (v SampleValue) IsNaN() bool {
	if v[1] == nil {
		return true
	}
	if str, ok := v[1].(string); ok {
		return str == "NaN" || str == "+Inf" || str == "-Inf"
	}
	return false
}

// QueryResult is a parsed query result.
type QueryResult struct {
	Ident  string            `json:"ident"`
	Value  float64           `json:"value"`
	Labels map[string]string `json:"labels"`
}

// ParseQueryResults converts QueryResponse to QueryResult slice.
func ParseQueryResults(resp *QueryResponse) ([]QueryResult, error) {
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("query failed: %s - %s", resp.ErrorType, resp.Error)
	}
	if !resp.Data.IsVector() {
		return nil, fmt.Errorf("unexpected result type: %s (expected vector)", resp.Data.ResultType)
	}

	results := make([]QueryResult, 0, len(resp.Data.Result))
	for _, sample := range resp.Data.Result {
		if sample.Value.IsNaN() {
			log.Log().Debugf("[VM] ParseQueryResults: skipping NaN/Inf sample for ident=%s", sample.GetIdent())
			continue
		}
		value, err := sample.Value.Value()
		if err != nil {
			log.Log().Debugf("[VM] ParseQueryResults: skipping invalid sample for ident=%s: %v", sample.GetIdent(), err)
			continue
		}
		results = append(results, QueryResult{
			Ident:  sample.GetIdent(),
			Value:  value,
			Labels: sample.Metric,
		})
	}
	return results, nil
}

// GroupResultsByIdent groups results by host identifier.
func GroupResultsByIdent(results []QueryResult) map[string]QueryResult {
	grouped := make(map[string]QueryResult, len(results))
	for _, r := range results {
		if r.Ident != "" {
			grouped[r.Ident] = r
		}
	}
	return grouped
}

// HostFilter defines filters for querying specific hosts.
type HostFilter struct {
	GroupIDs       []int64           // N9E target metadata scope; not injected into PromQL
	TargetTags     map[string]string // N9E target tags; not injected into PromQL
	BusinessGroups []string          // OR relation
	Tags           map[string]string // AND relation
}

// IsEmpty returns true if no filters are set.
func (f *HostFilter) IsEmpty() bool {
	return f == nil || (len(f.GroupIDs) == 0 && len(f.TargetTags) == 0 && len(f.BusinessGroups) == 0 && len(f.Tags) == 0)
}

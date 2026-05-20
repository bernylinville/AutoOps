// Package engine provides the inspection engine core.
package engine

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"dodevops-api/api/inspection/model"
	"dodevops-api/pkg/log"
	"dodevops-api/api/inspection/service/engine/vmclient"
	n9emodel "dodevops-api/api/n9e/model"
	n9eservice "dodevops-api/api/n9e/service"

	"golang.org/x/sync/errgroup"
)

// Collector collects host metadata and metrics from N9E and VictoriaMetrics.
type Collector struct {
	n9eClient   *n9eservice.N9EClient
	vmClient    *vmclient.Client
	metrics     []*model.MetricDefinition
	hostFilter  *vmclient.HostFilter
	concurrency int
}

// NewCollector creates a Collector.
func NewCollector(
	n9eClient *n9eservice.N9EClient,
	vmClient *vmclient.Client,
	metrics []*model.MetricDefinition,
	hostFilter *vmclient.HostFilter,
	concurrency int,
) *Collector {
	if concurrency <= 0 {
		concurrency = 20
	}
	return &Collector{
		n9eClient:   n9eClient,
		vmClient:    vmClient,
		metrics:     metrics,
		hostFilter:  hostFilter,
		concurrency: concurrency,
	}
}

// CollectAll executes the complete data collection workflow.
func (c *Collector) CollectAll(ctx context.Context) (*CollectionResult, error) {
	collectedAt := time.Now().Unix()
	log.Log().Info("[Collector] starting data collection")

	hosts, err := c.collectHostMetas(ctx)
	if err != nil {
		return nil, fmt.Errorf("collect host metas: %w", err)
	}

	if len(hosts) == 0 {
		log.Log().Warn("[Collector] no hosts found")
		return &CollectionResult{
			Hosts:       hosts,
			HostMetrics: make(map[string]*model.HostMetrics),
			CollectedAt: collectedAt,
		}, nil
	}

	hostMetrics, err := c.collectMetrics(ctx, hosts)
	if err != nil {
		return nil, fmt.Errorf("collect metrics: %w", err)
	}

	// Restrict hosts to metric scope when filter is active.
	if c.shouldRestrictHostsToMetricScope() {
		hosts, hostMetrics = c.restrictHostsToMetricScope(hosts, hostMetrics)
	}

	// Identify hosts with no metrics as failed.
	var failedHosts []FailedHost
	for _, host := range hosts {
		if hm, exists := hostMetrics[host.Hostname]; !exists || len(hm.Metrics) == 0 {
			failedHosts = append(failedHosts, FailedHost{
				Hostname: host.Hostname,
				Error:    "no metrics collected",
			})
		}
	}

	log.Log().Infof("[Collector] completed: total=%d, with_metrics=%d, failed=%d",
		len(hosts), len(hostMetrics), len(failedHosts))

	return &CollectionResult{
		Hosts:       hosts,
		HostMetrics: hostMetrics,
		FailedHosts: failedHosts,
		CollectedAt: collectedAt,
	}, nil
}

func (c *Collector) collectHostMetas(ctx context.Context) ([]*model.HostMeta, error) {
	targets, err := c.n9eClient.GetTargets(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch N9E targets: %w", err)
	}

	var hosts []*model.HostMeta
	for _, target := range targets {
		host := targetToHostMeta(target)
		if host != nil {
			hosts = append(hosts, host)
		}
	}

	log.Log().Infof("[Collector] collected %d host metas", len(hosts))
	return hosts, nil
}

// targetToHostMeta converts an N9E TargetData to HostMeta.
func targetToHostMeta(target n9emodel.TargetData) *model.HostMeta {
	hostname, cpuModel, memoryTotalStr, kernelVersion := n9eservice.ExtractTargetMetadata(target)

	ip := strings.TrimSpace(target.HostIP)
	if ip == "" {
		ip = strings.TrimSpace(target.RemoteAddr)
	}

	os := strings.TrimSpace(target.OS)
	if os == "" {
		os = "linux"
	}

	// Parse memory total (N9E reports in kB by default, but we normalize for safety).
	var memoryTotal int64
	if memoryTotalStr != "" {
		memoryTotal = parseMemoryToKB(memoryTotalStr)
	}

	hostMeta := &model.HostMeta{
		Ident:         target.Ident,
		Hostname:      hostname,
		IP:            ip,
		OS:            os,
		OSVersion:     strings.TrimSpace(target.Arch),
		KernelVersion: kernelVersion,
		CPUCores:      target.CPUNum,
		CPUModel:      cpuModel,
		MemoryTotal:   memoryTotal,
	}

	return hostMeta
}

func (c *Collector) collectMetrics(ctx context.Context, hosts []*model.HostMeta) (map[string]*model.HostMetrics, error) {
	hostMetricsMap := make(map[string]*model.HostMetrics, len(hosts))
	for _, host := range hosts {
		hostMetricsMap[host.Hostname] = model.NewHostMetrics(host.Hostname)
	}

	var pendingMetrics, activeMetrics []*model.MetricDefinition
	for _, m := range c.metrics {
		if m.IsPending() {
			pendingMetrics = append(pendingMetrics, m)
		} else {
			activeMetrics = append(activeMetrics, m)
		}
	}

	// Set N/A for pending metrics.
	for _, metric := range pendingMetrics {
		for _, hm := range hostMetricsMap {
			hm.SetMetric(model.NewNAMetricValue(metric.Name))
		}
	}

	// Collect active metrics concurrently.
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)

	var mu sync.Mutex

	for _, metric := range activeMetrics {
		metric := metric
		g.Go(func() error {
			var err error
			if metric.HasExpandLabel() {
				err = c.collectExpandedMetric(ctx, metric, hostMetricsMap, &mu)
			} else {
				err = c.collectSimpleMetric(ctx, metric, hostMetricsMap, &mu)
			}
			if err != nil {
				log.Log().Warnf("[Collector] metric %s failed: %v", metric.Name, err)
			}
			return nil // Single metric failure does not abort.
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("concurrent metric collection: %w", err)
	}

	return hostMetricsMap, nil
}

func (c *Collector) collectSimpleMetric(
	ctx context.Context,
	metric *model.MetricDefinition,
	hostMetricsMap map[string]*model.HostMetrics,
	mu *sync.Mutex,
) error {
	results, err := c.vmClient.QueryByIdentWithFilter(ctx, metric.Query, c.hostFilter)
	if err != nil {
		return fmt.Errorf("query %s: %w", metric.Name, err)
	}

	mu.Lock()
	defer mu.Unlock()

	for ident, result := range results {
		hostname := model.CleanIdent(ident)
		if hostMetrics, exists := hostMetricsMap[hostname]; exists {
			mv := model.NewMetricValue(metric.Name, result.Value)
			mv.Timestamp = time.Now().Unix()
			hostMetrics.SetMetric(mv)
		}
	}
	return nil
}

func (c *Collector) collectExpandedMetric(
	ctx context.Context,
	metric *model.MetricDefinition,
	hostMetricsMap map[string]*model.HostMetrics,
	mu *sync.Mutex,
) error {
	results, err := c.vmClient.QueryResultsWithFilter(ctx, metric.Query, c.hostFilter)
	if err != nil {
		return fmt.Errorf("query %s: %w", metric.Name, err)
	}

	hostMaxValues := make(map[string]float64)
	hostExpandedMetrics := make(map[string][]*model.MetricValue)

	for _, result := range results {
		hostname := model.CleanIdent(result.Ident)
		if hostname == "" {
			continue
		}
		if _, exists := hostMetricsMap[hostname]; !exists {
			continue
		}

		labelValue := result.Labels[metric.ExpandByLabel]
		if labelValue == "" {
			labelValue = "unknown"
		}

		// Filter non-physical disk paths.
		if metric.ExpandByLabel == "path" {
			if !isPhysicalDiskPath(labelValue) {
				continue
			}
		}

		expandedName := fmt.Sprintf("%s:%s", metric.Name, labelValue)
		mv := model.NewMetricValue(expandedName, result.Value)
		mv.Timestamp = time.Now().Unix()
		mv.Labels = map[string]string{metric.ExpandByLabel: labelValue}

		hostExpandedMetrics[hostname] = append(hostExpandedMetrics[hostname], mv)

		if current, exists := hostMaxValues[hostname]; !exists || result.Value > current {
			hostMaxValues[hostname] = result.Value
		}
	}

	mu.Lock()
	defer mu.Unlock()

	for hostname, metrics := range hostExpandedMetrics {
		if hostMetrics, exists := hostMetricsMap[hostname]; exists {
			for _, mv := range metrics {
				hostMetrics.SetMetric(mv)
			}
		}
	}

	if metric.Aggregate == model.AggregateMax {
		for hostname, maxValue := range hostMaxValues {
			if hostMetrics, exists := hostMetricsMap[hostname]; exists {
				mv := model.NewMetricValue(fmt.Sprintf("%s_max", metric.Name), maxValue)
				mv.Timestamp = time.Now().Unix()
				hostMetrics.SetMetric(mv)
			}
		}
	}

	return nil
}

func (c *Collector) shouldRestrictHostsToMetricScope() bool {
	if c.hostFilter == nil || c.hostFilter.IsEmpty() {
		return false
	}
	for _, metric := range c.metrics {
		if metric != nil && !metric.IsPending() {
			return true
		}
	}
	return false
}

func (c *Collector) restrictHostsToMetricScope(
	hosts []*model.HostMeta,
	hostMetrics map[string]*model.HostMetrics,
) ([]*model.HostMeta, map[string]*model.HostMetrics) {
	if len(hosts) == 0 || len(hostMetrics) == 0 {
		return hosts, hostMetrics
	}

	inScope := make(map[string]struct{}, len(hostMetrics))
	for hostname, metrics := range hostMetrics {
		if hasCollectedMetric(metrics) {
			inScope[hostname] = struct{}{}
		}
	}

	filteredHosts := make([]*model.HostMeta, 0, len(inScope))
	filteredMetrics := make(map[string]*model.HostMetrics, len(inScope))
	for _, host := range hosts {
		if host == nil {
			continue
		}
		if _, ok := inScope[host.Hostname]; !ok {
			continue
		}
		filteredHosts = append(filteredHosts, host)
		if m, exists := hostMetrics[host.Hostname]; exists {
			filteredMetrics[host.Hostname] = m
		}
	}

	log.Log().Infof("[Collector] host filter: %d -> %d", len(hosts), len(filteredHosts))
	return filteredHosts, filteredMetrics
}

func hasCollectedMetric(metrics *model.HostMetrics) bool {
	if metrics == nil {
		return false
	}
	for _, value := range metrics.Metrics {
		if value != nil && !value.IsNA {
			return true
		}
	}
	return false
}

func isPhysicalDiskPath(path string) bool {
	if path == "" || path == "unknown" {
		return false
	}

	// Skip Kubernetes and container paths.
	if strings.Contains(path, "/var/lib/kubelet/plugins/kubernetes.io/csi/") ||
		strings.Contains(path, "/var/lib/kubelet/pods/") ||
		strings.Contains(path, "/run/containerd/") ||
		strings.Contains(path, "/var/lib/docker/") ||
		strings.Contains(path, "/overlay/") ||
		strings.Contains(path, "/overlay2/") {
		return false
	}

	// Skip snap mounts.
	if strings.HasPrefix(path, "/snap/") {
		return false
	}

	// Skip system virtual filesystems.
	virtualPaths := []string{"/dev", "/dev/shm", "/proc", "/sys", "/run", "/run/lock", "/run/user"}
	for _, vp := range virtualPaths {
		if path == vp || strings.HasPrefix(path, vp+"/") {
			return false
		}
	}

	return true
}

// parseMemoryToKB parses a memory string with optional unit suffix and normalizes to kilobytes.
// Expected N9E format: "16000000kB". Also handles MB, GB, or bare numbers (assumed kB).
func parseMemoryToKB(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	upper := strings.ToUpper(s)
	unit := ""

	// Check for unit suffix (longest match first: GB > MB > KB > K).
	switch {
	case strings.HasSuffix(upper, "GB"):
		unit = "GB"
	case strings.HasSuffix(upper, "MB"):
		unit = "MB"
	case strings.HasSuffix(upper, "KB"):
		unit = "KB"
	case strings.HasSuffix(upper, "K"):
		unit = "K"
	}

	numPart := s
	if unit != "" {
		numPart = strings.TrimSpace(s[:len(s)-len(unit)])
	}

	val, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		log.Log().Warnf("[Collector] failed to parse memory string %q: %v", s, err)
		return 0
	}

	switch unit {
	case "GB":
		val *= 1024 * 1024 // GB → kB
	case "MB":
		val *= 1024 // MB → kB
	}

	return int64(val)
}

// Package engine provides Host inspection threshold configuration.
package engine

// ThresholdPair defines warning and critical thresholds for a metric.
type ThresholdPair struct {
	Warning  float64 `json:"warning"`
	Critical float64 `json:"critical"`
}

// ThresholdsConfig contains threshold configurations for Host inspection alerts.
type ThresholdsConfig struct {
	CPUUsage        ThresholdPair `json:"cpu_usage"`
	MemoryUsage     ThresholdPair `json:"memory_usage"`
	DiskUsage       ThresholdPair `json:"disk_usage"`
	ZombieProcesses ThresholdPair `json:"zombie_processes"`
	LoadPerCore     ThresholdPair `json:"load_per_core"`
	NTPOffset       ThresholdPair `json:"ntp_offset"`
}

// DefaultThresholds returns the default Host inspection thresholds.
func DefaultThresholds() *ThresholdsConfig {
	return &ThresholdsConfig{
		CPUUsage:        ThresholdPair{Warning: 70, Critical: 90},
		MemoryUsage:     ThresholdPair{Warning: 70, Critical: 90},
		DiskUsage:       ThresholdPair{Warning: 70, Critical: 90},
		ZombieProcesses: ThresholdPair{Warning: 1, Critical: 10},
		LoadPerCore:     ThresholdPair{Warning: 0.7, Critical: 1.0},
		NTPOffset:       ThresholdPair{Warning: 0.5, Critical: 1.0},
	}
}

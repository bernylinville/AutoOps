package model

import (
	"testing"
)

// ========================================
// Table Name Tests
// ========================================

func TestTableNames(t *testing.T) {
	tests := []struct {
		name     string
		table    string
		expected string
	}{
		{"CmdbHost", CmdbHost{}.TableName(), "cmdb_host"},
		{"Project", Project{}.TableName(), "cmdb_project"},
		{"CIChangeLog", CIChangeLog{}.TableName(), "ci_change_log"},
		{"NetworkInspection", NetworkInspection{}.TableName(), "network_inspection"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.table != tt.expected {
				t.Errorf("TableName() = %s, want %s", tt.table, tt.expected)
			}
		})
	}
}

// ========================================
// Host Lifecycle Transition Tests
// ========================================

func TestHostLifecycleAllowedTransitions(t *testing.T) {
	// Verify the transition map contains all expected states
	expectedStates := []int{0, 1, 3, 4, 5, 6, 7, 8, 9, 10}
	for _, state := range expectedStates {
		if _, exists := HostLifecycleAllowedTransitions[state]; !exists {
			t.Errorf("State %d missing from HostLifecycleAllowedTransitions", state)
		}
	}
}

func TestHostLifecycleValidTransitions(t *testing.T) {
	validCases := []struct {
		name string
		from int
		to   int
	}{
		{"Initial to Procurement", 0, 6},
		{"Procurement to Inventory", 6, 7},
		{"Inventory to Pending-Online", 7, 8},
		{"Pending-Online to Online", 8, 1},
		{"Online to Decommission-Request", 1, 9},
		{"Online to Degraded", 1, 5},
		{"Offline to Decommission-Request", 3, 9},
		{"Lost to Decommission-Request", 4, 9},
		{"Degraded to Decommission-Request", 5, 9},
		{"Degraded to Online", 5, 1},
		{"Decommission-Request to Scrapped", 9, 10},
		{"Decommission-Request to Online (recall)", 9, 1},
	}
	for _, tc := range validCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, ok := HostLifecycleAllowedTransitions[tc.from]
			if !ok {
				t.Fatalf("State %d not in transition map", tc.from)
			}
			found := false
			for _, target := range allowed {
				if target == tc.to {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Transition %d -> %d should be allowed but wasn't", tc.from, tc.to)
			}
		})
	}
}

func TestHostLifecycleInvalidTransitions(t *testing.T) {
	invalidCases := []struct {
		name string
		from int
		to   int
	}{
		{"Scrapped is terminal", 10, 1},
		{"Scrapped to procurement", 10, 6},
		{"Procurement cannot skip to online", 6, 1},
		{"Inventory cannot skip to online", 7, 1},
		{"Online cannot go to scrapped directly", 1, 10},
		{"Offline cannot go to online directly", 3, 1},
	}
	for _, tc := range invalidCases {
		t.Run(tc.name, func(t *testing.T) {
			allowed, ok := HostLifecycleAllowedTransitions[tc.from]
			if !ok {
				// State not in map is fine for "not allowed"
				return
			}
			for _, target := range allowed {
				if target == tc.to {
					t.Errorf("Transition %d -> %d should NOT be allowed but was", tc.from, tc.to)
				}
			}
		})
	}
}

// ========================================
// Project Model Struct Tests
// ========================================

func TestProjectVoFields(t *testing.T) {
	vo := ProjectVo{
		ID:        1,
		Name:      "TestProject",
		Code:      "TP001",
		HostCount: 10,
		DBCount:   3,
		AppCount:  5,
		Status:    1,
	}
	if vo.ID != 1 || vo.Name != "TestProject" || vo.Code != "TP001" {
		t.Error("ProjectVo basic fields not set correctly")
	}
	if vo.HostCount != 10 || vo.DBCount != 3 || vo.AppCount != 5 {
		t.Error("ProjectVo count fields not set correctly")
	}
}

func TestProjectStatsVo(t *testing.T) {
	stats := ProjectStatsVo{
		ProjectID:      1,
		TotalHosts:     100,
		OnlineHosts:    80,
		OfflineHosts:   20,
		TotalDatabases: 5,
		TotalApps:      3,
		HostsByGroup: []GroupCountVo{
			{GroupID: 1, GroupName: "Prod", Count: 60},
			{GroupID: 2, GroupName: "Test", Count: 40},
		},
		DBsByType: []TypeCountVo{
			{Type: 1, Count: 3},
			{Type: 2, Count: 2},
		},
	}
	if stats.TotalHosts != stats.OnlineHosts+stats.OfflineHosts {
		t.Error("OnlineHosts + OfflineHosts should equal TotalHosts")
	}
	if len(stats.HostsByGroup) != 2 {
		t.Error("Expected 2 groups in HostsByGroup")
	}
	if len(stats.DBsByType) != 2 {
		t.Error("Expected 2 types in DBsByType")
	}
}

// ========================================
// CIChangeLog Tests
// ========================================

func TestCIChangeLogQueryDefaults(t *testing.T) {
	dto := CIChangeLogQueryDto{
		EntityType: "ci_instance",
		EntityID:   42,
		Page:       1,
		PageSize:   20,
	}
	if dto.EntityType != "ci_instance" {
		t.Error("EntityType should be ci_instance")
	}
	if dto.Page < 1 {
		t.Error("Page should be >= 1")
	}
}

// ========================================
// NetworkInspection Tests
// ========================================

func TestNetworkInspectionTableName(t *testing.T) {
	n := NetworkInspection{}
	if n.TableName() != "network_inspection" {
		t.Errorf("Expected network_inspection, got %s", n.TableName())
	}
}

func TestNetworkInspectionVoMapping(t *testing.T) {
	vo := NetworkInspectionVo{
		ID:        1,
		MgmtIP:    "192.168.1.1",
		Reachable: true,
		LatencyMs: 15,
		Port:      22,
		Method:    "tcp",
	}
	if vo.MgmtIP != "192.168.1.1" || !vo.Reachable || vo.Method != "tcp" {
		t.Error("NetworkInspectionVo fields not mapped correctly")
	}
}

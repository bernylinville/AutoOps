package model

import (
	"encoding/json"
	"testing"
)

// ========================================
// UserIDs Value/Scan Tests
// ========================================

func TestUserIDsValueNil(t *testing.T) {
	var u UserIDs
	val, err := u.Value()
	if err != nil {
		t.Fatalf("UserIDs.Value() error: %v", err)
	}
	if val != nil {
		t.Errorf("Expected nil for empty UserIDs, got %v", val)
	}
}

func TestUserIDsValueWithData(t *testing.T) {
	u := UserIDs{1, 2, 3}
	val, err := u.Value()
	if err != nil {
		t.Fatalf("UserIDs.Value() error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Expected []byte, got %T", val)
	}
	var result []uint
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result) != 3 || result[0] != 1 || result[2] != 3 {
		t.Errorf("Unexpected result: %v", result)
	}
}

func TestUserIDsScanNil(t *testing.T) {
	var u UserIDs
	err := u.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
	if len(u) != 0 {
		t.Errorf("Expected empty UserIDs after Scan(nil), got %v", u)
	}
}

func TestUserIDsScanBytes(t *testing.T) {
	var u UserIDs
	data := []byte(`[10,20,30]`)
	err := u.Scan(data)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(u) != 3 || u[0] != 10 || u[1] != 20 || u[2] != 30 {
		t.Errorf("Expected [10,20,30], got %v", u)
	}
}

func TestUserIDsScanNonBytes(t *testing.T) {
	var u UserIDs
	err := u.Scan(12345)
	if err != nil {
		t.Fatalf("Scan(int) should not error, got: %v", err)
	}
}

// ========================================
// ResourceIDs Value/Scan Tests
// ========================================

func TestResourceIDsValueNil(t *testing.T) {
	var r ResourceIDs
	val, err := r.Value()
	if err != nil {
		t.Fatalf("ResourceIDs.Value() error: %v", err)
	}
	if val != nil {
		t.Errorf("Expected nil for empty ResourceIDs, got %v", val)
	}
}

func TestResourceIDsValueWithData(t *testing.T) {
	r := ResourceIDs{100, 200}
	val, err := r.Value()
	if err != nil {
		t.Fatalf("ResourceIDs.Value() error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Expected []byte, got %T", val)
	}
	var result []uint
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result) != 2 || result[0] != 100 {
		t.Errorf("Unexpected: %v", result)
	}
}

func TestResourceIDsScanBytes(t *testing.T) {
	var r ResourceIDs
	data := []byte(`[5,6]`)
	err := r.Scan(data)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(r) != 2 || r[0] != 5 || r[1] != 6 {
		t.Errorf("Expected [5,6], got %v", r)
	}
}

// ========================================
// DomainsJSON Value/Scan Tests
// ========================================

func TestDomainsJSONValueNil(t *testing.T) {
	var d DomainsJSON
	val, err := d.Value()
	if err != nil {
		t.Fatalf("DomainsJSON.Value() error: %v", err)
	}
	if val != nil {
		t.Errorf("Expected nil, got %v", val)
	}
}

func TestDomainsJSONValueWithData(t *testing.T) {
	d := DomainsJSON{"example.com", "test.cn"}
	val, err := d.Value()
	if err != nil {
		t.Fatalf("DomainsJSON.Value() error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Expected []byte, got %T", val)
	}
	var result []string
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result) != 2 || result[0] != "example.com" {
		t.Errorf("Unexpected: %v", result)
	}
}

func TestDomainsJSONScanBytes(t *testing.T) {
	var d DomainsJSON
	data := []byte(`["a.com","b.com"]`)
	err := d.Scan(data)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(d) != 2 || d[0] != "a.com" {
		t.Errorf("Expected [a.com, b.com], got %v", d)
	}
}

// ========================================
// OtherResources Value/Scan Tests
// ========================================

func TestOtherResourcesValueEmpty(t *testing.T) {
	o := OtherResources{}
	val, err := o.Value()
	if err != nil {
		t.Fatalf("OtherResources.Value() error: %v", err)
	}
	if val == nil {
		t.Error("OtherResources.Value() should not be nil for empty struct")
	}
}

func TestOtherResourcesValueWithData(t *testing.T) {
	o := OtherResources{
		Redis:     []string{"redis-1:6379", "redis-2:6379"},
		Kafka:     []string{"kafka-1:9092"},
		RabbitMQ:  []string{},
		Zookeeper: nil,
	}
	val, err := o.Value()
	if err != nil {
		t.Fatalf("OtherResources.Value() error: %v", err)
	}
	bytes, ok := val.([]byte)
	if !ok {
		t.Fatalf("Expected []byte, got %T", val)
	}
	var result OtherResources
	if err := json.Unmarshal(bytes, &result); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if len(result.Redis) != 2 || result.Redis[0] != "redis-1:6379" {
		t.Errorf("Unexpected Redis: %v", result.Redis)
	}
	if len(result.Kafka) != 1 {
		t.Errorf("Unexpected Kafka: %v", result.Kafka)
	}
}

func TestOtherResourcesScanNil(t *testing.T) {
	var o OtherResources
	err := o.Scan(nil)
	if err != nil {
		t.Fatalf("Scan(nil) error: %v", err)
	}
}

func TestOtherResourcesScanBytes(t *testing.T) {
	var o OtherResources
	data := []byte(`{"redis":["r1"],"kafka":["k1","k2"]}`)
	err := o.Scan(data)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(o.Redis) != 1 || o.Redis[0] != "r1" {
		t.Errorf("Redis: %v", o.Redis)
	}
	if len(o.Kafka) != 2 {
		t.Errorf("Kafka: %v", o.Kafka)
	}
}

// ========================================
// Application Table Name Tests
// ========================================

func TestApplicationTableName(t *testing.T) {
	a := Application{}
	if a.TableName() != "app_application" {
		t.Errorf("Expected app_application, got %s", a.TableName())
	}
}

func TestJenkinsEnvTableName(t *testing.T) {
	j := JenkinsEnv{}
	if j.TableName() != "app_jenkins_env" {
		t.Errorf("Expected app_jenkins_env, got %s", j.TableName())
	}
}

func TestQuickDeploymentTableName(t *testing.T) {
	q := QuickDeployment{}
	if q.TableName() != "quick_deployments" {
		t.Errorf("Expected quick_deployments, got %s", q.TableName())
	}
}

func TestQuickDeploymentTaskTableName(t *testing.T) {
	q := QuickDeploymentTask{}
	if q.TableName() != "quick_deployment_tasks" {
		t.Errorf("Expected quick_deployment_tasks, got %s", q.TableName())
	}
}

// ========================================
// Roundtrip Serialization Tests
// ========================================

func TestUserIDsRoundtrip(t *testing.T) {
	original := UserIDs{1, 2, 3, 100}
	val, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored UserIDs
	err = restored.Scan(val.([]byte))
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != len(original) {
		t.Fatalf("Length mismatch: %d vs %d", len(restored), len(original))
	}
	for i := range original {
		if original[i] != restored[i] {
			t.Errorf("Mismatch at %d: %d vs %d", i, original[i], restored[i])
		}
	}
}

func TestResourceIDsRoundtrip(t *testing.T) {
	original := ResourceIDs{10, 20, 30}
	val, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored ResourceIDs
	err = restored.Scan(val.([]byte))
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 3 || restored[1] != 20 {
		t.Errorf("Roundtrip failed: %v", restored)
	}
}

func TestDomainsJSONRoundtrip(t *testing.T) {
	original := DomainsJSON{"example.com", "test.cn", "inner.dev"}
	val, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored DomainsJSON
	err = restored.Scan(val.([]byte))
	if err != nil {
		t.Fatal(err)
	}
	if len(restored) != 3 || restored[2] != "inner.dev" {
		t.Errorf("Roundtrip failed: %v", restored)
	}
}

func TestOtherResourcesRoundtrip(t *testing.T) {
	original := OtherResources{
		Redis:     []string{"r1:6379"},
		Kafka:     []string{"k1:9092", "k2:9092"},
		Zookeeper: []string{"zk1:2181"},
		Other:     []string{"custom-svc"},
	}
	val, err := original.Value()
	if err != nil {
		t.Fatal(err)
	}
	var restored OtherResources
	err = restored.Scan(val.([]byte))
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Redis) != 1 || len(restored.Kafka) != 2 || len(restored.Zookeeper) != 1 || len(restored.Other) != 1 {
		t.Errorf("Roundtrip failed: %+v", restored)
	}
}

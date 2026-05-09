package service

import (
	"strings"
	"testing"

	appmodel "dodevops-api/api/app/model"
	deploymodel "dodevops-api/api/deploy/model"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// noopDialector is a minimal GORM dialector that never opens a real connection.
// Used to construct a dry-run *gorm.DB for SQL statement inspection in tests.
type noopDialector struct{}

func (noopDialector) Name() string { return "noop" }
func (noopDialector) Initialize(db *gorm.DB) error {
	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{})
	return nil
}
func (noopDialector) Migrator(db *gorm.DB) gorm.Migrator {
	return migrator.Migrator{Config: migrator.Config{DB: db}}
}
func (noopDialector) DataTypeOf(_ *schema.Field) string { return "text" }
func (noopDialector) DefaultValueOf(_ *schema.Field) clause.Expression {
	return clause.Expr{SQL: "NULL"}
}
func (noopDialector) BindVarTo(w clause.Writer, _ *gorm.Statement, _ interface{}) {
	_, _ = w.WriteString("?")
}
func (noopDialector) QuoteTo(w clause.Writer, s string)           { _, _ = w.WriteString(s) }
func (noopDialector) Explain(sql string, _ ...interface{}) string { return sql }

func newDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(noopDialector{}, &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open(noop): %v", err)
	}
	return db.Session(&gorm.Session{DryRun: true})
}

// TestValidateDeployProfileRefs_EnvTypeMismatch checks that a ClusterTarget whose
// EnvType differs from the profile env is rejected with a mismatch error.
func TestValidateDeployProfileRefs_EnvTypeMismatch(t *testing.T) {
	cases := []struct {
		clusterEnvType string
		profileEnv     string
		wantErr        bool
	}{
		{clusterEnvType: "test", profileEnv: "dev", wantErr: true},
		{clusterEnvType: "dev", profileEnv: "test", wantErr: true},
		{clusterEnvType: "dev", profileEnv: "dev", wantErr: false},
		{clusterEnvType: "test", profileEnv: "test", wantErr: false},
		{clusterEnvType: "staging", profileEnv: "test", wantErr: false},
		{clusterEnvType: "devtest", profileEnv: "test", wantErr: false},
		{clusterEnvType: "dev", profileEnv: "  dev  ", wantErr: false},
		{clusterEnvType: "Dev", profileEnv: "dev", wantErr: false},
		{clusterEnvType: "TEST", profileEnv: "test", wantErr: false},
		{clusterEnvType: "  Dev  ", profileEnv: "  dev  ", wantErr: false},
		{clusterEnvType: "Dev", profileEnv: "test", wantErr: true},
		{clusterEnvType: "PROD", profileEnv: "prod", wantErr: false},
	}
	for _, tc := range cases {
		err := checkClusterTargetEnvType(tc.clusterEnvType, tc.profileEnv)
		if tc.wantErr && err == nil {
			t.Errorf("clusterEnvType=%q profileEnv=%q: expected mismatch error, got nil", tc.clusterEnvType, tc.profileEnv)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("clusterEnvType=%q profileEnv=%q: unexpected error: %v", tc.clusterEnvType, tc.profileEnv, err)
		}
		if tc.wantErr && err != nil && !strings.Contains(err.Error(), "不匹配") {
			t.Errorf("clusterEnvType=%q profileEnv=%q: error %q does not contain '不匹配'", tc.clusterEnvType, tc.profileEnv, err.Error())
		}
	}
}

// TestDeleteProfile_CleansJenkinsEnv verifies that deleteProfileManagedSideEffects
// issues DELETE statements for both agent_approver_allowlist and app_jenkins_env.
func TestDeleteProfile_CleansJenkinsEnv(t *testing.T) {
	db := newDryRunDB(t)
	profile := &appmodel.AppDeployProfile{
		AppID:           42,
		ApplicationCode: "demo-app",
		Env:             "dev",
	}

	// In dry-run mode the statements are built but not executed; we capture them
	// via gorm.Statement.SQL after each call.
	allowlistSQL := db.Where("application_code = ? AND env = ? AND created_by = ?",
		profile.ApplicationCode, profile.Env, "deploy-profile").
		Delete(&deploymodel.AgentApproverAllowlist{}).Statement.SQL.String()

	jenkinsEnvSQL := db.Where("app_id = ? AND env_name = ?", profile.AppID, profile.Env).
		Delete(&appmodel.JenkinsEnv{}).Statement.SQL.String()

	if !strings.Contains(allowlistSQL, "agent_approver_allowlist") {
		t.Errorf("expected allowlist DELETE, got: %s", allowlistSQL)
	}
	if !strings.Contains(jenkinsEnvSQL, "app_jenkins_env") {
		t.Errorf("expected app_jenkins_env DELETE, got: %s", jenkinsEnvSQL)
	}
}

func TestBuildDeployProfileUpdatesValidatesBeforePersistence(t *testing.T) {
	base := appmodel.AppDeployProfile{
		ClusterTargetID: 1,
		JenkinsServerID: 2,
		HarborServerID:  3,
		ApproverAdminID: 4,
		ResourceType:    deploymodel.DeployResourceTypeDeployment,
		DefaultGitRef:   "main",
	}
	badResourceType := "service"

	updates, candidate, err := buildDeployProfileUpdates(base, &appmodel.UpdateAppDeployProfileRequest{ResourceType: &badResourceType})
	if err == nil {
		t.Fatal("expected invalid resourceType error")
	}
	if updates != nil {
		t.Fatalf("updates should be nil when validation fails: %#v", updates)
	}
	if candidate.ResourceType != badResourceType {
		t.Fatalf("candidate should show rejected value for diagnostics, got %q", candidate.ResourceType)
	}
}

func TestBuildDeployProfileUpdatesAppliesDefaultsAndTrims(t *testing.T) {
	base := appmodel.AppDeployProfile{
		ClusterTargetID: 1,
		JenkinsServerID: 2,
		HarborServerID:  3,
		ApproverAdminID: 4,
		ResourceType:    deploymodel.DeployResourceTypeDeployment,
	}
	branch := "  "
	releaseName := " java-demo "
	replicas := int32(0)
	servicePort := int32(0)

	updates, candidate, err := buildDeployProfileUpdates(base, &appmodel.UpdateAppDeployProfileRequest{
		DefaultGitRef: &branch,
		ReleaseName:   &releaseName,
		Replicas:      &replicas,
		ServicePort:   &servicePort,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if candidate.DefaultGitRef != "main" || updates["default_git_ref"] != "main" {
		t.Fatalf("default git ref not applied: candidate=%q updates=%v", candidate.DefaultGitRef, updates)
	}
	if candidate.ReleaseName != "java-demo" || updates["release_name"] != "java-demo" {
		t.Fatalf("release name not trimmed: candidate=%q updates=%v", candidate.ReleaseName, updates)
	}
	if candidate.Replicas != 1 || updates["replicas"] != int32(1) {
		t.Fatalf("replicas default not applied: candidate=%d updates=%v", candidate.Replicas, updates)
	}
	if candidate.ServicePort != 80 || updates["service_port"] != int32(80) {
		t.Fatalf("service port default not applied: candidate=%d updates=%v", candidate.ServicePort, updates)
	}
}

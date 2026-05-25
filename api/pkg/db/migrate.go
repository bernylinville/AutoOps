// pkg/db/migrate.go
package db

import (
	appmodel "dodevops-api/api/app/model"
	cmdbmodel "dodevops-api/api/cmdb/model"
	ccmodel "dodevops-api/api/configcenter/model"
	deploymodel "dodevops-api/api/deploy/model"
	inspectionmodel "dodevops-api/api/inspection/model"
	k8smodel "dodevops-api/api/k8s/model"
	monitormodel "dodevops-api/api/monitor/model"
	n9emodel "dodevops-api/api/n9e/model"
	systemmodel "dodevops-api/api/system/model"
	taskmodel "dodevops-api/api/task/model"
	toolmodel "dodevops-api/api/tool/model"

	"gorm.io/gorm"
)

// 注册所有需要自动建表的 model
var models = []interface{}{
	// System models
	&systemmodel.SysAdmin{},
	&systemmodel.SysRole{},
	&systemmodel.SysMenu{},
	&systemmodel.SysAdminRole{},
	&systemmodel.SysRoleMenu{},
	&systemmodel.SysDept{},
	&systemmodel.SysPost{},
	&systemmodel.SysLoginInfo{},
	&systemmodel.SysOperationLog{},
	&systemmodel.SysAuditLog{},
	// CMDB models
	&cmdbmodel.CmdbGroup{},
	&cmdbmodel.CmdbHost{},
	&cmdbmodel.CmdbSQL{},
	&cmdbmodel.CmdbSQLRecord{},
	// CI 动态模型
	&cmdbmodel.CIType{},
	&cmdbmodel.CITypeAttribute{},
	&cmdbmodel.CIInstance{},
	&cmdbmodel.CIRelation{},
	// Project 项目维度模型
	&cmdbmodel.Project{},
	// 变更日志（Phase 4）
	&cmdbmodel.CIChangeLog{},
	// 网络设备巡检（Phase 5）
	&cmdbmodel.NetworkInspection{},
	// Config center models
	&ccmodel.EcsAuth{},
	&ccmodel.KeyManage{},
	&ccmodel.SyncSchedule{},
	&ccmodel.AccountAuth{},
	// Task models
	&taskmodel.TaskTemplate{},
	&taskmodel.Task{},
	&taskmodel.TaskWork{},
	&taskmodel.TaskAnsible{},
	&taskmodel.TaskAnsibleWork{},
	// Monitor models
	&monitormodel.Agent{},
	// K8s models
	&k8smodel.KubeCluster{},
	// Deploy models
	&deploymodel.ClusterTarget{},
	&deploymodel.DeployRequest{},
	&deploymodel.ApprovalRecord{},
	&deploymodel.ExecutionRecord{},
	&deploymodel.ResourceOwner{},
	&deploymodel.DeployNotification{},
	&deploymodel.PipelineRun{},
	&deploymodel.PipelineStageRecord{},
	// App models
	&appmodel.Application{},
	&appmodel.JenkinsEnv{},
	&appmodel.QuickDeployment{},
	&appmodel.QuickDeploymentTask{},
	// Tool models
	&toolmodel.Tool{},
	&toolmodel.ServiceDeploy{},
	// N9E models
	&n9emodel.N9EConfig{},
	&n9emodel.N9EBusiGroup{},
	&n9emodel.N9EDataSource{},
	&n9emodel.N9ESyncLog{},
	// Alert models
	&n9emodel.AlertRule{},
	&n9emodel.AlertEvent{},
	&n9emodel.NotifyChannel{},
	// Inspection models
	&inspectionmodel.InspectionTask{},
	&inspectionmodel.InspectionRun{},
	&inspectionmodel.InspectionTargetResult{},
	&inspectionmodel.InspectionAlert{},
	&inspectionmodel.InspectionReportArtifact{},
	&inspectionmodel.InspectionNotification{},
}

// 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	// 执行 inspection 模块的自定义迁移（GORM 无法表达的部分索引）
	for _, sql := range inspectionmodel.InspectionMigrationSQL() {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}

	return nil
}

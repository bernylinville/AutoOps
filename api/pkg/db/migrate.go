// pkg/db/migrate.go
package db

import (
	cmdbmodel "dodevops-api/api/cmdb/model"
	ccmodel "dodevops-api/api/configcenter/model"
	monitormodel "dodevops-api/api/monitor/model"
	taskmodel "dodevops-api/api/task/model"
	k8smodel "dodevops-api/api/k8s/model"
	appmodel "dodevops-api/api/app/model"
	systemmodel "dodevops-api/api/system/model"
	toolmodel "dodevops-api/api/tool/model"
	n9emodel "dodevops-api/api/n9e/model"

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
}

// 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(models...)
}

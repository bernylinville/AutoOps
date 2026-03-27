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
	&cmdbmodel.CmdbGroup{},
	&ccmodel.EcsAuth{},
	&ccmodel.KeyManage{},
	&ccmodel.SyncSchedule{},
	&cmdbmodel.CmdbHost{},
	&cmdbmodel.CmdbSQLRecord{},
	&cmdbmodel.CmdbSQL{},
	&ccmodel.AccountAuth{},
	&taskmodel.TaskTemplate{},
	&taskmodel.Task{},
	&taskmodel.TaskWork{},
	&taskmodel.TaskAnsible{},
	&taskmodel.TaskAnsibleWork{},
	&monitormodel.Agent{},
	&k8smodel.KubeCluster{},
	&appmodel.Application{},
	&appmodel.JenkinsEnv{},
	&appmodel.QuickDeployment{},
	&appmodel.QuickDeploymentTask{},
	&systemmodel.SysOperationLog{},
	&systemmodel.SysAuditLog{},
	&toolmodel.Tool{},
	&toolmodel.ServiceDeploy{},
	// M7: N9E 模型
	&n9emodel.N9EConfig{},
	&n9emodel.N9EBusiGroup{},
	&n9emodel.N9EDataSource{},
	&n9emodel.N9ESyncLog{},
	// M8: 告警通知模型
	&n9emodel.AlertRule{},
	&n9emodel.AlertEvent{},
	&n9emodel.NotifyChannel{},
}

// 自动迁移所有模型
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(models...)
}

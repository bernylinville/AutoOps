package k8s

import (
	"dodevops-api/api/k8s/controller"
	"dodevops-api/common"
	"dodevops-api/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterK8sRoutes(router *gin.RouterGroup) {
	kubeClusterCtrl := controller.NewKubeClusterController(common.GetDB())
	k8sNodesCtrl := controller.NewK8sNodesController(common.GetDB())
	k8sNamespaceCtrl := controller.NewK8sNamespaceControllerWithCache(common.GetDB(), common.GetRedisClient())
	k8sWorkloadCtrl := controller.NewK8sWorkloadController(common.GetDB())
	k8sEventsCtrl := controller.NewK8sEventsController(common.GetDB())
	k8sTerminalCtrl := controller.NewK8sTerminalController(common.GetDB())
	k8sServiceCtrl := controller.NewK8sServiceController(common.GetDB())
	k8sIngressCtrl := controller.NewK8sIngressController(common.GetDB())
	k8sStorageCtrl := controller.NewK8sStorageController(common.GetDB())
	k8sConfigCtrl := controller.NewK8sConfigController(common.GetDB())
	
	// K8s集群管理路由 — H2-P0-3: 补 RBAC
	router.POST("/k8s/cluster", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:cluster:add"), kubeClusterCtrl.CreateCluster)
	router.GET("/k8s/cluster/:id", middleware.AuthMiddleware(), kubeClusterCtrl.GetCluster)
	router.GET("/k8s/cluster", middleware.AuthMiddleware(), kubeClusterCtrl.GetClusterList)
	router.PUT("/k8s/cluster/:id", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:cluster:edit"), kubeClusterCtrl.UpdateCluster)
	router.DELETE("/k8s/cluster/:id", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:cluster:delete"), kubeClusterCtrl.DeleteCluster)
	router.GET("/k8s/cluster/:id/status", middleware.AuthMiddleware(), kubeClusterCtrl.GetClusterStatus)
	router.GET("/k8s/cluster/:id/detail", middleware.AuthMiddleware(), kubeClusterCtrl.GetClusterDetail)
	router.POST("/k8s/cluster/:id/sync", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:cluster:sync"), kubeClusterCtrl.SyncCluster)

	// K8s节点管理路由
	router.GET("/k8s/cluster/:id/nodes", middleware.AuthMiddleware(), k8sNodesCtrl.GetNodes)        // 获取集群节点列表
	router.GET("/k8s/cluster/:id/nodes/:nodeName", middleware.AuthMiddleware(), k8sNodesCtrl.GetNodeDetail) // 获取节点详细信息
	router.GET("/k8s/cluster/:id/nodes/:nodeName/enhanced", middleware.AuthMiddleware(), k8sNodesCtrl.GetNodeDetailEnhanced) // 获取增强的节点详细信息
	router.POST("/k8s/cluster/:id/nodes/:nodeName/taints", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:edit"), k8sNodesCtrl.AddTaint)
	router.DELETE("/k8s/cluster/:id/nodes/:nodeName/taints", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:edit"), k8sNodesCtrl.RemoveTaint)
	router.POST("/k8s/cluster/:id/nodes/:nodeName/labels", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:edit"), k8sNodesCtrl.AddLabel)
	router.DELETE("/k8s/cluster/:id/nodes/:nodeName/labels", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:edit"), k8sNodesCtrl.RemoveLabel)
	router.POST("/k8s/cluster/:id/nodes/:nodeName/cordon", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:cordon"), k8sNodesCtrl.CordonNode)
	router.POST("/k8s/cluster/:id/nodes/:nodeName/drain", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:node:drain"), k8sNodesCtrl.DrainNode)
	router.GET("/k8s/cluster/:id/nodes/:nodeName/resources", middleware.AuthMiddleware(), k8sNodesCtrl.GetNodeResourceAllocation)

	// K8s命名空间管理路由
	router.GET("/k8s/cluster/:id/namespaces", middleware.AuthMiddleware(), k8sNamespaceCtrl.GetNamespaces)              // 获取指定集群的命名空间列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName", middleware.AuthMiddleware(), k8sNamespaceCtrl.GetNamespace) // 获取命名空间详情
	router.POST("/k8s/cluster/:id/namespaces", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:namespace:add"), k8sNamespaceCtrl.CreateNamespace)
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:namespace:edit"), k8sNamespaceCtrl.UpdateNamespace)
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:namespace:delete"), k8sNamespaceCtrl.DeleteNamespace)
	
	// ResourceQuota管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/resourcequotas", middleware.AuthMiddleware(), k8sNamespaceCtrl.GetResourceQuotas)    // 获取ResourceQuota列表
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/resourcequotas", middleware.AuthMiddleware(), k8sNamespaceCtrl.CreateResourceQuota) // 创建ResourceQuota
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/resourcequotas/:quotaName", middleware.AuthMiddleware(), k8sNamespaceCtrl.UpdateResourceQuota) // 更新ResourceQuota
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/resourcequotas/:quotaName", middleware.AuthMiddleware(), k8sNamespaceCtrl.DeleteResourceQuota) // 删除ResourceQuota
	
	// LimitRange管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/limitranges", middleware.AuthMiddleware(), k8sNamespaceCtrl.GetLimitRanges)    // 获取LimitRange列表
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/limitranges", middleware.AuthMiddleware(), k8sNamespaceCtrl.CreateLimitRange) // 创建LimitRange
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/limitranges/:limitRangeName", middleware.AuthMiddleware(), k8sNamespaceCtrl.UpdateLimitRange) // 更新LimitRange
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/limitranges/:limitRangeName", middleware.AuthMiddleware(), k8sNamespaceCtrl.DeleteLimitRange) // 删除LimitRange

	// ===================== K8s工作负载管理路由 =====================
	
	// 工作负载通用路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/workloads", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetWorkloads) // 获取工作负载列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/workloads/:type/:workloadName", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetWorkloadDetail) // 获取工作负载详情
	
	// Deployment管理路由
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:add"), k8sWorkloadCtrl.CreateDeployment)
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:edit"), k8sWorkloadCtrl.UpdateDeployment)
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:delete"), k8sWorkloadCtrl.DeleteDeployment)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/scale", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:scale"), k8sWorkloadCtrl.ScaleDeployment)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/restart", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:restart"), k8sWorkloadCtrl.RestartDeployment)

	// Deployment版本回滚管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/history", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetDeploymentHistory)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/revisions/:revision", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetDeploymentRevision)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/rollback", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:rollback"), k8sWorkloadCtrl.RollbackDeployment)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/pause", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:edit"), k8sWorkloadCtrl.PauseDeployment)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/resume", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:deployment:edit"), k8sWorkloadCtrl.ResumeDeployment)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/deployments/:deploymentName/rollout-status", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetDeploymentRolloutStatus) // 获取Deployment滚动发布状态

	// 工作负载Pod管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/workloads/:type/:workloadName/pods", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetWorkloadPods) // 获取工作负载下的Pod列表
	
	// Pod管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPods) // 获取Pod列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPodDetail) // 获取Pod详情
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:pod:delete"), k8sWorkloadCtrl.DeletePod)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/logs", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPodLogs)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/events", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPodEvents)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/yaml", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPodYaml)
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/yaml", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:pod:edit"), k8sWorkloadCtrl.UpdatePodYaml)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/pods/yaml", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:pod:add"), k8sWorkloadCtrl.CreatePodFromYAML)
	
	// K8s事件管理路由
	router.GET("/k8s/cluster/:id/events", middleware.AuthMiddleware(), k8sEventsCtrl.GetClusterEvents) // 获取集群事件列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/events", middleware.AuthMiddleware(), k8sEventsCtrl.GetEvents) // 获取命名空间事件列表
	
	// K8s容器终端路由 — H2-P0-3: exec 必须加 RBAC
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/terminal", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:pod:exec"), k8sTerminalCtrl.ConnectPodTerminal)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/containers", middleware.AuthMiddleware(), k8sTerminalCtrl.GetPodContainers)
	
	// ===================== K8s监控API路由 =====================
	
	// Pod监控路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pods/:podName/metrics", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetPodMetrics) // 获取Pod监控指标
	
	// 节点监控路由
	router.GET("/k8s/cluster/:id/nodes/:nodeName/metrics", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetNodeMetrics) // 获取节点监控指标
	
	// 命名空间监控路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/metrics", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetNamespaceMetrics) // 获取命名空间监控指标
	
	// ===================== YAML管理路由 =====================

	// YAML校验路由
	router.POST("/k8s/cluster/:id/yaml/validate", middleware.AuthMiddleware(), k8sWorkloadCtrl.ValidateYAML) // 校验YAML格式

	// 通用工作负载YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/workload-yaml/:workloadType/:workloadName", middleware.AuthMiddleware(), k8sWorkloadCtrl.GetWorkloadYaml) // 获取工作负载YAML
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/workload-yaml", middleware.AuthMiddleware(), k8sWorkloadCtrl.UpdateWorkloadYaml) // 更新工作负载YAML

	// ===================== K8s Service管理路由 =====================

	// Service基础CRUD路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/services", middleware.AuthMiddleware(), k8sServiceCtrl.GetServices)                  // 获取Service列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName", middleware.AuthMiddleware(), k8sServiceCtrl.GetServiceDetail) // 获取Service详情
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/services", middleware.AuthMiddleware(), k8sServiceCtrl.CreateService)               // 创建Service
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName", middleware.AuthMiddleware(), k8sServiceCtrl.UpdateService)   // 更新Service
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName", middleware.AuthMiddleware(), k8sServiceCtrl.DeleteService) // 删除Service

	// Service YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName/yaml", middleware.AuthMiddleware(), k8sServiceCtrl.GetServiceYaml)    // 获取Service YAML
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName/yaml", middleware.AuthMiddleware(), k8sServiceCtrl.UpdateServiceYaml) // 更新Service YAML

	// Service事件路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/services/:serviceName/events", middleware.AuthMiddleware(), k8sServiceCtrl.GetServiceEvents) // 获取Service事件

	// ===================== K8s Ingress管理路由 =====================

	// Ingress基础CRUD路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/ingresses", middleware.AuthMiddleware(), k8sIngressCtrl.GetIngresses)                  // 获取Ingress列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName", middleware.AuthMiddleware(), k8sIngressCtrl.GetIngressDetail) // 获取Ingress详情
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/ingresses", middleware.AuthMiddleware(), k8sIngressCtrl.CreateIngress)               // 创建Ingress
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName", middleware.AuthMiddleware(), k8sIngressCtrl.UpdateIngress)   // 更新Ingress
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName", middleware.AuthMiddleware(), k8sIngressCtrl.DeleteIngress) // 删除Ingress

	// Ingress YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName/yaml", middleware.AuthMiddleware(), k8sIngressCtrl.GetIngressYaml)    // 获取Ingress YAML
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName/yaml", middleware.AuthMiddleware(), k8sIngressCtrl.UpdateIngressYaml) // 更新Ingress YAML

	// Ingress事件和监控路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName/events", middleware.AuthMiddleware(), k8sIngressCtrl.GetIngressEvents)       // 获取Ingress事件
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/ingresses/:ingressName/monitoring", middleware.AuthMiddleware(), k8sIngressCtrl.GetIngressMonitoring) // 获取Ingress监控

	// ===================== K8s存储管理路由 =====================

	// PVC基础CRUD路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pvcs", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVCs)                // 获取PVC列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pvcs/:pvcName", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVCDetail) // 获取PVC详情
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/pvcs", middleware.AuthMiddleware(), k8sStorageCtrl.CreatePVC)             // 创建PVC
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/pvcs/:pvcName", middleware.AuthMiddleware(), k8sStorageCtrl.UpdatePVC)    // 更新PVC
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/pvcs/:pvcName", middleware.AuthMiddleware(), k8sStorageCtrl.DeletePVC) // 删除PVC

	// PVC YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/pvcs/:pvcName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVCYaml)    // 获取PVC YAML
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/pvcs/:pvcName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.UpdatePVCYaml) // 更新PVC YAML

	// PV基础CRUD路由
	router.GET("/k8s/cluster/:id/pvs", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVs)              // 获取PV列表
	router.GET("/k8s/cluster/:id/pvs/:pvName", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVDetail) // 获取PV详情
	router.POST("/k8s/cluster/:id/pvs", middleware.AuthMiddleware(), k8sStorageCtrl.CreatePV)           // 创建PV
	router.PUT("/k8s/cluster/:id/pvs/:pvName", middleware.AuthMiddleware(), k8sStorageCtrl.UpdatePV)    // 更新PV
	router.DELETE("/k8s/cluster/:id/pvs/:pvName", middleware.AuthMiddleware(), k8sStorageCtrl.DeletePV) // 删除PV

	// PV YAML管理路由
	router.GET("/k8s/cluster/:id/pvs/:pvName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.GetPVYaml)    // 获取PV YAML
	router.PUT("/k8s/cluster/:id/pvs/:pvName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.UpdatePVYaml) // 更新PV YAML

	// StorageClass基础CRUD路由
	router.GET("/k8s/cluster/:id/storageclasses", middleware.AuthMiddleware(), k8sStorageCtrl.GetStorageClasses)                        // 获取StorageClass列表
	router.GET("/k8s/cluster/:id/storageclasses/:storageClassName", middleware.AuthMiddleware(), k8sStorageCtrl.GetStorageClassDetail) // 获取StorageClass详情
	router.POST("/k8s/cluster/:id/storageclasses", middleware.AuthMiddleware(), k8sStorageCtrl.CreateStorageClass)                     // 创建StorageClass
	router.PUT("/k8s/cluster/:id/storageclasses/:storageClassName", middleware.AuthMiddleware(), k8sStorageCtrl.UpdateStorageClass)    // 更新StorageClass
	router.DELETE("/k8s/cluster/:id/storageclasses/:storageClassName", middleware.AuthMiddleware(), k8sStorageCtrl.DeleteStorageClass) // 删除StorageClass

	// StorageClass YAML管理路由
	router.GET("/k8s/cluster/:id/storageclasses/:storageClassName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.GetStorageClassYaml)    // 获取StorageClass YAML
	router.PUT("/k8s/cluster/:id/storageclasses/:storageClassName/yaml", middleware.AuthMiddleware(), k8sStorageCtrl.UpdateStorageClassYaml) // 更新StorageClass YAML

	// ===================== K8s配置管理路由 =====================

	// ConfigMap基础CRUD路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/configmaps", middleware.AuthMiddleware(), k8sConfigCtrl.GetConfigMaps)                      // 获取ConfigMap列表
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/configmaps/:configMapName", middleware.AuthMiddleware(), k8sConfigCtrl.GetConfigMapDetail) // 获取ConfigMap详情
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/configmaps", middleware.AuthMiddleware(), k8sConfigCtrl.CreateConfigMap)                   // 创建ConfigMap
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/configmaps/:configMapName", middleware.AuthMiddleware(), k8sConfigCtrl.UpdateConfigMap)    // 更新ConfigMap
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/configmaps/:configMapName", middleware.AuthMiddleware(), k8sConfigCtrl.DeleteConfigMap) // 删除ConfigMap

	// ConfigMap YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/configmaps/:configMapName/yaml", middleware.AuthMiddleware(), k8sConfigCtrl.GetConfigMapYaml)    // 获取ConfigMap YAML
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/configmaps/:configMapName/yaml", middleware.AuthMiddleware(), k8sConfigCtrl.UpdateConfigMapYaml) // 更新ConfigMap YAML

	// Secret基础CRUD路由 — H2-P0-3: Secret 必须加 RBAC
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/secrets", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:view"), k8sConfigCtrl.GetSecrets)
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/secrets/:secretName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:view"), k8sConfigCtrl.GetSecretDetail)
	router.POST("/k8s/cluster/:id/namespaces/:namespaceName/secrets", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:add"), k8sConfigCtrl.CreateSecret)
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/secrets/:secretName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:edit"), k8sConfigCtrl.UpdateSecret)
	router.DELETE("/k8s/cluster/:id/namespaces/:namespaceName/secrets/:secretName", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:delete"), k8sConfigCtrl.DeleteSecret)

	// Secret YAML管理路由
	router.GET("/k8s/cluster/:id/namespaces/:namespaceName/secrets/:secretName/yaml", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:view"), k8sConfigCtrl.GetSecretYaml)
	router.PUT("/k8s/cluster/:id/namespaces/:namespaceName/secrets/:secretName/yaml", middleware.AuthMiddleware(), middleware.RbacMiddleware("k8s:secret:edit"), k8sConfigCtrl.UpdateSecretYaml)
}
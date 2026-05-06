# Deploy Control Plane

## 1. 当前部署模型

AutoOps 发布中心支持两条路径：

1. **Direct**：直接对目标集群执行受限资源操作
2. **GitOps**：写入 `pukka-gitops` 工作树，再由 ArgoCD 同步

## 2. 当前仓库与 GitOps 仓库分工

### AutoOps 仓库

- 提供平台本体 Helm chart：`charts/autoops/`
- 提供 API / 前端 / 发布逻辑代码

### pukka-gitops 仓库

- 提供平台本体 ArgoCD Application：`argocd-apps/templates/autoops.yaml`
- 提供平台本体值覆盖：`apps/autoops/values.yaml`
- 提供 AutoOps 受管发布清单目录：`apps/autoops-managed-releases/`

## 3. 工作树约束

GitOps 模式下，AutoOps 进程需要可写的本地工作树：

```yaml
integrations:
  gitops:
    local_checkout_path: /workspace/pukka-gitops
```

开发环境：

- Docker Compose 直接挂载 sibling repo `../pukka-gitops`

生产环境：

- Helm chart 提供 `gitopsWorkingTree` PVC
- 可选 bootstrap clone
- 若要执行 `git push`，必须补充 Git 凭据

## 4. 监控与告警

- 生产环境依赖外部 N9E + VictoriaMetrics 提供指标
- 生产环境 FlashDuty 负责告警通知
- 开发环境 Docker Compose 含 Prometheus + Pushgateway 用于本地测试

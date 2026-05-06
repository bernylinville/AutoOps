# Architecture

## 1. 技术基线

| 层 | 技术 |
|---|---|
| Backend | Go 1.25 / Gin / GORM |
| Frontend | Vue 3 / Element Plus |
| DB | PostgreSQL 17 |
| Cache | Valkey 9 |
| Dev runtime | Docker Compose |
| Prod runtime | Kubernetes + ArgoCD |
| Monitoring | 外部 N9E + VictoriaMetrics |
| Alerting | 既有 FlashDuty |

## 2. 请求链路

```text
HTTP
  → Recovery
  → CORS
  → Logger
  ├─ 公开接口: /captcha /login /healthz /monitor/agent/heartbeat
  └─ JWT
      → AuthMiddleware
      → LogMiddleware
      → AuditMiddleware
      → RBAC per route
      → Controller
      → Service
      → DAO
      → PostgreSQL
```

## 3. 当前环境拓扑

### 3.1 开发环境

```text
Browser
  ├─ http://localhost:18088 → devops-web
  └─ http://localhost:18000 → devops-api

Docker Compose
  ├─ devops-web
  ├─ devops-api
  ├─ postgres
  └─ valkey

External / local integrations
  ├─ minikube (测试集群)
  ├─ ../pukka-gitops (本地 GitOps 工作树)
  └─ shared N9E / VictoriaMetrics / FlashDuty
```

### 3.2 生产环境

```text
ArgoCD Application (pukka-gitops)
  ├─ source: AutoOps repo / charts/autoops
  └─ values: pukka-gitops / apps/autoops/values.yaml

Kubernetes namespace: autoops
  ├─ autoops-api
  ├─ autoops-web
  ├─ autoops-postgres
  ├─ autoops-valkey
  ├─ upload PVC
  └─ gitops working tree PVC

Gateway / exposure
  └─ Envoy Gateway + HTTPRoute → 10.0.17.206
```

## 4. 监控架构

- 主机基础指标：AutoOps → N9E / VictoriaMetrics
- 告警：AutoOps / N9E → FlashDuty
- 进程 / 端口接口：兼容保留，外部监控模式下可返回空结果
- 开发环境 Docker Compose 含 Prometheus + Pushgateway（本地测试）
- 生产环境不部署内置监控组件，全部依赖外部 N9E

## 5. GitOps 架构

AutoOps 平台有两个 GitOps 相关面：

1. **平台本体部署**：通过 `pukka-gitops` 的 ArgoCD Application 部署
2. **AutoOps 管理的发布清单**：继续写入 `autoops-managed-releases`

两者必须分离，不要把平台本体 chart 和受管发布清单混在一个目录中。

## 6. 配置约束

- 配置加载：`gopkg.in/yaml.v2`
- Docker / K8s 运行时都通过 `config.template.yaml` 渲染最终配置
- `integrations.gitops.local_checkout_path` 必须指向可写工作树
- 生产若需要 `git push`，需提供 Git 凭据（SSH key 或 token）

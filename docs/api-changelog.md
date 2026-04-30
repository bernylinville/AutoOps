# API Changelog

> 理解 API 历史变更、排查接口兼容性问题时阅读。

## 概览

AutoOps CMDB 2.0 升级分 8 个 Phase（2026-03 ~ 2026-04），全部完成。详细进度记录见 `progress.md`（权威来源）。

## Phase 索引

| Phase | 主题 | 关键 API 变更 |
|-------|------|-------------|
| 0 | MySQL→PostgreSQL, Redis→Valkey | 基础设施迁移，API 接口无变化 |
| 1 | 动态 CI 模型 | 15 个 REST API：CIType/Attribute/Instance/Relation CRUD |
| 2 | 项目/应用维度 | Project CRUD + 资产统计 API |
| 3 | CI 拓扑 | 拓扑查询 + ECharts 可视化 |
| 4 | 资产生命周期 | 状态变更 + 变更日志 + 过期告警 |
| 5 | 网络设备 | SNMP 采集 + TCP 检测 |
| 6 | N9E 集成 | 同步/告警/数据源 9 个 API |
| 7 | SNMP 巡检 + 主机生命周期 | 10 种主机状态 + 批量操作 |
| 8 | 数据库变更日志 + UI 完善 | cmdbSQL changelog + SNMP 团体名 |

## 待合并来源

以下文档包含详细 API 说明，后续可按 Phase 合并到本文档：

- `progress.md` — 各 Phase API 端点清单
- `api/docs/task_execution_example.md` — 任务执行 API
- `api/docs/task_pause_feature_migration.md` — 任务暂停迁移
- `api/docs/toggle_status_fix.md` — 状态切换修复
- `api/docs/cron_format_fix.md` — Cron 格式标准化
- `api/docs/sync_schedule_example.md` — 同步调度
- `api/docs/task_list_with_details_example.md` — 任务列表增强
- `api/docs/scheduler_crash_fix.md` — 调度器崩溃修复
- `api/scripts/README_FIX.md` — 定时任务修复

## 2026-04 Deploy 控制平面

### 新增后端路由

#### 页面 / JWT 路由

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/deploy/cluster-targets` | 获取部署目标列表 |
| `GET` | `/api/v1/deploy/cluster-targets/:id` | 获取单个部署目标 |
| `POST` | `/api/v1/deploy/cluster-targets` | 创建部署目标 |
| `PUT` | `/api/v1/deploy/cluster-targets/:id` | 更新部署目标 |
| `POST` | `/api/v1/deploy/cluster-targets/:id/validate-direct-credential` | 校验 Direct 受限 kubeconfig |
| `POST` | `/api/v1/deploy/cluster-targets/:id/validate-gitops-repo` | 校验 GitOps 本地仓库分支 |
| `GET` | `/api/v1/deploy/gitops/validate-working-tree` | 校验 GitOps 本地工作树脚手架 |
| `POST` | `/api/v1/deploy/requests` | 创建部署申请 |
| `GET` | `/api/v1/deploy/requests` | 获取部署申请列表 |
| `GET` | `/api/v1/deploy/requests/:id` | 获取部署申请详情 |
| `GET` | `/api/v1/deploy/requests/:id/executions` | 获取执行记录 |
| `GET` | `/api/v1/deploy/requests/:id/notifications` | 获取通知记录 |
| `POST` | `/api/v1/deploy/requests/:id/dispatch-approval` | 重发钉钉 OA 审批实例 |
| `POST` | `/api/v1/deploy/requests/:id/sync-approval` | 手动同步审批状态 |
| `POST` | `/api/v1/deploy/requests/:id/approve` | 站内审批兜底：通过 |
| `POST` | `/api/v1/deploy/requests/:id/reject` | 站内审批兜底：拒绝 |
| `POST` | `/api/v1/deploy/requests/:id/execute` | 执行部署申请 |
| `POST` | `/api/v1/deploy/requests/:id/rollback` | 下线/回滚部署申请 |
| `POST` | `/api/v1/deploy/requests/:id/cleanup` | 清理 Direct 资源 |
| `GET` | `/api/v1/pipeline-runs/:id` | 获取 PipelineRun 详情 |
| `GET` | `/api/v1/pipeline-runs/by-request/:requestId` | 按部署申请获取 PipelineRun |

#### Agent / Hermes 机器入口

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/integrations/agent/deploy-requests` | 通过外部身份创建部署申请 |
| `GET` | `/api/v1/integrations/agent/deploy-requests/:requestNo` | 按申请单号查询部署申请 |
| `GET` | `/api/v1/integrations/agent/deploy-requests/:requestNo/status` | 按申请单号查询状态 |
| `POST` | `/api/v1/integrations/agent/deploy-requests/:requestNo/dispatch-approval` | 重发审批实例 |
| `POST` | `/api/v1/integrations/agent/deploy-requests/:requestNo/sync-approval` | 手动同步审批状态 |
| `POST` | `/api/v1/integrations/agent/deploy-requests/:requestNo/execute` | 按申请单号执行部署 |
| `POST` | `/api/v1/integrations/agent/deploy-requests-by-id/:id/execute` | 按申请 ID 执行部署 |

页面路由位于 JWT + Audit 保护下，并按路由配置 `RbacMiddleware`；Agent 入口位于 `AgentAuthMiddleware` + Agent Audit 保护下。

### 新增领域模型

| 模型 | 用途 |
|------|------|
| `ClusterTarget` | 部署目标与默认审批人 / 凭据引用 |
| `DeployRequest` | 部署申请主表 |
| `ApprovalRecord` | 审批动作留痕 |
| `ExecutionRecord` | 执行记录 |
| `ResourceOwner` | 资源唯一 owner 注册表 |

### 新增配置项

#### `config.yaml`

```yaml
dingtalkApproval:
  client_id: ""
  client_secret: ""
  process_code: ""
  poll_interval_seconds: 30
  field_mappings:
    request_no: ""
    release_name: ""
    cluster_target: ""
    deploy_mode: ""
    resource_type: ""
    image: ""
    namespace: ""
    ttl_hours: ""
    reason: ""

integrations:
  agent:
    bearer_token: "..."
  gitops:
    local_checkout_path: "/path/to/pukka-gitops"
```

### 行为变化

- 新增独立 `deploy` 领域模块，不再把申请/审批/执行逻辑混入 `k8s` 或 `app`
- `direct` 模式要求：
  - 受限 kubeconfig
  - `ao-direct-*` namespace
  - 强制 owner / deploy-mode / request-id / ttl 标签注解
- `gitops` 模式要求：
  - `autoops-managed-releases` Helm 脚手架存在
  - 本地仓库可写、可 commit、可 push

### 当前已验证

- Direct smoke：成功
- GitOps smoke：成功
- `go test ./...`：通过
- 前端 `npm run lint`：通过（仅存在既有 warning）

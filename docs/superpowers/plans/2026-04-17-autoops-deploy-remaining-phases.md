# AutoOps Deploy 控制平面 — 剩余阶段（P0/P1/P1/P2）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 AutoOps Deploy 控制平面从「主体已落地、但端到端未跑通」推到「真实钉钉 OA 审批 → 群聊结果回写 → Hermes 自然语言触发 → GitOps 回滚/下线全闭环」的 MVP 状态。

**Architecture:** 在现有 `api/api/deploy/` 模块基础上仅增量扩展 ——（1）打通钉钉真实 userId 映射；（2）新增 `notifier` 子服务消费 `chatContextJson` 回群；（3）在 `~/.hermes/hermes-agent/skills/devops/` 新增 `deploy-via-autoops/` 骨架；（4）在 `gitopsWriter` 上扩展 delete/rollback；（5）补齐 AuditMiddleware 与 pukka-gitops 契约 README。

**Tech Stack:** Go 1.25 + Gin + GORM / PostgreSQL 17 / Vue 3.5 + Element Plus / Python 3.11+ (Hermes) / ArgoCD App-of-Apps / DingTalk OA + Robot Webhook.

**Generated:** 2026-04-17 via `/all-plan` (Readiness 87/100, inspiration skipped, reviewer pending).

---

## 1. 需求澄清与关键假设

### 1.1 已明确的前提

| # | 前提 | 证据来源 |
|---|------|---------|
| P1 | AutoOps `deploy` 模块已落地：5 models、21 controller methods、双 router（RBAC + AgentAuth）、两个 scheduler | `api/api/deploy/model/deploy.go` L74–205；`controller/deploy.go` L24–221；`router/deploy/deploy.go` L14–43；`scheduler/deployApprovalSyncScheduler.go`；`scheduler/deployTTLReaper.go` |
| P2 | Direct mode 与 GitOps mode 主干链路都已跑通 | handoff 二-2/二-3；本地 smoke 验证通过 |
| P3 | 审批通过 → 自动执行已落地（`AutoExecuteApprovedDeployRequest`） | `service/deploy.go` L528；`scheduler/deployApprovalSyncScheduler.go` L131 |
| P4 | pukka-gitops 是 ArgoCD App-of-Apps，AutoOps 写 `apps/autoops-managed-releases/releases/*.yaml` | `pukka-gitops/bootstrap/root-app.yaml`；`apps/autoops-managed-releases/templates/releases.yaml` |
| P5 | 双模式策略：**direct 只用于 `ao-direct-*` 短生命周期**；GitOps 为正式业务主战场 | 用户选项确认 |
| P6 | 同一 `(cluster, namespace, kind, name)` 只允许一个 owner，通过 `deploy_resource_owner` 表保证 | `model/deploy.go` L188–205；`service/deploy.go` `reserveResourceOwners` L874 |
| P7 | `chatContextJson` 字段已建库但无消费者 | grep 确认 deploy.go:320/361 仅写入 |
| P8 | Hermes 在 `~/.hermes/hermes-agent`，Python 3.11+，skill 目录 `skills/devops/` | 第三方调研确认 |

### 1.2 仍需确认的点（待验证项）

| ID | 待确认 | 影响面 | 默认假设 | 何时验证 |
|----|--------|--------|---------|---------|
| Q1 | 钉钉应用是否开通「通讯录读取」权限？ | 影响能否自动解析 `sys_admin → userId`。handoff 2026-04-17 核验结果：**未开通** `qyapi_get_department_member` | MVP 内先**手工补字段**（管理员手动填 `sys_admin.dingtalk_user_id`），不依赖自动解析 | Phase A Task A-1 |
| Q2 | 群聊 webhook 需要按 cluster 维度配置还是全局一个？ | 多环境时影响路由 | 默认**全局 + 可按 cluster_target 覆盖**；表 schema 预留，不上 UI | Phase B Task B-1 |
| Q3 | GitOps "下线" 是删 release 文件还是改 CR `active=false`？ | 影响 autoops-managed-releases 模板是否需要改 | 默认**删除 release 文件 + git commit**（当前 Helm 模板不支持 inactive 语义）| Phase D Task D-1 |
| Q4 | Hermes 调 AutoOps 时是否需要二次授权（Approve 链接放回群聊）？ | 影响 bot 消息是否用 ActionCard | MVP 仅发文本结果，不做交互卡片 | Phase B Task B-4 |
| Q5 | 群聊机器人发送失败是否重试？重试多少次？ | 影响 deploy_notification 表字段 | MVP 记失败 1 次即止，写日志不重试 | Phase B Task B-2 |
| Q6 | 审计中间件与 RBAC 的执行顺序？是否要对 agent 路由也启用审计？ | 影响中间件链 | 页面路由：RBAC → Audit；Agent 路由：AgentAuth → Audit（记 source='agent'） | Phase X Task X-1 |

### 1.3 容易返工的点（红线）

| ID | 雷区 | 避雷策略 |
|----|------|---------|
| R1 | 在 `chat_context_json` 里塞 DingTalk 专有字段（如 conversationId/messageId），后续接 Lark/企微时被迫改 schema | 字段命名用 **provider-agnostic** 前缀（`provider`, `chat_id`, `at_mobiles`, `sender_external_id`），provider 枚举字符串化 |
| R2 | 把群聊机器人 webhook 硬编码进 config.yaml 全局配置 | 新增 `deploy_bot_endpoint` 表（或在 `ClusterTarget` 上扩字段），webhook_url 与 secret 落库，支持后续按 cluster 覆盖 |
| R3 | Hermes skill 直接调 K8s / 直接改 git | 严禁。skill 仅调 `/api/v1/integrations/agent/deploy-requests`。代码评审强制该边界 |
| R4 | GitOps 回滚实现成"强制覆盖 HEAD"；破坏其他 release | 回滚 = 新增一条 reverse commit（revert 或 delete 对应 release 文件），不重写历史 |
| R5 | 为 P2 GitOps 下线临时绕过 `resource_owner` 表 | 下线必须走 `DeactivateResourceOwnersByRequestID`；绕过就等于允许双主 |
| R6 | 在 AutoOps 里复制一份 DingTalk robot SDK | 新增 `api/pkg/dingtalkbot/`，**与 n9e 的 notifier 共享 payload 构造器**（`n9e/service/notifier.go` L96 的模式提炼到 pkg） |
| R7 | Agent bearer token 塞进 URL query 或 log | 只走 `Authorization: Bearer`；日志脱敏必须在 `middleware/logMiddleware.go` 里对该 header 做 redact（验证现状） |

---

## 2. 现状调研摘要

### 2.1 AutoOps 当前相关模块与能力（源码核实）

| 层 | 文件 | 核实结果 |
|----|------|---------|
| Model | `api/api/deploy/model/deploy.go` | ClusterTarget / DeployRequest / ApprovalRecord / ExecutionRecord / ResourceOwner 齐备；`DeployRequest.ChatContextJSON`（L132）已存在但无读取者 |
| DAO | `api/api/deploy/dao/deploy.go` | 18 个方法，包含 `ListPendingApprovalSyncRequests`（L140）、`ListExpiredDirectRequests`（L179）、`DeactivateResourceOwnersByRequestID`（L172）——回滚可复用 |
| Controller | `api/api/deploy/controller/deploy.go` | 21 方法，UI/Agent 共用 Service，仅入口不同 |
| Router | `api/router/deploy/deploy.go` | 页面：每条挂 `RbacMiddleware("deploy:*")`；Agent：`/integrations/agent` 组挂 `AgentAuthMiddleware`；**均未挂 AuditMiddleware**（缺口）|
| Service | 7 文件 | `deploy.go` 1228 行为核心编排；`tryDispatchApproval` L765 是审批发起钩子；`syncApprovalStatusInternal` L800 自动执行；`executeDeployRequestInternal` L537 真正执行 |
| Scheduler | `scheduler/deployApprovalSyncScheduler.go` / `deployTTLReaper.go` | 审批同步 30 s 一次；TTL 回收 60 s 一次；两者都会触发 service 层 |
| Middleware | `middleware/agentAuthMiddleware.go` | Bearer 校验读 `config.Config.Integrations.Agent.BearerToken`；未写则 403；header 缺失 401 |
| Migration | `api/pkg/db/migrate.go` L64–68 | 5 个 deploy 模型全部注册 ✅ |
| 钉钉 OA | `api/api/deploy/service/dingtalkApproval.go` | `IsConfigured` / `GetAccessToken` / `CreateProcessInstance`（POST `/v1.0/workflow/processInstances`）/ `GetProcessInstance`（GET） 都已实现 |
| 钉钉机器人 | `api/common/config/config.go` 有 `dingtalk.webhook_url`；`api/n9e/service/notifier.go` 有 `SendDingtalk`（L96） | **未抽象为通用 pkg，当前仅告警在用**。MVP 需提炼到 `api/pkg/dingtalkbot/` |
| 前端 | `web/src/views/K8s/K8sReleaseCenter.vue` / `web/src/api/deploy.js` | 两个 tab（申请、目标）+ 新建/执行/同步/审批对话框；**未暴露 chatContext 字段**（正确，UI 无需暴露）|

### 2.2 pukka-gitops 当前结构与约束

```
pukka-gitops/
├── bootstrap/root-app.yaml           # ArgoCD 根 App，指向 argocd-apps
├── argocd-apps/                      # App-of-Apps Helm chart
│   ├── Chart.yaml
│   └── templates/                    # 16 个子 Application (sync-wave 0..3)
│       ├── autoops-managed-releases.yaml
│       ├── example-app.yaml
│       └── ...
├── apps/
│   ├── example-app/                  # 示例 Helm chart
│   └── autoops-managed-releases/     # ★ AutoOps 写入目标
│       ├── Chart.yaml (v0.1.0)
│       ├── values.yaml
│       ├── templates/releases.yaml   # 用 Files.Glob "releases/*.yaml" 渲染 CR
│       └── releases/
│           └── smoke-gitops-235414.yaml   # 示例 Release CR
├── infra/                            # 基础设施组件 values 覆写
├── platform/                         # Gateway/HTTPRoute 原生 YAML
└── scripts/                          # mirror-images.sh 等
```

**约束**:
- 无 `.github/`, **无 CI**，所有校验靠本地 `helm template` / `kubectl apply --dry-run`
- 单集群（`destination.server: https://kubernetes.default.svc`）
- Release CR schema 仅在 `templates/releases.yaml` 中以隐式 Helm 模板约束存在，**无 README**
- AutoOps 写入的 namespace 约定：`ao-gitops-{releaseName}`（与 direct 的 `ao-direct-*` 区隔）

### 2.3 Hermes Agent 当前约束

| 项 | 值 |
|---|---|
| 路径 | `~/.hermes/hermes-agent/`（实测 `/home/kchou/Code/.hermes/hermes-agent`） |
| 语言 | Python 3.11+（`pyproject.toml` name=`hermes-agent` v0.9.0）|
| 技能入口 | `skills/<category>/<skill-name>/SKILL.md` |
| 示例 | `skills/devops/webhook-subscriptions/SKILL.md` |
| 配置 | `~/.hermes/cli-config.yaml`；`cli-config.yaml.example` 有 55 KB 现成模板 |
| 外部调用 | 自由（Python requests / httpx） |
| AutoOps 集成 | **无**。需新增 skill + 配置 |

### 2.4 可复用的部分

- `IDeployService` 现有方法：`CreateAgentDeployRequest`、`GetDeployRequestByRequestNo`、`SyncApprovalStatusByRequestNo` → Hermes 直接消费
- `executionDetailWith*JSON` 助手（`service/deploy.go` L1088–1181） → 结果通知 payload 里可提取
- `DeactivateResourceOwnersByRequestID`（DAO） → GitOps 下线路径复用
- `n9e/service/notifier.go` 的 markdown payload 构造逻辑 → 提取到 `pkg/dingtalkbot/`
- `middleware/agentAuthMiddleware.go` → 不改，Hermes 直接用

### 2.5 当前缺口（差距清单）

| # | 差距 | 阻塞 | 归属 |
|---|------|------|------|
| G1 | `sys_admin.dingtalk_user_id` 为占位值；OA 发起人/审批人映射错 | P0 | 数据 + Phase A |
| G2 | 钉钉机器人回群无实现 | P1 | Phase B |
| G3 | Hermes 侧无 `deploy-via-autoops` skill | P1 | Phase C |
| G4 | GitOps 无回滚/删除 release；失败二次回收缺 | P2 | Phase D |
| G5 | deploy 路由无 AuditMiddleware | 横切 | Phase X |
| G6 | pukka-gitops `autoops-managed-releases/` 无 README | 横切 | Phase X |
| G7 | DingTalk 机器人 SDK 未抽象 | 支撑 Phase B | Phase B 前置 |
| G8 | `chatContextJson` 未消费 | 支撑 Phase B | Phase B |
| G9 | Deploy routes swagger 注释可能过时（新增接口时需补） | 小 | 各 Phase |

---

## 3. 目标边界设计

### 3.1 AutoOps 职责

**承担**：
1. Deploy 域唯一的领域仓库 & 状态机 owner（`deploy_request`、`approval_record`、`execution_record`、`resource_owner`）
2. 钉钉 OA 审批实例发起/同步/自动执行
3. Kubernetes 操作的唯一入口：
   - Direct mode：直连 K8s API apply/delete
   - GitOps mode：写 `pukka-gitops` working tree → `git push`
4. 审计链路主干（AuditMiddleware 落库 + `execution_record.detail_json`）
5. 对外 API：页面 API（RBAC + Cookie JWT）+ Agent API（Bearer，给 Hermes）
6. 结果通知聚合器（给钉钉机器人、后续可扩到 Lark / 企微）
7. 资源 owner 登记表，防止双主

**不承担**：
- ❌ 不做 ArgoCD 的同步状态监听（ArgoCD 自治）
- ❌ 不直接订阅钉钉群聊消息（由 Hermes 代理）
- ❌ 不实现审批流的可视化编排（复用钉钉 OA 模板）
- ❌ 不存钉钉成员完整信息（仅 `dingtalk_user_id` 映射字段）

### 3.2 pukka-gitops 职责

**承担**：
1. 所有 GitOps 模式资源的声明式真相来源
2. ArgoCD 负责把 Git 状态同步到集群（reconcile / drift-detect / sync-wave）
3. `apps/autoops-managed-releases/` 目录作为 AutoOps 的"交付出口"
4. 基础设施组件（Jenkins / Harbor / Prometheus / Gateway）的长期管理

**不承担**：
- ❌ 不响应钉钉 / Hermes 的即时请求
- ❌ 不存审批记录（审批主数据在 AutoOps）
- ❌ 不管理 direct mode 资源（direct 命名空间 `ao-direct-*` 不写进仓库）

**新增契约（本计划新增）**：
- `apps/autoops-managed-releases/README.md` — Release CR schema 文档（Phase X）

### 3.3 Hermes 职责

**承担**：
1. 钉钉群聊消息接收（通过 Hermes 自身 webhook/gateway）
2. 自然语言意图解析：`部署 nginx 到 pukka 测试` → 结构化参数
3. 调用 AutoOps Agent API 创建 DeployRequest
4. 把 AutoOps 返回的 `requestNo` / 状态回写群聊（Hermes 内部发消息，不复用 AutoOps 的 bot）
5. 状态轮询（可选）：定时 GET `/api/v1/integrations/agent/deploy-requests/:requestNo` 反馈状态变化

**不承担**：
- ❌ 不直接调用 K8s
- ❌ 不直接写 pukka-gitops
- ❌ 不持有 kubeconfig / git 凭据
- ❌ 不做审批决策（只负责发起与播报）

### 3.4 钉钉群聊机器人

**两类机器人，各司其职**：

| 机器人 | 归属 | 输入方向 | 输出方向 | MVP 状态 |
|-------|------|---------|---------|---------|
| Hermes 入口 bot | Hermes Gateway | 群聊 @ 触发 → Hermes | 回发确认/状态到群 | Phase C |
| AutoOps 结果 bot | AutoOps | — | 执行结束后主动回发结果到群 | Phase B |

两者**可以是同一个 webhook token**（AutoOps 配置里把 webhook_url 与 Hermes 用的同一个填进去），但**程序上是独立发送方**。

### 3.5 审批系统位置

```
[群聊 / 页面] → [Hermes 解析] → AutoOps:CreateDeployRequest
                                      │
                                      ▼
                           tryDispatchApproval()
                                      │
                                      ▼
                           钉钉 OA (企业真实 userId)
                                      │
                      ┌───────────────┴───────────────┐
                      ▼                               ▼
             审批人 APP 操作              deployApprovalSyncScheduler
                      │                               │
                      └───────────────┬───────────────┘
                                      ▼
                           approval_status = approved
                                      │
                                      ▼
                           AutoExecuteApprovedDeployRequest
                                      │
                             ┌────────┴────────┐
                             ▼                 ▼
                         direct 执行        GitOps 写仓库 → git push
                             │                 │
                             └────────┬────────┘
                                      ▼
                           executeDeployRequestInternal → 回写 execution_record
                                      │
                                      ▼
                           ★ NEW: DeployNotifier → 钉钉群聊结果消息
```

审批系统是**必经关卡**，不允许任何路径绕过 `approval_status`。

---

## 4. 双模式部署策略

### 4.1 GitOps 发布模式

| 维度 | 内容 |
|------|------|
| 适用场景 | 正式业务、需要审计留痕、多人协同维护、需要 ArgoCD drift detection |
| 执行链路 | 审批通过 → `RenderGitOpsReleaseFile` → `WriteGitOpsReleaseToWorkingTree` → `CommitGitOpsWorkingTree` → `PushGitOpsBranch` → ArgoCD 拉取 → K8s 同步 |
| 审计链路 | `execution_record.detail_json` + `git_commit_sha` + Git commit 历史 + ArgoCD 同步历史 |
| 回滚链路（Phase D 新增）| 新增一次反向 commit（删除 release 文件 或 替换为上一版本 CR）→ ArgoCD 自动回收 |
| 优点 | 真相在 Git，审计完整；ArgoCD 自动纠偏；可 code review |
| 缺点 | 延迟（审批完 → git push → ArgoCD reconcile）；依赖 git/ArgoCD 健康 |
| Namespace 约定 | `ao-gitops-{releaseName}`（由 Release CR 渲染产生）|
| 资源标签 | `app.kubernetes.io/managed-by=argocd` + `autoops.io/deploy-mode=gitops` + `autoops.io/release-name=<name>` |
| 生命周期 | 由 GitOps 文件存在性决定；删除 release 文件 = 下线；无 TTL |
| 权限控制 | AutoOps 需要 git push 权限；不需要 K8s 权限（由 ArgoCD 持有）|

### 4.2 平台直连即时部署模式

| 维度 | 内容 |
|------|------|
| 适用场景 | 临时测试、短生命周期、POC、群聊即发即用、需要立即可见 |
| 执行链路 | 审批通过 → `RenderDirectManifest` → `NewDirectKubeClient` → `ApplyDirectResources` |
| 审计链路 | `execution_record.detail_json` + 资源上的 `autoops.io/request-id` 注解；无 git 审计 |
| 回收链路 | `deployTTLReaper` 每 60 s 扫 `execution_status=succeeded AND ttl_hours != null` → `CleanupDirectRequestByID` → 删 namespace |
| 回滚链路 | 等价于"清理"（整个 namespace 删除）；无"回退到上一版"语义 |
| 优点 | 延迟低；不影响 GitOps 仓库；适合一次性实验 |
| 缺点 | 审计停在 DB；若 AutoOps DB 丢失则现场只有注解；人肉 kubectl apply 可覆盖（靠治理+RBAC 防）|
| Namespace 约定 | `ao-direct-{prefix}-{releaseName}`（`ClusterTarget.DirectNamespacePrefix` 控制）|
| 资源标签 | `app.kubernetes.io/managed-by=autoops` + `autoops.io/deploy-mode=direct` + `autoops.io/owner-system=direct` + `autoops.io/request-id=<requestNo>` + `autoops.io/ttl-expire-at=<RFC3339>` |
| 生命周期 | TTL（`ttl_hours` 默认 72 h，上限由 `ClusterTarget.DefaultTtlHours` 控）|
| 权限控制 | AutoOps 持有 scoped kubeconfig（仅对 `ao-direct-*` 命名空间有权限）；通过 `ValidateDirectKubeconfigAccess` 的 `SelfSubjectAccessReview` 探测 |

### 4.3 两种模式的隔离（治理边界）

| 隔离维度 | GitOps | Direct |
|---------|--------|--------|
| Namespace 前缀 | `ao-gitops-*` | `ao-direct-*`（前缀可配）|
| `managed-by` label | `argocd` | `autoops` |
| `owner-system` label | `gitops` | `direct` |
| `deploy-mode` label | `gitops` | `direct` |
| 资源 owner 表 | `resource_owner.owner_system='gitops'` | `resource_owner.owner_system='direct'` |
| 生命周期策略 | Git 文件删除 = 下线 | TTL 或手动 cleanup |
| RBAC 权限码 | `deploy:request:create` + `deploy:request:execute` | 同上，但**额外**对 direct 加 `deploy:request:direct`（Phase X 新增）|
| kubeconfig | 不持有（由 ArgoCD 持有）| AutoOps 持有，scope 仅限 `ao-direct-*` |

**单 owner 原则实现**：
- `reserveResourceOwners()` 在执行前查询 `deploy_resource_owner WHERE cluster_target_id=? AND namespace=? AND kind=? AND name=? AND active=true`
- 若存在 `owner_system != 本次 mode` 的 active 记录 → 报错阻塞（现有逻辑，不改）
- 若存在同 mode 的 active 记录 → 视为更新当前 owner（现有逻辑）
- 下线（Phase D 新增）：必须调 `DeactivateResourceOwnersByRequestID` 释放占位

---

## 5. 风险与治理原则

| 风险 | 可能性 | 影响 | 缓解措施 |
|------|--------|------|---------|
| **R1: GitOps / direct 漂移或打架** | 中 | 高 | `resource_owner` 表硬约束；Phase X 补一条"owner 冲突告警"到 bot；review 前 smoke test 双写同 name 必须报错 |
| **R2: 审批绕过** | 低 | 致命 | `executeDeployRequestInternal` 入口必 check `approval_status=approved`（已在 L541）；Phase X 新增测试 `TestExecuteBlocksUnapprovedRequest` |
| **R3: 审计不完整** | 中 | 中 | Phase X-1 补 AuditMiddleware；agent 路由带 `source='agent'` 标记；结果通知发送记录入 `deploy_notification` 表（Phase B 新增）|
| **R4: K8s 权限过大** | 中 | 高 | direct kubeconfig 必须 scope 到 `ao-direct-*` 命名空间；`ValidateDirectKubeconfigAccess` 的 SSAR 检查作为 ClusterTarget 创建时的准入校验；**不得使用 cluster-admin kubeconfig** |
| **R5: Hermes API 被滥用** | 中 | 高 | Bearer token 全局 1 个 → Phase C 改为**按 Hermes 实例/agent 申请单 token**（`agent_token` 表，Phase C 可选）；短期用频率限制 middleware（Phase X-2）|
| **R6: 群聊误触发** | 高 | 中 | Hermes skill 解析后**必须回显确认卡**；用户需二次确认才调 AutoOps；误触发只会创建申请，不会绕过审批 |
| **R7: 缺回滚/状态回写** | 高 | 中 | Phase D 把 GitOps 回滚/下线做完；执行失败时 `execution_status=failed` + `request_status=failed` + bot 必须回群（不能静默）|
| **R8: 前端/后端/执行器先长歪** | 中 | 中 | 每个 Phase 先写 service 层集成测试（Go 单元测试 + testcontainer PG），前端先落 API 绑定再出 UI；Phase 内部任务按 "test → impl → verify → commit" 强制顺序 |
| **R9: pukka-gitops 里出现孤儿 release 文件** | 中 | 低 | Phase D 提供 reconcile 命令（`go run main.go -c config.yaml --reconcile-gitops`），定期比对 `deploy_request WHERE execution_status=succeeded AND mode=gitops` 与仓库里文件列表，列出孤儿（不自动删）|
| **R10: 钉钉 access_token 缓存失效导致雪崩** | 低 | 中 | `dingtalkApproval.GetAccessToken` 加 TTL 缓存（若没有）；Phase A Task A-2 验证 |

### 治理原则（硬约束）

1. **单 owner 原则**：同一 `(cluster, namespace, kind, name)` 在 `resource_owner` 里最多一条 `active=true` 记录。违反 = 执行被阻塞。
2. **审批不可绕过**：所有执行入口必 check `approval_status=approved`。Hermes 也不例外。
3. **Hermes 不碰 K8s/Git**：Hermes skill 的代码只能出现 HTTP 调用，不得 `import kubernetes`、`subprocess git`、写本地路径。
4. **direct 小范围**：direct mode 命名空间前缀固定；不允许在其他 namespace 创建 direct 资源（在 `ensureDirectNamespace` 里强制校验）。
5. **审计全量**：页面 / Agent / Scheduler 三种触发路径必须都在 `execution_record.detail_json` 里能溯源。

---

## 6. 建议的 MVP 范围

### 第一版支持范围

| 维度 | MVP 内 | 显式延后 |
|------|--------|---------|
| 资源类型 | Pod / Deployment / Service（已支持）| Ingress / Gateway API / StatefulSet / HPA / ConfigMap / Secret |
| 环境 | 仅 pukka 测试集群（`env_type=test`）| staging / prod |
| 审批流 | 钉钉 OA（1 个审批人）| 多级审批、串签、或签、抄送 |
| Hermes 能力 | 自然语言 → 结构化请求（nginx 场景）+ 状态查询 | 审批、取消、回滚、跨集群路由 |
| 结果通知 | 钉钉机器人文本消息（成功 / 失败）| ActionCard、交互按钮、失败重试、多平台（Lark/企微）|
| GitOps 回滚 | 删除 release 文件（下线语义）| 指定 revision 精准回滚 |
| 观测性 | `deploy_notification` 表 + `execution_record.detail_json` | Prometheus metrics、trace 注入、Grafana 面板 |
| 权限 | 现有 RBAC 粒度 | 按 cluster / namespace 的细粒度数据权限 |
| Agent Token | 全局 1 个 Bearer | 按 agent / scope / expiry 的多 token 体系 |

### 必须延后

- ❌ Helm chart/Kustomize overlay 类型的部署（Phase D 之后再议）
- ❌ 钉钉交互卡片 / ActionCard 支持
- ❌ 跨集群批量部署
- ❌ 自动扩缩容（HPA）配置入口
- ❌ 前端可视化拓扑 / drift 展示（ArgoCD 已有 UI，不重复造）

---

## 7. 领域模型设计建议

### 7.1 已存在模型（不改 schema，仅补注释）

| 模型 | 表 | 备注 |
|------|----|------|
| `ClusterTarget` | `deploy_cluster_target` | 本计划不改 schema |
| `DeployRequest` | `deploy_request` | `ChatContextJSON` 本计划起**正式消费**；建议在 model 注释里固化 JSON schema |
| `ApprovalRecord` | `deploy_approval_record` | 不改 |
| `ExecutionRecord` | `deploy_execution_record` | 不改；`detail_json` 约定扩展点 |
| `ResourceOwner` | `deploy_resource_owner` | 不改；`active` 字段在 Phase D 被回写 |

### 7.2 `ChatContextJSON` 的正式 schema（本计划固化）

```json
{
  "provider": "dingtalk",          // 枚举: dingtalk | lark | wecom（MVP 仅 dingtalk）
  "chat_id": "chatxxx",            // 会话 ID（钉钉 openConversationId 或 webhook 末段 token hash）
  "at_mobiles": ["13800138000"],   // 回发时 @ 谁
  "at_user_ids": [],               // 可选
  "sender_external_id": "ding_user_xxx",
  "origin_message_id": "msg_xxx",
  "extra": {}                      // provider 专有扩展
}
```

### 7.3 新增模型（Phase B 新增）

#### `DeployNotification`（表 `deploy_notification`）

```go
type DeployNotification struct {
    ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    RequestID    uint      `gorm:"not null;index" json:"requestId"`
    Channel      string    `gorm:"type:varchar(32);not null" json:"channel"`       // dingtalk_robot | lark_robot
    Stage        string    `gorm:"type:varchar(32);not null" json:"stage"`         // executed | failed
    PayloadJSON  string    `gorm:"type:text" json:"payloadJson"`                   // 发送的 payload 快照
    Status       string    `gorm:"type:varchar(32);default:'pending';index" json:"status"` // pending | sent | failed
    ErrorMessage string    `gorm:"type:text" json:"errorMessage,omitempty"`
    SentAt       *time.Time `json:"sentAt,omitempty"`
    CreatedAt    time.Time `json:"createdAt"`
}
```

用途：记录每一次群聊消息发送（含 payload + 结果），方便排查"为什么没收到通知"。

#### `DeployBotEndpoint`（表 `deploy_bot_endpoint`，Phase B 可选，MVP 先全局配置即可）

如 Q2 最终决定需要按 cluster 覆盖，则新增。MVP 默认**不建**，仅在 `config.yaml` 里加一段全局配置。

### 7.4 状态机字段建议

**`DeployRequest` 的三个状态字段（已存在，补约束）**：

```
request_status:      submitted → pending_approval → approved → executing → (succeeded|failed|cancelled|expired)
                                                                    ↓
                                                                 rolled_back (Phase D)
                                                                    ↓
                                                                 cleaned (TTL)

approval_status:     pending → (approved|rejected|expired)

execution_status:    pending → running → (succeeded|failed|rolled_back|cleaned)
```

**Phase D 新增状态**：`request_status: rolled_back`（在 rollback 链路末尾回写）；`execution_record.status: rolled_back` 已存在但尚未被写入，Phase D 把它激活。

### 7.5 Owner / managed-by 语义

| 来源 | `owner_system` | K8s label `managed-by` | K8s label `deploy-mode` | Git |
|------|---------------|------------------------|-------------------------|-----|
| direct 模式 | `direct` | `autoops` | `direct` | 不落 |
| GitOps 模式 | `gitops` | `argocd` | `gitops` | Release CR 文件 |

**关键不变量**：`resource_owner` 表中，任意 `(cluster_target_id, namespace, kind, name)` 组合下 `active=true` 的行数 ≤ 1。Phase D 的下线操作必须维护此不变量。

---

## 8. API 与页面建议

### 8.1 后端 API（新增 / 改造）

| 路由 | 方法 | 新增/改造 | 用途 | 消费方 |
|------|------|---------|------|--------|
| `/api/v1/deploy/requests/:id/rollback` | POST | **新增**（Phase D）| 发起 GitOps 下线（删除 release 文件并 push） | 页面 |
| `/api/v1/deploy/requests/:id/notifications` | GET | **新增**（Phase B）| 查该申请的通知发送历史 | 页面 |
| `/api/v1/integrations/agent/deploy-requests/:requestNo/status` | GET | **新增**（Phase C）| 精简状态查询（给 Hermes 轮询用）| Hermes |
| `/api/v1/integrations/agent/deploy-requests` | POST | **改造**（Phase A/C）| 支持 `chat_context` 字段正式解析；原已接收但未消费，需落地 provider/chat_id 校验 | Hermes |
| `/api/v1/deploy/requests/:id/rollback` | POST | **新增**（Phase D）| 管理员在 UI 上手动触发下线 | 页面 |
| `/api/v1/deploy/cluster-targets/:id/reconcile-gitops` | POST | **新增**（Phase D 可选）| 触发 GitOps 孤儿文件 reconcile 报告 | 页面（管理员）|

Swagger 注释同步更新。

### 8.2 页面改动

| 页面 | 改动 |
|------|------|
| `K8sReleaseCenter.vue` | 1. 新增「通知记录」子对话框（Phase B，按 request 查 `deploy_notification`）<br>2. 新增「下线」按钮（Phase D，仅 GitOps 且 `execution_status=succeeded`）<br>3. 执行详情 dialog 里展示 `chatContextJson` 解析后的 provider / chat_id（Phase B）|
| 新弹窗：通知记录 | 展示 channel / stage / status / sent_at / error / payload preview |
| 新弹窗：下线确认 | 展示"此操作会删除 Git 中的 release 文件并触发 ArgoCD 回收"，确认后调 `/rollback` |

### 8.3 接口分类

| 类别 | 路由前缀 | 中间件 | 消费 |
|------|---------|-------|------|
| 页面 | `/api/v1/deploy/*` | `JWT` → `Audit` → `Rbac(code)` | Vue 前端 |
| Agent | `/api/v1/integrations/agent/*` | `AgentAuth` → `Audit(source=agent)` | Hermes |

**审计中间件必须覆盖两类路由**（Phase X-1）。

### 8.4 Hermes ↔ AutoOps 契约

#### 创建申请（Hermes → AutoOps）

```http
POST /api/v1/integrations/agent/deploy-requests
Authorization: Bearer <agent_token>
Content-Type: application/json

{
  "clusterTargetName": "pukka-dev",
  "mode": "gitops",
  "resourceType": "deployment",
  "releaseName": "nginx-demo",
  "namespace": "ao-gitops-nginx-demo",
  "image": "nginx:1.27.4-alpine",
  "replicas": 1,
  "serviceEnabled": true,
  "ttlHours": null,
  "reason": "测试环境 nginx 部署（来自钉钉群聊 @wangsan）",
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "ding_user_xxx",
  "requesterDisplayName": "王三",
  "approverAdminId": 89,
  "chatContext": {
    "provider": "dingtalk",
    "chat_id": "chatxxx",
    "at_mobiles": ["13800138000"],
    "at_user_ids": [],
    "sender_external_id": "ding_user_xxx",
    "origin_message_id": "msg_xxx"
  }
}
```

返回：
```json
{
  "code": 0,
  "data": {
    "requestNo": "DR202604170001",
    "requestStatus": "pending_approval",
    "approvalStatus": "pending",
    "approvalDispatchStatus": "dispatched",
    "dingtalkProcessInstanceId": "..."
  }
}
```

#### 查询状态（Hermes 轮询）

```http
GET /api/v1/integrations/agent/deploy-requests/DR202604170001/status
```

返回精简：
```json
{
  "code": 0,
  "data": {
    "requestNo": "DR202604170001",
    "requestStatus": "succeeded",
    "approvalStatus": "approved",
    "executionStatus": "succeeded",
    "finishedAt": "2026-04-17T14:30:22Z",
    "executionSummary": "GitOps: commit abc123 pushed to main"
  }
}
```

---

## 9. 实施计划（按 writing-plans 风格）

### 阶段总览

| Phase | 主题 | 预计任务数 | 前置 |
|-------|------|----------|------|
| **Phase X-prep** | 横切基线（先收口）| 3 | — |
| **Phase A** | P0 钉钉真实 userId 打通 | 5 | X-prep |
| **Phase B** | P1 结果回群机器人 | 8 | A |
| **Phase C** | P1 Hermes custom skill | 5 | A |
| **Phase D** | P2 GitOps 回滚/下线 | 6 | B |

---

### Phase X-prep: 横切基线收口

#### Task X-1: 给 deploy 路由挂 AuditMiddleware

**目标**：所有 deploy 页面路由 & agent 路由都经过审计落库，补足 handoff 里声称的"审计完整"。

**Files:**
- Modify: `api/router/deploy/deploy.go:14-43`
- Read first: `api/middleware/auditMiddleware.go`（确认接口签名与 `source` 参数支持）

- [ ] **Step 1: 阅读现有 AuditMiddleware 签名**

Run:
```bash
grep -n "func AuditMiddleware" /home/kchou/Code/AutoOps/api/middleware/auditMiddleware.go
```
Expected: 确认是否支持传入 `source` 标签（默认 'ui'），若不支持则先改造为 `AuditMiddleware(source string)`。

- [ ] **Step 2: 写失败测试 — 页面路由应落审计**

Create `api/api/deploy/controller/deploy_audit_test.go`:
```go
package controller_test

import (
    "bytes"
    "encoding/json"
    "net/http/httptest"
    "testing"

    "github.com/stretchr/testify/require"
)

func TestCreateDeployRequest_WritesAuditLog(t *testing.T) {
    srv, db := newTestServer(t)  // helper 见下一步
    body, _ := json.Marshal(map[string]any{
        "clusterTargetName": "pukka-dev",
        "mode":              "gitops",
        // ... minimum valid payload
    })
    req := httptest.NewRequest("POST", "/api/v1/deploy/requests", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer "+testJWT)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    var count int64
    require.NoError(t, db.Table("sys_audit_log").Where("path = ?", "/api/v1/deploy/requests").Count(&count).Error)
    require.Equal(t, int64(1), count, "应该落一条审计日志")
}
```

Run:
```bash
cd api && mise exec go@1.25.0 -- go test ./api/deploy/controller/... -run TestCreateDeployRequest_WritesAuditLog -v
```
Expected: FAIL（路由未挂 AuditMiddleware，无审计记录）

- [ ] **Step 3: 在 router 上挂 AuditMiddleware**

Edit `api/router/deploy/deploy.go`, 在两个 group 上添加：
```go
// 页面路由
router.Use(middleware.AuditMiddleware("ui"))  // ← 新增或挂在外层 group

// Agent 路由
agentGroup := router.Group("/integrations/agent")
agentGroup.Use(middleware.AgentAuthMiddleware(), middleware.AuditMiddleware("agent"))
```

（具体挂法视 `AuditMiddleware` 现状而定，若原无 source 参数则调用 `middleware.AuditMiddleware()`。）

- [ ] **Step 4: 跑测试通过**

Run:
```bash
cd api && mise exec go@1.25.0 -- go test ./api/deploy/controller/... -run TestCreateDeployRequest_WritesAuditLog -v
```
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/controller/deploy_audit_test.go api/router/deploy/deploy.go
git commit -m "feat(deploy): wire AuditMiddleware on UI and Agent routes"
```

---

#### Task X-2: 在 pukka-gitops 写 autoops-managed-releases 契约 README

**目标**：文档化 Release CR schema，让后续手工修 YAML / AutoOps 升级时有 schema 可查。

**Files:**
- Create: `/home/kchou/Code/pukka-gitops/apps/autoops-managed-releases/README.md`

- [ ] **Step 1: 读当前模板理清字段**

Run:
```bash
cat /home/kchou/Code/pukka-gitops/apps/autoops-managed-releases/templates/releases.yaml
cat /home/kchou/Code/pukka-gitops/apps/autoops-managed-releases/releases/smoke-gitops-235414.yaml
```

- [ ] **Step 2: 写 README**

Write `/home/kchou/Code/pukka-gitops/apps/autoops-managed-releases/README.md`:
```markdown
# AutoOps Managed Releases

This directory is the AutoOps → GitOps handoff point.
**Do NOT hand-edit** `releases/*.yaml` under normal operation — AutoOps writes them.

## Ownership
- Writer: AutoOps `service/gitopsWriter.go` via `WriteGitOpsReleaseToWorkingTree`
- Reader: ArgoCD Application `autoops-managed-releases` (sync-wave 3)
- Renderer: `templates/releases.yaml` (Helm `.Files.Glob "releases/*.yaml"`)

## Release CR Schema (v1alpha1)

\`\`\`yaml
apiVersion: autoops.io/v1alpha1
kind: Release
metadata:
  name: <releaseName>       # 与 deploy_request.release_name 一致
spec:
  namespace: ao-gitops-<releaseName>
  resourceType: deployment  # deployment | pod
  image: <image:tag>
  replicas: 1
  service:
    enabled: true
    type: ClusterIP
    port: 80
    targetPort: 80
  labels:
    app.kubernetes.io/managed-by: argocd
    autoops.io/deploy-mode: gitops
  annotations:
    autoops.io/release-name: <releaseName>
    autoops.io/source: autoops-gitops
\`\`\`

## Lifecycle
- Create: AutoOps approves & pushes `releases/<name>.yaml`
- Update: AutoOps overwrites the file + git commit
- Delete (rollback): AutoOps removes the file + git commit (Phase D, 2026-04+)

## Namespace Convention
All AutoOps-managed workloads live in `ao-gitops-<releaseName>`.
Infrastructure components (Jenkins, Harbor, etc.) do NOT live here.
```

- [ ] **Step 3: Commit in pukka-gitops**

```bash
cd /home/kchou/Code/pukka-gitops
git add apps/autoops-managed-releases/README.md
git commit -m "docs(autoops-managed-releases): document Release CR contract"
```
Expected: commit created.

---

#### Task X-3: 抽象 DingTalk 机器人 SDK 到 pkg

**目标**：把 `api/n9e/service/notifier.go` 里的 DingTalk webhook 发送逻辑提炼到 `api/pkg/dingtalkbot/`，给 Phase B 复用；同时保持 n9e 原行为不变。

**Files:**
- Create: `api/pkg/dingtalkbot/client.go`
- Create: `api/pkg/dingtalkbot/client_test.go`
- Modify: `api/api/n9e/service/notifier.go`（把 SendDingtalk 内部实现改调 pkg）

- [ ] **Step 1: 写测试（table-driven）**

Create `api/pkg/dingtalkbot/client_test.go`:
```go
package dingtalkbot_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "dodevops-api/pkg/dingtalkbot"
)

func TestClient_SendText(t *testing.T) {
    var captured []byte
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        captured, _ = io.ReadAll(r.Body)
        w.Write([]byte(`{"errcode":0}`))
    }))
    defer srv.Close()

    c := dingtalkbot.NewClient(dingtalkbot.Config{WebhookURL: srv.URL, Secret: "test"})
    err := c.SendText("hello", []string{"13800138000"}, nil)
    if err != nil { t.Fatal(err) }

    if !strings.Contains(string(captured), "hello") {
        t.Errorf("payload missing text: %s", captured)
    }
    if !strings.Contains(string(captured), "13800138000") {
        t.Errorf("payload missing atMobile: %s", captured)
    }
}

func TestClient_SendMarkdown(t *testing.T) {
    // 同构 — 断言 msgtype=markdown + title + text
}
```

Run:
```bash
cd api && mise exec go@1.25.0 -- go test ./pkg/dingtalkbot/... -v
```
Expected: FAIL（包不存在）

- [ ] **Step 2: 实现 client**

Create `api/pkg/dingtalkbot/client.go`:
```go
package dingtalkbot

import (
    "bytes"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strconv"
    "time"
)

type Config struct {
    WebhookURL string
    Secret     string
    Timeout    time.Duration
}

type Client struct {
    cfg  Config
    http *http.Client
}

func NewClient(cfg Config) *Client {
    to := cfg.Timeout
    if to == 0 { to = 5 * time.Second }
    return &Client{cfg: cfg, http: &http.Client{Timeout: to}}
}

type textPayload struct {
    MsgType string `json:"msgtype"`
    Text    struct{ Content string `json:"content"` } `json:"text"`
    At      struct {
        AtMobiles []string `json:"atMobiles,omitempty"`
        AtUserIds []string `json:"atUserIds,omitempty"`
        IsAtAll   bool     `json:"isAtAll,omitempty"`
    } `json:"at"`
}

func (c *Client) SendText(content string, atMobiles, atUserIds []string) error {
    var p textPayload
    p.MsgType = "text"
    p.Text.Content = content
    p.At.AtMobiles = atMobiles
    p.At.AtUserIds = atUserIds
    return c.send(p)
}

type markdownPayload struct {
    MsgType  string `json:"msgtype"`
    Markdown struct{ Title, Text string } `json:"markdown"`
    At       struct{ AtMobiles, AtUserIds []string; IsAtAll bool } `json:"at"`
}

func (c *Client) SendMarkdown(title, text string, atMobiles, atUserIds []string) error {
    var p markdownPayload
    p.MsgType = "markdown"
    p.Markdown.Title = title
    p.Markdown.Text = text
    p.At.AtMobiles = atMobiles
    p.At.AtUserIds = atUserIds
    return c.send(p)
}

func (c *Client) send(payload any) error {
    body, _ := json.Marshal(payload)
    u := c.cfg.WebhookURL
    if c.cfg.Secret != "" {
        ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
        stringToSign := ts + "\n" + c.cfg.Secret
        mac := hmac.New(sha256.New, []byte(c.cfg.Secret))
        mac.Write([]byte(stringToSign))
        sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
        q := url.Values{"timestamp": {ts}, "sign": {sig}}
        if strings.Contains(u, "?") { u += "&" + q.Encode() } else { u += "?" + q.Encode() }
    }
    resp, err := c.http.Post(u, "application/json", bytes.NewReader(body))
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("dingtalk robot status=%d", resp.StatusCode)
    }
    var r struct{ Errcode int; Errmsg string }
    _ = json.NewDecoder(resp.Body).Decode(&r)
    if r.Errcode != 0 { return fmt.Errorf("dingtalk robot errcode=%d errmsg=%s", r.Errcode, r.Errmsg) }
    return nil
}
```

- [ ] **Step 3: 跑测试通过**

Run:
```bash
cd api && mise exec go@1.25.0 -- go test ./pkg/dingtalkbot/... -v
```
Expected: PASS

- [ ] **Step 4: 把 n9e/service/notifier.go 的 SendDingtalk 改调 pkg**

Edit `api/api/n9e/service/notifier.go` 把现有 SendDingtalk 里的 http 发送部分替换为：
```go
import "dodevops-api/pkg/dingtalkbot"
// ...
client := dingtalkbot.NewClient(dingtalkbot.Config{WebhookURL: config.WebhookURL, Secret: config.Secret})
return client.SendMarkdown(msg.Title, msg.Content, nil, nil)
```

Run现有 n9e 测试不能破：
```bash
cd api && mise exec go@1.25.0 -- go test ./api/n9e/... -v
```
Expected: PASS（n9e 回归通过）

- [ ] **Step 5: Commit**
```bash
git add api/pkg/dingtalkbot/ api/api/n9e/service/notifier.go
git commit -m "feat(dingtalkbot): extract robot SDK to pkg and reuse in n9e notifier"
```

---

### Phase A: P0 打通真实钉钉 OA 审批实例

#### Task A-1: sys_admin 表 dingtalk_user_id 字段盘点 + 手工补录指引

**目标**：确认数据库里每个需要做审批人/发起人的 `sys_admin` 记录都有真实 `dingtalk_user_id`；产出补录 SOP。

**Files:**
- Create: `docs/dingtalk-userid-bootstrap.md`
- Read: `api/api/system/model/sysAdmin.go`（确认字段存在）
- Run SQL: 数据盘点

- [ ] **Step 1: 核查字段与现有数据**

```bash
grep -n "DingtalkUserId\|dingtalk_user_id" /home/kchou/Code/AutoOps/api/api/system/model/sysAdmin.go
docker exec devops-postgres psql -U devops -d autoops -c "select id, username, nickname, dingtalk_user_id from sys_admin order by id;"
```
Expected: 字段存在；看到占位值 `smoke-dingtalk-admin` 等。

- [ ] **Step 2: 写 SOP 文档**

Create `docs/dingtalk-userid-bootstrap.md`（内容：如何在钉钉后台查真实 userId → psql UPDATE 语句模板 → 验证方式）。最少 3 段：如何查 userId、如何 UPDATE、如何验证。

- [ ] **Step 3: 手动补录 + 验证**

```bash
# 假设审批人真实 userId = "manager01"
docker exec devops-postgres psql -U devops -d autoops -c "UPDATE sys_admin SET dingtalk_user_id='manager01' WHERE id=89;"
docker exec devops-postgres psql -U devops -d autoops -c "select id, dingtalk_user_id from sys_admin where id=89;"
```
Expected: `dingtalk_user_id='manager01'`

- [ ] **Step 4: Commit**
```bash
git add docs/dingtalk-userid-bootstrap.md
git commit -m "docs(dingtalk): userid bootstrap SOP for real OA dispatch"
```

---

#### Task A-2: `dingtalkApproval.GetAccessToken` 加缓存

**目标**：避免每次 `CreateProcessInstance` 都拉一次 access token（钉钉有频控）。

**Files:**
- Modify: `api/api/deploy/service/dingtalkApproval.go:96-138`
- Test: `api/api/deploy/service/dingtalkApproval_test.go`

- [ ] **Step 1: 先读现状，确认是否已有缓存**

Run:
```bash
grep -n "accessToken\|AccessToken" /home/kchou/Code/AutoOps/api/api/deploy/service/dingtalkApproval.go
```
- 若已有 TTL 缓存 → 跳过本 task
- 若无 → 继续

- [ ] **Step 2: 写测试**

Create `dingtalkApproval_test.go` 中 `TestGetAccessToken_CachesUntilExpiry`：两次调用只发一次 HTTP（用 httptest server 计数）。

- [ ] **Step 3: 给 service 加 struct 字段 `tokenCache { token string; expiresAt time.Time; mu sync.Mutex }`**

在 `GetAccessToken` 开头 check；锁 + double-check；过期前 30 s 主动刷新。

- [ ] **Step 4: 跑测试通过**

```bash
cd api && mise exec go@1.25.0 -- go test ./api/deploy/service/... -run TestGetAccessToken -v
```

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/service/dingtalkApproval.go api/api/deploy/service/dingtalkApproval_test.go
git commit -m "feat(deploy): cache dingtalk access_token until near-expiry"
```

---

#### Task A-3: 真实 OA 实例 smoke test（脚本化）

**目标**：写一个可重复执行的 shell 脚本，用真实 userId 发起一条部署申请 → 断言 `dingtalkProcessInstanceId` 已落库。

**Files:**
- Create: `scripts/deploy/smoke-oa-dispatch.sh`

- [ ] **Step 1: 写脚本**

Create `scripts/deploy/smoke-oa-dispatch.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
API="${API:-http://127.0.0.1:18000}"
TOKEN="${TOKEN:-$(cat /tmp/admin.token)}"

REQ_BODY=$(cat <<JSON
{
  "clusterTargetId": 1,
  "mode": "direct",
  "resourceType": "deployment",
  "releaseName": "oa-smoke-$(date +%s)",
  "namespace": "ao-direct-smoke",
  "image": "nginx:1.27.4-alpine",
  "replicas": 1,
  "ttlHours": 1,
  "approverAdminId": 89,
  "reason": "OA smoke test"
}
JSON
)

echo "==> Creating deploy request..."
RESP=$(curl -sf -X POST "$API/api/v1/deploy/requests" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$REQ_BODY")
echo "$RESP" | jq

REQUEST_NO=$(echo "$RESP" | jq -r '.data.requestNo')
DISPATCH_STATUS=$(echo "$RESP" | jq -r '.data.approvalDispatchStatus')
PROC_ID=$(echo "$RESP" | jq -r '.data.dingtalkProcessInstanceId')

echo "requestNo=$REQUEST_NO dispatch=$DISPATCH_STATUS procId=$PROC_ID"

if [[ "$DISPATCH_STATUS" != "dispatched" ]]; then
  echo "FAIL: approvalDispatchStatus != dispatched"
  exit 1
fi
if [[ -z "$PROC_ID" || "$PROC_ID" == "null" ]]; then
  echo "FAIL: dingtalkProcessInstanceId empty"
  exit 1
fi
echo "PASS"
```

- [ ] **Step 2: 跑脚本**

```bash
chmod +x scripts/deploy/smoke-oa-dispatch.sh
./scripts/deploy/smoke-oa-dispatch.sh
```
Expected: `PASS`，且钉钉 APP 能看到该审批单。

- [ ] **Step 3: Commit**
```bash
git add scripts/deploy/smoke-oa-dispatch.sh
git commit -m "test(deploy): smoke script for real dingtalk OA dispatch"
```

---

#### Task A-4: 审批通过 → 自动执行 smoke（direct mode）

**目标**：在钉钉 APP 手动审批通过后，断言 AutoOps 能自动执行并落 K8s 资源。

**Files:**
- Create: `scripts/deploy/smoke-oa-autoexec.sh`

- [ ] **Step 1: 写脚本**

Create 脚本：等待 `sync-approval` 接口或 scheduler 触发后，轮询 `GET /deploy/requests/:id` 直至 `execution_status == succeeded` 或超时（3 分钟）。打印最终 `ExecutionRecord`。若失败 → 退出码 1。

- [ ] **Step 2: 执行一次完整链路**

```bash
# 1) 跑 Task A-3 创建申请
./scripts/deploy/smoke-oa-dispatch.sh > /tmp/req.json
REQUEST_NO=$(jq -r '.data.requestNo' /tmp/req.json)

# 2) 手动在钉钉 APP 审批通过

# 3) 调 sync 接口或等 scheduler
curl -sf -X POST "$API/api/v1/deploy/requests/$(...)/sync-approval" ...

# 4) 轮询
./scripts/deploy/smoke-oa-autoexec.sh "$REQUEST_NO"
```
Expected: `PASS`，K8s 里 `ao-direct-smoke` namespace 有 nginx pod。

- [ ] **Step 3: Commit**
```bash
git add scripts/deploy/smoke-oa-autoexec.sh
git commit -m "test(deploy): smoke script for OA-approved auto-execution"
```

---

#### Task A-5: 补 Phase A 单元测试 — `tryDispatchApproval` 对无效 userId 报错语义

**目标**：确保 `sys_admin.dingtalk_user_id` 为空或占位时，`tryDispatchApproval` 返回明确错误，不吞不 dispatch。

**Files:**
- Modify: `api/api/deploy/service/deploy.go:765` 附近的 `tryDispatchApproval`（若未实现）
- Test: `api/api/deploy/service/deploy_test.go`

- [ ] **Step 1: 写测试 `TestTryDispatchApproval_EmptyApproverUserID_Rejected`**

期望 `approval_dispatch_status=failed` 且 `approval_dispatch_message` 含 "approver dingtalk_user_id empty"。

- [ ] **Step 2: 跑测试 → FAIL**

- [ ] **Step 3: 在 `tryDispatchApproval` 里加前置校验**

```go
if approver.DingtalkUserID == "" {
    return req, fmt.Errorf("approver dingtalk_user_id empty (sys_admin.id=%d)", approver.ID)
}
```

调用方（`markApprovalDispatch`）写 `status=failed` + `message=err.Error()`。

- [ ] **Step 4: 跑测试 → PASS**

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/service/deploy.go api/api/deploy/service/deploy_test.go
git commit -m "feat(deploy): reject OA dispatch when approver dingtalk_user_id missing"
```

---

### Phase B: P1 结果回群机器人

#### Task B-0: 给 `config.yaml` 加 deploy robot 配置段

**目标**：全局配置一个机器人 webhook，MVP 先全局；后续按 cluster 覆盖走 B-6 可选。

**Files:**
- Modify: `api/common/config/config.go`
- Modify: `api/config.yaml.example`

- [ ] **Step 1: 加配置结构**

Edit `api/common/config/config.go`，在 `Integrations` 下加：
```go
type DeployBot struct {
    Provider   string `yaml:"provider"`   // dingtalk
    WebhookURL string `yaml:"webhook_url"`
    Secret     string `yaml:"secret"`
    Enabled    bool   `yaml:"enabled"`
}
type integrations struct {
    Agent     agent
    DeployBot DeployBot `yaml:"deploy_bot"`
}
```

- [ ] **Step 2: 加 example**

Edit `api/config.yaml.example`:
```yaml
integrations:
  agent:
    bearer_token: "..."
  deploy_bot:
    provider: dingtalk
    enabled: false
    webhook_url: "https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN"
    secret: ""
```

- [ ] **Step 3: Commit**
```bash
git add api/common/config/config.go api/config.yaml.example
git commit -m "feat(config): add deploy_bot section for result-back-to-chat"
```

---

#### Task B-1: DeployNotification 模型 + 迁移注册

**目标**：建 `deploy_notification` 表。

**Files:**
- Modify: `api/api/deploy/model/deploy.go`（加 `DeployNotification` struct）
- Modify: `api/pkg/db/migrate.go`（注册）

- [ ] **Step 1: 写测试 `TestMigrate_CreatesDeployNotificationTable`**

Create a migration test that runs AutoMigrate on a throwaway DB and asserts `deploy_notification` table exists with expected columns. Run → FAIL.

- [ ] **Step 2: 加 struct**

```go
type DeployNotification struct {
    ID           uint      `gorm:"primaryKey;autoIncrement" json:"id"`
    RequestID    uint      `gorm:"not null;index" json:"requestId"`
    Channel      string    `gorm:"type:varchar(32);not null" json:"channel"`
    Stage        string    `gorm:"type:varchar(32);not null" json:"stage"`
    PayloadJSON  string    `gorm:"type:text" json:"payloadJson"`
    Status       string    `gorm:"type:varchar(32);default:'pending';index" json:"status"`
    ErrorMessage string    `gorm:"type:text" json:"errorMessage,omitempty"`
    SentAt       *time.Time `json:"sentAt,omitempty"`
    CreatedAt    time.Time `json:"createdAt"`
}
func (DeployNotification) TableName() string { return "deploy_notification" }
```

- [ ] **Step 3: 注册到 migrate.go**

Edit `api/pkg/db/migrate.go` line ~65，加入 `&deploymodel.DeployNotification{}`。

- [ ] **Step 4: 跑测试 → PASS**

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/model/deploy.go api/pkg/db/migrate.go
git commit -m "feat(deploy): add deploy_notification model for bot send history"
```

---

#### Task B-2: `deploy/service/notifier.go` — DeployNotifier

**目标**：根据 `DeployRequest.ChatContextJSON` + 全局 bot 配置，构造并发送结果消息，记录到 `deploy_notification`。

**Files:**
- Create: `api/api/deploy/service/notifier.go`
- Test: `api/api/deploy/service/notifier_test.go`

- [ ] **Step 1: 写测试 `TestDeployNotifier_SendsSuccessText`**

用 httptest server 拦截 dingtalk 请求；断言消息内容包含 `requestNo`、`mode`、`namespace`、`结果: 成功`；断言 DB 落一条 `deploy_notification status=sent`。

- [ ] **Step 2: 实现**

```go
package service

import (
    "dodevops-api/common/config"
    "dodevops-api/pkg/dingtalkbot"
    // ...
)

type IDeployNotifier interface {
    NotifyExecutionResult(req *DeployRequest, exec *ExecutionRecord) error
}

type deployNotifier struct {
    db *gorm.DB
}

func NewDeployNotifier(db *gorm.DB) IDeployNotifier {
    return &deployNotifier{db: db}
}

func (n *deployNotifier) NotifyExecutionResult(req *DeployRequest, exec *ExecutionRecord) error {
    botCfg := config.Config.Integrations.DeployBot
    if !botCfg.Enabled || botCfg.WebhookURL == "" {
        return nil // no-op, 合法跳过
    }
    var chatCtx ChatContext
    if req.ChatContextJSON != "" {
        _ = json.Unmarshal([]byte(req.ChatContextJSON), &chatCtx)
    }
    title, text := n.buildMarkdown(req, exec)

    record := &DeployNotification{
        RequestID: req.ID, Channel: "dingtalk_robot",
        Stage: stageFromExecStatus(exec.Status),
        PayloadJSON: marshalPayload(title, text, chatCtx),
        Status: "pending",
    }
    n.db.Create(record)

    client := dingtalkbot.NewClient(dingtalkbot.Config{WebhookURL: botCfg.WebhookURL, Secret: botCfg.Secret})
    err := client.SendMarkdown(title, text, chatCtx.AtMobiles, chatCtx.AtUserIDs)
    if err != nil {
        record.Status = "failed"
        record.ErrorMessage = err.Error()
    } else {
        now := time.Now(); record.Status = "sent"; record.SentAt = &now
    }
    n.db.Save(record)
    return err
}

func (n *deployNotifier) buildMarkdown(req *DeployRequest, exec *ExecutionRecord) (string, string) {
    status := map[string]string{"succeeded":"✅ 成功","failed":"❌ 失败","rolled_back":"↩️ 已回滚","cleaned":"🧹 已回收"}[exec.Status]
    return fmt.Sprintf("AutoOps 部署结果 - %s", req.RequestNo),
        fmt.Sprintf("#### AutoOps 部署结果 %s\n\n- **申请号**: %s\n- **模式**: %s\n- **资源**: %s/%s\n- **集群**: %s\n- **状态**: %s",
            status, req.RequestNo, req.Mode, req.Namespace, req.ReleaseName, req.ClusterTargetID, exec.Status)
}
```

- [ ] **Step 3: 跑测试 → PASS**

```bash
cd api && mise exec go@1.25.0 -- go test ./api/deploy/service/... -run TestDeployNotifier -v
```

- [ ] **Step 4: Commit**
```bash
git add api/api/deploy/service/notifier.go api/api/deploy/service/notifier_test.go
git commit -m "feat(deploy): deploy notifier sends execution result to dingtalk group chat"
```

---

#### Task B-3: 在 `executeDeployRequestInternal` 结束后调用 notifier

**目标**：把 notifier 挂到执行结束后（成功 / 失败 都触发）。

**Files:**
- Modify: `api/api/deploy/service/deploy.go:629-663`（执行完成写回库之后）

- [ ] **Step 1: 写测试 `TestExecute_TriggersNotifierOnSuccess`**

用 mock notifier 断言被调用一次；断言参数里 `exec.Status=succeeded`。

- [ ] **Step 2: 把 notifier 注入 service**

在 `deployService` struct 加 `notifier IDeployNotifier` 字段；`NewDeployService` 签名加参；调用方（main.go / wire）同步。

- [ ] **Step 3: 在 `executeDeployRequestInternal` 末尾调用**

```go
// 在 L660 附近，写回 request 状态之后
if err := s.notifier.NotifyExecutionResult(req, executionRecord); err != nil {
    zap.L().Warn("notify execution result failed", zap.Error(err), zap.String("requestNo", req.RequestNo))
}
```

- [ ] **Step 4: 跑测试 → PASS；跑全量 `go test ./...`**

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/service/deploy.go api/api/deploy/service/deploy_test.go api/main.go
git commit -m "feat(deploy): notify chat after execution complete"
```

---

#### Task B-4: API 查询通知记录

**目标**：页面可以查 "这次申请发了什么消息"。

**Files:**
- Modify: `api/api/deploy/dao/deploy.go`（加 `ListNotificationsByRequestID`）
- Modify: `api/api/deploy/service/deploy.go`（加 `ListNotifications`）
- Modify: `api/api/deploy/controller/deploy.go`（加 `ListNotifications`）
- Modify: `api/router/deploy/deploy.go`（挂 `/deploy/requests/:id/notifications`）

- [ ] **Step 1: 写集成测试 `TestListNotifications_ReturnsHistory`**

- [ ] **Step 2: 加 DAO 方法**

```go
func (d *deployDao) ListNotificationsByRequestID(requestID uint) ([]DeployNotification, error) {
    var list []DeployNotification
    err := db.DB().Where("request_id = ?", requestID).Order("id asc").Find(&list).Error
    return list, err
}
```

- [ ] **Step 3: 加 controller + router**

```go
router.GET("/deploy/requests/:id/notifications",
    middleware.RbacMiddleware("deploy:request:view"),
    deployCtrl.ListNotifications)
```

- [ ] **Step 4: 测试 → PASS**

- [ ] **Step 5: Commit**
```bash
git add api/api/deploy/...
git commit -m "feat(deploy): API to list notification history per request"
```

---

#### Task B-5: 前端 — 通知记录按钮 + 对话框

**目标**：K8sReleaseCenter 页面里每条申请新增「通知记录」按钮。

**Files:**
- Modify: `web/src/api/deploy.js`
- Modify: `web/src/views/K8s/K8sReleaseCenter.vue`

- [ ] **Step 1: `web/src/api/deploy.js` 加**
```js
export function getDeployNotifications(id) {
  return request({ url: `/deploy/requests/${id}/notifications`, method: 'get' })
}
```

- [ ] **Step 2: K8sReleaseCenter.vue 加对话框**

新增 `notificationDialogVisible` ref + `notificationList`；在现有 action 列里加按钮 `通知记录` → 调 `getDeployNotifications(row.id)` → 弹出表格（channel/stage/status/sentAt/errorMessage/payloadJson preview）。

- [ ] **Step 3: `npm run serve` 手动验证**

```bash
cd web && npm run serve
# 浏览器打开 /k8s/release-center，点一条申请的 "通知记录"，确认弹窗
```

- [ ] **Step 4: Commit**
```bash
git add web/src/api/deploy.js web/src/views/K8s/K8sReleaseCenter.vue
git commit -m "feat(k8s-release-center): view notification history dialog"
```

---

#### Task B-6: e2e smoke — 直连模式部署结束触发消息

**目标**：跑一条完整流程：创建申请 → 审批通过 → direct 执行 → 群聊收到消息。

**Files:**
- Create: `scripts/deploy/smoke-notifier.sh`

- [ ] **Step 1: 写脚本**

组合 A-3/A-4 的 smoke，执行结束后查 `/notifications` 并断言至少一条 `status=sent`。

- [ ] **Step 2: 跑脚本 + 人工检查群聊**

- [ ] **Step 3: Commit**

---

#### Task B-7: （可选）失败分支验证

**目标**：直连 apply 失败时，bot 消息必须发（❌ 失败 + 简要原因）。

**Files:**
- Test: `api/api/deploy/service/deploy_test.go` 补用例

- [ ] **Step 1/2**: 写测试（mock `ApplyDirectResources` 返回 err）→ 断言 notifier 被调一次 + `stage=failed`
- [ ] **Step 3**: Commit

---

#### Task B-8: `chatContextJson` schema 文档化

**目标**：在 `docs/deploy-control-plane.md` 加一节，明确 ChatContext JSON schema + 跨 provider 扩展指引。

**Files:**
- Modify: `docs/deploy-control-plane.md`

- [ ] **Step 1**: 写一节 "ChatContext Schema"，列 fields + 示例
- [ ] **Step 2**: Commit
```bash
git add docs/deploy-control-plane.md
git commit -m "docs(deploy): document chat_context schema"
```

---

### Phase C: P1 Hermes custom skill

#### Task C-1: `/api/v1/integrations/agent/deploy-requests/:requestNo/status` 精简状态接口

**目标**：给 Hermes 提供一个轻量轮询接口（不返回大字段如 env_json / resources_json）。

**Files:**
- Modify: `api/api/deploy/service/deploy.go`（加 `GetAgentStatusByRequestNo`）
- Modify: `api/api/deploy/controller/deploy.go`
- Modify: `api/router/deploy/deploy.go`

- [ ] **Step 1: 写测试**

`TestAgentGetStatus_ReturnsSlimPayload` 断言响应不含 `env_json`、`resources_json`；含 `executionSummary`。

- [ ] **Step 2: 实现**

- [ ] **Step 3: 挂路由**
```go
agentGroup.GET("/deploy-requests/:requestNo/status", deployCtrl.GetAgentStatus)
```

- [ ] **Step 4: 测试 → PASS**

- [ ] **Step 5: Commit**

---

#### Task C-2: Hermes skill 骨架

**目标**：在 `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/` 生成 SKILL.md 与配置模板。

**Files:**
- Create: `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/SKILL.md`
- Create: `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/README.md`
- Modify: `~/.hermes/cli-config.yaml`（加 autoops 段；仅在用户本机，不 commit 密钥）

- [ ] **Step 1: 先读同类 skill 结构**

```bash
cat ~/.hermes/hermes-agent/skills/devops/webhook-subscriptions/SKILL.md
```

- [ ] **Step 2: 写 SKILL.md**

```markdown
---
name: deploy-via-autoops
description: Create a Kubernetes deployment request via AutoOps from natural-language chat.
tools: http_request
activation: when user requests to deploy a simple service to a test cluster
---

## Usage
When the user says: "帮我部署 nginx 到 pukka 测试集群":

1. Parse intent:
   - clusterTargetName (default: pukka-dev)
   - mode (default: gitops for persistent, direct for quick test)
   - resourceType (default: deployment)
   - releaseName (derived from sanitized service name + timestamp if unspecified)
   - namespace (derived: ao-gitops-{releaseName} or ao-direct-{releaseName})
   - image (if user gave image:tag; else require clarification)
   - replicas (default: 1)
   - serviceEnabled (default: true for HTTP services)
   - ttlHours (default: 72 for direct, null for gitops)
   - reason (auto-fill from group chat context)

2. Confirm with user:
   "将在 pukka-dev 集群以 gitops 模式部署 nginx:1.27.4-alpine 到 ao-gitops-nginx-demo，审批人：王经理。是否继续？"

3. On confirmation, call AutoOps:
   POST {autoops.base_url}/api/v1/integrations/agent/deploy-requests
   Authorization: Bearer {autoops.agent_token}
   Body: see schema in README.md

4. Report back: requestNo + "已发起审批，请在钉钉 APP 中处理"

5. (Optional) Poll GET /api/v1/integrations/agent/deploy-requests/{requestNo}/status every 30s
   until executionStatus in (succeeded, failed, rolled_back).

## Configuration
Requires cli-config.yaml:
    integrations:
      autoops:
        base_url: "http://autoops.internal:18000"
        agent_token: "..."
```

- [ ] **Step 3: 写 README.md（对 schema 的详细补充 + 错误码表）**

- [ ] **Step 4: 本地运行 Hermes**

```bash
# 伪代码 — 视 Hermes CLI 用法
hermes skill list | grep deploy-via-autoops
hermes chat --skill deploy-via-autoops "帮我部署 nginx 到 pukka 测试"
```
Expected: Hermes 解析后提示确认 → 用户确认 → 调 AutoOps → 返回 requestNo。

- [ ] **Step 5: Commit（在 Hermes 仓库，视其 git 习惯）**

---

#### Task C-3: Hermes → AutoOps 契约集成测试

**目标**：不依赖真实 Hermes 运行环境，写一个 Python 脚本模拟 Hermes 的 HTTP 调用，验证端到端契约。

**Files:**
- Create: `scripts/deploy/hermes-contract-test.py`

- [ ] **Step 1: 脚本**

```python
#!/usr/bin/env python3
import os, json, requests, sys
API = os.environ.get("AUTOOPS_API", "http://127.0.0.1:18000")
TOKEN = os.environ["AGENT_TOKEN"]

body = {
    "clusterTargetId": 1,
    "mode": "direct",
    "resourceType": "deployment",
    "releaseName": f"hermes-smoke-{os.getpid()}",
    "namespace": "ao-direct-hermes-smoke",
    "image": "nginx:1.27.4-alpine",
    "replicas": 1,
    "ttlHours": 1,
    "approverAdminId": 89,
    "reason": "hermes contract test",
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "ding_smoke",
    "requesterDisplayName": "Hermes Smoke",
    "chatContext": {
        "provider": "dingtalk",
        "chat_id": "smoke_chat",
        "at_mobiles": [],
        "sender_external_id": "ding_smoke"
    }
}

r = requests.post(f"{API}/api/v1/integrations/agent/deploy-requests",
                  headers={"Authorization": f"Bearer {TOKEN}"}, json=body, timeout=10)
r.raise_for_status()
data = r.json()["data"]
request_no = data["requestNo"]
print(f"created: {request_no}")

r2 = requests.get(f"{API}/api/v1/integrations/agent/deploy-requests/{request_no}/status",
                  headers={"Authorization": f"Bearer {TOKEN}"}, timeout=10)
r2.raise_for_status()
print(json.dumps(r2.json(), indent=2, ensure_ascii=False))
```

- [ ] **Step 2: 运行**

```bash
AGENT_TOKEN=<token> python3 scripts/deploy/hermes-contract-test.py
```
Expected: 打印 requestNo + slim 状态 JSON。

- [ ] **Step 3: Commit**
```bash
git add scripts/deploy/hermes-contract-test.py
git commit -m "test(hermes): contract test script for autoops agent API"
```

---

#### Task C-4: Hermes skill 确认卡 / 防误触发

**目标**：skill 在真正调 AutoOps **之前**必须向用户再确认一次（文本确认即可），避免群聊误触发。

**Files:**
- Modify: `SKILL.md`（把确认步骤写死）
- Create: `skills/devops/deploy-via-autoops/examples/confirm-flow.md`

- [ ] **Step 1**: 在 SKILL.md 加一条强制约束 "Before POST, always ask user to confirm with a full echo of the parsed parameters"
- [ ] **Step 2**: 在 README 里写一个误触发恢复 SOP
- [ ] **Step 3**: Commit

---

#### Task C-5: 文档把 Hermes 链路加入 AutoOps docs

**目标**：在 `docs/deploy-control-plane.md` 加一节 "通过 Hermes / 钉钉群聊触发" 并指向 Hermes skill 路径。

**Files:**
- Modify: `docs/deploy-control-plane.md`

- [ ] **Step 1**: 加节
- [ ] **Step 2**: Commit

---

### Phase D: P2 GitOps 回滚 / 下线 / 状态回写

#### Task D-1: `gitopsWriter.go` 新增 `DeleteGitOpsRelease`

**目标**：提供删除 `releases/{name}.yaml` + git commit + push 的能力。

**Files:**
- Modify: `api/api/deploy/service/gitopsWriter.go`
- Test: `api/api/deploy/service/gitopsWriter_test.go`

- [ ] **Step 1: 写测试 `TestDeleteGitOpsRelease_RemovesFileAndCommits`**

用临时 git repo（`git init` 再 commit 一个 release 文件）；调 `DeleteGitOpsRelease` → 断言文件不存在 + 新增一条 commit；断言 commit message 含 "Delete AutoOps release"。

- [ ] **Step 2: 实现**

```go
func DeleteGitOpsRelease(req *DeployRequest, clusterTargetName, releaseDir, branch string) (*GitOpsDeleteResult, error) {
    filePath := fmt.Sprintf("%s/%s.yaml", releaseDir, req.ReleaseName)
    full := filepath.Join(gitopsRoot, filePath)
    if err := os.Remove(full); err != nil { return nil, err }
    // git add -A; git commit -m "Delete AutoOps release <requestNo>"; git push origin branch
    // ...
    return &GitOpsDeleteResult{CommitSHA: sha, FilePath: filePath}, nil
}
```

- [ ] **Step 3: 测试 → PASS**

- [ ] **Step 4: Commit**

---

#### Task D-2: `service/deploy.go` 新增 `RollbackDeployRequest`

**目标**：编排"下线"：调 `DeleteGitOpsRelease` → 更新 execution_status=rolled_back → deactivate ResourceOwners → notifier 发消息。

**Files:**
- Modify: `api/api/deploy/service/deploy.go`

- [ ] **Step 1: 写测试 `TestRollbackDeployRequest_GitOps`**

- [ ] **Step 2: 实现**

```go
func (s *deployService) RollbackDeployRequest(c *gin.Context, id uint) gin.H {
    req := dao.GetDeployRequestByID(id)
    if req.Mode != DeployModeGitOps {
        return gin.H{"code": 1, "msg": "only gitops mode supports rollback"}
    }
    if req.ExecutionStatus != ExecutionStatusSucceeded {
        return gin.H{"code": 1, "msg": "only succeeded requests can be rolled back"}
    }

    result, err := DeleteGitOpsRelease(req, ..., ..., ...)
    // create ExecutionRecord (phase=rollback, status=succeeded|failed)
    // update DeployRequest: request_status=rolled_back, execution_status=rolled_back
    dao.DeactivateResourceOwnersByRequestID(req.ID)
    s.notifier.NotifyExecutionResult(req, rollbackExec)
    // ...
}
```

- [ ] **Step 3: 测试 → PASS**

- [ ] **Step 4: Commit**

---

#### Task D-3: 路由 + 控制器暴露 `/rollback`

**Files:**
- Modify: `api/api/deploy/controller/deploy.go`
- Modify: `api/router/deploy/deploy.go`

- [ ] **Step 1: 写 E2E 测试**

- [ ] **Step 2: 挂路由**

```go
router.POST("/deploy/requests/:id/rollback",
    middleware.RbacMiddleware("deploy:request:execute"),
    deployCtrl.RollbackDeployRequest)
```

- [ ] **Step 3: 测试 → PASS**

- [ ] **Step 4: Commit**

---

#### Task D-4: 前端下线按钮

**Files:**
- Modify: `web/src/api/deploy.js`
- Modify: `web/src/views/K8s/K8sReleaseCenter.vue`

- [ ] **Step 1: api 加 `rollbackDeployRequest(id)`**
- [ ] **Step 2: 页面加按钮（仅 `mode=gitops && execution_status=succeeded` 显示）+ 二次确认**
- [ ] **Step 3: `npm run serve` 手动验收**
- [ ] **Step 4: Commit**

---

#### Task D-5: GitOps 孤儿文件 reconcile 命令（可选，MVP 下沉）

**目标**：提供一个 CLI flag（`--reconcile-gitops`）扫描 pukka-gitops 里 `releases/*.yaml` 与 DB 比对，列出孤儿。不自动删，仅报表。

**Files:**
- Modify: `api/main.go`（加 flag）
- Create: `api/api/deploy/service/reconcileGitOps.go`

- [ ] **Step 1: 实现**
- [ ] **Step 2: Bash 验证**
```bash
cd api && mise exec go@1.25.0 -- go run . -c config.yaml --reconcile-gitops
```
Expected: 打印表格（requestNo / file / status / owner_system）。

- [ ] **Step 3: Commit**

---

#### Task D-6: 失败后状态回写与二次回收

**目标**：若 rollback 本身失败（git push 失败），执行记录写 `phase=rollback, status=failed`，request_status 维持 `succeeded`（没回滚成），notifier 发失败消息。

**Files:**
- Modify: `api/api/deploy/service/deploy.go:RollbackDeployRequest`

- [ ] **Step 1: 写测试 `TestRollback_WhenGitPushFails_RequestRemainsSucceeded`**
- [ ] **Step 2: 处理错误分支**
- [ ] **Step 3: 测试 → PASS**
- [ ] **Step 4: Commit**

---

### 阶段验收矩阵

| Phase | Must-pass 验收命令 | 预期 |
|-------|-------------------|------|
| X-prep | `go test ./api/deploy/...` | 全绿 |
| X-prep | `cd /home/kchou/Code/pukka-gitops && git log --oneline -1` | 最近 commit 是 README |
| A | `./scripts/deploy/smoke-oa-dispatch.sh` | PASS |
| A | 钉钉 APP | 能看到真实审批单 |
| A | `./scripts/deploy/smoke-oa-autoexec.sh <requestNo>` | PASS |
| B | `./scripts/deploy/smoke-notifier.sh` | 群聊收到消息 + DB `deploy_notification.status=sent` |
| B | 浏览器访问 `/k8s/release-center` | 通知记录对话框可见 |
| C | `AGENT_TOKEN=x python3 scripts/deploy/hermes-contract-test.py` | 返回 requestNo + 状态 |
| C | `hermes chat --skill deploy-via-autoops` | 能走完"意图→确认→下单"流程 |
| D | `curl -X POST /deploy/requests/:id/rollback` | `request_status=rolled_back` + git 仓库 release 文件消失 |
| D | K8s `kubectl get ns ao-gitops-<name>` | NotFound（ArgoCD 回收）|

---

## 10. 明确结论

### 10.1 AutoOps 直接管理部分部署是否可行？

**可行，但只能小范围长期存在**（与 handoff 结论一致，本计划确认此边界）。

### 10.2 适合什么范围

1. 临时测试环境、群聊即发即用的 POC
2. `ao-direct-*` 命名空间下、带 TTL、单体服务（nginx / simple API / test harness）
3. 需要极低延迟、不要求跨人审计的内部实验
4. 不涉及持久化数据（无 PVC / Secret 依赖跨环境复现）的资源

### 10.3 不适合什么范围

1. ❌ 正式业务（应一律走 GitOps）
2. ❌ 需要多人协同维护、code review、code freeze 的服务
3. ❌ 需要 ArgoCD drift detection / auto-sync 的生命周期
4. ❌ 跨集群、多环境（staging/prod）的统一配置
5. ❌ 涉及敏感凭据、持久卷、需要 seal 的资源

### 10.4 与 GitOps 长期共存的必要前提

1. **资源 owner 表硬约束永远生效** — 绝不允许任何代码路径绕过 `reserveResourceOwners` / `DeactivateResourceOwnersByRequestID`
2. **命名空间前缀物理隔离** — direct 只能在 `ao-direct-*`；代码里 `ensureDirectNamespace` 校验；GitOps 只能在 `ao-gitops-*`（由 Release CR 模板硬编码）
3. **`managed-by` label 强制区分** — 任何人肉 kubectl apply 落到这些 namespace 之前，必须先读 label 才能决定归属；review/on-call SOP 要明确
4. **direct kubeconfig 严格 scope** — 不得使用 cluster-admin；SSAR 探测作为 ClusterTarget 准入硬条件
5. **审计全路径覆盖** — UI / Agent / Scheduler 三条触发路径都要进 `sys_audit_log`；Phase X-1 必须先做
6. **reconcile 能力** — Phase D-5 的孤儿文件扫描要能跑起来，持续发现 "Git 里没有 但 DB 里 active" 或 "Git 里有 但 DB 里缺" 的异常
7. **双模式路由在 UI / Hermes 层显式化** — 前端表单必须让用户明确选 mode；Hermes skill 必须在确认卡中打印出 mode；不得默默选 direct

### 10.5 推荐长期演进路线

- **T+0（本计划）**：P0/P1/P1/P2 MVP 跑通
- **T+1**：引入 Prometheus metrics（`deploy_request_total`、`deploy_execution_duration_seconds` 等）；Grafana 面板
- **T+2**：ArgoCD Application 状态回写到 AutoOps（补充 `request_status=synced` 语义）
- **T+3**：多 tenant Agent Token（按 Hermes 实例 / 业务线）
- **T+4**：crypto 加密 `deploy_cluster_target.direct_kubeconfig_ref` 引用的凭据
- **T+5**：direct mode 逐步收缩 → 只保留 "试用沙箱" 场景；正式业务全迁 GitOps

---

## Inspiration Credits

本轮结构化方案未调用 inspiration 角色（理由：架构/状态机任务对创意价值有限；见 Appendix A3）。无 adopted / adapted 记录。

---

## Review Summary

Reviewer scoring 将在 Phase 4（下一步）由 `reviewer`（codex）通过 `/ask` 执行。本文档 v1 未经 reviewer 评分。

---

## Appendix

### A1. Clarification Summary

```
Readiness Score: 87/100

Dimensions:
- Problem Clarity: 27/30 ✓
- Functional Scope: 23/25 ✓  (all 4 streams P0+P1+P1+P2 in scope)
- Success Criteria: 14/20 ⚠  (MVP 跑通 + 手动验收脚本; 非 CI 闭环)
- Constraints: 14/15 ✓
- Priority/MVP: 9/10 ✓

Assumptions & Gaps:
- Success: 本计划以 "MVP 跑通" 为完成线；持续回归由后续 task 补
- Constraints: 钉钉通讯录权限未开通，以"手工补录 dingtalk_user_id"作为 MVP 降级路径
```

### A2. 代码调研摘要（源码已核实）

- AutoOps deploy 模块：5 models、DAO 18 methods、controller 21 methods、service 7 files、scheduler 2 files，均与 handoff 一致
- pukka-gitops：ArgoCD App-of-Apps + `apps/autoops-managed-releases/` 脚手架，Release CR schema 只在 Helm 模板里隐式约束
- 关键缺口：`chatContextJson` 仅写入未读取；deploy routes 无 AuditMiddleware；autoops-managed-releases 无 README；DingTalk robot 未抽象到 pkg

### A3. 未采纳的替代方案

| 方案 | 为何未采纳 |
|------|----------|
| 把 "结果回群" 做成 ArgoCD Notification Controller | 依赖 ArgoCD 扩展 + 无法覆盖 direct mode；失去 AutoOps 聚合视角 |
| 在 Hermes 里直接写 kubectl / git | 违反 "Hermes 不碰 K8s/Git" 硬约束；授权失控 |
| 把 `chat_context_json` 拆成专表 `deploy_request_chat` | 当前字段数量少、查询频次低，拆表反而增加 join；字段内结构化足够 |
| direct mode 默认长生命周期（无 TTL）| 违反"direct 只做短生命周期"治理 |
| 每集群一套机器人 webhook（Phase B 就上）| MVP 增加复杂度；先全局一个，Q2 在 Phase D 之后再评估 |
| GitOps 下线通过把 CR 置 `active=false` 而非删文件 | 需改 autoops-managed-releases Helm 模板；ArgoCD reconcile 语义更复杂；MVP 选最朴素路径 |

### A4. Acceptance Criteria（端到端）

- [ ] 钉钉 APP 能收到由 AutoOps 发起的真实 OA 审批单（非占位）
- [ ] 审批通过后，AutoOps 自动触发 direct 或 GitOps 执行（`request_status=succeeded`）
- [ ] 执行结束后，钉钉群聊收到一条含 requestNo / mode / namespace / 结果的消息
- [ ] `deploy_notification` 表落一条 `status=sent`
- [ ] 页面「通知记录」对话框可展示该条消息
- [ ] Hermes skill 能解析 "部署 nginx 到 pukka 测试" 并调 AutoOps，返回 requestNo
- [ ] GitOps 模式的 release 可通过 `POST /rollback` 被删除（Git 文件消失 + K8s 资源被 ArgoCD 回收）
- [ ] `sys_audit_log` 里能查到上述所有操作的条目（UI / Agent 两类 source 都存在）
- [ ] `go test ./...` 全绿
- [ ] pukka-gitops 仓库 `apps/autoops-managed-releases/README.md` 存在

---

# DingTalk → AutoOps 构建部署 v1.1 规划（Direct 模式重定位）

更新时间：2026-04-28 Asia/Shanghai
模式：ralplan 共识规划（基于 v1 已完成 P1+P2+P3.A 的再评估 + 新方向调整）
范围标识：dingtalk-build-deploy-v1.1（dev/test 双环境，新项目，Direct 部署链路）

---

## 0. Context（为什么再规划）

上一轮 ralplan 输出了 `.omx/plans/plan-dingtalk-build-deploy-v1-consensus.md`，已落地 P1（4 篇文档 + e2e-checklist）+ P2（envType 校验、jenkins_env 清理、buildParams UI、AgentOutboxEvent 全链路）+ P3.A（E2E checklist），代码 go test/build/npm build 全过。

本轮再规划的两个驱动：

1. **方向调整（用户最新指令）**：开发/测试环境的新服务部署**不再走 pukka-gitops 仓库 + ArgoCD**，改为 **AutoOps 直接调 K8s API 部署 Deployment**（Direct 模式），并通过 AutoOps 前端 UI 触发。
2. **冷启动评审发现的 5 个真实阻塞**：Agent Token 无 UI/无文档、ClusterTarget envType 无 UI、`checkClusterTargetEnvType` 大小写敏感、outbox 写入吞错误、java-demo Jenkinsfile 修复未明确序位。

**预期结果**：Hermes/Agent 路径默认走 Direct → AutoOps 直连 K8s 部署到 dev/test namespace；UI 路径已存在（K8sReleaseCenter.vue），补齐 Direct 模式后端真实功能缺失即可。GitOps 代码保留但标记 deprecated，不删除（删除引入新 bug 的概率高于保留成本）。

---

## 1. RALPLAN-DR 摘要

### 1.1 Principles（5 条）

1. **Direct 是 v1.1 的主路径，GitOps 保留但不再是默认**：Agent/Hermes 链路硬编码切换为 Direct；UI 仍保留 GitOps 选项以兼容存量。
2. **AutoOps 直连 K8s 用现有 client-go 实现**：`directExecutor.go`/`directManifest.go` 已存在且真实可用，不重新发明轮子；只补齐已知的 3 处功能缺失。
3. **不删既有 GitOps 代码**：`gitopsWriter.go` 等保留并继续过 CI；文档明示 deprecated。
4. **Hermes 仍是单轮 NLU + ID 透传**：v1.1 不在 Hermes 侧增加状态、不做多轮对话；所有部署语义仍由 AutoOps Profile + ClusterTarget 决定。
5. **故障定位先看真实错误**：Direct 部署失败先看 K8s API 返回 / Pod events，再看 AutoOps 日志，最后才考虑 RBAC / kubeconfig。

### 1.2 Decision Drivers（Top 3）

1. **真实跑通 Direct E2E**：用户最痛点是"用 AutoOps UI 就能部署到 dev/test，不靠 ArgoCD"。
2. **不引入新模型/新表**：Profile/ClusterTarget/PipelineRun 都已存在；只在硬编码点切换 Mode、修补 Direct 真实缺失。
3. **文档闭环**：v1 的 4 篇文档都假设 GitOps 链路，必须修订或附 v1.1 增量说明，否则 onboarding/checklist 会误导。

### 1.3 Viable Options（含失效证据）

| 选项 | 形态 | 优点 | 代价 | 适用场景 |
|---|---|---|---|---|
| **A. 切硬编码 Mode + Direct 功能补齐（推荐）** | `agentBuildDeploy.go:60` 硬编码改 Direct；补齐 EnvJSON/ResourcesJSON 注入；UI 现有，文档修订 | 最小改动；不动 Profile schema；不删既有代码 | Profile UI 暂不暴露 Mode 选择（管理员若想强制 GitOps 需 SQL） | v1.1 主路径 |
| **B. 给 Profile 加 `DeployMode` 字段 + 迁移** | AppDeployProfile 加列；UI 加选择；migrate.go 走 AutoMigrate | 干净；管理员可按 Profile 选 mode | 需要 schema 变更 + 前后端 + 测试 + 文档；引入"既有 Profile 默认值"问题 | 真有"按应用区分 Mode"需求时再做（v1.2） |
| **C. 删除 GitOps 代码，单轨 Direct** | rm `gitopsWriter.go`、删 `Mode == gitops` 分支 | 代码最简洁 | 高概率破坏现有测试 + 审批/通知流程；UI 选项要同步移除；revert 风险高 | 仅当 GitOps 已经在生产被废弃 ≥ 1 季度后 |

**选 A 的理由（含 B/C 失效证据）**：
- B 失效：当前用户需求是"开发/测试新服务都走 Direct"，没有"按应用区分"的诉求；schema 变更换不来即时收益。
- C 失效：`api/api/deploy/service/deploy.go:802-851` 的 Mode 路由还在多个流程里被引用（PipelineRun、ExecutionRecord、notifier），删除涉及 5+ 文件回归测试；与"留着 outbox 同理"——保留成本远低于删除风险。

---

## 2. 当前方案再评估（必要 / 过度 / 不足）

### 2.1 必要保留（已落地，不动）

| 能力 | 文件 | 为什么必要 |
|---|---|---|
| `validateDeployProfileRefs` envType 强校验 | `api/api/app/service/deployProfile.go` | Direct 模式下 envType 错配会把 dev 部署到 test 集群，比 GitOps 更危险 |
| `deleteProfileManagedSideEffects` 清 jenkins_env | 同上 | Profile 删除后 fallback 路径取到孤儿数据的问题与 Mode 无关 |
| AppDeployProfile + 4 篇文档 + e2e-checklist | `docs/`、`api/api/app/model/` | Profile 是 Hermes 的契约源，文档是对接基础；与 Direct/GitOps 选择无关 |
| `AgentOutboxEvent` 全链路 | `api/api/deploy/...` | 事件总线占位通道；保留成本低于删除风险（同上一轮 ADR） |
| `buildParamsJson` UI textarea | `web/src/views/app/application.vue` | Jenkins 构建参数透传，与部署模式无关 |
| Direct 模式核心实现 | `directExecutor.go`、`directManifest.go`、`directCredential.go` | 已用 k8s.io/client-go v0.34.1 真实实现，不是 stub |
| K8sReleaseCenter.vue 部署表单 | `web/src/views/K8s/K8sReleaseCenter.vue` | 已支持 mode 选择 + 镜像/副本/Service/TTL 表单 |

### 2.2 过度设计 → 砍 / 暂缓

| 方向 | 处置 | 说明 |
|---|---|---|
| `marshalProfileJSON` 加 object 校验 | **砍** | Gin BindJSON 在 `map[string]interface{}` 上已强制 object，传 array/string 会在 binding 阶段 422，重复加无收益 |
| `buildParamsJson` 前端 typeof 二次校验 | **砍** | `JSON.parse + try/catch` 已经覆盖；同样属于幻觉阻塞 |
| 给 Profile 加 `DeployMode` 字段 | **暂缓**（B 选项） | v1.1 用 A 方案足够；除非用户明确"按应用区分 Mode"再做 |
| 删除 GitOps 代码 | **暂缓**（C 选项） | revert 风险高；标 deprecated 即可，v1.2 再清理 |
| Agent Token UI 化 | **暂缓** | 单实例 token，UI 化 ≥ 2 天工作量；v1.1 仅文档化 |
| pipeline stage 5 段拆分 | **暂缓** | 当前 PipelineRun 状态足够 Direct 路径用；待 Phase B E2E 暴露需求 |

### 2.3 不足 → 必须在本轮补齐

| Gap | 影响 | 修复点 |
|---|---|---|
| Agent 链路硬编码 GitOps | Direct 主路径无法被 Hermes 触发 | A1 |
| Direct 模式 EnvJSON 未注入 container.Env | 应用启动时拿不到环境变量 | A2 |
| Direct 模式 ResourcesJSON 未读取（硬编码 25m-100m） | 资源不可调，超出硬编码 limit 即 OOM | A2 |
| `checkClusterTargetEnvType` 大小写敏感 | 运维误把 envType 写成 "Dev" 即校验失败，错误信息迷惑 | A3 |
| `notifier.go` outbox 写入 `_ =` 吞错 | DB 写失败时排错困难 | A3 |
| ClusterTarget envType 字段在 UI 无下拉框 | 撕破 onboarding "0 SQL" 承诺 | A4 |
| Agent Token 生成路径无文档 | Hermes 调用 API 的硬阻塞 | A5 |
| 4 篇文档 + checklist 都假设 GitOps 链路 | onboarding 误导管理员配 ArgoCD | A6 |
| **GitOps/ArgoCD 初始化文档遗漏**（Plan agent 揭示） | 误诊 ArgoCD 失败为 AutoOps bug | 不再适用（v1.1 不走 ArgoCD），但 v1-scope 必须明示"v1.1 不依赖 pukka-gitops" |
| java-demo Jenkinsfile 修复无硬序 | 用户先跑 smoke 会得到 Jib HTTP 失败的污染信号 | B 阶段硬序约束 |

---

## 3. 任务计划（4 阶段）

### Phase A — 代码 + 文档收束（仓库内，预计 1.5 天）

#### A1. Agent 链路切换为 Direct（核心 pivot）
- 文件：`api/api/deploy/service/agentBuildDeploy.go:60`
- 改动：`Mode: model.DeployModeGitOps` → `Mode: model.DeployModeDirect`
- 测试：`agentBuildDeploy_test.go`（新建或扩展）确认 `buildAgentDeployRequestFromProfile` 输出 Direct
- 影响：所有 Hermes 触发的部署默认走 K8s 直连；UI 路径不变

#### A2. Direct 模式补齐 EnvJSON / ResourcesJSON 注入
- 文件：`api/api/deploy/service/directManifest.go::directContainer` (line 153-172 区间)
- 改动 1：解析 `req.EnvJSON`（map[string]string）→ `container.Env []corev1.EnvVar`
- 改动 2：解析 `req.ResourcesJSON`（含 requests/limits）→ `container.Resources corev1.ResourceRequirements`，保留当前硬编码作为 fallback
- 测试：`directManifest_test.go` 添加用例：空 EnvJSON / 有 EnvJSON / 空 Resources / 自定义 Resources
- 滚动更新策略 **暂缓**（默认 Deployment 已是 RollingUpdate，仅 maxSurge/maxUnavailable 用默认值，v1.1 可接受）

#### A3. 边界硬化（2 处真改 1 处砍）
- A3.1 `api/api/app/service/deployProfile.go::checkClusterTargetEnvType` 加 `strings.ToLower` 归一化 + 测试用例（"Dev" vs "dev"、"TEST" vs "test"）
- A3.2 `api/api/deploy/service/notifier.go` 两处 `_ = dao.WriteOutboxEvent(...)` 改为：
  ```go
  if err := dao.WriteOutboxEvent(...); err != nil {
      log.Printf("[notifier] outbox write failed for requestNo=%s eventType=%s: %v", req.RequestNo, eventType, err)
  }
  ```
- A3.3（砍）`marshalProfileJSON` object 校验、前端 typeof 二次校验——不做

#### A4. ClusterTarget envType UI 编辑表单
- 文件：`web/src/views/K8s/K8sReleaseCenter.vue` 的"部署目标"Tab（不动 `k8s-clusters.vue`，因为 ClusterTarget 是部署中心概念，不是 K8s 集群概念）
- 改动：在新建/编辑 ClusterTarget 抽屉中加 `envType` 下拉框（dev/test/staging/prod）
- 同步：DirectKubeconfigRef 字段在 UI 上要能选（账户系统已存在）
- 退路：若改动 > 半天，停下，改在 onboarding 里给 SQL 模板 + curl 模板，并在 v1-scope 标注"ClusterTarget envType UI 是 v1.1 已知妥协"

#### A5. Agent Token 文档化
- 主位置：`docs/autoops-build-deploy-onboarding.md` 第 2 章末尾增加子节"Agent Bearer Token 配置"——明确说明当前阶段在 `api/config.yaml` 的 `integrations.agent.bearer_token` 配置（YAML 键使用下划线）；改后需重启 API
- 反链：`docs/dingtalk-hermes-api-contract.md` 的 Authentication 段加一行 "Token 来源见 onboarding §2.X"；`docs/e2e-checklist-java-demo.md` 第 8 项加同样反链
- 不要新建 `docs/agent-token.md`（避免文档碎片化）

#### A6. 文档去 GitOps 化（v1.1 主路径调整）
- A6.1 `docs/dingtalk-build-deploy-v1-scope.md`：
  - 目标链路图改：Harbor 扫描 → AutoOps Direct 部署到 dev/test namespace（不再写 GitOps repo）
  - 职责矩阵更新："AutoOps 管 Direct K8s 部署"；GitOps 标 deprecated（保留代码但不推荐）
  - 加段："v1.1 不依赖 pukka-gitops / ArgoCD"
- A6.2 `docs/autoops-build-deploy-onboarding.md`：
  - 删除/标注 ArgoCD 配置项；
  - ClusterTarget 章节强调 `directKubeconfigRef` 必填（指向 ConfigCenter 中预先录入的 kubeconfig 账户）
- A6.3 `docs/e2e-checklist-java-demo.md`：
  - 删 § 7（pukka-gitops ArgoCD App）
  - § 6 K8s namespace 验证保留（Direct 也需要 ns 存在）
  - § 8 Agent Token 加反链到 onboarding（A5）
  - § 9 Jenkins job 加硬序提示："必须先按 java-demo-jib-fix-instructions.md 修改 Jenkinsfile，再录入此项"
- A6.4 `docs/dingtalk-hermes-api-contract.md`：
  - "成功响应"段更新：accessUrl 在 v1.1 暂不返回（AccessInfo 结构体无此字段）；回应内容由 image/namespace/servicePort 替代
  - 错误码不变

#### A7. 验证
```bash
cd /home/kchou/Code/AutoOps/api && go build ./... && go test ./api/app/... ./api/deploy/... -count=1
cd /home/kchou/Code/AutoOps/web && npm run build
```

---

### Phase B — 真实 Direct E2E（用户主导，预计 1–2 天）

#### B0. 用户必做项 gate（在 e2e-checklist 顶部新增）
- [ ] 钉钉企业自建应用注册 + 凭据（参考 `docs/dingtalk-userid-bootstrap.md`）
- [ ] 钉钉 OA 审批模板创建（参考 `docs/dingtalk-oa-template.md`）
- [ ] 审批人钉钉 userid 抓取
- [ ] Jenkins 凭据（git ssh、harbor robot）预创建
- [ ] Harbor 项目 + robot account 预创建
- [ ] dev/test K8s RBAC：创建专用 ServiceAccount，绑定最小权限 Role（必须 allow: `create namespaces`（集群级）、`create/delete deployments.apps`、`create/delete pods`、`create/delete services`（命名空间级）；必须 deny: `create persistentvolumeclaims`、`create ingresses.networking.k8s.io`）；详见 `directCredential.go` 权限边界
- [ ] dev/test 集群 kubeconfig 录入 AutoOps ConfigCenter（账户系统）
- [ ] 钉钉群 + outgoing webhook 指向 Hermes
- [ ] Hermes 部署 + 反向调用 AutoOps（Hermes 是外部组件，不在 AutoOps 仓库交付）

#### B1. 修 java-demo Jenkinsfile（硬序：必须早于 B4）
- 用户在 `gayhub.seeingtv.com:.../java-demo` 仓库改 Jenkinsfile，应用 `-Djib.to.auth.sendCredentialsOverHttp=true`（参考 `docs/java-demo-jib-fix-instructions.md`）
- 不在 AutoOps 仓库交付物范围

#### B2. AutoOps UI 录入数据
- 系统管理 → 用户管理：填审批人钉钉 userid
- 配置中心 → 账户授权：录 Jenkins、Harbor、kubeconfig（dev/test 各一）
- 部署中心 → 部署目标（K8sReleaseCenter "部署目标"Tab）：
  - dev target：envType=dev（A4 完成后 UI 选）、directKubeconfigRef=`account:<dev-kubeconfig-id>`
  - test target：envType=test
- 应用管理 → 部署配置（Profile 抽屉）：
- dev profile：namespace=`ao-direct-java-demo`（Direct 模式要求 `ao-direct-` 前缀）、release=`java-demo`、harborProject=`library`、harborRepo=`java-demo`、jenkinsJob=`java-demo`、approver=审批人 ID
- test profile：同上 namespace=`ao-direct-java-demo`
- Validate Profile：返回 `{valid: true}` 才算通过

#### B3. 配 Agent Token + 重启
- 编辑 `api/config.yaml`：`integrations.agent.bearer_token: <随机 32 字节 base64>`
- 重启 AutoOps API
- 自验：`curl -H "Authorization: Bearer <token>" http://<api>/api/v1/integrations/agent/deploy-requests/nonexistent/status` 应返 HTTP 200 + JSON `{"code": 404, ...}`（非 HTTP 401/403）

#### B4. dev smoke
```bash
curl -X POST http://<api>/api/v1/integrations/agent/build-deploy-requests \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"requesterExternalType":"dingtalk","requesterExternalId":"<userid>","applicationCode":"java-demo","env":"dev","gitRef":"main","reason":"smoke"}'
```
观察点：
- 返回 `requestNo` + `workflowKind=build_deploy`
- 钉钉审批人收 OA 卡片 → 通过
- Jenkins build 成功（Jib HTTP 已修）
- Harbor `library/java-demo` 出现 artifact + 扫描通过
- **AutoOps 直接调 K8s API**（Direct 模式）创建 Deployment + Service 到 `ao-direct-java-demo` namespace
- Pod Running，可通过 Service 端口访问（v1.1 AccessInfo 返回 namespace/servicePort/targetPort，accessUrl 计划 v1.2 实现）
- `GET /api/v1/integrations/agent/deploy-requests/<requestNo>/events`（需 Bearer Token）返回 `execution_succeeded` 事件

#### B5. 通过 K8sReleaseCenter UI 手动 Direct 部署（验证 UI 路径）
- 不走 Hermes，直接在 UI 选 mode=Direct，填 image+namespace+replicas，提交
- 期望：与 B4 同样部署成功，Service 端口可达

#### B6. test smoke + envType 反向用例
- 同 B4 但 `env=test`
- 反向：故意把 test profile 配 dev target → 期望 P2.A envType 校验报错
- 反向：故意把 envType 写 "Dev" → 期望 A3.1 大小写归一化通过（不应报错）

#### B7. Findings 收集
- 所有问题分类记录到 `docs/e2e-findings-2026-04-28.md`，按"阻塞 / 错误信息可读 / UX 抖动"分级

---

### Phase C — 收敛硬化（条件触发，按 B7 findings）

- C1. 阻塞类 fix 优先（不预承诺范围）
- C2. 错误信息可读性（钉钉群回写文案、approval 拒绝原因、K8s API 报错的中文化）
- C3.（条件）pipeline stage 5 段拆分——仅当 B7 暴露"状态过粗看不懂卡在哪"
- C4.（条件）滚动更新策略字段化——仅当 B7 出现"部署中断流量"问题
- C5.（条件）Direct 模式补 ConfigMap/Secret/Ingress 支持——仅当 B7 出现真实需求

---

### Phase D — 不做项（显式声明）

- D1. **不删** GitOps 代码（`gitopsWriter.go` 等保留，UI 仍可选）
- D2. **不做** Profile.DeployMode 字段（v1.1 用 A1 硬编码足够）
- D3. **不做** Agent Token UI（v1.1 文档化，v1.2 再考虑）
- D4. **不做** Harbor HTTPS 切换（jib-fix workaround 即可，2 周技术债到期再启）
- D5. **不做** AgentOutboxEvent revert（保留作为 alpha 通道；v1-scope 标 alpha + may change）

---

## 4. 必须遵守的工作约束

1. **Git 范围**：本轮仅触动 §3 列出的文件 + 4 篇文档 + 1 个 e2e-findings。其他历史 dirty/untracked（`docker/prometheus/**`、`api/scripts/README_FIX.md` 等）**一律不动**。
2. **测试基线（每次提交前）**：
   ```bash
   cd /home/kchou/Code/AutoOps/api && go build ./... && go test ./api/app/... ./api/deploy/... -count=1
   cd /home/kchou/Code/AutoOps/web && npm run build
   ```
3. **不绕过 hook**：禁止 `--no-verify`；hook 失败 → 修代码不修 hook。
4. **Direct 失败先看 K8s API/Pod events**：再看 AutoOps 日志，最后才考虑 RBAC/kubeconfig。注意：kubeconfig 权限校验要求 PVC/Ingress create 必须 deny（安全边界），使用 cluster-admin 反而会导致校验失败。
5. **Hermes 侧零状态**：所有部署配置/状态/审批/记录都在 AutoOps；Hermes 只传文本 + ID。

---

## 5. Critical Files

### Phase A 改动
- `api/api/deploy/service/agentBuildDeploy.go`（A1 硬编码切 Direct）
- `api/api/deploy/service/directManifest.go`（A2 EnvJSON/ResourcesJSON 注入）
- `api/api/app/service/deployProfile.go`（A3.1 ToLower）
- `api/api/deploy/service/notifier.go`（A3.2 outbox warning 日志）
- `web/src/views/K8s/K8sReleaseCenter.vue`（A4 ClusterTarget envType 表单）
- `docs/autoops-build-deploy-onboarding.md`（A5 + A6.2）
- `docs/dingtalk-hermes-api-contract.md`（A5 反链 + A6.4）
- `docs/e2e-checklist-java-demo.md`（A5 反链 + A6.3 + B0 用户必做项 gate）
- `docs/dingtalk-build-deploy-v1-scope.md`（A6.1 去 GitOps）

### Phase A 新建测试
- `api/api/deploy/service/agentBuildDeploy_test.go`（如不存在）
- `api/api/deploy/service/directManifest_test.go`（扩展或新建）
- `api/api/app/service/deployProfile_test.go` 已存在 → 加 ToLower 用例

### Phase B 不在仓库内（用户主导）
- 外部 `gayhub.seeingtv.com:.../java-demo/Jenkinsfile`
- `api/config.yaml` 本地修改（不入库）

### 复用既有实现（不动）
- `api/api/deploy/service/directExecutor.go`（已用 client-go 真实实现）
- `api/api/deploy/service/directCredential.go`（kubeconfig 解析已就绪）
- `api/api/deploy/model/deploy.go`（DeployMode 常量、ClusterTarget.DirectKubeconfigRef）
- `web/src/views/K8s/K8sReleaseCenter.vue` 现有 mode 选择器和部署表单

---

## 6. Verification

### Phase A 完成判定
- [ ] `go test ./api/app/... ./api/deploy/... -count=1` 全过
- [ ] `go build ./...` 无 error
- [ ] `npm run build` 无 error
- [ ] 新建测试覆盖：Agent 路径输出 Direct mode、EnvJSON 注入到 container.Env、ResourcesJSON 解析、ToLower 大小写归一化
- [ ] K8sReleaseCenter ClusterTarget 抽屉能选 envType=dev/test 并保存

### Phase B 完成判定
- [ ] 用户在 e2e-findings-2026-04-28.md 记录至少 dev + test 两次完整 E2E 跑通日志
- [ ] dev smoke 全链路成功：审批 → Jenkins 构建 → Harbor 扫描 → Direct 部署 → Pod Running → Service 端口可达
- [ ] envType 反向用例报错文案为"部署目标与环境不匹配"
- [ ] 大小写反向用例（envType="Dev" 配 profile.Env="dev"）通过校验

### 失败兜底
- A4 envType UI 工作量超半天 → 退路：仅出 SQL/curl 模板，文档标注已知妥协
- A2 Direct 模式 ResourcesJSON 解析复杂度超预期 → 仅做 EnvJSON，Resources 暂保硬编码 + 文档备注

---

## 7. ADR（决策记录）

- **Decision**：v1.1 把 Agent 链路默认部署模式从 GitOps 切到 Direct；保留 GitOps 代码但标 deprecated。前端 K8sReleaseCenter 的 mode 选择器保留双轨。
- **Drivers**：用户最新指令（不依赖 pukka-gitops/ArgoCD）；Direct 实现已就绪（不是 stub）；前端表单已存在；切换最小改动是改硬编码一行 + 补齐 Direct 真实功能缺失。
- **Alternatives considered**：
  - B（Profile 加 DeployMode 字段）— 未来需要"按应用区分"时再做
  - C（删 GitOps 代码）— revert 风险 > 保留成本
- **Why chosen A**：最小改动覆盖用户需求；保留可逆性（如果未来要切回 GitOps 只需改回硬编码一行）；不影响 UI 路径已有的双轨支持。
- **Consequences**：
  - + Hermes 触发的部署不再依赖外部 ArgoCD/pukka-gitops，链路缩短
  - + Direct 模式真实功能补齐后，应用可读环境变量、自定义资源
  - + 前端 UI 路径与 Agent 路径都走同一套 Direct 实现
  - − GitOps 代码进入"保留但不推荐"状态，需文档维护其 deprecated 状态
  - − Profile 暂无 Mode 字段，强制走 GitOps 需要 SQL（v1.1 不暴露此能力）
- **Follow-ups**：
  - v1.2 候选：Profile.DeployMode 字段化、Agent Token UI、Direct 模式 ConfigMap/Secret/Ingress 支持
  - 监控点：B7 findings 中如果出现 Direct 模式独有问题（如滚动更新中断流量、kubeconfig 过期），优先进入 C 阶段

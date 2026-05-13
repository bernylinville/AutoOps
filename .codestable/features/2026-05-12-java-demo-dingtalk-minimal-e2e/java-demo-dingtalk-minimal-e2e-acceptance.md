# java-demo DingTalk 最小 E2E 验收报告

> 阶段：阶段 3（验收闭环）
> 验收日期：2026-05-13
> 关联方案 doc：`.codestable/features/2026-05-12-java-demo-dingtalk-minimal-e2e/java-demo-dingtalk-minimal-e2e-design.md`

## 1. 接口契约核对

对照方案第 2.1 节名词层核查。

**接口示例逐项核对**：

- [x] Agent 自动接入请求使用 `POST /api/v1/integrations/agent/project-onboard-build-deploy`。代码入口存在，路由锚点为 `api/router/deploy/deploy.go:47`。
- [x] 请求 DTO 包含 `gitRepoUrl`、`env`、`gitRef`、`exposureMode`、`chatContext`。代码锚点为 `api/api/deploy/model/deploy.go:332`、`api/api/deploy/model/deploy.go:341`、`api/api/deploy/model/deploy.go:343`。
- [x] Hermes 请求体使用 snake_case `chat_id` / `at_user_ids`，AutoOps 通知结构读取同名字段。代码锚点为 `api/api/deploy/service/notifier.go:17`、`api/api/deploy/service/notifier.go:19`、`api/api/deploy/service/notifier.go:21`。
- [x] Agent 请求不提交 `image`、`namespace`、`clusterTargetId`、`jenkinsJobName`、`harborProject` 等 Profile 字段。最终请求 `DR20260513135942.298666965` 由 AutoOps Profile 写入这些字段，证据见 evidence 第 6.4 节。

**名词层「现状 → 变化」逐项核对**：

- [x] Agent 路由存在并被真实使用：`DR20260513135942.298666965` 来源为 `agent`，`workflowKind=build_deploy`，`mode=direct`。
- [x] Git URL 自动接入复用 Application / Profile：最终 namespace 为 `ao-direct-java-demo-test`，release 为 `java-demo`。
- [x] Direct NodePort 输出存在：`Service/java-demo type=NodePort nodePort=32580`。
- [x] Jenkins 构建结果解析存在：`jenkinsBuildNumber=72`，`artifactTag=main-20260513-152618-c5bdf39`，`finalImageRef=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39`。
- [x] 钉钉回群结构存在：`deploy_notification` 中 `executed/sent`，payload 包含 request、状态、镜像、namespace、NodePort 和访问地址。

**流程图核对**：

- [x] DingTalk 群用户 → Hermes：真实入口来自钉钉群用户原话，Hermes 返回 requestNo。
- [x] Hermes → AutoOps：Agent API 创建 `DR20260513135942.298666965`。
- [x] AutoOps → OA：`approvalStatus=approved`，审批实例状态已回写。
- [x] Scheduler → Jenkins：`PipelineScheduler` 触发 `java-demo-build #72`。
- [x] Jenkins → Harbor：最终镜像被 Kubernetes 拉取，pod image digest 为 `sha256:213d38af25c5f6b2d44163f6a7594322f34da71c4670d9230b0e50f928100e4b`。
- [x] AutoOps → Kubernetes：`Deployment/java-demo` 更新并 ready。
- [x] AutoOps → Bot：`deploy_notification executed/sent`。

## 2. 行为与决策核对

对照方案第 1 节和第 2.2 节核查。

**需求摘要逐项验证**：

- [x] 钉钉群用户原话触发成功：最终 requestNo 为 `DR20260513135942.298666965`。
- [x] 创建申请后先进入审批：`approvalStatus` 从 pending 进入 approved 后，Pipeline 才执行。
- [x] Jenkins 构建成功：`java-demo-build #72`，`result=SUCCESS`。
- [x] Harbor 镜像可用：Kubernetes 实际拉取 Harbor digest。
- [x] Direct Kubernetes 部署成功：Deployment ready `1/1`，Service 为 NodePort。
- [x] deploy_bot 回群成功：`deploy_notification` 中 `executed/sent`。

**明确不做逐项核对**：

- [x] 未支持生产环境、`prod`、正式发布或复杂审批矩阵。
- [x] 未新增 Gateway / Ingress / MetalLB 业务应用暴露模式。Harbor HTTPRoute timeout 修复只作用于平台 Harbor 上传路径，不改变 `java-demo` 业务暴露模式。
- [x] Hermes 未追问或提交容器镜像、namespace、ClusterTarget、Jenkins job、Harbor project / repository。
- [x] 未写 `pukka-gitops` 业务 release 文件；未修改 `apps/autoops-managed-releases/releases/`。
- [x] `.codestable` 证据未记录真实 token、密码、PAT、webhook、kubeconfig 或 Secret value。
- [x] 未实现 `AUTOOPS_IMAGE` 优先解析，仍留给 `repo-jenkinsfile-build-contract`。

**关键决策落地**：

- [x] 先跑最小 E2E，而不是先做通用平台改造。证据包记录了真实失败、修复和最终成功。
- [x] 真实触发从钉钉群进入，裸 curl 仅用于定位和验证。
- [x] Hermes 只调 AutoOps，未直接操作 Jenkins、Harbor、Kubernetes 或 GitOps。
- [x] 战术修正只处理阻断项：Hermes `chatContext`、Harbor Gateway timeout、AutoOps deployer RBAC。
- [x] 证据优先于推断：每个阶段都有数据库、Jenkins、Kubernetes 或通知记录。

**编排层「现状 → 变化」逐项核对**：

- [x] Preflight 已记录 AutoOps、Jenkins、Harbor、Kubernetes、Hermes、ClusterTarget 和 Secret key 存在性。
- [x] Hermes `chatContext` 已从 camelCase 修为 snake_case。
- [x] Jenkins → Harbor 504 已通过 Harbor HTTPRoute timeout 修复，提交为 `03ff8ef`。
- [x] Direct redeploy RBAC 已通过 `deployerAccess` 修复，提交为 `876c0f0`、`cecd2cc`。
- [x] 真实触发后按状态推进，没有绕过审批。
- [x] E2E 证据包已追加最终成功证据。

**流程级约束核对**：

- [x] 错误语义可定位：失败时记录了 deploy stage forbidden 错误和 Jenkins → Harbor 504 原因。
- [x] 幂等性符合预期：复用 `Application(code=java-demo)` 和 test Profile，新建 request `DR20260513135942.298666965`。
- [x] 顺序约束符合预期：审批通过后触发 Jenkins，构建成功后部署 Kubernetes。
- [x] 扩展点位置未越界：`AUTOOPS_IMAGE`、固定 `nodeport_access_host`、Hermes skill 版本化副本未并入本 feature。
- [x] 可观测点齐全：requestNo、审批、Jenkins build、镜像、Kubernetes 资源、NodePort、通知记录均已写入 evidence。
- [x] 安全约束符合预期：证据只记录 key 存在性和资源状态，不记录 Secret value。

**挂载点反向核对**：

- [x] opsclaw Hermes skill / 脚本：只改 `chatContext` 字段名，证据见 evidence 第 2 节。
- [x] java-demo Jenkinsfile：当前已有 `IMAGE_TAG` 输出，本阶段未扩大 Jenkinsfile 契约。
- [x] pukka-gitops 运行配置：只改 Harbor HTTPRoute timeout 和 AutoOps deployerAccess RBAC，已补入 design 第 2.3 节。
- [x] feature-local 证据文件：新增最终成功证据，不记录凭据。
- [x] 反向 grep：本 feature 的外部配置改动均落在 design 第 2.3 节列出的挂载点内。
- [x] 拔除沙盘推演：移除 Hermes 字段修正会影响回群上下文；移除 Harbor timeout 会恢复 push 504 风险；移除 deployerAccess 会导致 Direct redeploy 失败；移除 evidence 不影响运行时能力。

## 3. 验收场景核对

对照方案第 3 节关键场景清单。

- [x] **S1 Preflight 正常**
  - 证据来源：手工只读检查、AutoOps DB、Kubernetes、opsclaw SSH、pukka-gitops / kubespray 文件。
  - 结果：通过。证据见 evidence 第 1 节，未记录 Secret 值。

- [x] **S2 意图解析正常**
  - 证据来源：Hermes dry-run、真实 request 的 `chatContextJson`。
  - 结果：通过。请求包含 `gitRepoUrl`、`env=test`、`gitRef=main`、`exposureMode=nodeport` 和 snake_case `chatContext`。

- [x] **S3 创建申请正常**
  - 证据来源：AutoOps DB。
  - 结果：通过。`DR20260513135942.298666965` 为 `source=agent`、`workflowKind=build_deploy`、`mode=direct`、`namespace=ao-direct-java-demo-test`。

- [x] **S4 审批正常**
  - 证据来源：AutoOps DB。
  - 结果：通过。最终 `approvalStatus=approved`，审批通过后 Pipeline 执行。

- [x] **S5 Jenkins 正常**
  - 证据来源：AutoOps PipelineRun、Jenkins build number。
  - 结果：通过。`java-demo-build #72` 成功，解析出 `main-20260513-152618-c5bdf39`。

- [x] **S6 Harbor 正常**
  - 证据来源：运行中 Pod 的 imageID。
  - 结果：通过。Kubernetes 成功拉取 `10.0.17.205:80/java-demo/java-demo@sha256:213d38af25c5f6b2d44163f6a7594322f34da71c4670d9230b0e50f928100e4b`。

- [x] **S7 Direct 部署正常**
  - 证据来源：`kubectl get deploy,svc,pod`。
  - 结果：通过。`Deployment/java-demo ready=1/1`，`Service/java-demo type=NodePort`。

- [x] **S8 对外访问正常**
  - 证据来源：6 个节点 IP 的 HTTP 访问。
  - 结果：通过。`10.0.17.40~45:32580` 均返回 HTTP 200 和 `AutoOps Java demo is running`。

- [x] **S9 回群正常**
  - 证据来源：`deploy_notification`。
  - 结果：通过。`executed/sent`，payload 包含申请号、状态、镜像、namespace、Service 类型、NodePort 和访问地址。

- [x] **S10 错误路径可定位**
  - 证据来源：evidence 第 4、5、6 节。
  - 结果：通过。已记录 Jenkins → Harbor 504 和 deployer RBAC forbidden，并关联修复提交。

**反向核对项**：

- [x] 未出现 Hermes 直接执行 `kubectl apply`、直接调 Jenkins、直接推 Harbor 或写 `pukka-gitops` release 文件。
- [x] Agent 请求 JSON 未包含应由 Profile 补齐的部署字段。
- [x] 证据包未包含凭据值。
- [x] 未修改 AutoOps Go 控制面来实现 Gateway / Ingress / `nodeport_access_host`。
- [x] 未把 `AUTOOPS_IMAGE` 优先解析作为完成前置。

## 4. 术语一致性

对照方案第 0 节和第 2.1 节核对。

- 「最小 E2E」：用于 feature design / evidence / acceptance，语义均指钉钉到回群的最窄真实流程。
- 「用户原话」：只用于本次样例，不扩展到其他 repo 或生产环境。
- 「Agent 自动接入请求」：代码落点为 `CreateAgentProjectOnboardBuildDeployRequest`，没有混用为普通 `deploy-requests`。
- 「Direct NodePort」：实际资源为 `Service(type=NodePort)`，未混成 Gateway / Ingress。
- 「战术修正」：仅覆盖 Hermes 字段、Harbor timeout、deployer RBAC 等最小阻断修复。
- 「证据包」：只写入 feature-local evidence，不进入运行时代码。
- 防冲突：`rg` 检查到的 `chat_id` / `at_user_ids` 与 AutoOps `ChatContext` 一致；未继续使用 `chatId` / `atUserIds` 作为 AutoOps 通知契约。

## 5. 架构归并

对照方案第 4 节，已实际更新架构文档。

- [x] 架构 doc：`.codestable/architecture/deploy-dingtalk-autoops-e2e.md`
  - 归并内容：DingTalk → Hermes → AutoOps Agent API → OA → Pipeline → Jenkins → Harbor → Direct Kubernetes → deploy_bot 的当前结构。
  - 已写入结构与职责、数据与状态、运行时配置、已知约束和验证记录。

- [x] 架构总入口：`.codestable/architecture/ARCHITECTURE.md`
  - 归并内容：新增 Deploy 集成现状文档索引，补充关键决定和 Secret 记录边界。

- [x] 名词归并
  - 已写入 Agent API、Project Onboarding、Direct NodePort、deployerAccess、deploy_bot 的职责边界。

- [x] 动词骨架归并
  - 已写入完整调用流程和状态对象。

- [x] 流程级约束归并
  - 已写入 Hermes 不直连下游、NodePort URL 来源、deployerAccess 与 clusterAccess 分离、失败 PipelineRun 无 retry/reset API 等约束。

归并后，未读过 feature design 的读者可以从 architecture 入口找到该能力的当前形态和约束。

## 6. requirement 回写

方案 frontmatter 的 `requirement` 为空，但本 feature 已形成用户可感能力：钉钉群发起测试环境部署并收到审批、构建、部署和回群结果。

- [x] 已执行 requirement backfill。
- [x] 新增 `.codestable/requirements/dingtalk-autoops-deploy-e2e.md`，`status: current`。
- [x] 新增 `.codestable/requirements/VISION.md`，将该能力列入 Current。
- [x] requirement 不写实现细节，只描述用户故事、痛点、解法和边界。

## 7. roadmap 回写

方案 frontmatter：

```yaml
roadmap: dingtalk-autoops-deploy-e2e
roadmap_item: java-demo-dingtalk-minimal-e2e
```

已完成回写：

- [x] `.codestable/roadmap/dingtalk-autoops-deploy-e2e/dingtalk-autoops-deploy-e2e-items.yaml`
  - `slug: java-demo-dingtalk-minimal-e2e`
  - `status: done`
  - `feature: 2026-05-12-java-demo-dingtalk-minimal-e2e`

- [x] `.codestable/roadmap/dingtalk-autoops-deploy-e2e/dingtalk-autoops-deploy-e2e-roadmap.md`
  - frontmatter `last_reviewed: 2026-05-13`
  - `related_requirements` 增加 `dingtalk-autoops-deploy-e2e`
  - `related_architecture` 增加 `deploy-dingtalk-autoops-e2e`
  - 第 5 节子 feature 清单对应条目状态改为 `done`
  - 第 9 节变更日志追加 2026-05-13 记录

- [x] YAML 校验通过：`validate-yaml.py --file ...items.yaml`。

## 8. attention.md 候选盘点

本 feature 暴露 1 条值得加入 `.codestable/attention.md` 的候选：

- 候选 1：AutoOps 当前没有失败 build-deploy 的专用 retry/reset API。若手工复用同一个 `deploy_pipeline_run` 重置，必须先处理旧 `deploy_pipeline_stage_record`，否则会命中唯一索引 `idx_pipeline_stage`。

本节只登记候选，未直接写入 `attention.md`。

## 9. 遗留

### 后续优化点

- `repo-jenkinsfile-build-contract`：固化 `AUTOOPS_IMAGE` / `AUTOOPS_IMAGE_TAG` 输出协议，并让 AutoOps 优先解析。
- `nodeport-access-feedback`：确认是否需要固定外部访问地址；当前实际 URL 来自节点 IP + NodePort。
- `autoops-production-config-readiness`：继续盘点生产 Helm runtime config、Secret key、ClusterTarget、Jenkins、Harbor 和 deploy_bot 配置。
- `hermes-deploy-skill-routing`：补齐 Hermes skill 版本化副本和路由治理。
- AutoOps deploy pipeline：后续可增加失败流水线 retry/reset API，避免手工 SQL 重置。

### 已知限制

- 当前只验证 `java-demo` 测试环境，不代表生产环境发布能力已经验收。
- 当前对外访问是 NodePort，不是固定域名或 Gateway。
- 当前构建仍依赖 `java-demo` 仓库 Jenkinsfile 的旧格式可解析输出。
- 本次存在对 `/home/kchou/Code/pukka-gitops` 的平台配置修复；这些提交已经推送，但 AutoOps 主仓仍有其他无关工作区改动。

### 实现阶段顺手发现

- `api/api/deploy/service/deploy.go`、`pipeline.go`、`agentBuildDeploy.go` 偏胖。若继续扩展 Agent / Pipeline 能力，应单独走 `cs-refactor`。
- AutoOps 仓库内缺少文档提到的 Hermes skill 副本路径，建议后续由 `hermes-deploy-skill-routing` 处理。
- 失败 PipelineRun 重置会撞旧 stage 唯一索引，建议形成运维注意事项或实现正式 retry API。

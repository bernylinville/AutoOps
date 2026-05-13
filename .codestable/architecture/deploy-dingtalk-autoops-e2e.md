---
doc_type: architecture
slug: deploy-dingtalk-autoops-e2e
status: current
last_reviewed: 2026-05-13
implements: [dingtalk-autoops-deploy-e2e]
tags: [deploy, dingtalk, hermes, jenkins, harbor, kubernetes]
---

# 钉钉触发 AutoOps 构建部署流程

## 1. 范围

本文记录当前已经跑通的测试环境部署流程：钉钉群用户发起部署请求，Hermes 将意图转成 AutoOps Agent 请求，AutoOps 创建审批和流水线，审批通过后触发 Jenkins 构建并推送 Harbor，最后由 AutoOps Direct 模式更新 Kubernetes `Deployment` 和 `Service(type=NodePort)`，并通过部署机器人回群。

该文档只记录已经在 `java-demo` 测试环境验证过的现状，不描述生产环境推广方案，也不描述未来通用 Jenkinsfile 契约。

## 2. 结构与职责

```text
DingTalk 群用户
  -> opsclaw Hermes skill
  -> AutoOps Agent API
  -> DingTalk OA 审批
  -> PipelineScheduler
  -> Jenkins java-demo-build
  -> Harbor 10.0.17.205
  -> Kubernetes Direct Deployment / NodePort Service
  -> deploy_bot 钉钉通知
```

| 组件 | 当前职责 | 代码或配置锚点 |
| --- | --- | --- |
| Hermes | 解析 Git URL、环境、分支和「对外访问」意图，只调用 AutoOps Agent API。 | 运行实例在 opsclaw `~/.hermes`；本仓记录见 feature evidence。 |
| AutoOps Agent API | 接收 Agent 请求，自动创建或复用 Application / Profile，创建 DeployRequest 和 PipelineRun。 | `api/router/deploy/deploy.go:47`；`api/api/deploy/service/agentBuildDeploy.go:52` |
| Project Onboarding 配置 | 限制 Git host、Jenkins job、Harbor project、ClusterTarget、默认端口和审批人。 | `api/api/deploy/service/agentBuildDeploy.go:214`；`/home/kchou/Code/pukka-gitops/apps/autoops/values.yaml:185` |
| DingTalk OA | 审批通过后由 AutoOps 同步任务回写 `approvalStatus=approved`。 | `api/api/deploy/service/deploy.go:1111`；`api/api/deploy/service/deploy.go:1234` |
| PipelineScheduler | 领取已审批且待执行的 PipelineRun，按 build → scan → deploy → notify 执行。 | `api/api/deploy/service/pipeline.go:123`；`api/api/deploy/service/pipeline.go:155` |
| Jenkins / Harbor | Jenkins 构建镜像并推送 Harbor，AutoOps 从 Console Output 解析镜像结果。 | `api/api/deploy/service/pipeline.go:221`；`api/api/deploy/service/jenkinsPipeline.go:188`；`api/api/deploy/service/jenkinsPipeline.go:541` |
| Direct Kubernetes | 使用 Direct kubeconfig 应用 Deployment / Service，等待 ready，收集 NodePort URL。 | `api/api/deploy/service/directExecutor.go:53`；`api/api/deploy/service/directExecutor.go:405`；`api/api/deploy/service/directExecutor.go:487` |
| deploy_bot | 读取 `chatContext` 和执行结果，发送钉钉 Markdown 通知。 | `api/api/deploy/service/notifier.go:17`；`api/api/deploy/service/notifier.go:44`；`api/api/deploy/service/notifier.go:151` |

## 3. 数据与状态

一次成功的 build-deploy 至少跨越以下状态对象：

- `deploy_request`：保存申请号、来源、审批状态、执行状态、镜像、namespace、releaseName 和 `chatContextJson`。
- `deploy_pipeline_run`：保存 Jenkins queue / build number、artifact tag、planned / final image ref、stage 状态。
- `deploy_pipeline_stage_record`：保存 build / deploy / notify 阶段的状态、外部 ID 和错误信息。
- `deploy_execution_record`：保存 Direct manifest 预览和实际 apply 结果。
- `deploy_notification`：保存回群通知 payload、channel、stage 和发送状态。

已验证的最终状态示例：

```text
requestNo=DR20260513135942.298666965
requestStatus=succeeded
approvalStatus=approved
executionStatus=succeeded
pipelineStatus=succeeded
jenkinsBuildNumber=72
finalImageRef=10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
namespace=ao-direct-java-demo-test
releaseName=java-demo
```

## 4. 运行时配置

AutoOps 生产部署由 `/home/kchou/Code/pukka-gitops` 管理。与本流程相关的稳定配置如下：

- AutoOps ArgoCD Application 读取 `charts/autoops` 和 `apps/autoops/values.yaml`。
- `projectOnboarding` 已启用，测试环境使用 `test_cluster_target_id` 对应的 Direct ClusterTarget。
- `skipPipelineScan: true` 当前用于绕开 Harbor scan，让最小 E2E 聚焦 build / deploy / notify。
- Harbor HTTPRoute 对 `/v2/` 等路径设置 `request: 10m` 和 `backendRequest: 10m`，避免大镜像上传在 Envoy Gateway 层超时。
- `deployerAccess` 独立于只读 `clusterAccess`，用于 Direct 部署和清理：
  - namespaces：`create/delete/get/list/watch`
  - pods/services：`create/delete/get/list/watch/update`
  - deployments：`create/delete/get/list/watch/update`
  - secrets：`create/get/list/watch/update`
  - nodes：`get/list/watch`

配置锚点：

```text
/home/kchou/Code/pukka-gitops/platform/harbor/harbor-httproute.yaml:14
/home/kchou/Code/pukka-gitops/charts/autoops/values.yaml:107
/home/kchou/Code/pukka-gitops/charts/autoops/templates/deployer-access-rbac.yaml:1
/home/kchou/Code/pukka-gitops/apps/autoops/values.yaml:65
```

## 5. 已知约束

- Hermes 不能直接调用 Jenkins、Harbor、Kubernetes 或写 GitOps release 文件；AutoOps 是唯一部署控制面。
- Agent 自动接入请求不提交 image、namespace、ClusterTarget、Jenkins job、Harbor project 这类 Profile 字段。
- `chatContext` 使用 snake_case 字段：`chat_id`、`at_user_ids`、`at_mobiles`。
- 当前 NodePort 访问地址来自 Kubernetes 节点 IP + NodePort。`nodeport_access_host` 虽然在 Helm values 中出现，但 Go 配置尚未消费，不能作为实际 URL 来源。
- Jenkinsfile 目前依赖旧格式 `IMAGE_TAG` 或完整 image ref 被 AutoOps 解析。`AUTOOPS_IMAGE` 优先解析仍属于后续 feature。
- 失败的 build-deploy 目前没有专用 retry/reset API。若复用同一个 PipelineRun 手工重置，必须处理旧 `deploy_pipeline_stage_record`，否则会命中唯一索引 `idx_pipeline_stage`。
- 文档和证据不得记录真实 token、密码、PAT、webhook、kubeconfig 或 Secret value。

## 6. 验证记录

本流程在 2026-05-13 完成一次真实 E2E 验证：

```text
DingTalk 用户原话
-> Hermes
-> AutoOps request DR20260513135942.298666965
-> DingTalk OA approved
-> Jenkins java-demo-build #72 SUCCESS
-> Harbor image 10.0.17.205:80/java-demo/java-demo:main-20260513-152618-c5bdf39
-> Deployment/java-demo ready 1/1
-> Service/java-demo NodePort 32580
-> 10.0.17.40~45:32580 HTTP 200
-> deploy_notification executed/sent
```

详细证据见：

- `.codestable/features/2026-05-12-java-demo-dingtalk-minimal-e2e/java-demo-dingtalk-minimal-e2e-evidence.md`

## 7. 变更日志

- 2026-05-13：基于 `java-demo` 钉钉最小 E2E 验收结果补充当前架构现状。

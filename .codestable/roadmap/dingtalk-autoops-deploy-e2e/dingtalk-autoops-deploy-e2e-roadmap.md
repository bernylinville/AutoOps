---
doc_type: roadmap
slug: dingtalk-autoops-deploy-e2e
status: active
created: 2026-05-12
last_reviewed: 2026-05-13
tags: [dingtalk, hermes, deploy, jenkins, harbor, kubernetes, e2e]
related_requirements: [dingtalk-autoops-deploy-e2e]
related_architecture: [ARCHITECTURE, deploy-dingtalk-autoops-e2e]
---

# DingTalk → Hermes → AutoOps 构建部署 E2E

## 1. 背景

本 roadmap 面向一次真实流程测试：在钉钉群中和 opsclaw 上的 Hermes agent 机器人对话，用户原话为：

```text
帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问
```

预期流程是：Hermes skill 将自然语言转成 AutoOps Agent API 的结构化请求；AutoOps 创建部署申请和钉钉 OA 审批；审批通过后 AutoOps 调 Jenkins 构建源码、推送镜像到 Harbor；构建成功后 AutoOps 使用 Direct 模式在 Kubernetes 中创建 `Deployment` / `Service`；最终通过部署结果机器人 webhook 回写钉钉群。

本次检查得到的结论是：**AutoOps 当前代码已经具备主流程骨架，可以支撑该目标的最小路径；但还不能直接判定生产环境一次必成**。主要原因是运行时配置、Hermes skill 字段契约、Jenkins 输出契约和 NodePort 访问地址仍需要在 E2E 前做检查或小修。

现有职责边界继续沿用：**Hermes 只理解用户意图并调用 AutoOps；AutoOps 是审批、Profile、Jenkins 调度、部署状态和 Direct Kubernetes 部署的控制面；Jenkinsfile 只负责编译、构建镜像、推 Harbor 并输出镜像结果。** 不让 Hermes 直接操作 Jenkins、Harbor、Kubernetes 或 GitOps 仓库。

## 2. 范围与明确不做

### 本 roadmap 覆盖

- 用用户原话跑通钉钉群 → Hermes → AutoOps → OA 审批 → Jenkins → Harbor → Direct Kubernetes → 钉钉回群的最小 E2E。
- 校验 AutoOps 生产部署在 `~/Code/pukka-gitops` 中的 Helm / ArgoCD / Gateway / Secret / runtime config 是否满足该流程。
- 校验 `~/Code/kubespray` 中 Pukka Kubernetes 集群网络、MetalLB、containerd、NodePort 和节点信息对该流程的影响。
- 固化 Hermes skill 到 AutoOps Agent API 的请求契约，尤其是 Git URL 自动接入、`nodeport` 暴露模式和钉钉回群上下文字段。
- 固化 repo 内 Jenkinsfile 与 AutoOps 的构建结果契约，让不同语言仓库通过自己的 Jenkinsfile 构建镜像，统一输出 AutoOps 可解析的镜像字段。
- 让 Direct 部署后的状态查询和部署机器人通知包含镜像、NodePort 和可访问 URL。
- 形成一份可重复执行的 E2E runbook 和证据清单。

### 明确不做

- 不让 Hermes 直接调用 Jenkins、Harbor、Kubernetes、ArgoCD 或写 `pukka-gitops`。
- 不让 AutoOps 自动生成或覆盖各业务仓库的 Jenkinsfile；通用能力用「Jenkins 参数 + 输出契约 + starter 模板」表达。
- 不在本 roadmap 内接入生产环境、多集群复杂审批矩阵、Gateway / Ingress / MetalLB 独立暴露模式。
- 不做 GitLab 仓库、Harbor project、Jenkins job 的全生命周期自动创建；只校验当前 `java-demo` 测试路径需要的配置。
- 不改 `.codestable/architecture/` 或 `.codestable/requirements/`，发现过时点只记为观察项。
- 不把任何真实 token、密码、PAT、kubeconfig 或 webhook URL 写入 roadmap。

## 3. 模块拆分（概设）

```text
DingTalk → Hermes → AutoOps 构建部署 E2E
├── A. Hermes 意图入口：把钉钉自然语言转成 AutoOps Agent API 请求
├── B. AutoOps Agent 控制面：项目自动接入、审批、PipelineRun 和状态查询
├── C. Jenkins / Harbor 构建契约：repo Jenkinsfile 构建镜像并输出结果
├── D. Direct Kubernetes 部署：创建 Deployment / Service / NodePort 并收集访问地址
├── E. 生产运行时配置：pukka-gitops Helm、Secret、ClusterTarget、Kubespray 集群约束
└── F. E2E 证据与回群：runbook、状态轮询、deploy_bot webhook 通知
```

### A. Hermes 意图入口

- **职责**：识别 Git URL、环境、分支和「对外访问」意图，调用 AutoOps Agent API；在创建成功、失败或用户追问进度时给出可读回复。
- **不负责**：保存部署配置、追问 namespace / image / Jenkins job / Harbor 仓库、直接操作外部系统。
- **承载的子 feature**：`java-demo-dingtalk-minimal-e2e`、`hermes-deploy-skill-routing`
- **触碰的现有代码 / 模块**：opsclaw `~/.hermes/skills/devops/deploy-via-autoops/` 与 `~/.hermes/scripts/autoops-deploy.sh`；AutoOps 仓库内当前未找到 `skills/devops/deploy-via-autoops/SKILL.md` 副本，这是观察项。

### B. AutoOps Agent 控制面

- **职责**：通过 Agent 路由接收请求，校验 Agent Token，自动创建 / 复用 Application 和 `AppDeployProfile`，创建部署申请和 PipelineRun，投递 OA 审批，暴露状态查询接口。
- **不负责**：自然语言理解、业务仓库构建逻辑、手工替用户审批。
- **承载的子 feature**：`autoops-production-config-readiness`、`java-demo-dingtalk-minimal-e2e`
- **触碰的现有代码 / 模块**：`api/router/deploy/deploy.go`、`api/api/deploy/service/agentBuildDeploy.go`、`api/api/deploy/service/deploy.go`、`api/scheduler/*`。

### C. Jenkins / Harbor 构建契约

- **职责**：AutoOps 传入标准 Jenkins 参数；repo Jenkinsfile 根据语言自行构建镜像并推送 Harbor；Jenkins Console Output 输出 AutoOps 可解析的镜像结果。
- **不负责**：审批、Direct 部署、Kubernetes 资源声明。
- **承载的子 feature**：`repo-jenkinsfile-build-contract`
- **触碰的现有代码 / 模块**：`api/api/deploy/service/pipeline.go`、`api/api/deploy/service/jenkinsPipeline.go`、外部 `java-demo` 仓库 Jenkinsfile。

### D. Direct Kubernetes 部署

- **职责**：使用 AutoOps 保存的 ClusterTarget / kubeconfig，通过 Kubernetes API 创建 namespace、`Deployment`、`Service`，等待 workload ready，并收集 NodePort、节点 IP 和访问 URL。
- **不负责**：ArgoCD 同步、GitOps release 文件写入、Gateway / Ingress 资源管理。
- **承载的子 feature**：`nodeport-access-feedback`
- **触碰的现有代码 / 模块**：`api/api/deploy/service/directManifest.go`、`api/api/deploy/service/directExecutor.go`、`api/api/deploy/service/deploy.go`、`api/api/deploy/service/notifier.go`。

### E. 生产运行时配置

- **职责**：确保 AutoOps 生产实例、Jenkins、Harbor、Kubernetes 集群、Secret 和 Helm values 的实际配置能够支撑 E2E。
- **不负责**：变更用户侧钉钉应用权限边界，或把凭据写入文档。
- **承载的子 feature**：`autoops-production-config-readiness`
- **触碰的现有代码 / 模块**：`~/Code/pukka-gitops/apps/autoops/values.yaml`、`~/Code/pukka-gitops/charts/autoops/templates/api-runtime-configmap.yaml`、`~/Code/kubespray/inventory/pukka-cluster/`。

### F. E2E 证据与回群

- **职责**：记录请求号、审批实例、Jenkins build、Harbor artifact、Kubernetes Deployment / Service、访问 URL 和钉钉群通知结果。
- **不负责**：为每个业务语言写完整构建教程。
- **承载的子 feature**：`e2e-runbook-and-evidence`
- **触碰的现有代码 / 模块**：`docs/` 或 `.codestable/features/*` 下游 feature 产物。

## 4. 模块间接口契约 / 共享协议（架构层详设）

本节是后续 feature-design 的硬约束输入。若实际实现需要改变字段名、端点或状态语义，必须先回到本 roadmap update。

### 4.1 用户原话 → Hermes 意图结构

**方向**：DingTalk 用户 → Hermes skill
**形式**：自然语言解析约定

**输入示例**：

```text
帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问
```

**解析结果**：

```json
{
  "intent": "project_onboard_build_deploy",
  "gitRepoUrl": "git@gayhub.seeingtv.com:ipaas/java-demo.git",
  "env": "test",
  "gitRef": "main",
  "exposureMode": "nodeport",
  "reason": "帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问"
}
```

**约束**：

- 只要用户提供 Git URL，优先走 `project_onboard_build_deploy`，不得退回只按 `applicationCode` 的 `build_deploy`。
- `测试环境` / `测试` / `test` 统一为 `test`；`开发环境` / `开发` / `dev` 统一为 `dev`。
- `需要能对外访问` / `对外访问` / `外网可访问` 统一为 `exposureMode=nodeport`。
- `生产` / `prod` / `正式环境` 必须明确拒绝，当前 Agent 自动接入仅支持 `dev/test`。
- 不向用户追问 `image`、`namespace`、`clusterTarget`、`Jenkins job`、`Harbor project/repository`。

### 4.2 Hermes → AutoOps Git URL 自动接入 API

**方向**：Hermes → AutoOps Agent API
**形式**：HTTP JSON

**请求**：

```http
POST /api/v1/integrations/agent/project-onboard-build-deploy
Authorization: Bearer <agent-token>
Content-Type: application/json
```

```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "<dingtalk-user-id>",
  "requesterDisplayName": "<display-name>",
  "gitRepoUrl": "git@gayhub.seeingtv.com:ipaas/java-demo.git",
  "env": "test",
  "gitRef": "main",
  "exposureMode": "nodeport",
  "reason": "帮我把 git@gayhub.seeingtv.com:ipaas/java-demo.git 的 main 分支重新部署到测试环境，需要能对外访问",
  "chatContext": {
    "provider": "dingtalk",
    "chat_id": "<dingtalk-chat-id>",
    "at_user_ids": ["<dingtalk-user-id>"],
    "sender_external_id": "<dingtalk-user-id>",
    "origin_message_id": "<message-id>",
    "extra": {
      "intent": "redeploy",
      "externalAccessRequired": true
    }
  }
}
```

**响应成功条件**：

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR...",
    "mode": "direct",
    "workflowKind": "build_deploy",
    "requestStatus": "pending_approval",
    "approvalStatus": "pending",
    "approvalDispatchStatus": "pending|dispatched",
    "namespace": "ao-direct-java-demo-test",
    "releaseName": "java-demo"
  }
}
```

**错误约定**：

- Agent 认证失败读取 `msg`：`agent integration is not configured`、`missing authorization header`、`invalid agent token`。
- 业务错误读取 `message`，且不能只看 HTTP 状态码；大多数业务错误是 HTTP 200 + JSON `code != 200`。
- 审批白名单拒绝可能返回 HTTP 403 + JSON `code: 471`。

**当前证据**：

- AutoOps 已注册 `POST /integrations/agent/project-onboard-build-deploy`，入口在 `api/router/deploy/deploy.go`。
- opsclaw 当前 skill 已能把用户原话映射到该端点，并把「对外访问」映射为 `nodeport`。
- opsclaw 当前脚本使用 `chatId` / `atUserIds`，而 AutoOps 通知结构体使用 `chat_id` / `at_user_ids`。下游 feature 必须统一为本节定义的 snake_case，避免回群通知缺少群和 @ 信息。

### 4.3 AutoOps 项目自动接入配置协议

**方向**：AutoOps runtime config → `agentBuildDeploy.go`
**形式**：YAML 配置

**配置 schema**：

```yaml
integrations:
  agent:
    bearer_token: "<from-secret>"
    project_onboarding:
      enabled: true
      allowed_git_hosts: ["gayhub.seeingtv.com"]
      shared_jenkins_job_name: "java-demo-build"
      default_business_group_id: 1
      default_business_dept_id: 1
      default_jenkins_server_id: 1
      default_harbor_server_id: 2
      default_harbor_credentials_id: "harbor-robot"
      default_harbor_project: "java-demo"
      default_approver_admin_id: 89
      dev_cluster_target_id: 1
      test_cluster_target_id: 2
      namespace_prefix: "ao-direct"
      default_service_port: 80
      default_target_port: 8080
  deploy_bot:
    provider: "dingtalk"
    enabled: true
    webhook_url: "<from-secret>"
    secret: "<from-secret>"
```

**生成的 Profile 约束**：

```text
applicationCode: repo 名派生，例如 java-demo
namespace:       <namespace_prefix>-<applicationCode>-<env>，例如 ao-direct-java-demo-test
releaseName:     <applicationCode>
mode:            direct
workflowKind:    build_deploy
resourceType:    deployment
serviceType:     exposureMode=nodeport 时为 NodePort，否则为 ClusterIP
servicePort:     default_service_port，当前 80
targetPort:      default_target_port，当前 8080
```

**约束**：

- `allowed_git_hosts` 必须包含 `gayhub.seeingtv.com`。
- `env=test` 必须选择 `test_cluster_target_id`，且目标 ClusterTarget 的 `envType` 必须匹配 `test`、`devtest` 或 `staging`。
- ClusterTarget 必须 `directEnabled=true` 且 `directKubeconfigRef` 非空。
- 自动接入只支持 `clusterip` / `nodeport`；`gateway` / `metallb` 必须返回明确不支持。
- `nodeport_access_host` 当前会被 Helm 渲染进 runtime config，但 Go config struct 没有对应字段；在未补齐前不能把它作为实际访问 URL 来源。

### 4.4 AutoOps → Jenkins 构建参数协议

**方向**：AutoOps PipelineService → Jenkins job / repo Jenkinsfile
**形式**：Jenkins parameterized build

**AutoOps 必须注入或透传的参数**：

```json
{
  "GIT_URL": "git@gayhub.seeingtv.com:ipaas/java-demo.git",
  "GIT_REPO_URL": "git@gayhub.seeingtv.com:ipaas/java-demo.git",
  "GIT_REF": "main",
  "APPLICATION_CODE": "java-demo",
  "ENV": "test",
  "RELEASE_NAME": "java-demo",
  "HARBOR_PROJECT": "java-demo",
  "HARBOR_REPOSITORY": "java-demo",
  "HARBOR_CREDENTIALS_ID": "harbor-robot",
  "SERVICE_PORT": 80,
  "TARGET_PORT": 8080
}
```

**Jenkinsfile 必须输出的推荐结果协议**：

```text
AUTOOPS_IMAGE=<registry>/<project>/<repository>:<tag>
AUTOOPS_IMAGE_TAG=<tag>
AUTOOPS_BUILD_LANGUAGE=<java|node|go|python|other>
AUTOOPS_BUILD_STRATEGY=<maven-jib|dockerfile|npm|go|python|other>
```

**当前兼容解析协议**：

AutoOps 当前解析器已经支持以下旧格式：

```text
IMAGE_TAG=<tag-or-image>
image tag: <tag-or-image>
docker push <image:tag>
podman push <image:tag>
pushed image: <image:tag>
镜像地址: <image:tag>
```

**约束**：

- Jenkins build 非 `SUCCESS` 时，pipeline 停在 build stage，不进入 Harbor scan 或 Direct deploy。
- Jenkins build 成功但没有解析到镜像结果时，pipeline 必须失败并给出可定位错误。
- repo Jenkinsfile 可以按语言自定义构建逻辑，但必须遵守输入参数和输出结果协议。
- AutoOps 不生成语言构建脚本；多语言通用性通过 starter Jenkinsfile / shared library / 输出契约实现。
- 下游 feature 必须补 `AUTOOPS_IMAGE` / `AUTOOPS_IMAGE_TAG` 优先解析，避免只依赖日志文本猜测。

### 4.5 Jenkins / Harbor → AutoOps 镜像解析和部署镜像协议

**方向**：Jenkins Console Output → AutoOps PipelineRun
**形式**：pipeline stage detail 与 `DeployRequest.image`

**状态写入约束**：

```json
{
  "pipelineStatus": "building|scanning|deploying|succeeded|failed",
  "currentPipelineStage": "build|scan|deploy|notify",
  "artifactTag": "<tag>",
  "plannedImageRef": "<registry>/<project>/<repo>:<tag>",
  "finalImageRef": "<registry>/<project>/<repo>:<tag>",
  "jenkinsBuildNumber": 28,
  "jenkinsBuildUrl": "<jenkins-build-url>"
}
```

**约束**：

- 若 Jenkins 只输出 tag，AutoOps 使用 Harbor account / project / repository 拼出最终镜像。
- 若 Jenkins 输出完整 image ref，AutoOps 优先使用该完整镜像。
- `AUTOOPS_SKIP_PIPELINE_SCAN=true` 时可跳过 scan stage；Pukka 当前 Helm values 已设置 `api.skipPipelineScan: true`，用于绕开 Harbor scan 阻塞最小 E2E。

### 4.6 AutoOps Direct Kubernetes 资源协议

**方向**：AutoOps → Kubernetes API
**形式**：Direct manifest / client-go apply

**Deployment 约束**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: <releaseName>
  namespace: <namespace>
  labels:
    app: <releaseName>
    app.kubernetes.io/managed-by: autoops
    autoops.io/deploy-mode: direct
spec:
  replicas: 1
  selector:
    matchLabels:
      app: <releaseName>
  template:
    metadata:
      labels:
        app: <releaseName>
    spec:
      imagePullSecrets:
        - name: harbor-pull-secret
      containers:
        - name: main
          image: <finalImageRef>
          ports:
            - containerPort: 8080
```

**Service 约束**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: <releaseName>
  namespace: <namespace>
spec:
  type: NodePort
  selector:
    app: <releaseName>
  ports:
    - name: http
      port: 80
      targetPort: 8080
```

**结果结构**：

```json
{
  "service": {
    "name": "java-demo",
    "type": "NodePort",
    "ports": [{"port": 80, "targetPort": "8080", "nodePort": 30278}]
  },
  "nodeIps": ["10.0.17.40", "10.0.17.41", "10.0.17.42"],
  "accessUrls": ["http://10.0.17.40:30278/"],
  "warnings": []
}
```

**约束**：

- Direct 部署必须等待 Deployment ready；失败时错误要包含 Pod / Deployment 状态信息。
- NodePort URL 当前由 Kubernetes Node IP + NodePort 生成；如果产品要求固定展示 `10.0.17.206`，必须新增被 Go 消费的 `nodeport_access_host` 或 Profile 级 `AccessURLTemplate`，不能只写 Helm values。
- Pukka 集群使用 Kubespray、Calico、kube-proxy IPVS、MetalLB L2；NodePort 默认可通过节点 IP 访问。`~/Code/kubespray` 当前未显示为 Harbor HTTP registry 配置 containerd insecure mirror，因此新 Pod 拉取 `10.0.17.205:80/...` 镜像需在 E2E 中重点验证。

### 4.7 AutoOps → 钉钉群最终通知协议

**方向**：AutoOps DeployNotifier → deploy_bot webhook → 钉钉群
**形式**：DingTalk markdown webhook

**配置**：

```yaml
integrations:
  deploy_bot:
    provider: dingtalk
    enabled: true
    webhook_url: "<from-secret>"
    secret: "<from-secret>"
```

**通知正文必须包含**：

```text
申请号: <requestNo>
模式: direct
发布名: java-demo
命名空间: ao-direct-java-demo-test
状态: succeeded|failed
镜像: <finalImageRef>
Service 类型: NodePort
Service 端口: 80 → 8080
NodePort: <nodePort>
对外访问地址: http://<node-ip>:<nodePort>/
```

**约束**：

- 回群依赖 `integrations.deploy_bot.enabled=true` 且 `webhook_url` 可用。
- `chatContext` 应使用 snake_case 的 `chat_id`、`at_user_ids`、`at_mobiles`；否则通知仍可能发到固定 webhook，但无法可靠 @ 发起人或关联群上下文。
- Hermes 创建请求后可先回写「申请已提交」；部署完成后的权威结果由 AutoOps deploy_bot webhook 回群。

### 4.8 AutoOps 生产部署和 Pukka 集群事实

**方向**：pukka-gitops / kubespray → E2E 验收
**形式**：只读事实约束

**当前事实**：

- AutoOps 自身由 `~/Code/pukka-gitops/argocd-apps/templates/autoops.yaml` 定义的 ArgoCD Application 部署到 `autoops` namespace。
- AutoOps Helm chart 通过 `apps/autoops/values.yaml` 配置 API、Web、PostgreSQL、Valkey、runtime Secret、Gateway 和 project onboarding。
- AutoOps Gateway 使用 Envoy Gateway + MetalLB，当前外部地址为 `http://10.0.17.206`。
- Jenkins 入口为 `http://10.0.17.204`，Harbor 入口为 `http://10.0.17.205`。
- Pukka Kubernetes 为 Kubespray 部署，3 个 control plane + 3 个 worker，节点 IP 范围为 `10.0.17.40`–`10.0.17.45`，MetalLB 池为 `10.0.17.200`–`10.0.17.220`。
- opsclaw Hermes 运行中，`~/.hermes/.env` 已存在 `HERMES_AUTOOPS_URL=http://10.0.17.206` 和 `HERMES_AUTOOPS_AGENT_TOKEN`。

**约束**：

- roadmap 和下游文档不得记录真实密码、PAT、token、webhook 或 kubeconfig 内容。
- 所有生产运行时检查只记录「存在 / 不存在 / 可用 / 不可用」和资源名，不记录 Secret 值。

## 5. 子 feature 清单

1. **java-demo-dingtalk-minimal-e2e** — 用用户原话跑通最窄 E2E，并只做阻塞该 E2E 的最小配置或脚本修正。
   - 所属模块：A / B / C / D / F 跨模块
   - 依赖：无
   - 状态：done
   - 对应 feature：`2026-05-12-java-demo-dingtalk-minimal-e2e`
   - 备注：已通过 `DR20260513135942.298666965` 验证钉钉请求、OA 审批、Jenkins build #72、Harbor 镜像、Direct NodePort 和钉钉回群；未做架构重构。

2. **autoops-production-config-readiness** — 校验并修正 AutoOps 生产 Helm runtime config、Secret、ClusterTarget、Jenkins/Harbor 账户、project onboarding 和 deploy_bot 配置。
   - 所属模块：B / E
   - 依赖：无
   - 状态：planned
   - 对应 feature：未启动
   - 备注：重点验证 `nodeport_access_host` 渲染但未被 Go 消费、`autoops-runtime` Secret key、`default_*_id` 与数据库实际记录一致。

3. **hermes-deploy-skill-routing** — 更新 opsclaw Hermes skill 和脚本，使 Git URL + `test` +「对外访问」稳定调用 `project-onboard-build-deploy`，并使用 snake_case `chatContext`。
   - 所属模块：A / F
   - 依赖：无
   - 状态：planned
   - 对应 feature：未启动
   - 备注：当前 opsclaw skill 方向正确，但脚本仍输出 `chatId` / `atUserIds`，需要与 AutoOps `ChatContext` 统一。

4. **repo-jenkinsfile-build-contract** — 固化 repo Jenkinsfile 的标准输入参数和输出结果协议，并让 AutoOps 优先解析 `AUTOOPS_IMAGE` / `AUTOOPS_IMAGE_TAG`。
   - 所属模块：C
   - 依赖：无
   - 状态：planned
   - 对应 feature：未启动
   - 备注：已有草稿 `.codestable/features/2026-05-12-repo-jenkinsfile-build-contract/repo-jenkinsfile-build-contract-design.md` 可复用；当前 AutoOps 只显式解析 `IMAGE_TAG` 和若干日志模式。

5. **nodeport-access-feedback** — 确保 Direct 部署状态查询和钉钉通知能返回稳定可用的外部访问 URL。
   - 所属模块：D / F
   - 依赖：`autoops-production-config-readiness`
   - 状态：planned
   - 对应 feature：未启动
   - 备注：优先确认节点 IP URL 是否满足测试环境；如需固定地址，再实现 Go config 中的 `nodeport_access_host` 或 Profile 级 URL 模板。

6. **e2e-runbook-and-evidence** — 形成可重复执行的测试手册和证据模板，记录请求号、审批、Jenkins、Harbor、Kubernetes 和钉钉通知结果。
   - 所属模块：F
   - 依赖：`java-demo-dingtalk-minimal-e2e`
   - 状态：planned
   - 对应 feature：未启动
   - 备注：不记录任何真实 Secret 值，只记录资源名、状态、时间和脱敏链接。

**最小闭环**：第 1 条 `java-demo-dingtalk-minimal-e2e` 做完后，可以在钉钉群输入用户原话，得到申请单号；审批通过后 Jenkins 构建成功、Harbor 出现镜像、AutoOps 在测试环境创建 `Deployment` / `NodePort Service`，并由 deploy_bot 在群里返回镜像和访问地址。

## 6. 排期思路

优先级按「先证明真实流程可跑，再补稳定契约」推进。

第一条直接跑 `java-demo` 最小 E2E，是因为当前 AutoOps 代码已经存在 Agent API、审批、Pipeline、Jenkins、Direct 部署和通知骨架；继续先做大规模抽象会延迟发现真实阻塞。最小 E2E 期间只允许做阻塞链路的小修，例如 Hermes 脚本字段名、生产 Secret 缺失、Jenkinsfile 输出不足或 NodePort URL 不可达。

第二、三、四、五条分别把最小 E2E 暴露出的运行时配置、Hermes 请求、Jenkins 输出和访问地址问题固化为可维护契约。第六条把操作证据写成手册，避免后续每次联调都从聊天记录中恢复上下文。

## 7. 当前代码是否满足需求

### 已满足

- Agent API 已存在：`POST /api/v1/integrations/agent/project-onboard-build-deploy`、`POST /build-deploy-requests`、状态查询和审批同步接口已注册。
- Git URL 自动接入已存在：支持解析 `git@gayhub.seeingtv.com:ipaas/java-demo.git`，可派生 `applicationCode=java-demo`，支持 `env=dev/test`。
- Direct 主路径已存在：Agent build-deploy 生成 `mode=direct`、`workflowKind=build_deploy`，部署时创建 `Deployment` / `Service`。
- `exposureMode=nodeport` 已存在：会生成 `ServiceType=NodePort`。
- Pipeline 已存在：审批通过后由 scheduler 执行 build → scan（可跳过）→ deploy → notify。
- Jenkins / Harbor 集成已存在：Pipeline 会触发 Jenkins，解析构建日志，拼接最终镜像并写回 `DeployRequest.image`。
- Direct 访问结果已存在：执行结果会收集 NodePort、节点 IP 和访问 URL，状态接口与通知可读取。
- AutoOps 生产 Helm values 已开启 `projectOnboarding`，配置了允许的 Git host、共享 Jenkins job、Jenkins/Harbor/审批人/ClusterTarget 默认 ID 和 `skipPipelineScan=true`。
- opsclaw Hermes 当前安装了 `deploy-via-autoops` skill，且运行环境中存在 AutoOps URL 和 Agent Token。

### 需要修正或验收

- **Hermes `chatContext` 字段命名不一致**：opsclaw 脚本当前输出 `chatId` / `atUserIds`，AutoOps 通知结构体读取 `chat_id` / `at_user_ids`。
- **Jenkins 输出契约不够稳**：AutoOps 当前未显式优先解析 `AUTOOPS_IMAGE` / `AUTOOPS_IMAGE_TAG`；若 Jenkinsfile 只输出这些新字段会失败。
- **`nodeport_access_host` 未生效**：Helm template 渲染了该字段，但 Go config struct 没有对应字段；当前访问 URL 来自 Kubernetes 节点 IP。
- **生产 Secret / webhook 需要运行时确认**：`deploy_bot.webhook_url`、OA 审批 client 信息、Agent token 来自 `autoops-runtime` Secret，roadmap 不记录值，只能在 E2E 前检查 key 存在和调用可用。
- **Harbor HTTP 镜像拉取要实测**：Kubespray containerd 配置中未看到针对 `10.0.17.205:80` 的 insecure registry 配置；如果 kubelet 无法拉取 Harbor HTTP 镜像，Direct 部署会停在 ImagePullBackOff。
- **AutoOps 仓库内 Hermes skill 副本缺失**：文档写着仓库内维护 `skills/devops/deploy-via-autoops/SKILL.md`，但当前 AutoOps 仓库未找到该目录；需要补齐或修正文档来源。

## 8. 观察项

- `.codestable/architecture/ARCHITECTURE.md` 目前只是骨架，Deploy 控制面现状仍主要在 `docs/deploy-control-plane.md`、`docs/dingtalk-hermes-api-contract.md` 和 `docs/dingtalk-build-deploy-v1-scope.md`，建议后续用 `cs-arch backfill` 补现状文档。
- `docs/dingtalk-hermes-api-contract.md` 中提示仓库内 skill 副本路径存在，但当前仓库没有 `skills/` 目录；这会影响 Hermes skill 的版本化评审。
- `~/Code/pukka-gitops` 与 `~/Code/kubespray` 当前都有非本任务修改；本 roadmap 不修改这些外部仓库。
- AutoOps 本仓当前已有若干非本任务工作区改动；本 roadmap 只新增 `.codestable/roadmap/dingtalk-autoops-deploy-e2e/`。
- 如果未来产品要求「对外访问」必须是固定 VIP / Gateway 地址而非节点 IP + NodePort，应另起 Gateway / Ingress 暴露 roadmap 或新增子 feature，不在最小 E2E 中扩大范围。

## 9. 变更日志

- 2026-05-13：`java-demo-dingtalk-minimal-e2e` 标记为完成，记录已验证的 request、Jenkins、Harbor、Kubernetes 和回群结果。
- 2026-05-12：创建 roadmap，基于 AutoOps 代码、pukka-gitops Helm values / chart、Kubespray inventory 和 opsclaw Hermes skill 做首次拆解。

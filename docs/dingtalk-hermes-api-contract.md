# Hermes-AutoOps Agent API 集成

## 概述

本文档规范了钉钉 Hermes 机器人与 AutoOps 的集成接口。Hermes 在解析用户自然语言后，通过 Agent 服务账号调用 AutoOps 的 build-deploy 端点发起构建部署流程。AutoOps 根据应用的 `AppDeployProfile` 配置自动补齐部署信息，并启动 OA 审批、Jenkins 构建、Harbor 镜像扫描和 **Direct K8s 部署**（v1.1 默认模式）的完整工作流。

> **v1.1 变更**：Agent 链路默认部署模式从 GitOps 切换为 Direct。AutoOps 直接调 K8s API 部署 Deployment/Service 到目标 namespace，不再依赖 pukka-gitops 仓库或 ArgoCD。

## 响应码约定

AutoOps API 使用两种响应格式：

### 业务响应（标准格式）

```json
{
  "code": <业务状态码>,
  "message": "<提示信息>",
  "data": <响应数据>
}
```

- **成功**：HTTP 200，JSON `code: 200`
- **业务错误**：HTTP 200，JSON `code: 400/404/500`（大部分错误通过 JSON code 区分，HTTP 状态码仍为 200）
- **审批白名单拒绝**：HTTP 403，JSON `code: 471`（使用 `FailedWithStatus` 返回）

### 认证错误（非标准格式）

Agent 认证中间件返回的响应**不使用**标准格式，而是使用 `msg` 字段（非 `message`）：

| 场景 | HTTP 状态码 | 响应体 |
|------|-----------|--------|
| Token 未配置 | 403 | `{"code": 403, "msg": "agent integration is not configured"}` |
| Authorization 头缺失 | 401 | `{"code": 401, "msg": "missing authorization header"}` |
| Token 无效 | 403 | `{"code": 403, "msg": "invalid agent token"}` |

> **⚠️ 关键**：Hermes 解析认证错误时应读取 `msg` 字段（不是 `message`）；解析业务响应时读取 `message` 字段。

## 认证

所有 Agent API 请求需在 `Authorization` Header 中提供 Bearer Token（Agent 服务账号令牌）：

```
Authorization: Bearer <token>
```

Token 配置方式详见 `docs/autoops-build-deploy-onboarding.md` §2.X（Agent Bearer Token 配置）。当前阶段在 `api/config.yaml` 的 `integrations.agent.bearer_token` 中配置（YAML 键使用下划线），修改后需重启 API。

## GitLab 项目自动接入 + 构建部署请求

**端点**：`POST /api/v1/integrations/agent/project-onboard-build-deploy`

**适用场景**：用户直接给出 GitLab 仓库地址，希望 AutoOps 自动创建/复用 Application 与 dev/test AppDeployProfile，然后触发 Jenkins → Harbor → AutoOps Direct 部署。

**Hermes 触发语义示例**：

```text
接入并部署 git@gayhub.seeingtv.com:demo/springboot-demo.git 到开发环境，分支 main，暴露 nodeport
部署 https://gayhub.seeingtv.com/demo/springboot-demo.git 到 test，clusterip
```

### 请求字段

| 字段名 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `requesterExternalType` | string | 是 | 固定 `"dingtalk"` |
| `requesterExternalId` | string | 是 | 请求人的钉钉 UserID |
| `requesterDisplayName` | string | 否 | 请求人显示名 |
| `gitRepoUrl` | string | 是 | GitLab 仓库地址，支持 `git@host:group/repo.git` 与 `https://host/group/repo.git` |
| `applicationCode` | string | 否 | 不传则从 repo 名派生，例如 `springboot-demo` |
| `env` | string | 是 | `dev` 或 `test` |
| `gitRef` | string | 否 | 分支/tag/commit，不传则使用 Profile 默认分支 `main` |
| `exposureMode` | string | 否 | `clusterip` 或 `nodeport`；`gateway`/`metallb` 当前会明确拒绝 |
| `reason` | string | 否 | 申请原因/用户原话摘要 |
| `buildParams` | object | 否 | 额外 Jenkins 参数；覆盖 Profile 同名参数 |
| `chatContext` | object | 否 | 钉钉群上下文 |

### 请求示例

```bash
curl -X POST "http://localhost:8000/api/v1/integrations/agent/project-onboard-build-deploy" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-agent-token" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "2853047193",
    "requesterDisplayName": "李明",
    "gitRepoUrl": "git@gayhub.seeingtv.com:demo/springboot-demo.git",
    "env": "dev",
    "gitRef": "main",
    "exposureMode": "nodeport",
    "reason": "接入并部署 springboot-demo 到开发环境，暴露 nodeport"
  }'
```

### AutoOps 后端自动补齐

该端点会从 `integrations.agent.project_onboarding` 配置中读取默认业务组/部门、共享 Jenkins Job、Harbor 项目、审批人、dev/test ClusterTarget 与 namespace 前缀，自动创建或复用：

- `app_application`：`code` 从 repo 名派生，`repo_url` 保存 GitLab 地址。
- `app_deploy_profile`：Direct 模式部署所需的 namespace、releaseName、Jenkins、Harbor、ServiceType、端口、审批人。
- `app_jenkins_env` 与 `agent_approver_allowlist`：保证后续 Agent build-deploy 可以通过白名单校验。

Hermes **不得**在该链路追问容器镜像、namespace、Jenkins Job、Harbor 仓库或 kubeconfig。

### 暴露模式

| exposureMode | 结果 |
|---|---|
| 省略 / `clusterip` | 创建 ClusterIP Service |
| `nodeport` | 创建 NodePort Service |
| `gateway` / `metallb` | v1.1 明确返回不支持，后续版本扩展 |

> v1.1 状态接口暂不返回实时分配的 `nodePort`；真实外部访问可通过 `kubectl -n <namespace> get svc <releaseName>` 查看 nodePort，再访问 `http://<nodeport_access_host>:<nodePort>/`。

## 发起构建部署请求

**端点**：`POST /api/v1/integrations/agent/build-deploy-requests`

**说明**：Hermes 在自然语言处理阶段收集必要参数后调用此端点，AutoOps 自动从 AppDeployProfile 补齐配置详情。

### 请求字段

| 字段名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| `requesterExternalType` | string | 是 | 固定值 `"dingtalk"` |
| `requesterExternalId` | string | 是 | 请求人的钉钉 UserID（示例：`"2853047193"`) |
| `requesterDisplayName` | string | 否 | 请求人显示名（示例：`"李明"`) |
| `applicationCode` | string | 是 | 应用代号（示例：`"java-demo"`），必须与 AutoOps 中已注册的应用代码匹配 |
| `env` | string | 是 | 部署环境，仅允许 `"dev"` 或 `"test"` |
| `gitRef` | string | 否 | Git 分支/tag/commit SHA（示例：`"main"`, `"v1.2.3"`, `"abc123def"`)；缺省使用 Profile 的 `defaultGitRef`（通常为 `"main"`) |
| `reason` | string | 否 | 发布原因说明（示例：`"修复客户报告的 bug"`）；缺省为 `"钉钉机器人触发构建部署"` |
| `buildParams` | object | 否 | 额外构建参数，键值对形式（示例：`{"SKIP_TESTS": false, "MAVEN_PROFILE": "prod"}`)；会合并到 Profile 的 `buildParams` 中，优先级更高 |
| `chatContext` | object | 否 | 钉钉群上下文，含 provider、chatId、atUserIds 等（示例：`{"provider":"dingtalk", "chatId":"chat123", "atUserIds":["user1","user2"]}`)；用于将回复发送回原群 |

### 请求示例

**Dev 环境示例**：

```bash
curl -X POST "http://localhost:8000/api/v1/integrations/agent/build-deploy-requests" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-agent-token" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "2853047193",
    "requesterDisplayName": "李明",
    "applicationCode": "java-demo",
    "env": "dev",
    "gitRef": "feature/new-api",
    "reason": "完成新 API 功能开发",
    "buildParams": {"MAVEN_PROFILE": "dev"},
    "chatContext": {
      "provider": "dingtalk",
      "chatId": "ching123456",
      "atUserIds": ["2853047193"]
    }
  }'
```

**Test 环境示例**：

```bash
curl -X POST "http://localhost:8000/api/v1/integrations/agent/build-deploy-requests" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-agent-token" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "2853047193",
    "requesterDisplayName": "李明",
    "applicationCode": "java-demo",
    "env": "test",
    "reason": "验收测试"
  }'
```

## Hermes skill 选路规则（必须先判定）

Hermes 的 `deploy-via-autoops` skill 必须先区分三条链路，不能把“源码构建部署”误判成“已有镜像部署”：

| 用户意图 | 典型自然语言 | AutoOps 端点 | 是否向用户索要 image/namespace |
|---|---|---|---|
| GitLab 项目自动接入 + build-deploy | “接入并部署 git@gayhub.seeingtv.com:demo/springboot-demo.git 到开发，暴露 nodeport” | `POST /api/v1/integrations/agent/project-onboard-build-deploy` | **禁止** |
| build-deploy / Profile 驱动（默认） | “部署 java-demo 到开发环境，main 分支”、“java-demo 发测试”、“构建并部署 java-demo” | `POST /api/v1/integrations/agent/build-deploy-requests` | **禁止** |
| existing-image direct（仅显式已有镜像） | “用 registry.example.com/api:20260428 部署临时服务”、“部署 nginx:1.27.4-alpine” | `POST /api/v1/integrations/agent/deploy-requests` | 允许在缺失时追问 |

### build-deploy 负面规则（本轮联调阻塞项）

对 `build-deploy` 链路，Hermes 只抽取并提交 `applicationCode`、`env`、可选 `gitRef`、请求人钉钉身份和可选原因。以下字段由 AutoOps 后端根据 `AppDeployProfile`/`ClusterTarget`/Jenkins 结果补齐，**不得向用户追问**：

- 容器镜像地址：镜像由 Jenkins 构建并推送后写回 DeployRequest，创建请求时通常为空。
- K8s namespace：来自 `AppDeployProfile.namespace`；Direct 的 `ao-direct-` 前缀是管理员配置/后端校验项。
- clusterTarget / kubeconfig：来自 `AppDeployProfile.clusterTargetId` 与 ClusterTarget 的 `directKubeconfigRef`。
- releaseName、Jenkins job、Harbor project/repository、approver：均来自 Profile。

错误行为示例（禁止）：

```text
skill_view: "deploy-via-autoops"，我需要以下信息才能继续：
容器镜像地址：例如 registry.example.com/java-demo:main-20250428 或类似格式。您是否有已构建好的镜像？
命名空间：开发环境的具体命名空间名称是什么？（direct 模式需要以 ao-direct- 开头）
```

正确行为示例：

```text
User: 部署 java-demo 到开发环境，main 分支
Hermes: 调用 /integrations/agent/build-deploy-requests，提交 applicationCode=java-demo、env=dev、gitRef=main。
```

对应请求体：

```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "<userid>",
  "applicationCode": "java-demo",
  "env": "dev",
  "gitRef": "main",
  "reason": "部署 java-demo 到开发环境，main 分支"
}
```

仓库内 skill 副本维护在 `skills/devops/deploy-via-autoops/SKILL.md`，更新 Hermes 运行环境时应以该文件为准同步到 opsclaw。

## Hermes 缺参追问规则

Hermes 应在自然语言处理阶段完成参数收集。对 build-deploy 链路，只有以下情况需要追问用户：

### 1. 缺少应用代号

**触发条件**：用户说 "部署 java 应用" 但未明确指定哪个应用

**追问示例**：
```
Hermes: "我找到了你的 java-demo 和 java-service 两个应用，请问要部署哪一个？"
User: "部署 java-demo"
Hermes: [继续请求 env 参数...]
```

**错误消息**（若仍未提供）：
```
Hermes: "部署失败：应用不存在或未接入 AutoOps。请检查应用代号是否正确。"
```

### 2. 缺少部署环境

**触发条件**：用户说 "部署到某环境" 时表述不清（例如说 "部署到生产"，但 Profile 只配置了 dev/test）

**追问示例**：
```
Hermes: "java-demo 支持部署到 dev（开发）或 test（测试）环境，请选择一个。"
User: "部署到 dev"
Hermes: [完成参数收集，发起请求]
```

**错误消息**（若环境不支持）：
```
Hermes: "部署失败：应用环境未配置部署 Profile。请联系管理员配置该环境的部署方案。"
```

### 3. Git 分支参数处理（无需追问）

**触发条件**：用户未明确指定分支/tag

**处理方式**：不追问用户，直接省略 `gitRef`，由 AutoOps 使用 Profile 配置的 `defaultGitRef`（通常为 `"main"`)

**示例对话**：
```
User: "部署 java-demo 到 test"
Hermes: "好的，将使用默认分支 main 部署。..." [直接发起请求]
```

## 错误回写格式

AutoOps 返回的错误响应，Hermes 应将 `message` 字段的内容回写到钉钉群。**重要**：AutoOps 大部分业务错误返回 **HTTP 200**，通过 JSON `code` 字段区分成功/失败。仅认证失败（无效/缺失 Token）返回真实 HTTP 401/403。

以下是完整的错误映射表及建议的中文回写消息：

| JSON code | HTTP | message | 建议回写消息 | 排查建议 |
|-----------|------|---------|-------------|--------|
| 400 | 200 | `"当前仅支持 dingtalk 外部身份类型"` | "系统配置错误：仅支持钉钉外部身份。" | 检查 Agent 服务配置中的 `requesterExternalType` |
| 404 | 200 | `"应用不存在或未接入 AutoOps"` | "应用代号错误或该应用尚未接入 AutoOps。请确认应用代号是否正确。" | 检查 applicationCode 是否与 AutoOps 注册的应用代码匹配 |
| 404 | 200 | `"应用环境未配置部署 Profile"` | "该应用的 %env% 环境尚未配置部署方案，请联系管理员配置。" | AutoOps 管理员需在对应应用的环境配置中创建 AppDeployProfile |
| 400 | 200 | `"应用环境部署 Profile 已禁用"` | "该应用的 %env% 环境部署方案已被禁用，请联系管理员启用。" | AutoOps 管理员需在配置中重新启用该 Profile 的 `enabled` 字段 |
| 400 | 200 | `"该钉钉用户未绑定 AutoOps 账号"` | "你的钉钉账号未在 AutoOps 中绑定，请联系管理员进行绑定。" | AutoOps 管理员需在系统管理 → 用户管理中为该钉钉 UserID 创建账号 |
| 471 | 403 | `"approverAdminId not in agent_approver_allowlist..."` | "审批人不在该环境的白名单中，请联系管理员添加。" | AutoOps 管理员需在审批人白名单中添加该审批人 |
| 500 | 200 | `"内部错误"` 或其他异常 | "部署服务暂时异常，请稍后重试。[错误详情: %errorMessage%]" | 检查 AutoOps 服务日志；如持续异常，联系后端团队 |

**认证错误（Agent 认证中间件返回，非标准格式，使用 `msg` 而非 `message`）：**

| HTTP 状态码 | 响应体 | 说明 |
|------------|--------|------|
| 401 | `{"code": 401, "msg": "missing authorization header"}` | 缺少 Authorization 头 |
| 403 | `{"code": 403, "msg": "invalid agent token"}` | Bearer Token 无效 |
| 403 | `{"code": 403, "msg": "agent integration is not configured"}` | Token 未配置（server 端未设置 bearer_token） |

> **注意**：大部分业务错误返回 HTTP 200 + JSON `code`（非 200 表示失败）。仅审批白名单拒绝（code 471）返回真实 HTTP 403，认证失败返回 HTTP 401/403。Hermes 应优先检查 JSON `code` 字段判断成功/失败。

## 成功响应

**HTTP 状态码**：`200`

**说明**：API 返回完整的 `DeployRequest` 对象（含所有字段）。以下仅列出 Hermes 需要关注的关键字段。

> **注意**：对于 `build_deploy` 工作流，`image` 字段在创建时通常为空（镜像地址由 Jenkins 构建后填入）。Hermes 应通过查询状态接口获取最终的 `accessInfo.image`。

**响应体关键字段**：

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR-20260428-001",
    "mode": "direct",
    "workflowKind": "build_deploy",
    "requestStatus": "pending_approval",
    "approvalStatus": "pending",
    "approvalDispatchStatus": "pending",
    "approvalDispatchMessage": "已提交审批",
    "dingtalkProcessInstanceId": "abc123...",
    "namespace": "ao-direct-java-demo",
    "releaseName": "java-demo",
    "replicas": 1,
    "serviceEnabled": true,
    "...": "（其他 DeployRequest 字段，image 字段在 build_deploy 模式创建时为空）"
  }
}
```

**Hermes 重点关注字段**：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `requestNo` | string | 部署申请单号，唯一标识本次部署请求（用于查询状态） |
| `workflowKind` | string | 固定值 `"build_deploy"` |
| `requestStatus` | string | 申请当前状态 |
| `approvalStatus` | string | 审批状态 |
| `approvalDispatchStatus` | string | 审批投递状态（`pending`/`dispatched`/`approved`/`rejected`） |
| `approvalDispatchMessage` | string | 审批投递说明 |
| `dingtalkProcessInstanceId` | string | 钉钉审批实例 ID，可用此构造审批链接 |
| `message` | string | 用户友好的提示消息 |

**回写示例**：

```
Hermes 回复钉钉群：
"✓ 部署请求已提交！
申请单号：DR-20260428-001
应用：java-demo / test 环境
分支：feature/new-api
审批实例：[点击审批]（使用 dingtalkProcessInstanceId 构造链接）
系统将在审批通过后自动执行 Jenkins 构建、Harbor 扫描和 Direct K8s 部署。"
```

## 查询部署状态

**端点**：`GET /api/v1/integrations/agent/deploy-requests/{requestNo}/status`

**说明**：Hermes 或用户可调用此端点查询部署进度。AutoOps 返回当前状态及中间结果（如部署访问信息）。

> **注意**：状态接口返回的字段为 `requestNo`、`requestStatus`、`approvalStatus`、`executionStatus`、`finishedAt`、`executionSummary`、`accessInfo`、`errorMessage`。如需 `pipelineStatus`（构建/扫描/部署进度）或 `mode`/`workflowKind`，请通过 `GET /api/v1/integrations/agent/deploy-requests/{requestNo}` 查询完整 DeployRequest 对象。

### 查询示例

```bash
curl -X GET "http://localhost:8000/api/v1/integrations/agent/deploy-requests/DR-20260428-001/status" \
  -H "Authorization: Bearer your-agent-token"
```

### 响应示例（等待审批阶段）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR-20260428-001",
    "requestStatus": "pending_approval",
    "approvalStatus": "pending",
    "executionStatus": "pending",
    "finishedAt": null,
    "executionSummary": null,
    "accessInfo": null,
    "errorMessage": null
  }
}
```

### 响应示例（构建/扫描中）

> **注意**：Jenkins 构建和 Harbor 扫描阶段，`requestStatus` 更新为 `"executing"`，但 `executionStatus` 仍为 `"pending"`（Direct 部署的执行尚未开始）。流水线进度需通过完整 DeployRequest 查询的 `pipelineStatus` 字段获取（`building`/`scanning`/`deploying`）。

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR-20260428-001",
    "requestStatus": "executing",
    "approvalStatus": "approved",
    "executionStatus": "pending",
    "finishedAt": null,
    "executionSummary": null,
    "accessInfo": null,
    "errorMessage": null
  }
}
```

### 响应示例（部署成功）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR-20260428-001",
    "requestStatus": "succeeded",
    "approvalStatus": "approved",
    "executionStatus": "succeeded",
    "finishedAt": "2026-04-28T15:30:45Z",
    "executionSummary": "Direct 执行成功",
    "accessInfo": {
      "image": "harbor.example.com/library/java-demo:main-abc123",
      "namespace": "ao-direct-java-demo",
      "releaseName": "java-demo",
      "serviceEnabled": true,
      "serviceType": "ClusterIP",
      "servicePort": 80,
      "targetPort": 8080
    },
    "errorMessage": null
  }
}
```

### 响应示例（部署失败）

```json
{
  "code": 200,
  "message": "成功",
  "data": {
    "requestNo": "DR-20260428-001",
    "requestStatus": "failed",
    "approvalStatus": "approved",
    "executionStatus": "failed",
    "finishedAt": "2026-04-28T15:35:12Z",
    "executionSummary": "执行失败",
    "accessInfo": null,
    "errorMessage": "Direct 部署失败：Pod 创建异常，imagePullBackOff"
  }
}
```

## 状态值语义

注意：API 返回 3 个独立的状态字段，Hermes 应综合判断：

### requestStatus（申请状态）

| 值 | 说明 | 后续动作 |
|---|---|---|
| `pending_approval` | 等待审批人确认 | Hermes 可提示用户审批 |
| `approved` | 审批通过 | 后台自动执行 |
| `rejected` | 审批拒绝 | 流程终止 |
| `executing` | 部署执行中 | 后台自动执行 |
| `succeeded` | 部署成功 | 返回 accessInfo（v1.1 含 namespace/servicePort 等） |
| `failed` | 部署失败 | 见 errorMessage |
| `rolled_back` | 已回滚 | 仅 GitOps 模式 |
| `cancelled` | 已取消 | — |
| `expired` | 已过期 | — |

### executionStatus（执行状态）

| 值 | 说明 |
|---|---|
| `pending` | 尚未执行 |
| `running` | 执行中 |
| `succeeded` | 执行成功 |
| `failed` | 执行失败 |
| `rolled_back` | 已回滚 |
| `cleaned` | 资源已回收 |

### pipelineStatus（流水线状态，仅 workflowKind=build_deploy 时有值）

| 值 | 说明 |
|---|---|
| `pending` | 流水线未开始 |
| `building` | Jenkins 构建中 |
| `scanning` | Harbor 镜像扫描中 |
| `deploying` | K8s 部署中 |
| `succeeded` | 流水线完成 |
| `failed` | 流水线失败 |

## 成功结果字段说明

当 `executionStatus == "succeeded"` 时，响应体包含以下字段：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `requestNo` | string | 部署申请单号 |
| `requestStatus` | string | 申请状态 |
| `approvalStatus` | string | 审批状态 |
| `executionStatus` | string | 执行状态（`"succeeded"`） |
| `finishedAt` | string | 完成时间（ISO 8601） |
| `executionSummary` | string | 执行摘要文本（如 `"Direct 执行成功"`） |
| `accessInfo` | object \| null | 部署访问信息（仅成功时非 null） |
| `errorMessage` | string \| null | 错误信息（仅失败时非 null） |

`accessInfo` 对象字段（v1.1）：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `image` | string | 部署的容器镜像 |
| `namespace` | string | K8s 命名空间（以 `ao-direct-` 开头） |
| `releaseName` | string | 发布名称 |
| `serviceEnabled` | bool | 是否创建了 Service |
| `serviceType` | string | Service 类型（如 ClusterIP），仅 serviceEnabled=true 时返回 |
| `servicePort` | int | Service 端口，仅 serviceEnabled=true 时返回 |
| `targetPort` | int | 目标端口，仅 serviceEnabled=true 时返回 |

> **⚠️ v1.1 accessUrl 现状**：`buildAccessInfo()` 当前仅返回 image、namespace、releaseName、serviceEnabled、serviceType、servicePort、targetPort，`accessUrl` 字段在 v1.1 的 `AccessInfo` 结构体中不存在，API 响应中不会出现此字段。如需访问地址，请在 AppDeployProfile 中配置 `accessUrlTemplate`，或根据 namespace + servicePort 自行拼接。

---

**文档版本**：1.2 | **更新时间**：2026-04-29 | **变更**：新增 GitLab 项目自动接入端点；v1.1 Agent 链路切换为 Direct 模式；accessUrl 在 v1.1 暂不返回（AccessInfo 无此字段）

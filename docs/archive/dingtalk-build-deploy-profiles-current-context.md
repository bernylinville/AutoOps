# DingTalk/Hermes 构建部署链路当前上下文与新窗口提示词

更新时间：2026-04-28 16:24 Asia/Shanghai

## 1. 用户原始诉求（以此为最高优先级）

用户这次要解决的不是“搭一个泛化 CI/CD 平台”，而是一个清晰的开发自助构建部署场景：

> 支持两套环境（开发和测试），在 K8s 里面通过 namespace 区分，能够自动触发构建和部署，不需要人工干预；前期只考虑新项目；结合 Hermes Agent（钉钉机器人），开发在群里 @机器人用自然语言发需求；用上 K8s 集群里的 Jenkins 和 Harbor；GitLab 是自建的 `gayhub.seeingtv.com`。

目标链路：

```text
钉钉群里 @机器人发送自然语言构建/部署需求
-> Hermes/Agent 解析意图和参数
-> Hermes 通过 skill / AutoOps API 调用 AutoOps 编排能力
-> AutoOps 根据应用 + 环境 Profile 补齐 Jenkins/Harbor/K8s/审批配置
-> AutoOps 发起钉钉 OA 审批
-> 审批通过后触发 Jenkins 源码构建与打包
-> Jenkinsfile 推送镜像到 Harbor 并触发/等待镜像安全及漏洞扫描
-> 扫描通过后 AutoOps 更新环境版本并由 GitOps/ArgoCD 落地 K8s 部署
-> AutoOps/Hermes 把最终构建部署结果和访问信息回写钉钉群
```

关键边界：

- v1 只支持 **新项目**，不处理老项目迁移/兼容的复杂场景。
- v1 只支持 **开发、测试两套环境**，在 K8s 里通过 namespace 区分。
- GitLab 自建地址：`gayhub.seeingtv.com`。
- Jenkins、Harbor 已在 K8s 集群中，CI/CD 逻辑主要由各项目 `Jenkinsfile` 承担。
- Harbor 的 `library` 等项目可以通过 UI/API 创建，不需要把 Harbor 项目全生命周期 provisioning 做成 v1 主线。
- Hermes/钉钉机器人只负责自然语言理解和调用 AutoOps，不在机器人侧保存部署配置和编排状态。
- AutoOps 需要提供后端接口和前端页面，用于配置“服务部署所需信息和可选信息”。

这和之前“钉钉群内通过聊天发送业务需求指令 -> Agent 分析意图并调用 AutoOps API -> AutoOps 发起发布任务及钉钉 OA 审批 -> 审批通过后由 AutoOps/ArgoCD 落地 K8s 部署 -> Agent 将最终部署结果及访问详情同步响应至钉钉内”的业务发布场景相比，复杂度不同：当前场景多了 **源码构建、镜像打包、Harbor 扫描、版本更新**，但仍不应该升级成全功能平台建设。

相关外部路径：

- `/home/kchou/Code/pukka-gitops/scripts/mirror-images.sh`
- `/home/kchou/Code/pukka-gitops/CLUSTER-ACCESS.md`

## 2. 通过 Chrome DevTools 已确认的 Jenkins / Harbor 状态

### Jenkins

访问：`http://10.0.17.204`

已通过 MCP Chrome DevTools 登录/检查，结论：

- 当前账号是 admin。
- Jenkins 授权策略是：`Logged-in users can do anything`。
- 匿名访问未开放。
- 因此 **Jenkins 权限不是当前阻塞点**。

发现的真实构建失败点：

- Jenkins build #20 已失败。
- Maven 构建阶段通过。
- 失败发生在 Jib 推送镜像到 HTTP Harbor 时：Jib 拒绝通过 HTTP 发送认证信息。
- 解决方向：
  - Jenkinsfile / Maven 参数添加：`-Djib.to.auth.sendCredentialsOverHttp=true`
  - 或者把 Harbor 改成 HTTPS。

### Harbor

访问：`http://10.0.17.205`

已确认：

- Harbor 可登录。
- 已存在项目：`library`、`petclinic`、`java-demo`。
- 默认 scanner 已安装。
- `java-demo` 当前无 artifact。
- Harbor 项目可以通过 UI/API 创建；不需要在 AutoOps v1 里过度建设 Harbor 基础设施 provisioning。

## 3. 方案判断：当前方案是否过度设计，哪里还不足

### 3.1 当前方案的核心判断

原 v7 方案偏重“全链路平台化 / 基础设施 provisioning / 多系统泛化编排”，对当前诉求有过度设计倾向。

当前更合适的 v1 方案是：

- **AutoOps 保存“应用 + 环境”的部署 Profile**，作为服务部署权威配置。
- Profile 里保存 Jenkins/Harbor/namespace/release/审批人/端口/访问地址等信息。
- **Hermes 只传自然语言解析结果**，例如 `applicationCode + env + gitRef + requesterExternalId + reason`。
- AutoOps 根据 Profile 解析成内部 deploy request / pipeline run。
- Jenkinsfile 继续承载项目自身构建、打包、镜像推送和扫描触发逻辑。
- AutoOps 负责审批、状态、流水线记录、GitOps 版本更新、最终结果回写。

### 3.2 哪些内容在 v1 里属于过度设计，应暂缓

| 方向 | v1 判断 | 原因 |
| --- | --- | --- |
| AutoOps 自动创建 GitLab repo | 暂缓 | 用户已有自建 GitLab，当前重点是构建部署打通，不是代码仓库生命周期管理。 |
| AutoOps 自动生成/管理 Jenkinsfile | 暂缓 | 用户明确“其他 CI/CD 都通过 Jenkinsfile”，AutoOps 只保存 job/参数/状态。 |
| AutoOps 深度 provisioning Jenkins job | 暂缓/可手工 | 初期可在 Jenkins 里已有 job 或按约定创建；不要阻塞主链路。 |
| Harbor 项目全生命周期治理 | 暂缓 | `library` 等项目可通过 MCP Chrome DevTools/UI/API 创建；v1 只需要配置 project/repository 并校验。 |
| 多集群、多环境、生产审批矩阵 | 暂缓 | 当前只要 dev/test namespace。 |
| StatefulSet/复杂工作负载模板 | 暂缓 | v1 只考虑新项目，优先 deployment/service。 |
| 通用自然语言平台/复杂多轮对话 | 暂缓 | Hermes 只需把明确意图转为结构化 API 请求。 |

### 3.3 哪些内容不是过度设计，属于必要能力

| 能力 | 判断 | 原因 |
| --- | --- | --- |
| App Deploy Profile | 必要 | 不然 Hermes 每次都要猜 Jenkins/Harbor/namespace/审批人。 |
| Agent build-deploy API | 必要 | Hermes 需要稳定结构化入口。 |
| AutoOps 前端配置 Profile | 必要 | 用户明确需要管理界面配置服务部署信息。 |
| DingTalk UserID 绑定 | 必要 | OA 审批和 requester/approver 映射需要。 |
| Approver allowlist/RBAC | 必要 | 防止机器人/普通用户随意指定审批人或部署目标。 |
| Pipeline run 状态记录 | 必要 | 需要闭环返回构建/扫描/部署结果。 |

### 3.4 当前方案仍不足的地方

1. **真实 E2E 尚未打通**：还缺一次从 Hermes/Agent 请求到 OA、Jenkins、Harbor、GitOps/ArgoCD、钉钉回写的完整验证。
2. **Jenkinsfile/Jib HTTP Harbor 问题未修**：Jib 需要 `-Djib.to.auth.sendCredentialsOverHttp=true` 或 Harbor HTTPS。
3. **Profile UI 高级字段未全部暴露**：后端预留了 `envVars/resources/buildParams/scanPolicy`，UI v1 暂未配置这些高级字段。
4. **dev/test 与 ClusterTarget envType 未强绑定**：目前只校验 ClusterTarget 存在，后续应限制 dev profile 只能选 dev target/test profile 只能选 test target，或明确映射规则。
5. **Hermes skill/API contract 需要固化**：要写清自然语言解析后的字段、缺参追问规则、错误回写格式。
6. **结果闭环策略需要明确**：Hermes 是轮询 AutoOps status、接收 webhook，还是由 AutoOps 直接发送钉钉通知，需要在 E2E 中定下来。

## 4. 已创建的计划/上下文文档

- `.omx/context/dingtalk-build-deploy-pipeline-20260428T073000Z.md`
- `.omx/plans/prd-dingtalk-build-deploy-profiles.md`
- `.omx/plans/test-spec-dingtalk-build-deploy-profiles.md`

## 5. 本轮已实现的主要能力

### 5.1 Backend：应用部署 Profile

新增/修改：

- `api/api/app/model/deployProfile.go`
  - `AppDeployProfile`
  - `CreateAppDeployProfileRequest`
  - `UpdateAppDeployProfileRequest`
  - `AppDeployProfileValidation`
- `api/api/app/service/deployProfile.go`
  - Profile CRUD/list/validate。
  - 校验 ClusterTarget、Jenkins 凭据、Harbor 凭据、审批人钉钉 UserID。
  - 创建/更新 Profile 时同步 `app_jenkins_env`。
  - 创建/更新 Profile 时同步 `agent_approver_allowlist`。
  - 创建/更新/删除与副作用已改为 DB transaction，副作用失败会回滚。
- `api/api/app/controller/deployProfile.go`
- `api/api/app/service/application.go`
  - `IApplicationService` 增加 deploy profile 方法。
- `api/router/app/application.go`
  - 新增路由：
    - `GET /apps/:id/deploy-profiles`
    - `POST /apps/:id/deploy-profiles`
    - `PUT /apps/:id/deploy-profiles/:profile_id`
    - `DELETE /apps/:id/deploy-profiles/:profile_id`
    - `POST /apps/:id/deploy-profiles/:profile_id/validate`
  - 路由已加：`AuthMiddleware()` + `RbacMiddleware("app:application:env")`。
- `api/pkg/db/migrate.go`
  - 注册 `AppDeployProfile`。
  - 注册 `AgentApproverAllowlist`。

### 5.2 Backend：Hermes/Agent build-deploy API

新增/修改：

- `api/api/deploy/model/deploy.go`
  - `CreateDeployRequest` 和 `CreateAgentDeployRequest` 增加：
    - `jenkinsServerId`
    - `jenkinsJobName`
    - `harborServerId`
    - `scanPolicy`
  - 新增 `CreateAgentBuildDeployRequest`。
- `api/api/deploy/service/agentBuildDeploy.go`
  - 新 endpoint service：按 `applicationCode + env + gitRef` 解析 Profile。
  - 拼装内部 `CreateAgentDeployRequest`。
- `api/api/deploy/service/deploy.go`
  - `CreateAgentDeployRequest` 和 UI `CreateDeployRequest` wrapper 已修复：不再丢失 Jenkins/Harbor/GitRef/BuildParams/ScanPolicy 等 build_deploy 字段。
  - pipeline run 创建时优先使用请求/Profile 提供的 Jenkins/Harbor/job，缺失才 fallback 到 cluster target / app env。
  - 保留 approver allowlist gate。
- `api/api/deploy/controller/deploy.go`
  - 新增 `CreateAgentBuildDeployRequest` handler。
- `api/router/deploy/deploy.go`
  - 新增 Agent 路由：
    - `POST /integrations/agent/build-deploy-requests`
- `api/api/deploy/model/approverAllowlist.go`
- `api/api/deploy/dao/approverAllowlist.go`

Agent 请求示例：

```json
{
  "requesterExternalType": "dingtalk",
  "requesterExternalId": "ding-user-id",
  "requesterDisplayName": "张三",
  "applicationCode": "java-demo",
  "env": "dev",
  "gitRef": "main",
  "reason": "构建并部署到开发环境",
  "chatContext": {
    "provider": "dingtalk",
    "chatId": "xxx",
    "atUserIds": ["xxx"]
  }
}
```

### 5.3 Frontend：AutoOps 管理界面支持

新增/修改：

- `web/src/api/app.js`
  - Harbor 服务器列表 API。
  - App deploy profile CRUD/validate API。
- `web/src/views/app/application.vue`
  - 应用列表操作新增“部署配置”。
  - 可配置 dev/test profile：
    - env
    - enabled
    - clusterTarget
    - namespace
    - releaseName
    - resourceType
    - Jenkins server/job
    - Harbor server/project/repository
    - defaultGitRef
    - approverAdminId
    - replicas
    - service enabled/type/port/targetPort
    - accessUrlTemplate
    - healthCheckPath
    - description
  - 审批人下拉显示钉钉 UserID；未绑定会显示“未绑定钉钉”。
- `web/src/views/system/Admin.vue`
  - 用户列表新增“钉钉UserID”列。
  - 新增/编辑用户表单新增 `dingtalkUserId` 字段。

审批人在 AutoOps 管理界面添加方式：

1. 系统管理 → 用户管理。
2. 新增或编辑用户。
3. 填写 `钉钉UserID`。
4. 应用管理 → 部署配置 → 选择该用户作为审批人。

## 6. Architect 审查结果

第一次 Architect 审查发现 3 个 blocker：

1. Profile 解析出来的 Jenkins/Harbor 字段在 `CreateAgentDeployRequest -> CreateDeployRequest` wrapper 中被丢弃。
2. UI build_deploy 请求 wrapper 也丢弃 build 字段。
3. Deploy Profile CRUD 只有 Auth，没有 RBAC。

已修复后再次 Architect 复审，结果：

```text
APPROVED — no concrete remaining blockers found in the reviewed scope.
```

复审确认：

- UI deploy requests 通过 `cloneCreateDeployRequest` 保留完整 build_deploy 字段。
- Agent deploy requests 通过 `createDeployRequestFromAgent` 保留完整 build_deploy 字段。
- pipeline run 使用 request-level Jenkins/Harbor 优先。
- deploy profile 路由已加 RBAC。
- profile side effects 已事务化且返回错误。

## 7. 已通过的验证

Backend：

```bash
cd api
go test ./api/app/... ./api/deploy/... -count=1
go test ./... -count=1
go build ./...
```

Frontend：

```bash
cd web
npm run build -- --dest /tmp/autoops-web-build-ralph
npm run build
./node_modules/.bin/eslint src/api/app.js src/views/app/application.vue src/views/system/Admin.vue
```

结果：

- 后端测试通过。
- 后端构建通过。
- 前端临时目录构建通过。
- 用户执行 `sudo rm -rf web/dist` 后，正常 `web/dist` 构建也已通过。
- ESLint 目标文件 0 error，仅存在既有 warning：
  - `View` unused。
  - `system/Admin.vue` 里一些旧 icon/参数 unused。

前端构建仍有非阻塞 warning：

- Browserslist 数据过旧。
- `/deep/` / `>>>` deprecated。
- bundle size 超过 Vue CLI 推荐阈值。

## 8. 当前已知风险 / 后续待做

1. 还未跑真实 E2E：Hermes → AutoOps → OA → Jenkins → Harbor scan → GitOps/ArgoCD → DingTalk。
2. Jenkinsfile 仍需处理 Harbor HTTP 推送认证问题：
   - `-Djib.to.auth.sendCredentialsOverHttp=true`
   - 或 Harbor HTTPS。
3. Profile UI v1 还未暴露高级字段：
   - `envVars`
   - `resources`
   - `buildParams`
   - `scanPolicy`
   但后端模型已预留。
4. 当前 dev/test profile 没有强校验 cluster target envType 与 profile env 的映射，只校验 cluster target 存在。
5. Hermes skill/API contract 还需要形成明确文档：自然语言字段抽取、缺参追问、API 请求体、错误回写、状态查询。
6. 工作区中存在不少与本任务无关的历史 dirty/untracked 文件，不要随意 revert。

## 9. 新任务窗口可直接使用的提示词

把下面整段复制到新任务窗口：

```text
你在 /home/kchou/Code/AutoOps 仓库继续工作。请先阅读：

- docs/dingtalk-build-deploy-profiles-current-context.md
- .omx/context/dingtalk-build-deploy-pipeline-20260428T073000Z.md
- .omx/plans/prd-dingtalk-build-deploy-profiles.md
- .omx/plans/test-spec-dingtalk-build-deploy-profiles.md

请用中文回复。当前任务不是重新从零设计 CI/CD 平台，而是围绕下面这个真实诉求重新评估当前方案是否过度设计、还有哪些不足，并继续规划/推进下一步：

用户核心需求：支持两套环境（开发和测试），在 K8s 里面通过 namespace 区分，能够自动触发构建和部署，不需要人工干预；前期只考虑新项目；结合 Hermes Agent（钉钉机器人），开发在群里 @机器人用自然语言发需求；用上 K8s 集群里的 Jenkins 和 Harbor；GitLab 是自建的 gayhub.seeingtv.com。

目标链路：
钉钉群 @机器人自然语言提出构建/部署需求 -> Hermes/Agent 解析 -> 调用 AutoOps API/skill -> AutoOps 根据应用+环境 Profile 补齐配置 -> 发起钉钉 OA 审批 -> 审批通过后触发 Jenkinsfile 源码构建和镜像打包 -> Harbor 镜像扫描通过 -> AutoOps/GitOps/ArgoCD 更新 dev/test namespace 版本并部署 -> 结果回写钉钉群。

已确认：
- Jenkins http://10.0.17.204 是 admin，授权策略为 Logged-in users can do anything，Jenkins 权限不是阻塞点。
- Jenkins build #20 失败原因是 Jib 不允许通过 HTTP 向 Harbor 发送凭据，不是 Jenkins 权限问题。修复方向是 Jenkinsfile 添加 -Djib.to.auth.sendCredentialsOverHttp=true 或 Harbor 改 HTTPS。
- Harbor http://10.0.17.205 已有 library、petclinic、java-demo 项目，默认 scanner 已安装。Harbor library/project 可通过 MCP Chrome DevTools/UI/API 创建，不要把 Harbor provisioning 做成 v1 主线。

当前方案判断：
- v1 不应该做泛化平台、自动建 GitLab repo、自动生成 Jenkinsfile、多集群/生产复杂审批矩阵、StatefulSet 等。
- v1 应该保留 AutoOps App Deploy Profile、Agent build-deploy API、AutoOps 前端部署配置、DingTalk UserID 绑定、审批白名单/RBAC、pipeline run 状态记录。
- Hermes 只负责自然语言解析和调用 AutoOps；AutoOps 保存配置、编排、审批、状态和部署结果。

已完成实现：
- 后端 AppDeployProfile CRUD/list/validate。
- Agent 新接口 POST /api/v1/integrations/agent/build-deploy-requests。
- Profile 解析 applicationCode + env + gitRef 到内部 build_deploy request。
- Profile 同步 app_jenkins_env 和 agent_approver_allowlist。
- Profile 路由已加 RBAC app:application:env。
- UI 应用管理新增“部署配置”。
- UI 系统用户新增/编辑钉钉UserID。
- 修复 build_deploy wrapper 丢失 Jenkins/Harbor/GitRef/ScanPolicy 字段的问题。
- Architect 复审已 APPROVED。

已通过验证：
cd api && go test ./api/app/... ./api/deploy/... -count=1
cd api && go test ./... -count=1
cd api && go build ./...
cd web && npm run build
cd web && ./node_modules/.bin/eslint src/api/app.js src/views/app/application.vue src/views/system/Admin.vue

请继续做下面的事：
1. 先检查当前 git diff，只关注本任务相关文件，不要 revert unrelated changes。
2. 重新评估当前方案：哪些是必要能力，哪些是过度设计要砍掉，哪些不足必须补齐。
3. 输出/更新一个可执行任务计划，必须围绕 dev/test namespace、新项目、Hermes 自然语言入口、Jenkinsfile、Harbor scan、AutoOps Profile/API/UI、GitOps/ArgoCD 结果闭环。
4. 补齐文档/API/操作说明：AutoOps 如何添加审批人、如何配置 deploy profile、Hermes 请求体是什么、缺参怎么处理、结果怎么回写。
5. 准备真实 E2E 前置数据：用户钉钉 UserID、dev/test deploy profile、Jenkins/Harbor config_account、cluster target、java-demo namespace。
6. 重点验证 Hermes 调用示例：applicationCode=java-demo, env=dev/test, gitRef=main 能创建 build_deploy request，并生成带 Jenkins/Harbor snapshot 的 pipeline run。
7. 如果进入真实 Jenkins 构建，优先修 Jenkinsfile/Jib HTTP Harbor 凭据问题，而不是 Jenkins RBAC。

关键文件：
- api/api/app/model/deployProfile.go
- api/api/app/service/deployProfile.go
- api/api/app/controller/deployProfile.go
- api/api/app/service/deployProfile_test.go
- api/api/deploy/service/agentBuildDeploy.go
- api/api/deploy/service/agentBuildDeploy_test.go
- api/api/deploy/service/deploy.go
- api/api/deploy/model/deploy.go
- api/api/deploy/model/approverAllowlist.go
- api/api/deploy/dao/approverAllowlist.go
- api/router/app/application.go
- api/router/deploy/deploy.go
- api/pkg/db/migrate.go
- web/src/api/app.js
- web/src/views/app/application.vue
- web/src/views/system/Admin.vue

外部参考：
- /home/kchou/Code/pukka-gitops/scripts/mirror-images.sh
- /home/kchou/Code/pukka-gitops/CLUSTER-ACCESS.md

注意：仓库存在较多历史 dirty/untracked 文件。只处理本任务相关文件，不要随意 revert unrelated changes。
```

## 10. 重新规划后的任务拆分

### Phase 0：范围冻结与方案去复杂化

目标：明确 v1 只做“新项目 + dev/test namespace + 自然语言触发构建部署”。

任务：

- 明确不做：GitLab repo 自动创建、Jenkinsfile 自动生成、多集群生产发布矩阵、StatefulSet、复杂 Harbor 项目治理。
- 明确保留：AutoOps Profile、Agent API、OA 审批、Jenkinsfile 构建、Harbor scan、GitOps/ArgoCD 部署、钉钉结果闭环。
- 输出一页 v1 scope 文档，防止后续继续扩散。

验收：

- 文档里能清楚回答“为什么不是 Jenkins 权限问题”。
- 文档里能清楚回答“AutoOps 管什么，Hermes 管什么，Jenkinsfile 管什么”。

### Phase 1：AutoOps 配置闭环补齐

目标：管理员可以不写 SQL，通过 UI 配出一个新项目 dev/test 自动构建部署所需信息。

任务：

- 完善/确认用户 DingTalk UserID 配置说明。
- 完善/确认 Jenkins/Harbor config_account 配置说明。
- 完善/确认 ClusterTarget dev/test 配置说明。
- 完善/确认 App Deploy Profile dev/test 配置说明。
- 评估是否需要在 UI 暴露 `buildParams`、`scanPolicy`、`envVars`、`resources` 的简版 JSON 编辑区。

验收：

- 能配置 `java-demo` 的 dev/test profile。
- Profile validate 通过。
- approver 有真实 DingTalk UserID。

### Phase 2：Hermes/Agent API Contract 固化

目标：Hermes 自然语言解析后的结构化调用稳定可测。

任务：

- 固化请求字段：`requesterExternalType/requesterExternalId/applicationCode/env/gitRef/reason/chatContext/buildParams`。
- 定义缺参追问规则：缺应用、缺环境、缺分支时 Hermes 如何追问。
- 定义 AutoOps 错误回写格式：未绑定 DingTalk、Profile 不存在、审批人未绑定、Jenkins/Harbor 未配置等。
- 写 curl 示例和 Hermes skill 调用示例。

验收：

- 用 curl 可创建 build_deploy request。
- 错误场景返回信息对开发可理解。

### Phase 3：真实 E2E Smoke

目标：跑通 `java-demo` 从钉钉/Hermes 请求到部署结果闭环。

任务：

- 准备 dev/test namespace。
- 确认 Jenkins job 和 Jenkinsfile。
- 修复 Jib HTTP Harbor 推送问题。
- 确认 Harbor project/repository 和 scanner。
- 用 Agent API 创建 request。
- 审批通过后触发 Jenkins。
- 确认 Harbor artifact + scan。
- 确认 GitOps/ArgoCD 更新并部署。
- 确认结果能被 Hermes/钉钉展示。

验收：

- `applicationCode=java-demo, env=dev, gitRef=main` 能完整跑通一次。
- 至少能拿到 requestNo、Jenkins build URL、Harbor artifact、部署 namespace/release、访问地址。

### Phase 4：收敛和硬化

目标：把 smoke 中暴露的问题固化到 AutoOps 中，避免手工补洞。

任务：

- cluster target envType 与 profile env 映射校验。
- Profile 高级字段 UI 简版支持。
- pipeline stage 状态更细：approval/build/scan/deploy/notify。
- Hermes status 查询或 webhook 回调策略明确。
- 补充服务端集成测试。

验收：

- dev/test 都可重复触发。
- 常见错误能被 Hermes 清晰回写。
- 不需要人工进入 Jenkins/Harbor/数据库补配置。

# AutoOps Deploy 控制平面交接与 VibeCoding 执行手册

> 最后更新：2026-04-17  
> 适用场景：在新环境或新会话中继续推进 AutoOps `deploy` 控制平面、钉钉 OA 审批、Hermes 对接与结果通知。

## 文档目的

本文档用于把当前这轮实现的「完整计划、当前进度、关键上下文、执行边界、验证命令、推荐 Prompt」收敛到一个入口，便于在新环境中继续执行，不需要重新梳理上下文。

建议先阅读：

1. `docs/deploy-control-plane.md`
2. `docs/dingtalk-oa-template.md`
3. `progress.md`
4. 本文档

---

## 一、目标总览

本轮目标不是做一个新的发布系统，而是在现有 AutoOps 基础上补齐一个平台化运维控制平面，满足以下链路：

```text
钉钉群聊用户 @ Hermes
  → Hermes 调 AutoOps agent API
  → AutoOps 建 DeployRequest
  → AutoOps 发起钉钉 OA 审批
  → 审批人在钉钉 OA 中同意 / 拒绝
  → AutoOps 同步审批结果
  → 审批通过后 AutoOps 自动执行
      ├─ direct mode：直连 Kubernetes
      └─ gitops mode：写 pukka-gitops 并 push
  → AutoOps 结果通知机器人回钉钉群聊
```

---

## 二、当前状态总表

### 已完成

#### 1. AutoOps 后端能力

- 已新增独立 `deploy` 领域模块：
  - `api/api/deploy/controller/`
  - `api/api/deploy/dao/`
  - `api/api/deploy/model/`
  - `api/api/deploy/service/`
- 已落地核心模型：
  - `ClusterTarget`
  - `DeployRequest`
  - `ApprovalRecord`
  - `ExecutionRecord`
  - `ResourceOwner`
- 已注册页面接口与 Agent 接口：
  - 页面接口：集群目标、申请、审批、执行、清理
  - Agent 接口：Hermes 可调用的 Bearer Token 入口

#### 2. Direct mode

- 已支持 `Pod` / `Deployment` / 可选 `Service`
- 已支持受限 kubeconfig 引用：`direct_kubeconfig_ref`
- 已支持 Kubernetes 权限探测：`SelfSubjectAccessReview`
- 已支持真实资源 apply
- 已支持 TTL 回收：
  - 手动清理
  - 调度器自动清理
- 已强制 direct 资源标签 / 注解：
  - `app.kubernetes.io/managed-by=autoops`
  - `autoops.io/owner-system=direct`
  - `autoops.io/deploy-mode=direct`
  - `autoops.io/request-id=<requestNo>`
  - `autoops.io/ttl-expire-at=<RFC3339>`

#### 3. GitOps mode

- 已支持 release 文件渲染
- 已支持写本地 `pukka-gitops` working tree
- 已支持本地 `git add / commit / push`
- 已支持 GitOps release 删除 / 下线
- 已支持 `--reconcile-gitops` 对账报告
- 已落地 `autoops-managed-releases` 脚手架到 GitOps 仓库
- 已完成 GitOps smoke test

#### 4. 审批链路

- 已实现钉钉 OA client 抽象
- 已实现审批实例发起
- 已实现审批状态同步：
  - 手动同步
  - 调度器同步
- 已修正 `approvers` 为钉钉要求的对象数组格式
- 已加 `access_token` 缓存，避免重复请求钉钉 token
- 已实现「审批通过后自动执行」：
  - 手动同步接口触发自动执行
  - 调度器同步触发自动执行

#### 5. 通知链路

- 已新增 `DeployNotification`
- 已实现 `chatContextJson` 正式消费
- 已抽取 `api/pkg/dingtalkbot`
- 已实现结果通知发送器
- 已提供通知记录查询 API
- 已在发布中心页面增加 `通知记录` 弹窗

#### 6. Hermes 集成

- 已新增 Agent 精简状态接口
- 已新增 Hermes skill 骨架：
  - `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/SKILL.md`
  - `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/README.md`
  - `~/.hermes/hermes-agent/skills/devops/deploy-via-autoops/examples/confirm-flow.md`
- 已在 `~/.hermes/config.yaml` 加 `integrations.autoops` 占位配置

#### 7. 前端

- 已新增 `K8s Release Center`
- 已新增 `web/src/api/deploy.js`
- 已新增路由 `/k8s/release-center`
- 已新增 GitOps `下线` 按钮
- 已新增 `通知记录` 弹窗

#### 8. 文档

- 已更新：
  - `docs/deploy-control-plane.md`
  - `docs/dingtalk-oa-template.md`
  - `docs/dingtalk-userid-bootstrap.md`
  - `docs/architecture.md`
  - `docs/backend-guide.md`
  - `docs/frontend-guide.md`
  - `docs/deployment.md`
  - `docs/api-changelog.md`
  - `README.md`
  - `progress.md`

### 未完成

#### P0：真实钉钉 OA 实例仍未完全打通

当前代码已经能真实调用钉钉审批实例接口，但还缺一条关键前置条件：

- `originatorUserId` 必须是真实钉钉成员 `userId`
- `approvers[].userIds` 也必须是真实钉钉成员 `userId`

当前本地测试数据仍存在占位值：

- `sys_admin.id=89`
- `sys_admin.dingtalk_user_id='smoke-dingtalk-admin'`

这会导致钉钉返回：

- `invalidParameter`
- 含义为：企业 ID、模板 code、发起人状态或成员身份参数不合法

另外，2026-04-17 本地核验结果显示，应用尚未开通通讯录读权限：

- `qyapi_get_department_member`

因此当前还不能依赖 AutoOps 自动发现真实成员 `userId`。

#### P1：结果回群机器人未实现

目标行为：

- Hermes 在群聊里发起申请
- AutoOps 审批并执行
- AutoOps 自己的钉钉机器人把结果回发到原群聊

当前状态：

- `DeployRequest.chatContextJson` 已预留
- 结果通知模型、发送器、查询 API 与前端弹窗已落代码
- 仍需用真实 webhook 做一次运行态验证

#### P1：Hermes 仓库内 custom tool / skill 未实现

当前共识：

- Hermes 本身没有业务接口
- AutoOps 暴露 API
- Hermes 后续通过 custom tool / skill 调 AutoOps agent API

当前状态：

- AutoOps Agent API 已有
- Hermes skill 骨架与本地配置占位已补
- 仍需在真实 Hermes 会话里走一遍确认链路

#### P2：GitOps 回滚 / 下线未实现

当前已有：

- GitOps create / write / commit / push
- GitOps delete / rollback API 与前端按钮
- GitOps working tree reconcile CLI

当前未有：

- 回滚指定 revision
- 失败后状态回写与二次回收

---

## 三、关键结论

### 1. Direct mode 可行，但只能小范围长期存在

适合：

- 临时测试环境
- 短生命周期资源
- 独立 `ao-direct-*` namespace
- TTL 明确、自动回收的试验性部署

不适合：

- 长期核心业务
- 需要多人协同维护的正式环境
- 依赖 Git 审计、回滚、变更评审的主业务发布

### 2. GitOps 与 direct mode 必须严格单 owner

当前已落地原则：

- 同一 `(cluster, namespace, kind, name)` 只能有一个 owner
- 通过 `deploy_resource_owner` 保证 owner 唯一
- 不允许 GitOps 与 direct 双主控制同一资源

### 3. 当前真实阻塞不在部署执行，而在钉钉身份映射

已验证通过：

- direct 真实执行
- gitops 真实提交
- 审批通过后自动执行

尚未验证通过：

- 用真实企业成员 `userId` 发起审批实例

---

## 四、关键文件与模块

### 后端核心

- `api/api/deploy/model/deploy.go`
- `api/api/deploy/dao/deploy.go`
- `api/api/deploy/controller/deploy.go`
- `api/api/deploy/service/deploy.go`
- `api/api/deploy/service/dingtalkApproval.go`
- `api/api/deploy/service/directCredential.go`
- `api/api/deploy/service/directExecutor.go`
- `api/api/deploy/service/directManifest.go`
- `api/api/deploy/service/gitopsRender.go`
- `api/api/deploy/service/gitopsWriter.go`
- `api/router/deploy/deploy.go`
- `api/scheduler/deployApprovalSyncScheduler.go`
- `api/scheduler/deployTTLReaper.go`

### 前端

- `web/src/api/deploy.js`
- `web/src/views/K8s/K8sReleaseCenter.vue`
- `web/src/router/k8s.js`

### 文档

- `docs/deploy-control-plane.md`
- `docs/dingtalk-oa-template.md`
- `docs/vibecoding-handoff.md`
- `progress.md`

### 外部参考仓库

- `/home/kchou/Code/pukka-gitops`
- `/home/kchou/Code/pukka-gitops/scripts/mirror-images.sh`

### Hermes 参考目录

- `~/.hermes/hermes-agent`

---

## 五、本地环境与运行上下文

### 代码仓库

- AutoOps：`/home/kchou/Code/AutoOps`
- pukka-gitops：`/home/kchou/Code/pukka-gitops`

### 工具链

- Go：`mise exec go@1.25.0 -- ...`
- Node：`mise exec node@24.14.1 -- ...`

### Docker 服务

- API：`http://127.0.0.1:18000`
- Web：`http://127.0.0.1:18088`
- PostgreSQL：`127.0.0.1:15432`

### 当前本地配置现状

- `api/config.yaml` 与 `docker/api/config.yaml` 为本地运行配置
- 这两个文件包含本地集成配置，但不应把密钥重新写入文档或提交
- `docker/api/config.yaml` 需要与容器实际运行保持一致
- 当前本地 `deploy_bot` 未配置真实 webhook，因此通知发送会落 `deploy_notification.status=skipped`
- 2026-04-17 一次 `docker build` 因拉取 `docker.io/library/alpine:3.23` 超时失败；本机当前运行中的 `devops-api` 通过静态编译二进制热替换方式更新到了最新代码

### 当前本地测试数据

- `deploy_cluster_target.id=1`：direct smoke target
- `sys_admin.id=89`：管理员测试账号
- `/tmp/admin.token`：本地测试 JWT

说明：

- 上述数据仅用于当前机器上的联调环境
- 在新环境中不要假设这些 ID 一定存在，应先检查数据库现状

---

## 六、最新验证记录

### 已验证通过

#### 后端测试

```bash
cd api
mise exec go@1.25.0 -- go test ./...
```

结果：

- 2026-04-17 通过

#### direct 执行回归

已在新构建镜像下验证：

- 手工将测试申请改为 `approved`
- 调 `POST /api/v1/deploy/requests/:id/execute`
- direct 资源创建成功
- 随后执行 cleanup 成功

#### Hermes 契约脚本

```bash
AGENT_TOKEN=change-me-agent-token python3 scripts/deploy/hermes-contract-test.py
```

结果：

- 2026-04-17 已成功创建申请并读取精简状态

#### GitOps reconcile

```bash
cd api
mise exec go@1.25.0 -- go run . -c /tmp/autoops-reconcile-config.yaml --reconcile-gitops
```

结果：

- 2026-04-17 已输出 `matched` 报告

#### 服务状态

```bash
docker compose -f docker/docker-compose.yml ps
```

结果：

- `devops-api` healthy
- `postgres` healthy
- `valkey` healthy

### 尚未通过

#### 真实钉钉 OA 实例创建

当前钉钉返回：

- `invalidParameter`

高概率原因：

1. `sys_admin.dingtalk_user_id` 仍为占位值
2. 发起人 `originatorUserId` 不是企业真实成员
3. 审批人 `approvers[].userIds` 不是企业真实成员
4. 应用没有通讯录读权限，无法自动校验成员

#### 真实 webhook 回群

当前结果：

- 通知记录已落库
- 本地因 `deploy_bot` 未配置真实 webhook，记录状态为 `skipped`
- 还未做真实群聊可见性验证

---

## 七、推荐继续顺序

### 阶段 A：先打通真实 OA 审批实例

目标：

- 让 AutoOps 能真实发起一个钉钉 OA 审批实例

步骤：

1. 检查 `sys_admin` 表中的测试审批人
2. 将 `dingtalk_user_id` 改为真实企业成员 `userId`
3. 如需要，申请通讯录读权限
4. 重试 Agent 接口：
   - 创建申请
   - 重发审批
5. 确认 `approvalDispatchStatus=dispatched`
6. 确认 `dingtalkProcessInstanceId` 已落库

完成标准：

- 钉钉 OA 中可见真实审批单

### 阶段 B：验证「审批通过 → 自动执行」

目标：

- 不再依赖手工执行 API

步骤：

1. 在钉钉 OA 中审批通过
2. 调用同步接口或等待调度器
3. 确认 AutoOps 自动执行
4. direct mode 检查 K8s 资源
5. gitops mode 检查 commit / push / ArgoCD 同步

完成标准：

- `approvalStatus=approved`
- `executionStatus=succeeded`
- `ExecutionRecord` 已落库

### 阶段 C：实现结果回群机器人

目标：

- 部署结果回写到原钉钉群聊

最小范围建议：

1. 先只支持文本通知
2. 只在执行完成后通知一次
3. 只处理成功 / 失败两种状态
4. 复用 `chatContextJson` 保存群聊上下文

完成标准：

- 群里收到申请编号、模式、命名空间、结果摘要
- `deploy_notification.status=sent`

### 阶段 D：Hermes custom tool / skill

目标：

- Hermes 不直接懂业务，只调用 AutoOps Agent API

最小范围建议：

1. Hermes 收到自然语言
2. 解析最少参数：
   - cluster target
   - deploy mode
   - image
   - release name
   - reason
3. 调 AutoOps
4. 返回申请单号与当前状态

当前代码基础已经具备：

- Agent 创建申请接口
- Agent 精简状态接口
- Hermes skill skeleton
- 本地配置占位

---

## 八、执行边界与约束

继续实现时必须遵守：

1. 不绕过审批
2. 不让 Hermes 直接操作 Kubernetes 或 Git
3. direct mode 创建的资源必须带完整标签 / 注解
4. 同一资源只能有一个 owner
5. GitOps 与 direct mode 不能双主控制同一资源
6. 优先复用现有模块，不重复造轮子
7. 遇到 GitOps 边界、权限边界、审批边界不明确时先停下来确认

---

## 九、新环境启动检查清单

在新环境接手时，先执行以下检查：

### 1. 读文档

- `docs/deploy-control-plane.md`
- `docs/dingtalk-oa-template.md`
- `docs/vibecoding-handoff.md`
- `progress.md`

### 2. 看代码结构

```bash
find api/api/deploy -maxdepth 3 -type f | sort
find api/router/deploy -maxdepth 2 -type f | sort
find api/scheduler -maxdepth 1 -type f | sort
```

### 3. 跑后端测试

```bash
cd api
mise exec go@1.25.0 -- go test ./...
```

### 4. 看 Docker 服务

```bash
docker compose -f docker/docker-compose.yml ps
docker logs --tail 100 devops-api
```

### 5. 看本地数据库关键数据

```bash
docker exec devops-postgres psql -U devops -d autoops -c "select id, name, direct_kubeconfig_ref from deploy_cluster_target order by id;"
docker exec devops-postgres psql -U devops -d autoops -c "select id, username, nickname, dingtalk_user_id from sys_admin order by id;"
docker exec devops-postgres psql -U devops -d autoops -c "select id, request_no, approval_status, execution_status, request_status from deploy_request order by id desc limit 20;"
```

### 6. 核实钉钉前置条件

- `processCode` 是否已配置
- 字段映射是否一致
- 审批人是否为真实钉钉成员 `userId`
- 是否具备通讯录读权限

---

## 十、推荐 Prompt

以下 Prompt 可直接给新环境中的编码代理使用。

### Prompt A：完整接手

```text
当前项目是 /home/kchou/Code/AutoOps，参考 GitOps 仓库是 /home/kchou/Code/pukka-gitops。

先不要盲目编码，先阅读以下文档并基于真实代码继续执行：
- docs/deploy-control-plane.md
- docs/dingtalk-oa-template.md
- docs/vibecoding-handoff.md
- progress.md

任务目标是继续推进 AutoOps deploy 控制平面，不要重做已经完成的部分。当前已完成：
- deploy 模块、direct mode、gitops mode
- 钉钉 OA client、审批同步
- 审批通过后自动执行

当前主要未完成项：
1. 用真实钉钉 userId 打通 OA 审批实例
2. AutoOps 结果通知机器人回群聊
3. Hermes custom tool / skill 对接 AutoOps Agent API

硬性约束：
1. 同一资源只能有一个 owner，不能 GitOps/direct 双主控制
2. 所有 direct mode 资源都必须带 managed-by / owner-system / deploy-mode / request-id / ttl 注解标签
3. 不绕过审批
4. 不让 Hermes 直接操作 Git 或 Kubernetes
5. 优先复用现有 deploy、RBAC、审计、scheduler、configcenter 模块

执行方式：
- 一次只做一个小任务
- 先读对应文件，再改
- 每完成一个任务就自检
- 如果权限边界、GitOps 边界或审批边界不明确，暂停并说明问题
```

### Prompt B：只打通钉钉 OA

```text
继续 AutoOps 的钉钉 OA 联调，只处理 OA 审批实例这一个任务，不要扩范围。

先阅读：
- docs/dingtalk-oa-template.md
- docs/vibecoding-handoff.md
- api/api/deploy/service/dingtalkApproval.go
- api/api/deploy/service/deploy.go

当前现状：
- approvers 已改为对象数组
- 审批通过自动执行已实现
- 当前真实阻塞是 sys_admin.dingtalk_user_id 还是占位值，钉钉要求真实 userId

目标：
1. 核查当前本地配置与数据库映射
2. 用真实 userId 重试发起审批实例
3. 给出最小修复方案
4. 不要动无关模块

验证要求：
- 给出实际调用命令
- 给出实际返回结果
- 如果失败，明确是代码问题还是环境问题
```

### Prompt C：实现结果回群机器人

```text
继续 AutoOps deploy 控制平面，只做“执行结果通知钉钉群聊”这个 MVP，不要扩到复杂卡片交互。

先阅读：
- docs/deploy-control-plane.md
- docs/vibecoding-handoff.md
- api/api/deploy/model/deploy.go
- api/api/deploy/service/deploy.go
- 现有钉钉机器人相关代码

目标：
1. 复用 chatContextJson 保存群聊上下文
2. 在 direct / gitops 执行结束后发送一条文本结果通知
3. 先只支持成功 / 失败
4. 输出最小实现方案、涉及文件、验证命令

约束：
- 不改变审批边界
- 不把通知逻辑耦合进 Hermes
- 不引入新依赖，优先复用现有钉钉机器人能力
```

### Prompt D：对接 Hermes custom tool / skill

```text
继续 AutoOps / Hermes 对接，只处理 Hermes 调 AutoOps Agent API 的最小闭环。

参考：
- ~/.hermes/hermes-agent
- docs/deploy-control-plane.md
- docs/vibecoding-handoff.md
- AutoOps 的 agent deploy API

目标：
1. 在 Hermes 侧实现一个最小 custom tool / skill
2. 输入自然语言部署意图
3. 调 AutoOps Agent API 创建 DeployRequest
4. 返回 requestNo、审批状态、执行状态摘要

约束：
- Hermes 不直接调用 Kubernetes
- Hermes 不直接操作 GitOps 仓库
- 所有部署都必须走 AutoOps 审批与执行链路
```

---

## 十一、一句话交接结论

当前 AutoOps deploy 控制平面的主体能力已经落地，真实运行链路里最大的剩余阻塞不是部署执行，而是钉钉真实成员 `userId` 映射。下一步优先打通 OA 审批实例，再补结果回群与 Hermes 侧适配。

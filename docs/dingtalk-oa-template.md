# DingTalk OA 审批模板接入方案

## 目标

本文档用于指导 AutoOps 接入钉钉原生 OA 审批模板，支撑以下链路：

1. AutoOps 创建部署申请
2. AutoOps 发起钉钉 OA 审批实例
3. 审批人在钉钉待办 / OA 审批中查看并处理审批
4. AutoOps 查询审批实例状态并回写
5. 审批通过后，AutoOps 执行 direct / GitOps 发布

当前 AutoOps 已具备：

- 部署申请建模与执行闸门
- 钉钉 OA 审批客户端抽象
- 审批状态同步能力

但要真正发起审批实例，仍需在钉钉侧准备一个可用的审批模板，并拿到 `processCode`。

---

## 先分清两个方案

钉钉里和“审批”相关，至少有两种容易混淆的接入方式：

### 方案 A：原生 OA 审批模板 + 审批实例

特点：

- 在 **OA 审批管理后台** 手工创建审批表单 / 审批模板
- AutoOps 只负责：
  - 发起审批实例
  - 查询审批实例状态
- 审批人直接在钉钉 OA 审批里点同意 / 拒绝
- **不需要** AutoOps 自己创建“待办卡片”

适合：

- 你现在已经能进入 OA 审批管理后台
- 想先把“审批模板 + 审批实例 + 状态回写”跑通

当前 AutoOps 代码更接近这个方案。

### 方案 B：流程中心集成审批卡片 / 待办

特点：

- 通过 `processCentres` 系列 API 创建 / 更新自有审批模板
- 除了审批实例，还要显式创建流程中心待处理任务
- 钉钉待办中心 / 审批详情页中会出现集成出来的审批任务和按钮

适合：

- 你明确要做“审批卡片 / 待办卡片 / 流程中心集成”
- 你希望把三方审批任务同步进钉钉待办中心

这个方案比方案 A 更复杂，所需权限也更多。

### 你现在应该选哪个

如果你的目标是：

- 先让 AutoOps 有一个可用的部署审批流
- 审批人在钉钉里处理审批
- AutoOps 再同步审批结果

那么建议先做 **方案 A**。

如果你的目标是：

- 必须看到“流程中心待办卡片 / 审批卡片”
- 或必须让三方审批任务同步进钉钉待办

那就要做 **方案 B**。

---

## 必需权限

### 方案 A（原生 OA 审批模板）当前实现必需

1. **工作流模板写权限**
   - 用途：创建或更新 OA 审批模板
   - API：
     - `POST /v1.0/workflow/processCentres/schemas`
   - 说明：
     - 创建完成后返回的 `processCode` 必须保存下来，后续发起审批实例依赖这个值

2. **工作流实例写权限**
   - 用途：发起审批实例
   - API：
     - `POST /v1.0/workflow/processInstances`

3. **工作流实例读权限**
   - 用途：查询审批实例状态
   - API：
     - `GET /v1.0/workflow/processInstances?processInstanceId=...`

### 当前联调状态（2026-04）

当前 AutoOps 与钉钉 OA 的联调进度如下：

- 已拿到真实 `processCode`
- 已按推荐字段名配置 `field_mappings`
- 已验证 AutoOps 能拿到 access token
- 已验证 AutoOps 能真实调用“发起审批实例”接口
- 已修正 `approvers` 参数为钉钉要求的对象数组格式
- 已实现审批通过后的自动执行入口
- 当前正在联调：
  - 真实钉钉 `userId` 映射
  - 审批实例成功创建后的状态同步

这说明：

- 现在不再是“还没接 OA”
- 而是“已经进入真实 OA 用户身份与模板参数联调阶段”

当前已知阻塞：

- AutoOps 测试数据里的 `sys_admin.dingtalk_user_id` 仍是 `smoke-dingtalk-admin` 这类占位值
- 钉钉 `originatorUserId` 和 `approvers[].userIds` 必须是企业内真实有效的钉钉 `userId`
- 若钉钉返回 `processInstanceInvalidParameter`，并提示“发起人不在发起部门中 / 发起部门ID错误”，则还需要给 AutoOps 配置发起人的真实钉钉部门 ID：`dingtalkApproval.originator_dept_id`
- 如果希望 AutoOps 自动从通讯录查询 `userId`，应用还需要开通 `qyapi_get_department_member` 等通讯录读权限
- 如果不开放通讯录读权限，也可以先手工把测试审批人的真实 `userId` 写入 AutoOps 用户表

配套补录文档：

- `docs/dingtalk-userid-bootstrap.md`

### 方案 A 当前实现非必需

1. **审批流数据管理权限**
   - 仅在你要把三方审批进一步同步为钉钉待办任务时需要
   - 当前 AutoOps 并未实现待办任务同步，因此不是必需

2. **审批任务操作能力**
   - 如需由 AutoOps 通过 API 代替审批人执行“同意/拒绝”时才需要额外使用
   - 当前设计要求审批动作由审批人本人在钉钉完成，因此不建议启用这条能力

### 方案 B（流程中心集成审批卡片 / 待办）额外需要

如果你要做流程中心集成审批卡片 / 待办，还需要：

1. **工作流模板读权限**
   - 用途：按模板名称查 `processCode`、查询表单 schema
   - 典型接口：
     - `GET /v1.0/workflow/processCentres/schemaNames/processCodes`
     - `GET /v1.0/workflow/forms/schemas/processCodes`

2. **审批流数据管理权限**
   - 用途：创建 / 查询 / 更新流程中心待办任务
   - 典型接口：
     - `POST /v1.0/workflow/processCentres/tasks`
     - `GET /v1.0/workflow/processCentres/todoTasks`
     - `PUT /v1.0/workflow/processCentres/tasks`

也就是说：

- 你看到“审批卡片 / 待办卡片”这个目标，更接近 **流程中心集成**
- 这不是当前 AutoOps 代码已经落地的路径

---

## 推荐应用载体

钉钉开放平台里，“企业内部应用”不是一种具体应用类型；你能创建的企业内部应用载体通常包括：

- 酷应用
- 机器人
- 小程序
- 网页应用

本方案推荐使用：

- **网页应用**

原因：

- 与当前 AutoOps 组织内审批场景最匹配
- 支持工作流模板写权限
- 支持工作流实例读写权限
- 更适合企业内定向审批人与内部待办流转

说明：

- “审批 / 工作流”在这里是应用可申请的能力与接口权限，不是企业内部应用类型。
- Hermes 群聊机器人与 AutoOps 审批应用建议分开：Hermes 用机器人承接对话，AutoOps 审批集成用网页应用承接 OA 工作流权限。

---

## 推荐审批模板设计

建议创建一个 **单级审批** 模板，AutoOps 在发起审批实例时动态传入审批人。

### 模板名称建议

- `AutoOps 部署申请`

### 适用目标链路

该模板用于承接以下链路：

```text
钉钉群聊用户 @ Hermes
  → Hermes 调 AutoOps
  → AutoOps 建申请
  → AutoOps 发起 OA 审批实例
  → 审批人在钉钉 OA 中审批
  → AutoOps 查询审批结果
  → 审批通过后继续执行
```

### 模板字段建议

以下字段建议直接作为模板中的表单控件名称（即 AutoOps 后续 `field_mappings` 要填写的值）：

| 显示名称 / 字段名 | 是否必填 | 说明 |
|---|---:|---|
| `申请单号` | 是 | AutoOps `requestNo` |
| `发布名称` | 是 | AutoOps `releaseName` |
| `部署目标` | 是 | `ClusterTarget.name` |
| `部署模式` | 是 | `gitops` / `direct` |
| `资源类型` | 是 | `deployment` / `pod` / `service` |
| `镜像` | 是 | 镜像地址 |
| `命名空间` | 是 | 最终 namespace |
| `TTL小时` | 否 | direct 模式时使用 |
| `申请原因` | 是 | 申请说明 |

> 建议以上字段名称保持稳定，不要随意改动。
> 当前 AutoOps 代码里的 `dingtalkApproval.field_mappings.*` 就是拿这些字段名称做映射。

### 完整推荐字段配置（手工创建时照着填）

下面是首版最推荐的“后台手工建模板”配置方式。

| 字段显示名 | 字段类型建议 | 必填 | AutoOps 映射键 | 备注 |
|---|---|---:|---|---|
| 申请单号 | 单行文本 | 是 | `request_no` | 用于唯一追踪部署申请 |
| 发布名称 | 单行文本 | 是 | `release_name` | 对应 `releaseName` |
| 部署目标 | 单行文本 | 是 | `cluster_target` | 对应 `ClusterTarget.name` |
| 部署模式 | 单行文本 | 是 | `deploy_mode` | `gitops` / `direct` |
| 资源类型 | 单行文本 | 是 | `resource_type` | `deployment` / `pod` / `service` |
| 镜像 | 单行文本 | 是 | `image` | 容器镜像地址 |
| 命名空间 | 单行文本 | 是 | `namespace` | 最终 namespace |
| TTL小时 | 单行文本 或 数字 | 否 | `ttl_hours` | 仅 direct 模式使用 |
| 申请原因 | 多行文本 | 是 | `reason` | 人工审批判断依据 |

#### 字段默认建议

- 字段顺序就按上表顺序
- 所有字段建议对审批人可见
- 首版不建议加附件、明细子表、联系人、日期控件、条件分支字段

#### 字段命名要求

请尽量**严格使用上面中文字段名**。

例如：

```text
申请单号
发布名称
部署目标
部署模式
资源类型
镜像
命名空间
TTL小时
申请原因
```

如果你实际在钉钉后台里用了不同名称，后续必须把这些实际名称填回：

```yaml
dingtalkApproval:
  field_mappings:
    request_no: "..."
    release_name: "..."
    cluster_target: "..."
    deploy_mode: "..."
    resource_type: "..."
    image: "..."
    namespace: "..."
    ttl_hours: "..."
    reason: "..."
```

---

## 审批流建议

### 首版建议

- 单级审批
- 审批人由 AutoOps 在发起审批实例时通过 `approvers` 动态传入
- 不在审批模板后台固定写死审批人

### 原因

当前 AutoOps 已设计：

- 请求可显式指定审批人
- 若未指定，则回退到 `ClusterTarget.defaultApproverAdminId`

因此模板应该尽量只负责“承载审批单”，不要在钉钉后台再做一套独立的审批人路由逻辑。

---

## AutoOps 配置映射

模板创建后，需要把以下配置填入 AutoOps：

### `api/config.yaml` / `docker/api/config.yaml`

```yaml
dingtalkApproval:
  client_id: "your_app_key"
  client_secret: "your_app_secret"
  process_code: "PROC-XXXXXXXXXXXXXXXX"
  microapp_agent_id: 0
  redirect_url: ""
  poll_interval_seconds: 30
  field_mappings:
    request_no: "申请单号"
    release_name: "发布名称"
    cluster_target: "部署目标"
    deploy_mode: "部署模式"
    resource_type: "资源类型"
    image: "镜像"
    namespace: "命名空间"
    ttl_hours: "TTL小时"
    reason: "申请原因"
```

### 字段解释

- `process_code`
  - 钉钉审批模板唯一标识
- `field_mappings.*`
  - 必须与钉钉 OA 模板中配置的表单字段名称完全一致
- `poll_interval_seconds`
  - AutoOps 后台轮询审批状态的周期

---

## 当前代码对模板的依赖点

### 模板创建 / 更新

- 当前尚未在 AutoOps 中实现“调用 API 自动创建模板”
- 当前已实现“使用 `processCode` 发起审批实例”

因此首版推荐流程是：

1. 先在钉钉侧准备模板
2. 拿到 `processCode`
3. 回填 AutoOps 配置
4. 用 AutoOps 发起审批实例

### 代码中已对接的位置

- 钉钉审批客户端：
  - `api/api/deploy/service/dingtalkApproval.go`
- 部署申请发起审批：
  - `api/api/deploy/service/deploy.go`
- 审批状态同步：
  - `api/api/deploy/service/deploy.go`
  - `api/scheduler/deployApprovalSyncScheduler.go`

---

## 推荐实施步骤

### 步骤 1：创建 AutoOps 审批应用

在钉钉开放平台：

1. 进入开发者后台
2. 选择企业内部开发
3. 新建应用，推荐选择 **网页应用**
4. 应用名称建议：`AutoOps 审批`

### 步骤 2：开通权限

给 `AutoOps 审批` 应用申请：

- 工作流模板写权限
- 工作流实例写权限
- 工作流实例读权限

### 步骤 3：创建 OA 审批模板

使用上文建议字段创建一个模板：

- 模板名：`AutoOps 部署申请`
- 字段名与 `field_mappings` 一致

#### 推荐首版做法：在钉钉后台手工创建模板

如果你现在已经创建好了企业内部 **网页应用**，但还不清楚如何配置单级审批模板，建议按下面步骤操作。

### 手工创建单级审批模板（推荐）

#### 1. 进入 OA 审批管理后台

在钉钉管理后台中，进入 **OA 审批管理后台**。

你要找的入口通常叫：

- OA 审批
- 审批管理
- 审批表单
- 创建审批表单

你要创建的是 **审批表单 / 审批模板**。

不要进入：

- 机器人消息卡片
- 互动卡片模板
- 酷应用卡片
- 群机器人配置

当前链路使用的是钉钉 OA 审批实例，不是机器人互动卡片。

#### 2. 创建审批表单

在 OA 审批管理后台点击：

```text
创建审批表单
```

创建一个普通审批表单 / 审批模板：

- 模板名称：`AutoOps 部署申请`

如果系统要求选择分类，选一个最普通、最接近“通用审批 / IT 申请 / 运维审批”的分类即可。

推荐分类：

- 通用审批
- IT 审批
- 运维审批

三者任选其一即可，分类本身不会影响 AutoOps 对接。

#### 3. 添加表单字段

按下面顺序添加字段，**字段显示名称尽量完全一致**：

| 顺序 | 显示名称 | 类型建议 | 必填 | 说明 |
|---|---|---|---:|---|
| 1 | `申请单号` | 单行文本 | 是 | AutoOps `requestNo` |
| 2 | `发布名称` | 单行文本 | 是 | AutoOps `releaseName` |
| 3 | `部署目标` | 单行文本 | 是 | `ClusterTarget.name` |
| 4 | `部署模式` | 单行文本 | 是 | `gitops` / `direct` |
| 5 | `资源类型` | 单行文本 | 是 | `deployment` / `pod` / `service` |
| 6 | `镜像` | 单行文本 | 是 | 镜像地址 |
| 7 | `命名空间` | 单行文本 | 是 | 最终 namespace |
| 8 | `TTL小时` | 单行文本 或 数字 | 否 | Direct 模式时使用 |
| 9 | `申请原因` | 多行文本 | 是 | 申请说明 |

> 强烈建议字段显示名保持不变。  
> 如果你改成别的名字，后面必须同步修改 AutoOps 的 `dingtalkApproval.field_mappings.*`。

#### 推荐操作顺序

你在后台创建审批表单时，可以直接按下面顺序点：

1. 添加单行文本：`申请单号`
2. 添加单行文本：`发布名称`
3. 添加单行文本：`部署目标`
4. 添加单行文本：`部署模式`
5. 添加单行文本：`资源类型`
6. 添加单行文本：`镜像`
7. 添加单行文本：`命名空间`
8. 添加数字或单行文本：`TTL小时`
9. 添加多行文本：`申请原因`

#### 4. 配置审批流程为单级审批

审批流程部分，建议这样配：

- 只保留 **1 个审批节点**
- 不做多级会签
- 不加抄送、条件分支、加签、转审等复杂逻辑

流程形态建议：

```text
发起人 → 审批人 → 结束
```

首版建议：

- 只保留这 1 个审批节点
- 不加抄送
- 不加条件分支
- 不加会签
- 不加加签
- 不加转审
- 不做抄送群或并行审批

#### 5. 审批人配置原则

当前 AutoOps 的设计目标是：

- 审批人由 AutoOps 在发起审批实例时动态传入
- 不是在模板里永久写死某个固定审批人

因此你在审批人节点里应优先选择以下能力（如果后台有）：

- 发起时指定审批人
- 接口传入审批人
- 发起时选择审批人

如果页面没有这些选项，而是**必须先选一个审批人**，首版可以这样处理：

- 先选一个测试审批人作为占位
- 保存模板拿到 `processCode`
- 后续再根据钉钉实际能力决定是否继续动态传 `approvers`

AutoOps 当前传给钉钉的 `approvers` 格式为：

```json
[
  {
    "actionType": "AND",
    "userIds": ["<审批人钉钉 userId>"]
  }
]
```

注意事项：

- `approvers` 不是字符串，也不是简单的 `["userId"]`
- `userIds` 里的值必须是企业内部通讯录里的钉钉 `userId`
- `originatorUserId` 同样必须是真实成员 `userId`，占位值会导致钉钉返回 `invalidParameter`

常见审批人节点选项的建议：

| 看到的选项 | 建议 |
|---|---|
| `接口指定` / `API传入审批人` / `发起时指定审批人` | 优先选择 |
| `发起人自选` | 可用于测试，但不如接口指定稳定 |
| `指定成员` | 没有动态选项时先选一个测试审批人 |
| `部门主管` / `角色` | 不建议作为首版默认方案 |
| `表单内联系人` | 只有当模板里额外增加“审批人”联系人字段时才考虑 |

> 如果你在这一步看到的选项不确定，请把界面上的原文发我，我可以继续帮你判断。

#### 当前最推荐的选择策略

优先级从高到低：

1. `接口指定`
2. `API传入审批人`
3. `发起时指定审批人`
4. `发起人自选`
5. `指定成员（测试占位）`

如果后台**只能**固定审批人：

- 先用 `指定成员`
- 选择一个测试审批人（例如管理员自己）
- 先把 OA 模板与实例链路跑通
- 后续再评估是否需要升级到流程中心集成方案

#### 6. 保存并启用审批表单

保存并发布 / 启用后，系统通常会生成一个模板标识。

你必须拿到：

- `processCode`

这个值是 AutoOps 发起审批实例时必须填写的。

在后台界面中，`processCode` 也可能被展示为：

- 审批流编码
- 模板编码
- 表单编码
- 审批模板唯一标识
- processCode

如果界面上找不到，可到开发者后台或 API 调试工具中查询这个审批表单对应的 `processCode`。

#### 建好模板后你要记录的内容

至少把这 3 类信息保存下来：

1. `processCode`
2. 审批人节点配置方式（例如：接口指定 / 发起人自选 / 指定成员）
3. 实际表单字段显示名

#### 7. 记录模板字段名称

保存模板后，确认你实际创建出来的字段显示名称是否和以下内容**完全一致**：

```text
申请单号
发布名称
部署目标
部署模式
资源类型
镜像
命名空间
TTL小时
申请原因
```

如果不一致，记录下真实字段名，后续填入 AutoOps：

```yaml
dingtalkApproval:
  field_mappings:
    request_no: "..."
    release_name: "..."
    cluster_target: "..."
    deploy_mode: "..."
    resource_type: "..."
    image: "..."
    namespace: "..."
    ttl_hours: "..."
    reason: "..."
```

创建模板有两种方式：

1. **推荐首版：在钉钉管理后台手工创建模板**
   - 适合先验证链路
   - 创建后复制模板编码 `processCode`
2. **后续增强：通过开放 API 创建/更新自有审批模板**
   - API：`POST /v1.0/workflow/processCentres/schemas`
   - 适合把模板也纳入代码化管理

### 步骤 4：保存 `processCode`

无论通过开发者后台还是 API 创建模板，都必须记录：

- `processCode`

### 步骤 5：更新 AutoOps 配置

填写：

- `client_id`
- `client_secret`
- `process_code`
- `field_mappings`

### 步骤 6：验证

按如下顺序验证：

1. 创建部署申请
2. 确认 `approvalDispatchStatus` 从 `skipped` 变为 `dispatched`
3. 在钉钉 OA 审批 / 待办中看到实例
4. 同意或拒绝审批
5. 调用 AutoOps 审批同步接口或等待轮询同步
6. 校验申请状态回写

### 最终你需要提供给 AutoOps 的信息

模板建好后，至少准备以下信息：

1. `client_id`
2. `client_secret`
3. `processCode`
4. 字段名称映射（若你没有完全使用推荐名称）

最理想情况是你直接回填：

```yaml
dingtalkApproval:
  client_id: "..."
  client_secret: "..."
  process_code: "..."
  field_mappings:
    request_no: "申请单号"
    release_name: "发布名称"
    cluster_target: "部署目标"
    deploy_mode: "部署模式"
    resource_type: "资源类型"
    image: "镜像"
    namespace: "命名空间"
    ttl_hours: "TTL小时"
    reason: "申请原因"
```

### 一个完整可用的首版配置目标

如果你完全按本文档手工建模板，那么最终你希望得到的配置应该长这样：

```yaml
dingtalkApproval:
  client_id: "你的网页应用 AppKey"
  client_secret: "你的网页应用 AppSecret"
  process_code: "钉钉审批模板 processCode"
  microapp_agent_id: 0
  redirect_url: ""
  poll_interval_seconds: 30
  field_mappings:
    request_no: "申请单号"
    release_name: "发布名称"
    cluster_target: "部署目标"
    deploy_mode: "部署模式"
    resource_type: "资源类型"
    image: "镜像"
    namespace: "命名空间"
    ttl_hours: "TTL小时"
    reason: "申请原因"
```

如果审批人节点不是“接口指定”，请额外记录这个差异，后续需要评估是否对 AutoOps 发起审批实例参数做调整。

---

## 当前不建议做的事

1. **不建议把 Hermes 群聊机器人和 AutoOps 审批应用做成同一个应用**
   - 聊天入口和审批权限边界应分开

2. **不建议让 AutoOps 代替审批人直接调用“同意/拒绝审批任务”**
   - 会破坏审批边界

3. **不建议一开始就引入流程中心待办同步**
   - 会额外引入审批流数据管理权限和更复杂的数据对齐

4. **不建议把模板字段名设计成易变的文案**
   - 一旦字段名变更，`field_mappings` 就需要同步修改

---

## 后续可扩展方向

1. 在 AutoOps 内实现：
   - 创建/更新审批模板 API
   - 模板版本校验
   - 模板字段一致性自检

2. 增加：
   - 审批拒绝自动终止
   - 更完整的审批记录明细

3. 如有组织级待办统一要求，再考虑：
   - 流程中心接入
   - 钉钉待办同步

---

## 一句话结论

当前 AutoOps 接钉钉 OA 的最优落地方式是：

- 用企业内部 **网页应用** 作为 AutoOps 审批应用载体
- 申请 **工作流模板写 / 实例写 / 实例读** 三个权限
- 创建一个 **字段名称稳定的单级审批模板**
- 将返回的 `processCode` 和字段名映射填回 AutoOps 配置

这样就能把当前已经写好的审批发起与状态同步代码真正跑起来。

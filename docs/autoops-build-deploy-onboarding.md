# AutoOps 构建部署配置新手上路

为新项目配置自动构建部署，管理员需完成以下5个步骤。按顺序执行，确保每步验证通过后再进行下一步。

## 第一章：添加审批人（绑定钉钉 UserID）

路径：系统管理 → 用户管理

### 操作步骤

1. 进入「系统管理」→「用户管理」
2. 找到要设为审批人的用户，点击「编辑」
3. 在表单中找到「钉钉 UserID」字段，填入该用户的钉钉 UserID
4. 点击「保存」
5. 确认该字段已保存（可再次进入编辑查看）

### 注意事项

- 钉钉 UserID 可在钉钉管理后台「组织管理」→「成员管理」中查询
- 未填写钉钉 UserID 的用户无法作为审批人（Profile 校验会拒绝）
- 每个应用部署 Profile 只能配置一个审批人

---

## 第二章：配置 Jenkins / Harbor 凭据

路径：配置中心 → 账号授权（AccountAuth）

### Jenkins 凭据

1. 进入「配置中心」→「账号授权」
2. 点击「新建」
3. 填写表单字段：
   - **名称**：如 `jenkins-prod`
   - **类型**：选择 **Jenkins**（type=4）
   > ⚠️ UI 下拉选项 type=5 标签为"Zabbix"，但后端常量 `HarborAccountType=5`。创建 Harbor 凭据时，请选择 type=5（UI 显示 "Zabbix"），后端会将其作为 Harbor 类型验证
   - **地址**：`http://10.0.17.204`
   - **用户名**：`admin`
   - **密码/Token**：Jenkins API Token
4. 点击「保存」

### Harbor 凭据

1. 同上路径点击「新建」
2. 填写表单字段：
   - **名称**：如 `harbor-prod`
   - **类型**：选择 **Zabbix**（type=5，即后端 Harbor 类型）⚠️ UI 显示为"Zabbix"，但后端验证为 Harbor
   - **地址**：`http://10.0.17.205`
   - **用户名**：`admin`（或专用账号）
   - **密码**：Harbor 登录密码
3. 点击「保存」

### K8s Kubeconfig 凭据（Direct 模式必需）

Direct 模式通过 `directKubeconfigRef` 引用预先录入的 kubeconfig 凭据。创建步骤：

1. 同上路径点击「新建」
2. 填写表单字段：
   - **名称**：如 `dev-kubeconfig`
   - **类型**：选择 **通用账号**（type=6）
   - **地址**：K8s API Server 地址（如 `10.0.17.43:6443`）
   - **用户名**：可填任意标识（如 `dev-cluster`）
   - **密码**：粘贴完整的 kubeconfig YAML 内容（此字段加密存储）
3. 点击「保存」
4. 记下凭据 ID 或别名，在 ClusterTarget 的 `directKubeconfigRef` 字段中填入 `account:<id或别名>`

> **⚠️ 重要**：kubeconfig 内容存储在密码字段中，AES 加密。创建后不可查看原文，建议备份。

> **⚠️ 权限边界**：AutoOps Direct 模式在部署前会校验 kubeconfig 权限（`directCredential.go`）。要求如下：
> - **必须 allow**：`create namespaces`（集群级权限）、`create/delete deployments.apps`、`create/delete pods`、`create/delete services`（命名空间级）
> - **必须 deny**：`create persistentvolumeclaims`、`create ingresses.networking.k8s.io`（安全边界，防止资源蔓延）
>
> 建议使用受限制的 ServiceAccount（非 cluster-admin），仅绑定上述最小权限。如果使用 cluster-admin 或权限过宽的 kubeconfig，权限校验会因 PVC/Ingress create 被允许而**失败**。

### 注意事项

- Jenkins 和 Harbor 各需单独创建一条凭据
- 保存后在 Profile 配置中可从下拉选择这两条凭据
- 凭据类型必须正确（误选会导致 Profile validate 报"凭据类型不匹配"）

### 2.X Agent Bearer Token 配置

Hermes 通过 Agent API 调用 AutoOps 时需要 Bearer Token 认证。当前阶段 Token 在 API 配置文件中管理：

1. 编辑 `api/config.yaml`，找到或新增 `integrations.agent.bearer_token` 字段（注意：YAML 使用下划线命名，不是驼峰）：
   ```yaml
   integrations:
     agent:
       bearer_token: "<随机生成的 32 字节 base64 编码字符串>"
   ```
2. 生成 Token：
   ```bash
   openssl rand -base64 32
   ```
3. 将生成的字符串填入配置文件，保存后重启 AutoOps API
4. 验证 Token 生效：
   ```bash
   curl -s -H "Authorization: Bearer <你配置的token>" \
     http://localhost:8000/api/v1/integrations/agent/deploy-requests/nonexistent/status | jq '{code: .code, message: .message}'
   ```
   期望返回 `{"code": 404, "message": "..."}`（HTTP 状态码 200，JSON code 404，表示 Token 认证通过但记录不存在。若 HTTP 返回 401/403 则 Token 无效或未配置）

> ⚠️ v1.1 阶段 Token 管理：Token 通过配置文件管理，修改后需重启 API 才能生效。Token 不支持 UI 管理（v1.2 候选）。

### 2.Y GitLab 项目自动接入配置（可选但推荐）

如果希望用户只在 Hermes 里提供 GitLab 地址，就自动创建 Application/Profile 并部署，需要启用 `integrations.agent.project_onboarding`：

```yaml
integrations:
  agent:
    bearer_token: "<agent-token>"
    project_onboarding:
      enabled: true
      allowed_git_hosts:
        - gayhub.seeingtv.com
      shared_jenkins_job_name: autoops-springboot-build
      default_business_group_id: 1
      default_business_dept_id: 1
      default_jenkins_server_id: 3
      default_harbor_server_id: 4
      default_harbor_project: library
      default_approver_admin_id: 9
      dev_cluster_target_id: 7
      test_cluster_target_id: 8
      namespace_prefix: ao-direct
      default_service_port: 80
      default_target_port: 8080
      nodeport_access_host: "<k8s-node-ip-or-vip>"
```

配置说明：

| 字段 | 说明 |
|---|---|
| `allowed_git_hosts` | 允许自动接入的 GitLab host 白名单 |
| `shared_jenkins_job_name` | 共享参数化 Jenkins Job 名，负责 clone/build/push 镜像 |
| `default_business_group_id` / `default_business_dept_id` | 自动创建 Application 时使用的默认业务归属 |
| `default_jenkins_server_id` / `default_harbor_server_id` | AutoOps 账号授权中的 Jenkins/Harbor 凭据 ID |
| `default_harbor_project` | 默认 Harbor 项目，例如 `library` |
| `default_approver_admin_id` | 自动创建 Profile 时使用的审批人 SysAdmin ID |
| `dev_cluster_target_id` / `test_cluster_target_id` | dev/test Direct ClusterTarget ID |
| `namespace_prefix` | 自动生成 namespace 前缀，默认 `ao-direct` |
| `default_service_port` / `default_target_port` | Service 端口与容器端口默认值 |
| `nodeport_access_host` | NodePort 对外访问提示使用的节点 IP/VIP（v1.1 状态接口暂不自动返回 nodePort） |

共享 Jenkins Job 至少应支持 `GIT_URL`、`GIT_REF`、`APPLICATION_CODE`、`HARBOR_PROJECT`、`HARBOR_REPOSITORY`、`ENV`、`RELEASE_NAME` 参数，并在 Console 输出 `IMAGE_TAG=<tag>`，供 AutoOps 提取镜像标签。

用户对话示例见 `docs/springboot-demo-autoops-onboarding.md`。

---

## 第三章：配置集群部署目标（ClusterTarget）

路径：部署中心 → 集群目标（或相关 K8s 集群配置区域）

需要为 dev 和 test 各创建一条记录。

> ⚠️ **v1.1 重要变更**：默认部署模式已从 GitOps 切换为 **Direct**（AutoOps 直连 K8s API 部署）。v1.1 **不依赖 pukka-gitops 仓库或 ArgoCD**。若需保留 GitOps 能力，代码未删除但标记为 deprecated。

### dev ClusterTarget

- **名称**：如 `pukka-dev`
- **环境类型（envType）**：`dev`（下拉框选择）
- **Direct 启用**：勾选（默认）
- **直连凭据引用（directKubeconfigRef）**：**必填**，格式为 `account:<kubeconfig账户ID>`，指向在「配置中心 → 账号授权」中预先录入的 kubeconfig 账户。例如 `account:1` 或 `account:dev-kubeconfig`
- **直连命名空间前缀**：`ao-direct`（默认）
- **K8s 集群**：选择 dev 集群
- **GitOps 相关字段**（如 `gitOpsRepo`、`gitOpsBranch`、`gitOpsReleaseDir`）：**v1.1 中不再必填**，标记为 deprecated

### test ClusterTarget

- **名称**：如 `pukka-test`
- **环境类型（envType）**：`test`
- **直连凭据引用**：指向 test 集群的 kubeconfig 账户
- 其余字段同 dev

### 注意事项

- envType 字段必须正确填写（dev/test），大小写不敏感（"Dev"/"DEV" 等同 "dev"）
- dev profile 必须选 envType=dev 的 ClusterTarget，test profile 同理
- 如果已有历史 ClusterTarget 但 envType 为空或错误，需先修正
- **directKubeconfigRef 必填**：Direct 模式要求 ClusterTarget 配置有效的 kubeconfig 引用，否则部署会失败
- **Namespace 命名约束**：Direct 模式下 namespace 必须以 `ao-direct-` 开头（运行时校验）。Profile 中 namespace 为必填字段，建议填 `ao-direct-java-demo`

---

## 第四章：配置应用部署 Profile（AppDeployProfile）

路径：应用管理 → 选中应用 → 点击「部署配置」

一个应用最多可配置两个 Profile（dev 和 test 各一）。

### 新增 Profile 步骤

1. 在应用列表找到目标应用（如 java-demo），操作列点击「部署配置」
2. 点击「新增」
3. 填写 dev Profile 字段：

| 字段 | 值示例 | 说明 |
|------|--------|------|
| 环境 | `dev` | 必填 |
| 启用 | 勾选 | 启用此 Profile |
| 集群目标 | `pukka-dev` | 需选 envType=dev 的 ClusterTarget |
| Namespace | `ao-direct-java-demo` | Direct 模式要求以 `ao-direct-` 开头，**必填**（留空校验不通过） |
| Release Name | `java-demo` | Helm release 名称 |
| 资源类型 | `deployment` | 部署资源类型 |
| Jenkins 服务器 | （下拉选择） | 第二章配置的 Jenkins 凭据 |
| Jenkins Job 名 | `java-demo` | 必须与 Jenkins 中实际 Job 名一致 |
| Harbor 服务器 | （下拉选择） | 第二章配置的 Harbor 凭据 |
| Harbor 项目 | `library` | Harbor 项目名 |
| Harbor 仓库 | `java-demo` | 镜像仓库名 |
| 默认分支 | `main` | Git 默认分支 |
| 审批人 | （下拉选择） | 第一章配置了钉钉 UserID 的用户 |
| 副本数 | `1` | K8s 副本数 |
| Service 类型 | `ClusterIP` | Kubernetes Service 类型 |
| 服务端口 | `80` | 暴露的服务端口 |
| 目标端口 | `8080` | Pod 内目标端口 |
| 访问地址模板 | `http://java-demo.dev.internal` | 可选，用于访问提示 |
| 构建参数 JSON | `{"PROFILE":"local"}` | 可选，高级配置 |

4. 点击「保存」
5. 保存后点击「校验」（validate），确认返回 `valid: true`

### test Profile

重复上述步骤，仅修改以下字段：

- **环境**：改为 `test`
- **Namespace**：改为 `ao-direct-java-demo`
- **集群目标**：选 envType=test 的 ClusterTarget

---

## 第五章：常见校验失败原因与修复

| 错误信息 | 原因 | 修复步骤 |
|---------|------|---------|
| `审批人未配置钉钉 UserID` | 选中的审批人用户未在系统管理绑定钉钉 UserID | 回到第一章，为该用户填写钉钉 UserID |
| `Jenkins 凭据不存在` | Profile 里选的 Jenkins ID 已被删除 | 重新选择有效的 Jenkins 凭据 |
| `Harbor 凭据类型不匹配` | 把 Jenkins 凭据误选为 Harbor，或反之 | 检查凭据类型，选正确类型的凭据 |
| `部署目标不存在` | ClusterTarget 已被删除 | 重新配置 ClusterTarget 并选择 |
| `部署目标环境类型与 Profile 环境不匹配` | dev profile 选了 envType=test 的 ClusterTarget | 修改 ClusterTarget 的 envType，或选正确的目标 |
| `Jenkins job 未配置` | Jenkins Job 名为空 | 填写 Jenkins 中实际存在的 Job 名 |
| `namespace 未配置` | namespace 字段为空 | 填写 K8s namespace 名（需与集群中实际 namespace 对应） |

### 校验通过后

即可通过 Hermes 或 AutoOps 前端触发 build_deploy 请求，开始自动构建部署流程。

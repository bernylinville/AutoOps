# E2E Smoke 前置数据 Checklist — java-demo

更新时间：2026-04-28

在执行 P3.B（dev）/ P3.C（test）真实 E2E 前，请逐项确认以下数据已录入，并打勾。

---

## 0. 外部前置 gate（B0 — 必须先完成）

以下项不属于 AutoOps 数据库，但缺少任何一项都会导致 E2E 失败：

- [ ] **钉钉企业自建应用**已注册，获取 AgentId / AppKey / AppSecret（参考 `docs/dingtalk-userid-bootstrap.md`）
- [ ] **钉钉 OA 审批模板**已创建，模板 code 与 AutoOps 配置一致（参考 `docs/dingtalk-oa-template.md`）
- [ ] **审批人钉钉 userid** 已抓取并填入 `sys_admin.dingtalk_user_id`
- [ ] **Jenkins 凭据**：Git SSH key + Harbor robot account 已在 Jenkins 中预创建
- [ ] **Harbor 项目 + robot account** 已创建（供镜像推送/扫描）
- [ ] **dev/test K8s RBAC ServiceAccount**：已创建受限制的 ServiceAccount，绑定最小权限 Role（必须 allow: create/delete deployments.apps、create/delete pods、create/delete services、create namespaces；必须 deny: create persistentvolumeclaims、create ingresses.networking.k8s.io）
- [ ] **钉钉群 + outgoing webhook** 已配置，指向 Hermes 回调地址
- [ ] **Hermes** 已部署并配置反向调用 AutoOps 的 Agent API

---

## 1. 用户数据

- [ ] **请求人**（用户 A）：SysAdmin 行存在，`dingtalk_user_id` 字段已填写
- [ ] **审批人**（用户 B）：SysAdmin 行存在，`dingtalk_user_id` 字段已填写
- 验证 SQL：
  ```sql
  SELECT id, username, dingtalk_user_id FROM sys_admin WHERE dingtalk_user_id IS NOT NULL AND dingtalk_user_id <> '';
  ```
  期望：至少 2 行（请求人 + 审批人）

---

## 2. 凭据（config_account 表）

- [ ] **Jenkins server**：`type=4`（jenkins），`host=http://<jenkins-host>`，`port=8080`，凭据已填写
- [ ] **Harbor server**：`type=5`（⚠️ UI 显示"Zabbix"，后端 Harbor 类型），`host=http://<harbor-host>`，`port=80`，用户名/密码已填写
- [ ] **dev K8s kubeconfig**：`type=6`（通用账号），密码字段粘贴完整 kubeconfig YAML，`host` 填 K8s API Server 地址
- [ ] **test K8s kubeconfig**：`type=6`（通用账号），密码字段粘贴完整 kubeconfig YAML，`host` 填 K8s API Server 地址
- [ ] **kubeconfig 权限校验**：调用直连凭据校验 API（`POST /api/v1/deploy/cluster-targets/:id/validate-direct-credential`），确认返回 `"valid": true`；若返回 `"valid": false`，检查 `permissions` 数组中哪些权限未满足（必须 allow: create/delete deployments.apps、create/delete pods、create/delete services、create namespaces（集群级）；必须 deny: create persistentvolumeclaims、create ingresses.networking.k8s.io）
- 验证 SQL：
  ```sql
  SELECT id, name, type, host, port FROM config_account WHERE type IN (4, 5, 6);
  ```
  期望：各至少 1 行

---

## 3. 集群目标（deploy_cluster_target 表）

- [ ] **dev target**：`env_type='dev'`，`direct_enabled=true`，`direct_kubeconfig_ref` 指向 ConfigCenter 中预录入的 dev 集群 kubeconfig 账户（格式 `account:<id或别名>`）
- [ ] **test target**：`env_type='test'`，`direct_enabled=true`，`direct_kubeconfig_ref` 指向 ConfigCenter 中预录入的 test 集群 kubeconfig 账户
- 验证 SQL：
  ```sql
  SELECT id, name, env_type, direct_enabled, direct_kubeconfig_ref FROM deploy_cluster_target WHERE env_type IN ('dev', 'test');
  ```
  期望：各 1 行，`direct_enabled=true`，`direct_kubeconfig_ref` 非 空

---

## 4. 应用（application 表）

- [ ] Application 行：`code='java-demo'`，`name='java-demo'`
- 验证 SQL：
  ```sql
  SELECT id, name, code FROM application WHERE code = 'java-demo';
  ```
  期望：1 行

---

## 5. AppDeployProfile（app_deploy_profile 表）

### 5.1 dev profile

- [ ] `env='dev'`，`enabled=true`
- [ ] `cluster_target_id` = dev target 的 ID（env_type=dev）
- [ ] `namespace='ao-direct-java-demo'`（Direct 模式要求 namespace 以 `ao-direct-` 开头，`directExecutor` 会在运行时校验）
- [ ] `release_name='java-demo'`
- [ ] `resource_type='deployment'`
- [ ] `jenkins_server_id` = Jenkins 凭据 ID
- [ ] `jenkins_job_name='java-demo'`
- [ ] `harbor_server_id` = Harbor 凭据 ID
- [ ] `harbor_project='library'`
- [ ] `harbor_repository='java-demo'`
- [ ] `default_git_ref='main'`
- [ ] `approver_admin_id` = 审批人（用户 B）的 ID
- [ ] `service_enabled=true`，`service_port=80`，`target_port=8080`

### 5.2 test profile

- [ ] `env='test'`，`enabled=true`
- [ ] `cluster_target_id` = test target 的 ID（env_type=test）
- [ ] `namespace='ao-direct-java-demo'`（同 dev，Direct 模式要求 `ao-direct-` 前缀）
- [ ] 其余字段同 dev profile

- 验证 SQL：
  ```sql
  SELECT p.id, p.env, p.namespace, p.jenkins_job_name, p.harbor_project, p.harbor_repository, t.env_type
  FROM app_deploy_profile p
  JOIN deploy_cluster_target t ON t.id = p.cluster_target_id
  WHERE p.application_code = 'java-demo';
  ```
  期望：2 行，dev 行的 env_type=dev，test 行的 env_type=test

- Profile validate API（可选但推荐）：
  ```bash
  # 获取 appId（返回分页格式，需 Admin JWT）
  curl -s -H "Authorization: Bearer <admin-token>" \
    http://localhost:8000/api/v1/apps | jq '.data.list[] | select(.code=="java-demo") | .id'
  # validate dev profile
  curl -s -X POST http://localhost:8000/api/v1/apps/<appId>/deploy-profiles/<profileId>/validate \
    -H "Authorization: Bearer <admin-token>" | jq '.data.valid'
  ```
  期望：`true`（`.data.valid` 为 true 表示校验通过）

---

## 6. K8s Namespace

Direct 模式下，AutoOps 会尝试创建目标 namespace（kubeconfig 必须有 `create namespaces` 权限），若 namespace 已存在则跳过创建。**注意**：Direct 模式强制要求 namespace 以 `ao-direct-` 前缀开头（`directExecutor.go:31-32` 硬编码校验），虽然在 UI 中 `directNamespacePrefix` 默认为 `ao-direct`，但后端不会使用 UI 的值做前缀校验，请务必保持前缀为 `ao-direct-`。

- [ ] `ao-direct-java-demo` namespace 在 dev 集群已存在（或 kubeconfig 有权创建）
  ```bash
  kubectl get ns ao-direct-java-demo --context=<dev-context>
  ```
- [ ] `ao-direct-java-demo` namespace 在 test 集群已存在（如需独立 test 命名空间）
  ```bash
  kubectl get ns ao-direct-java-demo --context=<test-context>
  ```

---

## 7. Agent Token

- [ ] AutoOps 中已配置 Agent Bearer Token（详见 `docs/autoops-build-deploy-onboarding.md` §2.X）
- 验证：
  ```bash
  curl -s -H "Authorization: Bearer <agent-token>" \
    http://localhost:8000/api/v1/integrations/agent/deploy-requests/nonexistent/status | jq '{code: .code, message: .message}'
  ```
  期望：`{"code": 404, "message": "..."}`（HTTP 状态码 200，JSON code 404，表示 Token 有效但请求不存在。若 HTTP 返回 401/403 则 Token 无效或未配置）

---

## 8. Hermes skill 行为（deploy-via-autoops）

> 本检查用于防止机器人把 build-deploy/Profile 链路误判成已有镜像 Direct 链路。skill 仓库内副本见 `skills/devops/deploy-via-autoops/SKILL.md`。

- [ ] opsclaw 当前 Hermes 已同步最新 `deploy-via-autoops` skill。
- [ ] 自然语言：`部署 java-demo 到开发环境，main 分支`。
- [ ] 期望：Hermes 调用 `POST /api/v1/integrations/agent/build-deploy-requests`，字段包含 `applicationCode=java-demo`、`env=dev`、`gitRef=main`。
- [ ] 期望：Hermes **不追问** 容器镜像地址、namespace、clusterTarget、kubeconfig、releaseName。
- [ ] 若缺应用，只追问应用；若缺环境，只追问 dev/test；若缺分支，不追问并使用 Profile 默认分支。
- [ ] 失败条件：出现类似“请提供容器镜像地址”或“Direct 模式命名空间需以 ao-direct- 开头”的追问。

---

## 9. Jenkins job

> ⚠️ **硬序**：必须先按 `docs/java-demo-jib-fix-instructions.md` 修改 Jenkinsfile，再录入此项。

- [ ] Jenkins `java-demo` job 存在，且 job 配置中 Jenkinsfile 已包含 `-Djib.to.auth.sendCredentialsOverHttp=true`（参考 `docs/java-demo-jib-fix-instructions.md`）
- [ ] Jenkins 可以连接 Harbor `http://<harbor-host>`（真实联调时替换为当前环境 Harbor 地址）

---

## 运行 E2E Smoke（dev）

所有 checklist 项目确认完毕后，执行：

```bash
curl -X POST http://localhost:8000/api/v1/integrations/agent/build-deploy-requests \
  -H "Authorization: Bearer <agent-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "<requester-dingtalk-userid>",
    "requesterDisplayName": "张三",
    "applicationCode": "java-demo",
    "env": "dev",
    "gitRef": "main",
    "reason": "smoke-test"
  }'
```

**验证点（按顺序）：**

1. 返回 `requestNo` + `workflowKind=build_deploy`
2. 审批人钉钉收到 OA 审批卡片 → 审批通过
3. Jenkins job `java-demo` 触发，Console 无 Jib HTTP 报错，build 成功
4. Harbor `library/java-demo` 出现新 artifact，扫描通过
5. **AutoOps 直接调 K8s API**（Direct 模式）创建 Deployment + Service 到 `ao-direct-java-demo` namespace
6. Pod Running，Service 端口可达（注意：ClusterIP 类型 Service 仅在集群内可达，如需从外部访问需通过 `kubectl port-forward` 或 Ingress）
8. `GET /api/v1/integrations/agent/deploy-requests/<requestNo>/events`（需 Bearer Token）返回 `execution_succeeded` 事件

**故障记录：** 将问题分类记录到 `docs/e2e-findings-2026-04-28.md`。

---

## 10. GitLab 项目自动接入 smoke（新增）

适用于“用户只告诉机器人 GitLab 地址”的链路。前置：`integrations.agent.project_onboarding.enabled=true` 且默认 ID 均已配置。

```bash
curl -X POST http://localhost:8000/api/v1/integrations/agent/project-onboard-build-deploy \
  -H "Authorization: Bearer <agent-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "<requester-dingtalk-userid>",
    "requesterDisplayName": "张三",
    "gitRepoUrl": "git@gayhub.seeingtv.com:demo/springboot-demo.git",
    "env": "dev",
    "gitRef": "main",
    "exposureMode": "nodeport",
    "reason": "自动接入并部署 springboot-demo"
  }'
```

期望：

- 自动创建/复用 `app_application.code=springboot-demo`。
- 自动创建/复用 dev `app_deploy_profile`。
- Profile namespace 默认为 `ao-direct-springboot-demo`。
- Profile serviceType 为 `NodePort`（如果请求中 `exposureMode=nodeport`）。
- 返回 `requestNo` + `workflowKind=build_deploy`，后续审批、Jenkins、Harbor、Direct K8s 部署流程与第 9 节一致。

若需要外部访问：

```bash
kubectl -n ao-direct-springboot-demo get svc springboot-demo
curl http://<nodeport_access_host>:<nodePort>/
```

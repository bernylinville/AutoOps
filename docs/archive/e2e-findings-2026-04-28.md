# DingTalk → AutoOps Direct E2E Findings（2026-04-28）

> 目的：记录 Phase B 真实联调证据。只有本文件填入 dev + test 两次完整 smoke 结果后，`docs/dingtalk-build-deploy-v1.1-plan.md` 的 Phase B 才能标记完成。

## 0. 运行前置

- [ ] 已按 `docs/e2e-checklist-java-demo.md` 完成外部前置 gate
- [ ] java-demo Jenkinsfile 已按 `docs/java-demo-jib-fix-instructions.md` 修复 Jib HTTP 凭据参数
- [ ] AutoOps API 已配置 `integrations.agent.bearer_token` 并重启
- [ ] dev/test ClusterTarget 均已配置 `direct_enabled=true`
- [ ] dev/test ClusterTarget 均已配置 `direct_kubeconfig_ref=account:<id或别名>`
- [ ] App Deploy Profile 已通过 UI 保存，且构建参数/环境变量/资源配置能正确持久化

## 1. Dev Smoke

### 请求

```bash
curl -X POST http://<api>/api/v1/integrations/agent/build-deploy-requests \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "<userid>",
    "applicationCode": "java-demo",
    "env": "dev",
    "gitRef": "main",
    "reason": "dev direct smoke"
  }'
```

### 证据

- requestNo：
- 审批实例 ID：
- Jenkins build number / URL：
- Harbor artifact：
- K8s namespace：
- Deployment：
- Service：
- Pod 状态：
- Agent events：

### 结论

- [ ] 通过
- [ ] 失败

失败详情：

## 2. Test Smoke

### 请求

```bash
curl -X POST http://<api>/api/v1/integrations/agent/build-deploy-requests \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "requesterExternalType": "dingtalk",
    "requesterExternalId": "<userid>",
    "applicationCode": "java-demo",
    "env": "test",
    "gitRef": "main",
    "reason": "test direct smoke"
  }'
```

### 证据

- requestNo：
- 审批实例 ID：
- Jenkins build number / URL：
- Harbor artifact：
- K8s namespace：
- Deployment：
- Service：
- Pod 状态：
- Agent events：

### 结论

- [ ] 通过
- [ ] 失败

失败详情：

## 3. 反向用例

### envType 不匹配

- 操作：将 test profile 指向 dev target，或将 dev profile 指向 test target
- 期望：返回“部署目标与环境不匹配”
- 实际：
- 结论：
  - [ ] 通过
  - [ ] 失败

### envType 大小写归一

- 操作：将 target envType 设置为 `Dev` / `TEST` 后校验对应 profile
- 期望：大小写不导致误报
- 实际：
- 结论：
  - [ ] 通过
  - [ ] 失败

### dev/test GitOps 阻断

- 操作：在 K8s Release Center 对 dev/test target 尝试选择 GitOps
- 期望：UI 禁用或提交前阻断，提示开发/测试新服务使用 Direct
- 实际：
- 结论：
  - [ ] 通过
  - [ ] 失败

### Hermes skill build-deploy 选路

- 操作：在钉钉群对 opsclaw Hermes 发送 `部署 java-demo 到开发环境，main 分支`
- 期望：Hermes 调用 `/api/v1/integrations/agent/build-deploy-requests`；只传 `applicationCode/env/gitRef/requester`；不追问 image/namespace
- 已观察到的错误回复：
  ```text
  skill_view: "deploy-via-autoops"，我需要以下信息才能继续：
  容器镜像地址：例如 registry.example.com/java-demo:main-20250428 或类似格式。您是否有已构建好的镜像？
  命名空间：开发环境的具体命名空间名称是什么？（direct 模式需要以 ao-direct- 开头）
  ```
- 初步定位：opsclaw 远端 skill v1.4.0 只有 existing-image direct 参数模型，把 build-deploy/Profile 链路误判为已有镜像部署。
- 修正来源：`skills/devops/deploy-via-autoops/SKILL.md` 与 `docs/dingtalk-hermes-api-contract.md` 已补充 build-deploy 负面规则。
- 实际：
- 结论：
  - [ ] 通过
  - [ ] 失败

## 4. Findings 分级

### 阻塞

| 编号 | 现象 | 影响 | 初步定位 | 处理状态 |
|---|---|---|---|---|
| B-001 | Hermes 对 `部署 java-demo 到开发环境，main 分支` 追问容器镜像和 namespace | 阻塞钉钉自然语言 build-deploy 联调 | `deploy-via-autoops` skill 缺少 build-deploy/Profile 选路规则 | 仓库 skill/契约已修正，待同步 opsclaw 后复测 |

### 错误信息可读性

| 编号 | 现象 | 期望文案 | 当前文案 | 处理状态 |
|---|---|---|---|---|
| M-001 |  |  |  |  |

### UX 抖动

| 编号 | 现象 | 影响 | 建议 | 处理状态 |
|---|---|---|---|---|
| U-001 |  |  |  |  |

## 5. Phase B 完成判定

- [ ] dev smoke 完整通过
- [ ] test smoke 完整通过
- [ ] envType 不匹配反向用例通过
- [ ] envType 大小写归一用例通过
- [ ] dev/test GitOps 阻断用例通过
- [ ] 所有阻塞类 findings 已关闭或明确进入 Phase C


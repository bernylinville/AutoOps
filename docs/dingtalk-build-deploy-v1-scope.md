# DingTalk → AutoOps 构建部署 v1 范围声明

> 本文档定义 DingTalk 集成构建部署功能的 v1 范围，防止功能蔓延，作为评审/复盘时的单一事实来源。

---

## 第一部分：目标链路

```
钉钉群 @机器人 
  → Hermes 解析 
  → AutoOps Agent API 
  → OA 审批 
  → Jenkins 构建 + Harbor 推送 
  → Harbor 扫描 
  → AutoOps Direct 部署到 K8s dev/test namespace 
  → 结果回写钉钉群
```

> **v1.1 变更**：默认部署模式从 GitOps/ArgoCD 切换为 **Direct**（AutoOps 直连 K8s API）。v1.1 不依赖 pukka-gitops 仓库或 ArgoCD。GitOps 代码保留但标记为 deprecated，UI 中仍可选 GitOps 模式以兼容存量。

---

## 第二部分：职责矩阵

| 组件 | 负责 | 不负责 |
|-----|------|-------|
| **AutoOps** | 保存应用+环境部署 Profile；接收 Agent 请求并解析 Profile；发起 OA 审批；记录 pipeline run 状态；**Direct 模式直接调 K8s API 部署 Deployment/Service**；将结果（访问地址等）写入 outbox | 自然语言理解；CI 构建逻辑；镜像打包；Harbor 项目创建；GitLab repo 创建 |
| **Hermes/Agent** | 接收钉钉自然语言消息；解析意图（applicationCode/env/gitRef）；缺参追问；调用 AutoOps Agent API；将 AutoOps 结果回写到钉钉群 | 保存部署配置；维护编排状态；控制 Jenkins/Harbor/K8s |
| **Jenkinsfile** | 源码编译；Maven/Jib 构建镜像；推送镜像到 Harbor；（可选）触发 Harbor 漏洞扫描 | 部署配置管理；OA 审批；GitOps 写入 |
| **GitOps/ArgoCD** | ~~ deprecated（v1.1 不依赖）~~ 保留代码不推荐使用 | 构建镜像；审批流程 |

---

## 第三部分：v1 不做清单

以下内容属于 **超出 v1 范围，不实现**：

1. **AutoOps 自动建 GitLab 仓库**  
   用户已有 gayhub.seeingtv.com，代码仓库生命周期由 GitLab 侧管理。

2. **AutoOps 自动生成或管理 Jenkinsfile**  
   Jenkinsfile 是项目自身 CI 契约，由研发团队维护。

3. **Harbor 项目全生命周期 provisioning**  
   library/java-demo 等项目可通过 UI/API 手动创建，v1 不做自动化。

4. **多集群路由、生产环境复杂审批矩阵**  
   v1 只支持 dev/test 双环境，单集群。

5. **StatefulSet / 复杂工作负载**  
   v1 只支持 Deployment（resourceType=deployment/pod）。

6. **Hermes 多轮对话、复杂 NLU 平台**  
   Hermes 只需单轮意图解析，输出结构化参数后调 AutoOps API。

7. **老项目迁移/兼容**  
   v1 只服务新项目，不处理历史遗留配置。

---

## 第四部分："Jenkins 权限不是阻塞点"事实记录

### 事件
Jenkins Build #20 失败

### 误判方向
有人可能认为是 Jenkins RBAC/权限问题

### 真实根因

- Jenkins 授权策略已是 **Logged-in users can do anything**，管理员账号 admin，无访问限制
- Build #20 失败发生在 Maven 构建成功之后的 **Jib 镜像推送阶段**
- 错误是 Jib 拒绝通过 HTTP（明文）发送 Harbor 凭据（`credentials over plain HTTP`）
- Harbor 地址 `harbor.harbor.svc.cluster.local` / `http://10.0.17.205` 均为 HTTP

### 修复方向
Jenkinsfile 中 Maven 调用加 `-Djib.to.auth.sendCredentialsOverHttp=true`  
详见 `docs/java-demo-jib-fix-instructions.md`

### 结论
**不要把 Jenkins 权限作为构建失败的默认假设，先看 Console Output。**

---

## 第五部分：技术债与运维约定

### 技术债

| 债项 | 当前状态 | 截止时间 | 处理方式 |
|-----|---------|---------|---------|
| Jib 通过 HTTP 发送 Harbor 凭据 | 临时 workaround（`sendCredentialsOverHttp=true`）| E2E 打通后 2 周内 | 切换 Harbor HTTPS（ingress + cert-manager）|

### 运维约定

**不做则 AutoOps 数据一致性损坏：**

1. **`agent_approver_allowlist.created_by='deploy-profile'` 是 Profile 专属所有权标记**  
   运维手工维护白名单必须使用其他 `created_by` 值（如 `manual`），否则 Profile 更新时会自动清除手工条目。

2. **Application.Code 改名期间禁止 Hermes 触发 build_deploy**  
   改名会导致 allowlist 用旧 code 查，产生 403 错误。改名后需手动检查并更新 deploy profile 和 allowlist。

3. **Hermes 必须只走 `/api/v1/integrations/agent/build-deploy-requests` 路径**  
   禁止绕过 Profile 直接调 `/deploy-requests`（UI 路径），否则 Hermes 无法利用 Profile 补齐 Jenkins/Harbor/namespace 等配置。

---

## 第六部分：v1.1 变更声明

- **v1.1 不依赖 pukka-gitops 仓库或 ArgoCD**。Agent 链路硬编码切换为 Direct 模式，AutoOps 直连 K8s API 部署到 dev/test namespace。
- GitOps 代码（`gitopsWriter.go` 等）保留在代码库中，标记为 deprecated。UI 仍可选 GitOps 模式以兼容存量场景，但 v1.1 新服务默认走 Direct。
- Direct 部署的环境变量（EnvJSON）和资源配置（ResourcesJSON）通过 Profile 传入，不再使用硬编码默认值。
- 详见 `docs/dingtalk-build-deploy-v1.1-plan.md` 获取完整变更清单。

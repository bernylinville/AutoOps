# Deployment Guide

## 1. 开发环境：Docker Compose

### 1.1 前置条件

- Docker Engine
- Docker Compose v2
- 本机具备 `docker compose` 运行权限
- GitOps 联调需要本机存在 sibling repo：`../pukka-gitops`

### 1.2 快速启动

```bash
cd docker
cp .env.example .env

docker compose up -d --build
./import-dev-data.sh --force
```

启动 6 个服务：

| 服务 | 端口 | 用途 |
|------|------|------|
| `devops-api` | `8000`（内部） / `18000`（宿主机） | Go API |
| `devops-web` | `80`（内部） / `18088`（宿主机） | Vue 前端（Nginx 反代） |
| `postgres` | `5432`（内部） / `15432`（宿主机） | PostgreSQL 17 |
| `valkey` | `6379`（内部） / `16379`（宿主机） | Valkey 9（缓存） |
| `prometheus` | `9090`（内部） / `19090`（宿主机） | 指标收集 |
| `pushgateway` | `9091`（内部） / `19091`（宿主机） | 短期指标网关 |

### 1.3 验证

```bash
docker compose ps
curl -sf http://127.0.0.1:18000/api/v1/healthz
curl -I http://127.0.0.1:18088
```

### 1.4 持久化

| Volume | 用途 |
|--------|------|
| `autoops-postgres-data` | PostgreSQL 数据 |
| `autoops-valkey-data` | Valkey 数据 |
| `autoops-prometheus-data` | Prometheus 指标 |
| `autoops-pushgateway-data` | Pushgateway 短期指标 |
| `autoops-api-log` | API 日志 |
| `autoops-api-upload` | 上传文件 |

清空环境（危险）：

```bash
cd docker && docker compose down -v
```

### 1.5 数据备份

```bash
cd docker

# PostgreSQL dump
docker compose exec -T postgres pg_dump -U devops autoops > autoops-dev.sql

# 上传目录
docker run --rm \
  -v autoops-api-upload:/data:ro \
  -v "$PWD":/backup \
  alpine:3.23 \
  sh -c 'tar czf /backup/autoops-upload.tar.gz -C /data .'
```

### 1.6 常见操作

```bash
cd docker

# 重建 API / Web
docker compose up -d --build devops-api devops-web

# 查看 API 日志
docker compose logs -f devops-api

# 重导基础数据
./import-dev-data.sh --force
```

## 2. 生产环境：Kubernetes + ArgoCD

AutoOps 生产环境通过 ArgoCD 从两处仓库拉取部署：

| 来源 | 路径 | 内容 |
|------|------|------|
| AutoOps 仓库 | `charts/autoops/` | Helm chart（Deployment、Service、PVC、Secret、Gateway 模板） |
| pukka-gitops 仓库 | `apps/autoops/values.yaml` | 生产环境 values（镜像 tag、Secret 引用、资源配额、集成配置） |

### 2.1 部署架构

```
┌─ ArgoCD Application ──────────────────────────────────┐
│  name: autoops, namespace: argocd                       │
│  syncPolicy: automated (prune + selfHeal, retry 3×)    │
│                                                         │
│  ┌─ source 1: AutoOps repo ──────────────────────────┐ │
│  │  path: charts/autoops                              │ │
│  │  helm values: apps/autoops/values.yaml (from src2) │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  ┌─ source 2: pukka-gitops repo ─────────────────────┐ │
│  │  ref: values, path: apps/autoops/                  │ │
│  └────────────────────────────────────────────────────┘ │
│                                                         │
│  destination: autoops namespace                         │
└─────────────────────────────────────────────────────────┘
```

### 2.2 组件清单

| 组件 | Kind | 说明 |
|------|------|------|
| `autoops-api` | Deployment | Go API，replica=1，Recreate 策略 |
| `autoops-web` | Deployment | Vue 前端（Nginx + default.conf），replica=1 |
| `autoops-postgres` | Deployment | PostgreSQL 17.4-alpine，Recreate 策略 |
| `autoops-valkey` | Deployment | Valkey 9.0.3-alpine，Recreate 策略 |
| `autoops-api` / `autoops-web` / ... | Service（ClusterIP） | 内部 DNS 发现 |
| `autoops` | Gateway + HTTPRoute | Envoy Gateway，MetalLB `10.0.17.206:80` |
| PVC ×4 | PersistentVolumeClaim | Postgres 20 Gi / Valkey 5 Gi / Upload 10 Gi / GitOps 2 Gi |
| `autoops-runtime` | Secret | DB 密码、JWT 密钥、Agent token、DingTalk 凭据等 17 个 key |

### 2.3 镜像仓库

所有镜像走火山引擎代理缓存：

```
pukka-all-images-cn-shanghai.cr.volces.com/proxy/
```

当前生产 tag：

| 组件 | Tag 示例 |
|------|---------|
| `autoops-api` | `20260430-1544-a2c3c49-crumbcookie` |
| `autoops-web` | `20260429-100647-bcb36b7` |
| `postgres` | `17.4-alpine` |
| `valkey` | `9.0.3-alpine` |

Tag 格式：`{date}-{time}-{git-short-sha}-{descriptor}`。CI 构建时自动生成。

### 2.4 Secret 管理

生产使用 `secret.mode=existing`，指向预先创建的 `autoops-runtime` Secret：

```yaml
secret:
  mode: existing
  existingSecret:
    name: autoops-runtime
    keys:
      dbPassword: POSTGRES_PASSWORD
      redisPassword: REDIS_PASSWORD
      jwtSecret: JWT_SECRET
      agentBearerToken: AGENT_BEARER_TOKEN
      heartbeatToken: HEARTBEAT_TOKEN
      flashdutyAppKey: FLASHDUTY_APP_KEY
      flashdutyIntegrationKey: FLASHDUTY_INTEGRATION_KEY
      dingtalkWebhookURL: DINGTALK_WEBHOOK_URL
      dingtalkApprovalClientID: DINGTALK_APPROVAL_CLIENT_ID
      dingtalkApprovalClientSecret: DINGTALK_APPROVAL_CLIENT_SECRET
      dingtalkApprovalProcessCode: DINGTALK_APPROVAL_PROCESS_CODE
      dingtalkApprovalOriginatorDeptID: DINGTALK_APPROVAL_ORIGINATOR_DEPT_ID
      deployBotEnabled: DEPLOY_BOT_ENABLED
      deployBotWebhookURL: DEPLOY_BOT_WEBHOOK_URL
      deployBotSecret: DEPLOY_BOT_SECRET
```

### 2.5 集群访问 RBAC

API Pod 通过 `autoops-cluster-reader` ServiceAccount 获得对所在集群的只读权限：

- 资源：nodes、namespaces、pods、services、deployments、jobs、ingresses 等
- 动作：get、list、watch
- Token Secret：`autoops-cluster-reader-token`（含 `kubernetes.io/service-account-token` 注解）

### 2.6 GitOps 工作树

API Pod 启动时通过 init container 执行 `git clone` 将 pukka-gitops 仓库拉取到 `/workspace/pukka-gitops`（PVC 持久化）。Deploy 模块通过 Direct / GitOps 两条路径操作该工作树。

生产配置：

```yaml
gitopsWorkingTree:
  enabled: true
  mountPath: /workspace/pukka-gitops
  repoURL: http://gayhub.seeingtv.com/ipaas/pukka-gitops.git
  branch: main
  bootstrapMode: clone
  persistence:
    enabled: true
    size: 2Gi
```

### 2.7 集成的外部服务

| 服务 | 用途 | 配置段 |
|------|------|--------|
| FlashDuty | 告警集成 | `flashduty.*` |
| DingTalk Webhook | 通知推送 | `dingtalk.webhookURL` |
| DingTalk 审批 | OA 审批流 | `dingtalkApproval.*`（含字段映射） |
| DingTalk 部署 Bot | Agent 部署交互 | `integrations.deployBot` |
| Monitor Agent | Agent 心跳 | `monitor.agent.heartbeatServerURL` |

### 2.8 生产注意事项

- `gitopsWorkingTree.bootstrapMode` 当前为 `clone`；Git 凭据不嵌入 `repoURL`，通过 `sshSecretName` 引用外部 Secret
- PostgreSQL / Valkey / Upload PVC 按需通过 `useDefaultStorageClass`、`storageClassName` 或 `existingClaim` 表达
- MetalLB IP 固定为 `10.0.17.206`，对应 `gateway.loadBalancerIP`
- `api.podSecurityContext` 配置 `runAsNonRoot: true`、`runAsUser: 100`
- `api.imageHost` 设为 `http://10.0.17.206`，用于构造 Docker 镜像引用前缀

## 3. 开发集群：minikube

### 3.1 启动

```bash
minikube start --driver=docker
minikube ip
```

### 3.2 接入 AutoOps

1. 导出 kubeconfig：
   ```bash
   minikube kubectl config view --raw > /tmp/minikube.kubeconfig
   ```
2. 在 AutoOps 界面新增 Kubernetes 集群
3. 粘贴 kubeconfig 内容
4. 若 apiserver 地址为 `127.0.0.1:<port>`：
   - 改为 `https://$(minikube ip):8443`
   - 或改为 `https://host.docker.internal:<port>`

## 4. 外部监控与告警

开发与生产共享 N9E + VictoriaMetrics + FlashDuty。

平台约束：

- `monitor/hosts/*` 兼容接口保留
- CPU / 内存 / 磁盘指标通过 VictoriaMetrics 查询
- `top-processes` / `ports` 接口在外部监控模式下返回空数组

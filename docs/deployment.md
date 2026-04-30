# Deployment Guide

## 1. 开发环境：Arch Linux + Docker Compose

当前推荐开发方式：

- AutoOps 平台本体：Docker Compose
- 测试集群：本地 minikube
- 监控 / 告警：复用已有 N9E + VictoriaMetrics + FlashDuty

> AutoOps 已移除内置 Prometheus / Pushgateway，因此本地 Compose 只保留 `postgres`、`valkey`、`devops-api`、`devops-web` 四个服务。

### 1.1 前置条件

- Docker Engine
- Docker Compose v2
- 本机具备 `docker compose` 运行权限
- 本机已存在 sibling repo：`../pukka-gitops`（供 GitOps 联调）

### 1.2 快速启动

```bash
cd docker
cp .env.example .env

docker compose up -d --build
./import-dev-data.sh --force
```

### 1.3 验证

```bash
cd docker

docker compose config >/dev/null
docker compose ps
curl -sf http://127.0.0.1:18000/api/v1/healthz
curl -I http://127.0.0.1:18088
```

### 1.4 持久化策略

开发环境已切换到 **Docker named volumes**：

| Volume | 用途 |
|---|---|
| `autoops-postgres-data` | PostgreSQL 数据 |
| `autoops-valkey-data` | Valkey 数据 |
| `autoops-api-log` | API 日志 |
| `autoops-api-upload` | 上传文件 |

查看：

```bash
docker volume ls | grep autoops
```

清空环境（危险）：

```bash
cd docker
docker compose down -v
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
# 重建 API / Web
cd docker
docker compose up -d --build devops-api devops-web

# 查看 API 日志
docker compose logs -f devops-api

# 重导基础数据
./import-dev-data.sh --force
```

## 2. 开发集群：minikube

### 2.1 启动 minikube

```bash
minikube start --driver=docker
minikube status
minikube ip
```

### 2.2 将 minikube 接入 AutoOps

推荐步骤：

1. 导出 kubeconfig：
   ```bash
   minikube kubectl config view --raw > /tmp/minikube.kubeconfig
   ```
2. 在 AutoOps 中新增 Kubernetes 集群
3. 粘贴 kubeconfig 内容
4. 如果 kubeconfig 中 apiserver 地址为 `127.0.0.1:<port>`：
   - 优先改成 `https://$(minikube ip):8443`
   - 或改成 `https://host.docker.internal:<port>`（Compose 已注入 host-gateway）

## 3. 生产环境：ArgoCD / pukka-gitops

### 3.1 Helm chart

AutoOps 平台 chart 位于：

- `charts/autoops/`

主要能力：

- API / Web / PostgreSQL / Valkey
- Upload PVC
- GitOps working tree PVC
- Envoy Gateway + HTTPRoute
- 运行时 config.template 渲染
- `secret.mode=generated|existing`，生产固定走 `existing`
- 通用 PVC contract：`useDefaultStorageClass` / `storageClassName` / `existingClaim`

### 3.2 GitOps 对接

`~/Code/pukka-gitops` 中已维护：

- `argocd-apps/templates/autoops.yaml`
- `apps/autoops/values.yaml`

其中 `autoops.yaml` 使用 **ArgoCD 多源 Application**：

- source 1：AutoOps 仓库 `charts/autoops`
- source 2：pukka-gitops 仓库 `apps/autoops/values.yaml`

### 3.3 生产注意事项

- AutoOps 平台在生产仍使用 `pukka-gitops` 工作树，但固定为 **existing PVC** 挂载
- 生产 `gitopsWorkingTree.bootstrapMode=disabled`，应用 Pod 不在启动时隐式 clone / fetch / reset 仓库
- 若需要平台在集群内执行 `git push`，Git 凭据必须通过现有 Secret 挂载（例如 `sshSecretName`）；`repoURL` 本身不得携带凭据
- PostgreSQL / Valkey / Upload PVC 通过 `useDefaultStorageClass=true`、显式 `storageClassName` 或 `existingClaim` 表达；不要再把 `local-path` / `nfs-client` 当作 AutoOps 默认
- 生产对外访问 MetalLB IP 为 `10.0.17.206`

## 4. 外部监控 / 告警

开发与生产共享：

- N9E
- VictoriaMetrics
- FlashDuty

当前平台约束：

- `monitor/hosts/*` 兼容接口仍保留
- CPU / 内存 / 磁盘实时/历史指标通过 VM 查询
- `top-processes` / `ports` 接口在外部监控模式下返回空数组

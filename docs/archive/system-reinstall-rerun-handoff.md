# AutoOps 重装系统后重执行任务交接文档

> 截至时间：2026-04-24
> 目的：开发机重装系统后，能够快速恢复上下文，并把本轮 AutoOps 环境重建 / 精简 / GitOps 化任务继续执行到完成。

## 1. 任务背景

当前 AutoOps 需要完成一轮开发环境与部署体系收敛：

- 开发机已经重装为 **Arch Linux**。
- 需要用 **Docker** 重建 AutoOps 本地开发环境，并确保项目能正确运行。
- 本地开发环境需要接入 **本机 minikube** 启动的 Kubernetes 集群做测试。
- 生产环境不再走项目内零散清单，改为通过 `~/Code/pukka-gitops` 的 **ArgoCD GitOps** 方式部署。
- AutoOps 平台本身需要维护可用于生产部署的 **Helm Chart**。
- 项目内不再需要 **Prometheus / Pushgateway**，无论代码、开发环境还是生产环境都要去掉。
- 开发与生产的 AutoOps 都对接**同一套外部监控与告警系统**：
  - N9E
  - VictoriaMetrics
  - FlashDuty
- Docker 开发环境的数据持久化，用户明确要求：**优先使用 Docker named volumes，不要使用本地目录 bind mount 存数据**。

## 2. 已锁定的关键决策

1. **Helm 是 AutoOps 平台部署的唯一规范化产物**。
2. **Prometheus / Pushgateway 已被判定为不需要保留的内部组件**。
3. 开发 / 生产统一复用外部 **N9E + VictoriaMetrics + FlashDuty**。
4. `autoops-managed-releases` 与 AutoOps 平台自身部署保持分离。
5. 生产 GitOps 仓库是：`~/Code/pukka-gitops`。
6. 生产侧接入的是 `pukka-gitops` 管理的 K8s 集群。
7. 开发侧接入的是本机 **minikube**。
8. Docker Compose 里数据库、缓存、日志、上传目录等持久化使用 **named volumes**。
9. **不要把本机 sudo 密码、JWT_SECRET、私钥等敏感信息写入文档或提交仓库**。
10. AutoOps chart 生产配置固定使用 `secret.mode=existing`，运行时敏感项仅通过已有 Secret 名称 / key 注入。
11. AutoOps chart 生产配置固定使用 `gitopsWorkingTree = existing PVC + Secret 挂载 Git 凭据 + bootstrapMode=disabled`。
12. AutoOps chart 不再把 `local-path` / `nfs-client` 作为默认存储契约，生产通过 `useDefaultStorageClass=true`、显式 StorageClass 或 `existingClaim` 绑定。
13. `pukka-gitops` 仓库内已经有额外约束：
   - 在 `apps/<name>/` 增加应用 values
   - 在 `argocd-apps/templates/<name>.yaml` 增加 ArgoCD Application 模板
   - 更新 `argocd-apps/values.yaml`
   - 更新 `docs/onboarding.md` 与 `docs/architecture.md`
   - 资源 requests / limits 必填
   - AutoOps 生产 LB IP 预期为 `10.0.17.206`

## 3. 本轮执行时已经落下去的修改（需复查并延续）

### 3.1 AutoOps 仓库内

#### 开发环境 / Docker
- `docker/docker-compose.yml`
  - 精简为：`postgres`、`valkey`、`devops-api`、`devops-web`
  - 使用 named volumes：
    - `autoops-postgres-data`
    - `autoops-valkey-data`
    - `autoops-api-log`
    - `autoops-api-upload`
  - 通过 `LOCAL_GITOPS_REPO_PATH` 挂载本地 `../pukka-gitops`
- `docker/.env.example`
  - 移除 prom/push 相关变量
  - 新增 `LOCAL_GITOPS_REPO_PATH`
- `docker/devops-start.sh`
  - 只保留当前需要的依赖启动流程
- `docker/api/config.template.yaml`
- `docker/api/config.yaml`
  - 已改为对接当前精简后的运行时配置

#### 监控精简 / 外部监控对接
- `api/common/config/config.go`
  - 已移除 Prometheus / Pushgateway 配置结构
- `api/config.yaml`
- `api/config.yaml.example`
- `docker/api/config.template.yaml`
  - 已改为外部 N9E / VictoriaMetrics / FlashDuty 的配置方向
- `api/common/agent/agent.go`
  - 已收敛为 heartbeat-only 逻辑
- `api/api/monitor/service/monitorService.go`
  - 已改成依赖外部 N9E / VM
- `api/api/monitor/service/agent.go`
  - 清理过无用导入

#### GitOps / Deploy 校验
- `api/api/deploy/service/gitopsWriter.go`
  - 已增强，对 AutoOps 平台应用 values / app 文件进行校验

#### Helm Chart
已新增：
- `charts/autoops/Chart.yaml`
- `charts/autoops/values.yaml`
- `charts/autoops/templates/_helpers.tpl`
- `charts/autoops/templates/secret.yaml`
- `charts/autoops/templates/api-runtime-configmap.yaml`
- `charts/autoops/templates/nginx-configmap.yaml`
- `charts/autoops/templates/persistentvolumeclaims.yaml`
- `charts/autoops/templates/services.yaml`
- `charts/autoops/templates/api-deployment.yaml`
- `charts/autoops/templates/web-deployment.yaml`
- `charts/autoops/templates/postgres-deployment.yaml`
- `charts/autoops/templates/valkey-deployment.yaml`
- `charts/autoops/templates/gateway.yaml`

#### 文档
已大幅重写 / 更新：
- `README.md`
- `docs/deployment.md`
- `docs/architecture.md`
- `docs/backend-guide.md`
- `docs/deploy-control-plane.md`

### 3.2 `~/Code/pukka-gitops` 仓库内

已新增 / 修改：
- `argocd-apps/templates/autoops.yaml`
- `apps/autoops/values.yaml`
- `argocd-apps/values.yaml`
- `docs/onboarding.md`
- `docs/architecture.md`

## 4. 上次执行时的验证结果

### 已验证
- `docker compose config` 已可正常渲染。
- Docker Compose 文件已切到 named volumes 方案。
- 后端 `go test ./...` 已通过 Docker 容器执行并全量通过。
- 前端已通过 Docker 容器完成生产构建（存在体积与过时 browserslist 警告，但构建成功）。
- AutoOps Helm Chart 已通过 **dockerized Helm** 的 `lint` 与 `template` 验证。
- `~/Code/pukka-gitops` 中的 ArgoCD apps chart 已通过 **dockerized Helm** 渲染验证。
- 当前主机已确认安装 `minikube`，版本为 `v1.38.1`。

使用的验证命令：

```bash
cd ~/Code/AutoOps
docker run --rm -v "$PWD/api":/src -w /src golang:1.25 sh -lc 'export PATH=/usr/local/go/bin:$PATH; go test ./...'

# 前端生产构建
docker run --rm -v "$PWD/web":/app -w /app node:22-bookworm sh -lc 'npm install && npm run build'

# AutoOps Helm Chart 验证
docker run --rm -v "$PWD":/work -w /work alpine/helm:3.17.3 lint charts/autoops
docker run --rm -v "$PWD":/work -w /work alpine/helm:3.17.3 template autoops charts/autoops

# pukka-gitops ArgoCD apps 渲染
docker run --rm -v ~/Code/pukka-gitops:/work -w /work alpine/helm:3.17.3 template pukka-argocd-apps argocd-apps -f argocd-apps/values.yaml
```

### 未完成 / 当前已知阻塞

#### 阻塞 1：主机缺少 Helm / kubectl
上次会话里主机 PATH 中没有：
- `helm`
- `kubectl`

说明：

- 目前 **host PATH** 中仍没有 `helm` / `kubectl`
- 但 **Helm 渲染验证已经通过 dockerized Helm 补齐**
- 如果后续要做更顺手的本机调试、Argo 模板检查或 Kubernetes 手工排障，仍建议在主机安装 `helm` 与 `kubectl`

以下是仍建议具备的本机命令：

```bash
helm lint charts/autoops
helm template autoops charts/autoops
cd ~/Code/pukka-gitops && helm template pukka-argocd-apps argocd-apps -f argocd-apps/values.yaml
kubectl --context minikube ...
```

#### 阻塞 2：Compose 运行态验收还没完整做完
尚未完成的目标验收：
- `docker compose up -d --build`
- `curl -sf http://127.0.0.1:18000/api/v1/healthz`
- `curl -I http://127.0.0.1:18088`
- 导入开发种子数据
- 实测 AutoOps 到 minikube 的 K8s 接入是否正常

补充说明：

- 本轮已经启动过 `docker compose up -d --build`
- 但该执行过程在会话中被中断，**当前不应假设运行态验收已经完成**
- 重启任务时应重新执行一次完整的 compose 启动、健康检查与 seed 导入流程

#### 已解决事项：Jenkins Pipeline 测试阻塞

此前 `go test ./...` 曾失败于 Jenkins pipeline 相关测试，问题已经在本地修正并完成重新验证：

- `api/api/deploy/service/jenkinsPipeline.go`
  - 为 `clientCache` 增加了惰性初始化，避免零值 adapter 在测试或直接构造场景下出现 nil map panic
- `api/api/deploy/service/jenkinsPipeline_test.go`
  - 测试 server 现在会兼容 `/crumbIssuer/api/json`
  - 镜像 tag fallback 断言已与当前实现保持一致

因此，**当前后端测试已不再是阻塞项**。

#### 已确认但尚未清理完的残留点

在代码库中继续 grep 时，仍能看到 `prometheus` / `pushgateway` 相关文本残留，主要位于以下非当前主路径位置：

- `api/common/templates/**`
- `api/common/config/images.yaml`
- `api/logs/**`
- `api/sql/**`
- `docker/mysql/**`

这说明：

- **当前主开发环境 / Helm / 文档主路径已经基本切换到“无内置 Prometheus/Pushgateway”方向**
- 但仓库内仍存在一些模板、样例、历史 SQL/日志数据中的相关字样
- 后续如果目标是“受支持路径完全去残留”，需要再判断这些目录是否属于本项目当前仍支持的功能面，再决定继续清理

## 5. 后续重执行时的推荐顺序

1. 安装基础工具
   - Docker / Docker Compose
   - Helm
   - kubectl
   - minikube
2. 恢复代码目录
   - `~/Code/AutoOps`
   - `~/Code/pukka-gitops`
3. 先跑静态渲染验证
   - `docker compose config`
   - `helm lint charts/autoops`
   - `helm template autoops charts/autoops`
   - `cd ../pukka-gitops && helm template ...`
4. 启动本地开发环境
   - `docker compose up -d --build`
5. 导入开发数据并联调
   - DB seed
   - 登录 / 健康检查
   - K8s(minikube) 接入
6. 验证生产交付物
   - `charts/autoops` 可渲染
   - `pukka-gitops` 中 autoops app 可渲染
7. 做最终文档收尾
   - 更新部署文档 / 开发文档 / 迁移说明
   - 记录仍存风险

如果只是为了快速恢复工作流，可直接跳过“安装 host helm”这一步，先使用 dockerized Helm 做渲染验证，再补主机工具链。

## 6. 系统重装后的首轮准备清单

建议在重新开工前，先完成以下最小准备动作：

1. 恢复仓库目录
   - `~/Code/AutoOps`
   - `~/Code/pukka-gitops`
2. 确认运行时工具可用
   - `docker`
   - `docker compose`
   - `helm`
   - `kubectl`
   - `minikube`
3. 确认 Docker daemon 已正常启动，当前用户可直接执行 `docker ps`
4. 确认本机 `minikube start --driver=docker` 可正常启动
5. 确认 `~/Code/pukka-gitops` 工作树存在，不要让 AutoOps 的 GitOps 校验指向空目录
6. 开始任何清理前先执行一次 `git status`，避免误覆盖仓库内已有脏改动

建议先跑这组快速检查：

```bash
docker --version
docker compose version
helm version
kubectl version --client
minikube version
docker ps
```

当前已知实际状态（截至本文件更新时）：

- `docker`：已安装
- `docker compose`：已安装
- `minikube`：已安装
- `helm`：host PATH 中缺失，但可用 dockerized Helm 代替
- `kubectl`：host PATH 中缺失

## 7. 关键文件清单（后续 agent 优先检查）

### AutoOps
- `docker/docker-compose.yml`
- `docker/.env.example`
- `docker/devops-start.sh`
- `docker/api/config.template.yaml`
- `docker/api/config.yaml`
- `api/common/config/config.go`
- `api/common/agent/agent.go`
- `api/api/monitor/service/agent.go`
- `api/api/monitor/service/monitorService.go`
- `api/api/deploy/service/gitopsWriter.go`
- `charts/autoops/**`
- `README.md`
- `docs/deployment.md`
- `docs/architecture.md`
- `docs/backend-guide.md`
- `docs/deploy-control-plane.md`

### pukka-gitops
- `apps/autoops/values.yaml`
- `argocd-apps/templates/autoops.yaml`
- `argocd-apps/values.yaml`
- `docs/onboarding.md`
- `docs/architecture.md`

## 8. 可直接复用的提示词

下面这段可以在系统重装后，直接发给新的 Codex / Claude / 其它代码代理继续执行：

```text
你现在在处理 AutoOps 仓库的续做任务。请不要重新做需求澄清，直接基于以下上下文继续执行，直到完成验证。

仓库：
- ~/Code/AutoOps
- ~/Code/pukka-gitops

目标：
1. 用 Docker 在 Arch Linux 上重建 AutoOps 开发环境，并确保可正常运行。
2. 开发环境接入本机 minikube Kubernetes 集群做测试。
3. 生产环境不再使用项目内零散 YAML，改为通过 ~/Code/pukka-gitops 的 ArgoCD GitOps 部署。
4. 为 AutoOps 平台维护可用于生产部署的 Helm Chart。
5. 从 AutoOps 的代码、开发环境、生产部署、文档中彻底移除 Prometheus / Pushgateway。
6. 开发和生产环境统一接入已有的外部 N9E、VictoriaMetrics、FlashDuty。
7. Docker 持久化使用 named volumes，不要用本地目录挂载数据目录。
8. 完成后补齐和修正文档。

已确认约束：
- Helm 是 AutoOps 平台的规范化部署产物。
- autoops-managed-releases 和 AutoOps 平台部署分离。
- 生产 GitOps 仓库是 ~/Code/pukka-gitops。
- pukka-gitops 中要维护：apps/autoops/values.yaml、argocd-apps/templates/autoops.yaml、argocd-apps/values.yaml，并同步更新 docs/onboarding.md 和 docs/architecture.md。
- 生产 LB IP 预期为 10.0.17.206。
- 不要把 sudo 密码、JWT_SECRET、私钥等敏感信息写入仓库或文档。

上次已经做过但需要你复查并继续：
- docker/docker-compose.yml 已精简为 postgres、valkey、devops-api、devops-web，并改为 named volumes。
- api/common/config/config.go、api/common/agent/agent.go、api/api/monitor/service/monitorService.go 等文件已朝“外部监控接入、去除 Prometheus/Pushgateway”方向修改。
- charts/autoops 已创建一套 Helm Chart。
- pukka-gitops 中已加入 autoops ArgoCD app 和 values。
- README.md、docs/deployment.md、docs/architecture.md、docs/backend-guide.md、docs/deploy-control-plane.md 已有较大改动。
- Jenkins pipeline 的 nil map 测试阻塞已修复，`go test ./...` 已重新全量通过。
- dockerized Helm 已确认：
  - `charts/autoops` 可 lint/template
  - `~/Code/pukka-gitops/argocd-apps` 可渲染出 `autoops` Application

上次已知阻塞：
- 主机当时缺少 helm 和 kubectl；不过 Helm/ArgoCD 渲染验证后来已经通过 dockerized Helm 补齐。
- Docker Compose 运行态验收、种子数据导入、minikube 联调尚未完成。
- Prometheus / Pushgateway 相关文本在模板、SQL、日志、样例路径中仍有残留，需要后续判断是否属于支持范围并继续清理。

请按下面顺序执行：
1. 检查 git 状态与已改文件，避免覆盖无关改动。
2. 安装或确认存在 docker、docker compose、helm、kubectl、minikube。
3. 验证 docker compose config。
4. 如果 host 没有 helm，优先用 dockerized Helm 重新验证 Helm Chart 与 pukka-gitops ArgoCD apps。
5. 启动 docker compose 开发环境，验证 API / Web 健康。
6. 导入开发数据，验证 AutoOps 接入 minikube。
7. grep 审计支持路径，确认 Prometheus / Pushgateway 已从当前受支持代码、配置、文档中清理干净；同时单独评估模板/样例/SQL/日志目录是否要纳入清理。
8. 完成文档收尾，输出变更文件、验证证据、剩余风险。

如果发现上次改动有问题，可以直接修正，但不要擅自扩大范围到无关模块。
```

## 9. 建议的最小执行命令集

```bash
# 1) Compose 静态检查
cd ~/Code/AutoOps/docker
docker compose config

# 2) 后端测试（不依赖主机 Go）
cd ~/Code/AutoOps
docker run --rm -v "$PWD/api":/src -w /src golang:1.25 sh -lc 'export PATH=/usr/local/go/bin:$PATH; go test ./...'

# 3) Helm 渲染
cd ~/Code/AutoOps
helm lint charts/autoops
helm template autoops charts/autoops

# 4) ArgoCD apps 渲染
cd ~/Code/pukka-gitops
helm template pukka-argocd-apps argocd-apps -f argocd-apps/values.yaml

# 如果主机没装 helm，可使用 dockerized Helm
cd ~/Code/AutoOps
docker run --rm -v "$PWD":/work -w /work alpine/helm:3.17.3 lint charts/autoops
docker run --rm -v "$PWD":/work -w /work alpine/helm:3.17.3 template autoops charts/autoops

cd ~/Code/pukka-gitops
docker run --rm -v "$PWD":/work -w /work alpine/helm:3.17.3 template pukka-argocd-apps argocd-apps -f argocd-apps/values.yaml

# 5) 启动本地环境
cd ~/Code/AutoOps/docker
docker compose up -d --build
curl -sf http://127.0.0.1:18000/api/v1/healthz
curl -I http://127.0.0.1:18088
```

## 10. 风险提醒

- 当前仓库里存在一些与本任务无关的脏改动，续做时必须先看 `git status`，避免误清理。
- Jenkins pipeline 测试阻塞已经修复，但仍应保留 Docker 内 `go test ./...` 作为回归动作。
- Helm Chart 与 ArgoCD values 已有 `lint/template` 实证，但仍缺少真实集群级部署验收。
- minikube 接入验证属于真实联调，不能只靠静态配置判断完成。
- `prometheus` / `pushgateway` 在模板、SQL、日志等目录仍有字样残留；若要求“仓库层面彻底无残留”，还需要额外清理判断。
- 本轮 `docker compose up -d --build` 过程曾被中断，不能把它当作已完成运行验收。

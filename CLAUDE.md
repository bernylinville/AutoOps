# AutoOps Agent Harness

本文件是 AutoOps 的根级 agent 入口。它只放跨任务都会用到的规则、路由和验证命令；细节放到 `docs/` 专题文档中维护。

## 项目定位

AutoOps 是 Go + Vue 3 构建的统一运维管理平台，覆盖 CMDB、Kubernetes、任务自动化、N9E / FlashDuty 监控告警、部署控制台和配置中心。

| 层 | 技术 |
|---|---|
| Backend | Go 1.25 / Gin / GORM / PostgreSQL 17 |
| Frontend | Vue 3.5 / Element Plus 2.10 / Vuex / Vue Router |
| Infra | PostgreSQL 17.4, Valkey 9, Prometheus, Pushgateway |
| Deploy | Docker Compose, Helm, Kubernetes, ArgoCD |

## 信息来源优先级

1. **源码和测试最高优先级**。文档与代码冲突时，以当前代码为准，并同步修正文档。
2. **本文件只做入口**。不要把架构、接口和排障细节复制进 `CLAUDE.md`。
3. **`docs/README.md` 是文档导航**。开始任务前先按任务类型读取对应专题文档。
4. **`README.md` 面向项目介绍**，不是开发约束的权威来源。
5. **`docs/archive/` 只读作历史背景**，不要把归档内容当作当前实现。

## 执行纪律

这些规则用于非平凡任务；明显的一行修正可以按需放轻。

- **先澄清再实现**：不确定时停止并说明困惑；存在多种解释时列出选项，不要默默选择。
- **先定义成功标准**：把「修复」「优化」「增加」转成可验证目标，必要时写成 `步骤 → 验证` 的短计划。
- **简洁优先**：只实现当前请求需要的能力，不添加未要求的抽象、配置、兼容层或未来扩展点。
- **精准修改**：每一行改动都应能追溯到用户请求；不要顺手重构、改格式、改注释或清理无关旧代码。
- **只清理自己的影响**：删除因本次改动产生的无用导入、变量、函数；预先存在的死代码只记录，不擅自删除。
- **验证驱动闭环**：Bug 修复先复现再修；行为变更先写测试或明确手动验证；重构前后运行同一组检查。
- **交付说明要有边界**：说明改了什么、刻意没碰什么、执行了哪些验证，无法验证时说明原因。

### 自检

交付前用这些具体检查验证纪律是否执行到位：

| 检查项 | 问题 |
|---|---|
| 假设可见性 | 我在实现中是否做了用户没有明确说出的假设？如果有，是否已列出并确认？ |
| 代码量 | 200 行能用 50 行解决吗？是否引入了单次使用的接口、结构体或配置项？ |
| 风格漂移 | 改动是否改了引号风格、缩进、命名约定、加了未要求的 type hint 或 docstring？ |
| 无关改动 | diff 里每一行是否都能追溯到用户请求？是否有「顺手改」的注释、格式化或重构？ |
| 投机功能 | 是否加了用户没要求的缓存、校验、通知、兼容层、「以后可能用到」的参数？ |
| 孤儿清理 | 是否删除了因本次改动而不再使用的 import/变量/函数？是否误删了预先存在的死代码？ |
| 验证闭环 | 是否有可运行的测试或手动验证步骤证明改动有效且无回归？ |

### AutoOps 常见反模式

| 反模式 | 应该做 |
|---|---|
| 新增 GORM 模型但未注册到 `migrate.go` | 模型 + 注册一起提交 |
| 前端 API URL 写了 `/api/v1` 前缀 | 让 `request.js` 自动添加 |
| 在修复 bug 时「顺便」增强相邻逻辑 | 只改触发 bug 的那几行 |
| 新增接口但未加 RBAC middleware | 逐路由检查权限码 |
| 用 Redis 语法写 Valkey 相关代码 | 按 Valkey 9 文档确认命令兼容性 |
| 一次性代码写成 Strategy / Factory 模式 | 先写一个函数，需要多态时再重构 |
| 为「未来可能的中间件巡检」预留接口和配置 | 当前只做 Host 巡检，中间件等需求确认后再加 |

## 任务路由

| 任务类型 | 先读 |
|---|---|
| 理解系统边界、请求链路、运行拓扑 | `docs/architecture.md` |
| 后端模型、路由、DAO、Service、错误码、测试 | `docs/backend-guide.md` |
| 前端页面、API 封装、路由、状态、样式 | `docs/frontend-guide.md`, `docs/design-system.md` |
| Docker、Kubernetes、ArgoCD、GitOps 工作树 | `docs/deployment.md`, `docs/deploy-control-plane.md` |
| 安全、认证、RBAC、凭据、审计、WebSocket/SSE | `docs/security-audit.md` |
| N9E / VictoriaMetrics / FlashDuty 集成 | `docs/n9e-integration-review.md` |
| 巡检功能 | `docs/intent/inspection.md`, `docs/TECH-inspection.md`, `docs/PLAN-inspection.md` |
| 规格驱动的新功能或跨模块改动 | `docs/specs/README.md` |
| 本地或生产问题排查 | `docs/troubleshooting.md` |

## 硬约束

- 数据库是 PostgreSQL 17，缓存是 Valkey 9。不要按 MySQL / Redis 语义设计新代码。
- GORM 新模型必须注册到 `api/pkg/db/migrate.go`，并运行 `./scripts/check-migrations`。
- 路由按 `router/{module}/{module}.go` 注册，再接入 `router/router.go`。
- RBAC 权限码格式为 `module:sub:action`，新增敏感接口必须逐路由使用 `middleware.RbacMiddleware("code")`。
- 错误码定义在 `api/common/constant/constant.go`。新增功能优先使用未占用的 470+ 区间；巡检已使用 471–477。
- 配置加载使用 `gopkg.in/yaml.v2`，启动参数为 `-c config.yaml`。
- 前端 API URL 不写 `/api/v1` 前缀，`web/src/utils/request.js` 会自动添加。
- 前端当前 localStorage 命名空间实际 key 是字符串 `"undefined"`，原因见 `docs/frontend-guide.md`。
- `web/vue.config.js` 的 devServer proxy 仍有旧 IP，本地开发需按 `docs/frontend-guide.md` 调整。
- 涉及 JWT、RBAC、Secret、SSH 凭据、SQL 执行、Kubernetes Secret、WebSocket/SSE 的改动必须先读 `docs/security-audit.md`。

## 代码组织约定

后端模块遵循：

```text
api/api/{module}/
  controller/
  service/
  dao/
  model/
```

模型文件通常同时包含 GORM Model、DTO 和 VO。参考 `api/api/cmdb/model/ciType.go`。

前端模块遵循：

```text
web/src/
  api/        # Axios API 模块
  router/     # 模块路由
  store/      # Vuex
  views/      # 页面组件
  utils/      # request.js, storage.js 等
```

## 本地开发环境

本地开发有两条路径。**日常开发用路径 B**（源码热重载）；路径 A 用于部署验证或全栈联调。

### 前置条件

```bash
make bootstrap    # 检查 Go 1.25+、Node 22+、Docker 27+、golangci-lint、psql、kubectl
```

Go 版本由 `mise.toml` 管理（`mise install`）；Node.js 需要自行安装 22+。

### 配置文件准备（首次）

```bash
# 1. Docker 环境变量
cp docker/.env.example docker/.env
# 按需修改端口、密码（默认密码 CHANGE_ME 必须改）

# 2. 后端配置
cp api/config.yaml.example api/config.yaml
# 按需修改 db.password、redis.password、integrations.gitops.local_checkout_path 等
```

两个配置文件的密码必须一致：`docker/.env` 的 `POSTGRES_PASSWORD` = `api/config.yaml` 的 `db.password`；`REDIS_PASSWORD` = `redis.password`。

### 路径 A：Docker 全栈（部署验证用）

所有服务都在容器里运行，通过 `18088`（前端）和 `18000`（API）访问。

```bash
cd docker
docker compose up -d --build
./import-dev-data.sh --force    # 导入 seed 数据（admin / 123456）
```

验证：`curl -sf http://127.0.0.1:18000/api/v1/healthz`

### 路径 B：本地源码开发（推荐）

只有数据库和缓存在容器里，后端和前端直接在本机跑，支持热重载。

```bash
# 终端 1：启动基础服务
make dev-check                  # 启动 postgres/valkey/prometheus/pushgateway + 运行 smoke test

# 终端 2：启动后端（端口 8000）
cd api && go run main.go -c config.yaml

# 终端 3：启动前端（端口 8080）
cd web && npm install && npm run serve
```

**注意**：`web/vue.config.js` 的 devServer proxy 硬编码了旧 IP，本地开发需要改为 `http://localhost:8000`。详见 `docs/frontend-guide.md`。

### 日常操作

```bash
# 格式化 + 检查
make fmt                        # go fmt
make lint                       # golangci-lint + vue lint

# 测试
make test                       # go test + npm lint/build

# 提交前完整检查（等同 CI）
./scripts/presubmit

# 查看 API 日志（Docker 模式）
cd docker && docker compose logs -f devops-api

# 重建单个服务
cd docker && docker compose up -d --build devops-api
```

### 更新（git pull 后）

```bash
# 1. 同步 Go 依赖（go.mod 变更时）
cd api && go mod download

# 2. 同步前端依赖（package.json 变更时）
cd web && npm install

# 3. 数据库迁移（GORM 自动迁移在启动时执行，无需手动操作）
# 启动后端即触发迁移

# 4. 重导 seed 数据（数据库结构破坏性变更时）
cd docker && ./import-dev-data.sh --force

# 5. 重建 Docker 镜像（Dockerfile 或依赖变更时）
cd docker && docker compose up -d --build devops-api devops-web
```

### Docker Compose 环境更新（最常用）

用户说「更新本地开发环境」「更新 Docker 环境」「更新开发环境」时，执行以下流程：

```bash
# 1. 停止需要重建的容器
cd /home/kchou/Code/AutoOps/docker && docker compose down devops-api devops-web

# 2. 重建镜像并启动
docker compose up -d --build devops-api devops-web

# 3. 等待健康检查通过（约 30s）
sleep 20 && docker ps --filter "name=devops" --format "table {{.Names}}\t{{.Status}}"

# 4. 验证服务可用
curl -sf http://127.0.0.1:18000/api/v1/healthz && echo " API_OK"
curl -sf -o /dev/null -w "WEB_HTTP_%{http_code}\n" http://127.0.0.1:18088/
```

**注意**：基础服务（postgres、valkey、prometheus、pushgateway）通常不需要重建，只重建 `devops-api` 和 `devops-web` 即可。如果 `docker/.env` 或 `api/config.yaml` 有改动，才需要 `docker compose down && docker compose up -d --build`。

### 数据管理

```bash
# 备份 PostgreSQL
cd docker && docker compose exec -T postgres pg_dump -U devops autoops > backup.sql

# 恢复 PostgreSQL
cd docker && docker compose exec -T postgres psql -U devops autoops < backup.sql

# 清空所有数据（危险：删除 volume）
cd docker && docker compose down -v

# 重导 seed 数据（保留 volume，只重建系统表）
cd docker && ./import-dev-data.sh --force
```

### 服务端口速查

| 服务 | Docker 全栈 | 本地开发 |
|---|---|---|
| 前端 | `http://127.0.0.1:18088` | `http://localhost:8080` |
| API | `http://127.0.0.1:18000` | `http://localhost:8000` |
| PostgreSQL | `127.0.0.1:15432` | `127.0.0.1:15432` |
| Valkey | `127.0.0.1:16379` | `127.0.0.1:16379` |

完整部署文档见 `docs/deployment.md`，排障见 `docs/troubleshooting.md`。

## 验证规则

- Bug 修复必须包含能复现问题的回归测试，除非现有测试设施确实无法覆盖。
- 非平凡逻辑必须有单元测试或等价验证。
- 前端用户流程改动需要实际启动页面验证；无法验证时在交付说明中明确说明。
- 数据库、RBAC、安全、部署相关改动完成前运行 `./scripts/presubmit`，或说明无法运行的原因。
- 不要为规避失败跳过测试、hook、lint 或安全检查。

## 文档规则

- 默认不新增子目录级 `AGENTS.md` 或 `CLAUDE.md`。只有目录存在独立且长期有效的本地规则时才新增。
- 根级 `AGENTS.md` 仅作为非 Claude harness 的薄入口，内容应指向本文件和 `docs/README.md`。
- 修改行为、接口、架构或安全边界时，同步更新对应 `docs/` 专题文档。
- 中文技术文档使用克制、准确、可扫读的写法；中文与英文、数字之间保留必要空格。

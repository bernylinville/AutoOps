# Backend Development Guide

后端开发时阅读：新增模型、路由、中间件、错误码和测试。

## 1. 基础约束

- 数据库：PostgreSQL 17
- 缓存：Valkey 9（配置中 key 名沿用 `redis`，实际连接 Valkey）
- 配置加载：`gopkg.in/yaml.v2`，启动参数 `-c config.yaml`
- 模块名：`dodevops-api`
- 新 GORM 模型必须注册到 `api/pkg/db/migrate.go`
- 错误码下一个可用段：`470+`（定义在 `api/common/constant/constant.go`）
- 日志：优先使用 `log/slog` 替代 `log.Println` 和 `fmt.Println`

## 2. 模块布局

```
api/api/{module}/
  controller/
  service/
  dao/
  model/
```

路由注册：

- `router/{module}/{module}.go`
- `router/router.go`

参考实现：`api/api/cmdb/model/ciType.go`、`dao/ciType.go`、`controller/ciType.go`

## 3. RBAC

权限码格式：`module:sub:action`（如 `cmdb:sql:select`），通过 `middleware.RbacMiddleware("code")` 逐路由应用。新增 API 端点必须声明权限码。

## 4. 配置文件

关键配置段：

```yaml
db:
  dialects: postgres
  host: postgres
  port: 5432
  db: autoops
  username: devops
  password: ...

redis:                    # 注意：key 名为 redis，实际连接 Valkey
  address: valkey:6379
  password: ...

monitor:
  agent:
    heartbeat_server_url: "http://autoops-api:8000/api/v1/monitor/agent/heartbeat"
    heartbeat_token: "..."

integrations:
  gitops:
    local_checkout_path: /workspace/pukka-gitops
```

## 5. 监控相关约束

Agent 仅保留心跳能力。后端兼容层规则：

- `api/api/monitor/service/monitorService.go` 从外部 N9E / VictoriaMetrics 读取主机基础指标
- `GetTopProcesses` / `GetHostPorts` 允许返回空数组
- Docker Compose 开发环境包含 Prometheus + Pushgateway 用于本地测试；生产依赖外部 N9E

## 6. Direct / GitOps 发布约束

- Direct 模式：限定在受控 namespace / 资源类型内执行
- GitOps 模式：要求 `integrations.gitops.local_checkout_path` 指向有效 `pukka-gitops` 工作树
- 工作树校验检查以下路径：
  - `apps/autoops-managed-releases/`
  - `argocd-apps/templates/autoops-managed-releases.yaml`
  - `apps/autoops/values.yaml`
  - `argocd-apps/templates/autoops.yaml`

## 7. 测试与格式化

```bash
cd api
go fmt ./...        # 格式化
go vet ./...        # 静态分析
golangci-lint run   # Lint（CI 强制执行）
go test ./... -count=1  # 测试
```

完整提交流程：

```bash
./scripts/presubmit
```

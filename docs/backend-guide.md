# Backend Development Guide

## 1. 基础约束

- 数据库：**PostgreSQL 17**
- 缓存：**Valkey 9**
- 配置加载：`gopkg.in/yaml.v2`
- 模块名：`dodevops-api`
- 新模型必须注册到 `api/pkg/db/migrate.go`
- 错误码下一个可用段：`470+`

## 2. 模块布局

```text
api/api/{module}/
  controller/
  service/
  dao/
  model/
```

路由注册：

- `router/{module}/{module}.go`
- `router/router.go`

## 3. 监控相关约束

AutoOps 当前 **不维护内置 Prometheus / Pushgateway**。

后端兼容层规则：

- `api/api/monitor/service/monitorService.go` 从外部 N9E / VictoriaMetrics 读取主机基础指标
- `GetTopProcesses` / `GetHostPorts` 允许返回空数组
- Agent 仅保留 **心跳** 能力，不再负责 Pushgateway 指标推送

相关配置只保留：

```yaml
monitor:
  agent:
    heartbeat_server_url: "http://127.0.0.1:8000/api/v1/monitor/agent/heartbeat"
    heartbeat_token: "..."
```

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

redis:
  address: valkey:6379
  password: ...

integrations:
  gitops:
    local_checkout_path: /workspace/pukka-gitops
```

## 5. Direct / GitOps 发布约束

- Direct 模式：限定在受控 namespace / 资源类型内执行
- GitOps 模式：要求 `integrations.gitops.local_checkout_path` 指向有效 `pukka-gitops` 工作树
- 工作树校验现在同时检查：
  - `apps/autoops-managed-releases/`
  - `argocd-apps/templates/autoops-managed-releases.yaml`
  - `apps/autoops/values.yaml`
  - `argocd-apps/templates/autoops.yaml`

## 6. 测试 / 格式化

```bash
cd api
gofmt -w ./...
go test ./...
```

开发过程中至少验证：

- 受影响包编译通过
- 关键回归测试通过
- GitOps / 监控改动无配置回归

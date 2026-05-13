# Attention

本文件是 CodeStable 技能启动必读的项目注意事项入口。所有 CodeStable 子技能开始工作前必须读取它。

## 项目碎片知识

<!-- cs-note managed: 用 cs-note 维护，新条目按下面分节追加 -->

### 编译与构建

- 后端：Go 1.25 / Gin / GORM；模块名 `dodevops-api`。
- 前端：Vue 3.5 / Element Plus 2.10 / Vuex / Vue Router。
- PR 或推送更新前运行 `./scripts/presubmit` 或 `make presubmit`，必须完整通过。
- 前端开发服务器命令是 `npm run serve`；手动设置中也可见 `npm run dev`，以实际 `web/package.json` 为准。

### 运行与本地起服务

- 推荐全栈开发入口：`cd docker && docker compose up -d`。
- 后端启动：`cd api && go run main.go -c config.yaml`，端口 8000。
- 前端 API URL 不写 `/api/v1` 前缀，`request.js` 拦截器会自动添加。
- `web/vue.config.js` 的 proxy target 可能硬编码为旧 IP，本地开发前需检查。

### 测试

- Bug 修复必须包含能捕获该问题的回归测试。
- 算法或非平凡逻辑需要单元测试。
- 用户可见流程能自动化验证时应补充端到端覆盖。

### 命令与脚本陷阱

- 配置加载使用 `gopkg.in/yaml.v2`，不是 Viper；启动参数为 `-c config.yaml`。
- 缓存配置 key 可能沿用 `redis` 命名，但实际缓存组件是 Valkey 9。

### 路径与目录约定

- 数据库是 PostgreSQL 17，不是 MySQL；缓存是 Valkey 9，不是 Redis。
- 新 GORM 模型必须注册到 `api/pkg/db/migrate.go`，否则不会自动建表。
- 路由注册在 `router/{module}/{module}.go`，并在 `router/router.go` 中调用。
- RBAC 权限码格式为 `module:sub:action`，逐路由应用 `middleware.RbacMiddleware("code")`。
- 错误码：通用 400-434，CI 440-456，项目 460-465，下一个可用段 470+。
- 前端 localStorage 命名空间实际 key 当前为 `"undefined"`。
- CMDB 2.0 Phase 0-8 已完成；新 CMDB 变更参考 `api/api/cmdb/model/ciType.go`、`dao/ciType.go`、`controller/ciType.go`。

### 环境变量与凭证

- 生产环境必须设置 `JWT_SECRET`，否则服务应 panic。
- 不要提交真实凭证、密钥、token 或 kubeconfig。

### 其他

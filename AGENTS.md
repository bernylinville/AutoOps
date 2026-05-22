# AutoOps Agent Entry

本文件面向读取 `AGENTS.md` 的 agent harness。Claude Code 的权威入口是 [CLAUDE.md](CLAUDE.md)；项目文档导航是 [docs/README.md](docs/README.md)。

## 使用方式

1. 先读 [CLAUDE.md](CLAUDE.md) 获取项目硬约束、任务路由、命令和验证规则。
2. 再按任务类型读取 [docs/README.md](docs/README.md) 中对应的专题文档。
3. 修改代码前读取相关源码、测试和已有实现示例，不要只依据文档或记忆。

## 关键提醒

- 数据库是 PostgreSQL 17，缓存是 Valkey 9。
- 新 GORM 模型必须注册到 `api/pkg/db/migrate.go`。
- 前端 API URL 不写 `/api/v1` 前缀。
- 安全、认证、RBAC、凭据、SQL 执行和 Kubernetes Secret 改动必须先读 `docs/security-audit.md`。
- 默认不新增子目录级 `AGENTS.md` 或 `CLAUDE.md`；目录级规则只有在长期、独立且必须自动加载时才新增。

## 常用验证

```bash
./scripts/presubmit
cd api && go test ./... -v -count=1
cd web && npm run lint && npm run build
./scripts/check-migrations
```

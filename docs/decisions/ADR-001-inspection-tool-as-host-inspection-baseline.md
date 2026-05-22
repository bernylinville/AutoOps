# ADR-001: Treat inspection-tool as the Host 巡检 compatibility baseline

## Status
Accepted

## Date
2026-05-22

## Context
AutoOps 的 Host 巡检功能来自 `~/Code/inspection-tool` CLI 平台化改造。用户反馈 AutoOps 当前巡检效果和原 CLI 有出入，尤其是 Excel 报告结构、字段和基线检查内容不一致。

关键约束：
- 运维已经习惯原 CLI 报告，平台化后下载报告应保持可比对、可复用。
- AutoOps 不能直接 import CLI 入口：Cobra/Viper/`os.Exit` 与后端服务模型不兼容。
- AutoOps 仍需要保留平台侧能力：任务管理、DB 持久化、报告文件路径隔离、JWT 下载审计、钉钉通知。
- `inspection-tool` 后期新增了 Host 安全/基线指标，AutoOps 简化版只覆盖资源指标会导致报告缺列。

## Decision
以 `~/Code/inspection-tool` 的 Host 巡检核心语义作为 AutoOps Host 巡检的权威兼容基线：

1. **引擎指标对齐**：AutoOps `api/api/inspection/service/engine/metrics.go` 维护与 `inspection-tool/configs/metrics.yaml` 等价的 Host 指标定义，包含资源指标和基线/安全指标：
   - `public_network`
   - `password_expiry`
   - `password_policy`
   - `open_files`
   - `max_files`
   - `sysctl_params`
2. **报告语义对齐**：AutoOps `WriteHostReport` 必须生成与 CLI Host 报告一致的 workbook 契约：
   - Sheet 顺序：`巡检概览`、`基线检查`、`详细数据`，存在告警时追加 `异常汇总`。
   - 标题：`系统巡检报告`。
   - `基线检查` 是密码、文件句柄、公网访问、sysctl 等基线检查，不再用于普通资源概览。
   - `详细数据` 是“一台主机一行”，不是“一指标一行”。
   - `异常汇总` 使用统一 8 列格式：来源类型、实例标识、告警级别、指标名称、当前值、警告阈值、严重阈值、告警消息。
3. **平台适配边界**：AutoOps 只在必要处偏离 CLI：
   - 不 import CLI/internal 包；移植纯逻辑并去掉 Cobra/Viper/zerolog/`os.Exit`。
   - 报告路径改为 `/data/inspection/{run_id}/inspection_report_{run_id}_{date}.xlsx`，避免多业务组文件名冲突。
   - 下载仍通过 AutoOps JWT + 审计接口。
4. **回归测试约束**：报告 workbook 契约、基线指标定义、筛选兜底逻辑必须用单元测试锁定。

## Alternatives Considered

### 保持 AutoOps 简化报告
- Pros: 实现更少，页面展示和报告可以按平台体验重新设计。
- Cons: 与运维已使用的 CLI 报告不一致，字段缺失，无法直接对比历史巡检。
- Rejected: 用户明确反馈“效果和原来的 inspection-tool 有出入，报告也不一样”，平台化目标不是重新定义巡检结果语义。

### 直接把 inspection-tool 作为外部 CLI 调用
- Pros: 报告天然一致，迁移成本低。
- Cons: 后端难以控制 context、超时、错误、结构化持久化；CLI 的 Cobra/Viper/`os.Exit` 不适合作为服务内依赖。
- Rejected: 不符合 AutoOps 需要 Web 管理、DB 结果查询和通知编排的架构边界。

### 把 inspection-tool 改成共享 Go module 后由 AutoOps import
- Pros: 长期可减少双份实现漂移。
- Cons: 当前 `inspection-tool` 是独立 CLI/internal 结构，拆 module 会扩大变更面，并影响已经可用的 CLI。
- Rejected for now: 后续可以作为单独重构决策；当前优先恢复平台功能与报告兼容。

## Consequences
- AutoOps Host 巡检的正确性判断应优先对照 `inspection-tool` 当前 Host 输出，而不是平台自定义 MVP 输出。
- 后续修改 `inspection-tool/configs/metrics.yaml` 或 `internal/report/excel/writer.go` 时，需要同步评估 AutoOps `metrics.go` 和 `report/excel.go` 是否需要更新。
- 报告 writer 代码会比简化版更长，但换来稳定的用户可见契约和可比对报告。
- 中间件巡检（MySQL/Redis/Nginx/Tomcat/Elasticsearch）仍不属于首版范围；如果后续接入，应单独记录 ADR 或扩展本 ADR。

## Verification
- `go test ./api/inspection/report ./api/inspection/service/engine`
- `go test ./api/inspection/...`
- `go test ./...`
- `git diff --check`

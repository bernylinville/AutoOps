# Implementation Plan: AutoOps 巡检功能

**对应规格**: [TECH-inspection.md](TECH-inspection.md)  
**日期**: 2026-05-19

---

## Overview

将 `~/Code/inspection-tool` CLI 的 Host 巡检核心逻辑直接移植到 AutoOps 后端，按 N9E 业务组组织巡检任务，支持定时自动巡检和手动触发，结果支持 Web 查看 + Excel 下载，异常通过钉钉通知。

## Architecture Decisions

1. **不 import inspection-tool**：直接复制核心逻辑到 `api/api/inspection/`，移除 Cobra/Viper/zerolog 依赖，适配 AutoOps 日志和错误处理
2. **VM Client 独立移植**：AutoOps 现有 `QueryPromQL` 是 `query_range`，巡检需要 `instant query` + `label matcher` 注入，因此独立移植 VM client
3. **PostgreSQL partial unique index**：用 `run_date` + partial index 实现 cron 任务幂等，替代 MySQL 的 `DATE() + FOR UPDATE`
4. **报告存储独立 volume**：`/data/inspection/` 不走 upload 静态目录，下载走 JWT 接口

---

## Task List

### Phase 0: Engine Contract Spike（必须先跑通）

#### Task 1: 创建引擎核心模型
**Description:** 移植 inspection-tool 的 model 类型到 AutoOps，适配包名和日志。

**Acceptance criteria:**
- [ ] `Alert`, `AlertLevel`, `AlertSummary` 类型可用
- [ ] `HostMeta`, `HostResult`, `HostStatus`, `DiskMountInfo` 类型可用
- [ ] `InspectionResult`, `InspectionSummary` 类型可用
- [ ] `MetricValue`, `HostMetrics`, `MetricDefinition`, `MetricStatus` 类型可用
- [ ] 所有类型放在 `api/api/inspection/model/engine.go`

**Verification:**
- [ ] `go build ./api/inspection/model` 编译通过

**Dependencies:** None

**Files:**
- `api/api/inspection/model/engine.go`

**Estimated scope:** M

---

#### Task 2: 移植 VM Client
**Description:** 移植 inspection-tool 的 VM client（instant query + label matcher 注入 + ident 分组解析），移除 zerolog，改用 AutoOps `pkg/log`。

**Acceptance criteria:**
- [ ] `vmclient.Client` 支持 `QueryWithFilter()` 和 `QueryByIdentWithFilter()`
- [ ] `HostFilter` 支持 `busigroup` label matcher 注入
- [ ] 重试策略继承 AutoOps N9E client 配置（3 次重试，指数退避）
- [ ] 日志使用 `pkg/log` 而非 zerolog

**Verification:**
- [ ] `go build ./api/inspection/service/engine/vmclient` 编译通过

**Dependencies:** Task 1

**Files:**
- `api/api/inspection/service/engine/vmclient/client.go`

**Estimated scope:** M

---

#### Task 3: 移植 Collector
**Description:** 移植 inspection-tool 的 Collector，从 N9E 获取主机列表（复用 AutoOps `n9e/service/client.go`），并发查询 VM 指标，标记采集失败主机。

**Acceptance criteria:**
- [ ] `Collector` 实现 `HostCollector` 接口
- [ ] 支持并发采集（默认 20 并发，可配置）
- [ ] 支持 `HostFilter` 过滤（busigroup + tags）
- [ ] 支持 expanded metrics（如 disk by path）
- [ ] 单指标失败不中断整体采集
- [ ] 移除 zerolog，改用 `pkg/log`

**Verification:**
- [ ] `go build ./api/inspection/service/engine` 编译通过
- [ ] 单元测试：`go test ./api/inspection/service/engine/...`

**Dependencies:** Task 1, Task 2

**Files:**
- `api/api/inspection/service/engine/collector.go`
- `api/api/inspection/service/engine/interfaces.go`

**Estimated scope:** L

---

#### Task 4: 移植 Evaluator
**Description:** 移植 inspection-tool 的 Evaluator，评估阈值生成告警，支持 Host 巡检的 6 个指标（cpu_usage, memory_usage, disk_usage, zombie_processes, load_per_core, ntp_offset）。

**Acceptance criteria:**
- [ ] `Evaluator` 实现 `HostEvaluator` 接口
- [ ] 支持 warning/critical 两级阈值
- [ ] NTP 特殊处理（stratum=0 直接 critical）
- [ ] expanded metrics 只展示不告警
- [ ] 移除 zerolog，改用 `pkg/log`

**Verification:**
- [ ] `go build ./api/inspection/service/engine` 编译通过
- [ ] 单元测试通过

**Dependencies:** Task 1

**Files:**
- `api/api/inspection/service/engine/evaluator.go`

**Estimated scope:** M

---

#### Task 5: 移植 Inspector
**Description:** 移植 inspection-tool 的 Inspector，编排 Collector + Evaluator，生成结构化 `InspectionResult`。

**Acceptance criteria:**
- [ ] `Inspector.Run(ctx)` 完成完整巡检流程
- [ ] 支持 `context.Context` 取消和超时
- [ ] 生成包含 Summary + Hosts + Alerts 的 `InspectionResult`
- [ ] 时区固定 `Asia/Shanghai`
- [ ] 移除 zerolog，改用 `pkg/log`

**Verification:**
- [ ] `go build ./api/inspection/service/engine` 编译通过

**Dependencies:** Task 3, Task 4

**Files:**
- `api/api/inspection/service/engine/inspector.go`

**Estimated scope:** S

---

#### Task 6: Engine Spike 验收
**Description:** 编写一个临时测试入口，对一个业务组执行完整巡检，验证过滤、指标解析、结果结构正确。

**Acceptance criteria:**
- [ ] 能对一个业务组执行巡检
- [ ] 产出结构化 `InspectionResult`（含 hosts, metrics, alerts, summary）
- [ ] 无 DB 依赖，纯 engine 测试

**Verification:**
- [ ] 运行测试入口，输出结果 JSON 正确
- [ ] `go test ./api/inspection/service/engine/...` 全部通过

**Dependencies:** Task 5

**Files:**
- `api/api/inspection/service/engine/engine_test.go`（临时，spike 后可保留核心用例）

**Estimated scope:** S

---

### Checkpoint: Phase 0 完成
- [ ] Engine 编译通过
- [ ] 能对一个业务组跑通巡检
- [ ] 指标解析和阈值评估结果正确
- [ ] 评审通过后再进入 Phase 1

---

### Phase 1: 基础框架

#### Task 7: 创建 GORM 数据模型
**Description:** 创建 6 张表的 GORM 模型，遵循 AutoOps Models + DTOs + VOs 同文件约定。

**Acceptance criteria:**
- [ ] `InspectionTask` 模型（含 n9e_group_id, n9e_group_name, target_query, notify 配置）
- [ ] `InspectionRun` 模型（含 run_date 字段）
- [ ] `InspectionTargetResult` 模型（metrics JSON 字段）
- [ ] `InspectionAlert` 模型
- [ ] `InspectionReportArtifact` 模型（含 expires_at）
- [ ] `InspectionNotification` 模型
- [ ] DTOs: `TaskVO`（脱敏）, `UpdateTaskDto`（保留原密钥）

**Verification:**
- [ ] `go build ./api/inspection/model` 编译通过

**Dependencies:** Phase 0

**Files:**
- `api/api/inspection/model/task.go`
- `api/api/inspection/model/run.go`
- `api/api/inspection/model/result.go`
- `api/api/inspection/model/alert.go`
- `api/api/inspection/model/report.go`
- `api/api/inspection/model/notification.go`

**Estimated scope:** M

---

#### Task 8: 注册模型和创建索引
**Description:** 注册模型到 `pkg/db/migrate.go`，通过 GORM 自动迁移 + 手动 SQL 创建 PostgreSQL partial unique index。

**Acceptance criteria:**
- [ ] 6 个模型注册到 `migrate.go`
- [ ] `AutoMigrate` 成功创建表
- [ ] 手动 SQL 创建 `idx_run_unique_daily` partial unique index
- [ ] 所有 GORM 关联和级联删除配置正确

**Verification:**
- [ ] `go run main.go` 启动成功，表自动创建
- [ ] 连接数据库验证表结构和索引

**Dependencies:** Task 7

**Files:**
- `api/pkg/db/migrate.go`
- `api/api/inspection/model/migration.go`（手动 SQL）

**Estimated scope:** S

---

#### Task 9: 创建路由骨架和 Task CRUD API
**Description:** 创建 inspection 模块的路由、controller、service、dao 骨架，实现 Task 的列表、详情、更新 API（密钥脱敏）。

**Acceptance criteria:**
- [ ] `router/inspection/inspection.go` 注册路由
- [ ] `router/router.go` 加入 `inspection.RegisterInspectionRoutes`
- [ ] GET `/inspection/tasks` 返回列表（webhook_url 脱敏）
- [ ] GET `/inspection/tasks/:id` 返回详情（secret 不返回）
- [ ] PUT `/inspection/tasks/:id` 支持更新（空字符串保留原密钥）
- [ ] 错误码使用 471-472

**Verification:**
- [ ] `go build` 编译通过
- [ ] Swagger 文档正确生成
- [ ] 手动 curl 测试 API

**Dependencies:** Task 8

**Files:**
- `router/inspection/inspection.go`
- `api/api/inspection/controller/task.go`
- `api/api/inspection/service/task.go`
- `api/api/inspection/dao/task.go`

**Estimated scope:** M

---

### Checkpoint: Phase 1 完成
- [ ] DB 表结构正确
- [ ] Task CRUD API 可用
- [ ] 密钥脱敏和保留逻辑正确

---

### Phase 2: 巡检引擎固化

#### Task 10: 集成 Engine 到 Service 层
**Description:** 将 Phase 0 的 engine 集成到 inspection service，实现从 DB 任务到 engine 执行的完整链路。

**Acceptance criteria:**
- [ ] `InspectionService` 能根据 `InspectionTask` 创建 engine 配置
- [ ] 从 N9E 配置读取 endpoint/token 创建 N9E client
- [ ] 从 task 的 `target_query` 创建 VM HostFilter
- [ ] 执行巡检并返回 `InspectionResult`

**Verification:**
- [ ] `go build` 编译通过
- [ ] 单元测试通过

**Dependencies:** Task 9, Phase 0

**Files:**
- `api/api/inspection/service/inspection.go`

**Estimated scope:** M

---

#### Task 11: 实现运行记录持久化
**Description:** 巡检完成后，将结果持久化到 DB（run + target_result + alert）。

**Acceptance criteria:**
- [ ] 创建 `InspectionRun` 记录
- [ ] 批量创建 `InspectionTargetResult` 记录
- [ ] 批量创建 `InspectionAlert` 记录
- [ ] 汇总统计正确（total/normal/warning/critical/failed/alert_count）
- [ ] 使用事务保证一致性

**Verification:**
- [ ] 手动触发一次巡检，DB 中数据完整
- [ ] 关联查询正确

**Dependencies:** Task 10

**Files:**
- `api/api/inspection/dao/run.go`
- `api/api/inspection/dao/result.go`
- `api/api/inspection/dao/alert.go`

**Estimated scope:** M

---

#### Task 12: 实现并发控制和跳过重入
**Description:** 全局信号量限制并发（上限 5），单任务跳过重入。

**Acceptance criteria:**
- [ ] 全局信号量 `maxConcurrent = 5`
- [ ] 同一任务运行中时，新触发直接跳过
- [ ] 手动触发不受跳过重入限制
- [ ] 超时控制：单次巡检 10 分钟全局超时

**Verification:**
- [ ] 并发测试：同时触发多个任务，不超过 5 个并发
- [ ] 重入测试：同一任务连续触发，第二次跳过

**Dependencies:** Task 11

**Files:**
- `api/api/inspection/service/scheduler.go`

**Estimated scope:** S

---

### Checkpoint: Phase 2 完成
- [ ] 手动触发巡检，结果正确持久化到 DB
- [ ] 并发和重入控制正常工作

---

### Phase 3: 调度与执行

#### Task 13: 实现定时调度器
**Description:** 引入 `github.com/robfig/cron/v3`，每个启用的 task 注册 cron job，时区固定 Asia/Shanghai。

**Acceptance criteria:**
- [ ] cron scheduler 使用 `cron.WithLocation(Asia/Shanghai)`
- [ ] 服务启动时加载所有 enabled task 注册调度
- [ ] 任务配置变更时重新注册
- [ ] 服务停止时优雅关闭 scheduler

**Verification:**
- [ ] 设置 cron 为每分钟，验证定时触发
- [ ] 验证时区正确（北京时间）

**Dependencies:** Task 12

**Files:**
- `api/api/inspection/scheduler/scheduler.go`
- `api/scheduler/inspectionScheduler.go`（或集成到现有 scheduler）

**Estimated scope:** M

---

#### Task 14: 实现手动触发 API
**Description:** 实现 POST `/inspection/tasks/:id/trigger`，支持前端手动触发巡检。

**Acceptance criteria:**
- [ ] 接口返回 run ID
- [ ] 异步执行，不阻塞 HTTP 响应
- [ ] 前端可轮询运行状态
- [ ] 记录 triggered_by 用户 ID

**Verification:**
- [ ] curl 测试手动触发，返回 run ID
- [ ] DB 中记录正确

**Dependencies:** Task 12

**Files:**
- `api/api/inspection/controller/inspection.go`

**Estimated scope:** S

---

#### Task 15: N9E 同步后自动创建/停用任务
**Description:** N9E 全量同步完成后，自动为新增业务组创建任务，为删除业务组停用任务。

**Acceptance criteria:**
- [ ] 同步完成后遍历 `n9e_busi_group`
- [ ] 不存在对应 task 的业务组，创建默认任务（enabled=false）
- [ ] 业务组从 N9E 删除时，对应 task 标记 enabled=false
- [ ] 不删除 task 和 history，保留审计

**Verification:**
- [ ] 模拟新增业务组，验证自动创建 task
- [ ] 模拟删除业务组，验证自动停用

**Dependencies:** Task 13

**Files:**
- `api/api/inspection/service/sync.go`
- `api/api/n9e/service/sync.go`（hook 点）

**Estimated scope:** M

---

### Checkpoint: Phase 3 完成
- [ ] 定时调度正常触发
- [ ] 手动触发可用
- [ ] N9E 同步后任务自动同步

---

### Phase 4: 报告与通知

#### Task 16: 适配 Excel 报告生成
**Description:** 移植 inspection-tool 的 Excel writer，修复多业务组文件名冲突，适配 AutoOps 路径。

**Acceptance criteria:**
- [ ] 移植 Excel writer 到 `api/api/inspection/report/`
- [ ] 文件名规则：`{output_dir}/{run_id}/inspection_report_{run_id}_{date}.xlsx`
- [ ] 包含概览、基线检查、详细数据、异常汇总 4 个 sheet
- [ ] 条件格式：warning 黄色，critical 红色，normal 绿色

**Verification:**
- [ ] 生成 Excel 文件，格式正确
- [ ] 中文显示正常

**Dependencies:** Task 11

**Files:**
- `api/api/inspection/report/excel.go`

**Estimated scope:** L

---

#### Task 17: 实现报告下载 API
**Description:** 实现 `GET /inspection/runs/:id/report`，JWT 鉴权 + 审计 + 流式下载。

**Acceptance criteria:**
- [ ] 通过 run_id 查找文件路径
- [ ] 校验文件路径在 `/data/inspection/` 根目录内（防路径遍历）
- [ ] JWT 鉴权
- [ ] 审计日志记录下载行为
- [ ] Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet

**Verification:**
- [ ] curl 测试下载，文件完整
- [ ] 尝试路径遍历攻击，被阻止

**Dependencies:** Task 16

**Files:**
- `api/api/inspection/controller/report.go`

**Estimated scope:** S

---

#### Task 18: 实现钉钉通知
**Description:** 复用 `deploy/service/notifier.go` 的 dingtalkbot，按任务配置发送 Markdown 汇总通知，落库通知记录。

**Acceptance criteria:**
- [ ] critical 必通知，warning/失败 按配置
- [ ] Markdown 格式：任务名、触发方式、耗时、主机统计、异常汇总、链接
- [ ] 通知结果落库 `inspection_notification`
- [ ] 通知失败不阻塞主流程，记录 error

**Verification:**
- [ ] 触发含异常巡检，验证钉钉消息格式
- [ ] DB 中 `inspection_notification` 记录正确

**Dependencies:** Task 11

**Files:**
- `api/api/inspection/service/notification.go`

**Estimated scope:** M

---

### Checkpoint: Phase 4 完成
- [ ] Excel 报告生成正确
- [ ] 报告下载可用
- [ ] 钉钉通知格式正确，落库完整

---

### Phase 5: 前端与收尾

#### Task 19: 前端页面开发
**Description:** 开发 4 个前端页面：概览、任务管理、运行历史、运行详情。

**Acceptance criteria:**
- [ ] 巡检概览页：今日统计、最近异常、快速入口
- [ ] 任务管理页：列表、启用开关、通知配置
- [ ] 运行历史页：列表、状态筛选、分页
- [ ] 运行详情页：Tab 切换（概览/主机/异常）、报告下载按钮

**Verification:**
- [ ] 页面功能测试
- [ ] 响应式布局正常

**Dependencies:** Phase 4

**Files:**
- `web/src/views/inspection/` 下多个 Vue 文件

**Estimated scope:** L

---

#### Task 20: 实现清理任务
**Description:** 每天凌晨 3 点执行，删除 30 天前的运行记录和报告文件。

**Acceptance criteria:**
- [ ] 定时任务扫描 `inspection_run` 删除 30 天前记录
- [ ] 级联删除 target_result, alert, report_artifact, notification
- [ ] 删除对应文件系统中的报告文件
- [ ] 使用事务或独立清理任务

**Verification:**
- [ ] 手动插入旧数据，验证清理正确

**Dependencies:** Phase 4

**Files:**
- `api/api/inspection/service/cleanup.go`

**Estimated scope:** S

---

#### Task 21: 集成测试与收尾
**Description:** 端到端测试，修复 bug，补充文档。

**Acceptance criteria:**
- [ ] 定时巡检自动触发并正确执行
- [ ] 手动触发可用
- [ ] 报告下载可用
- [ ] 钉钉通知正常
- [ ] 前端页面功能完整
- [ ] 清理任务正常

**Verification:**
- [ ] 全流程测试
- [ ] `go test ./api/inspection/...` 通过

**Dependencies:** Task 19, Task 20

**Files:**
- 多文件修复

**Estimated scope:** M

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Phase 0 engine spike 失败 | 高 | 先验证 VM/N9E 过滤和指标解析，确认后再固化 |
| Excel 报告生成依赖 excelize | 中 | 确认 AutoOps go.mod 已含或允许添加 |
| cron 库引入 | 低 | `github.com/robfig/cron/v3` 是标准库，风险低 |
| 前端开发时间不确定 | 中 | Phase 5 可独立进行，不阻塞后端 |

## Open Questions

- [ ] Phase 0 验收时，用哪个业务组做 spike 测试？
- [ ] 钉钉 Webhook URL 是全局配置还是按任务配置？（当前按任务）
- [ ] 前端框架是否有现成的表格/分页组件可用？

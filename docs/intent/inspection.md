# AutoOps 巡检功能 — 确认版意图

## Outcome

在 AutoOps 平台新增 Host 巡检功能，将 `~/Code/inspection-tool` CLI 的核心巡检逻辑抽取为可被调用的库后内嵌到 AutoOps 后端。运维人员在 Web 上管理巡检任务、查看结构化结果、下载 Excel 报告。

## User

已登录的 AutoOps 运维人员。

## Why Now

已有成熟的 CLI 巡检工具，但缺乏 Web 化管理能力（任务调度、结果可视化、报告下载），需要平台化整合。

## Success Criteria

1. 每个 N9E 业务组每天北京时间 10 点自动执行巡检
2. 支持手动触发巡检
3. 巡检结果结构化存储，支持 Web 列表查看、按业务组/主机/异常级别筛选、异常汇总
4. 支持下载 Excel 报告
5. 巡检异常通过钉钉 Webhook 按任务维度汇总通知
6. 历史数据（运行记录、目标结果、异常明细、报告文件）保留 30 天自动清理

## Constraint

- 首版仅 Host 巡检，中间件巡检（MySQL/Redis/Nginx/Tomcat/Elasticsearch）后续迭代
- 巡检目标限定为 N9E 监控纳管的机器，按业务组组织
- Excel 报告生成以 CLI 当前 Host 报告为兼容基线：sheet、字段和一主机一行明细保持一致；AutoOps 仅修复多业务组文件名冲突和下载鉴权/审计
- Web 页面展示重新设计
- 权限：所有已登录用户可见，下载接口走 JWT + 审计
- 调度防并发冲击：队列化执行、最大并发上限 5 组、超时控制、跳过重入、失败处理
- CLI 核心代码需先抽取为库（接收 `context.Context`，返回结构化结果，不依赖 Cobra/Viper/`os.Exit`），再被 AutoOps 调用
- 通知规则：critical 必通知，warning 可选（默认通知），采集失败/无主机通知任务负责人

## Out of Scope

- 不做修复执行 / 自动处置
- 不做报表模板在线编辑
- 不做非 N9E 资产巡检
- 不做多通知渠道（仅钉钉 Webhook）
- 不做复杂跨业务组权限隔离
- 不做巡检阈值热更新（配置文件变更后需重启/重载生效）
- 中间件巡检（MySQL/Redis/Nginx/Tomcat/Elasticsearch）纳入后续迭代

## 关键决策记录

| 决策 | 选择 | 原因 |
|------|------|------|
| 首版巡检范围 | Host only | 最核心场景，降低首版风险，中间件后续迭代 |
| 调度并发策略 | 并发 + 上限 5 组 | 避免 72 个业务组同时启动压垮系统，同时不拖太久 |
| 通知规则 | critical 必通知，warning 默认通知，失败/无主机通知 | 可配置，默认覆盖主要异常场景 |
| 巡检兼容基线 | inspection-tool Host 报告/指标语义 | 运维已有报告依赖，平台化不重新定义巡检结果；详见 `docs/decisions/ADR-001-inspection-tool-as-host-inspection-baseline.md` |

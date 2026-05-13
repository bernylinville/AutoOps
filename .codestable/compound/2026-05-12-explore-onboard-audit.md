---
doc_type: explore
type: module-overview
status: current
summary: CodeStable onboarding scan and legacy documentation mapping candidates for AutoOps.
tags:
  - onboarding
  - codestable
---

# CodeStable Onboarding Audit — AutoOps

扫描日期：2026-05-12

## 扫描结论

- `.codestable/`：本次创建，之前不存在。
- 旧版目录：未发现 `easysdd/`、`codestable/`、`.easysdd/`。
- 现有文档：仓库已有较完整的 `docs/`、根目录说明和历史归档，因此按「迁移路径」处理。
- 安全策略：本次未移动、删除或重命名任何已有文档；仅创建 CodeStable 骨架、复制共享工具/参考，并记录候选映射。

## 高置信度映射候选

这些文件语义明确，适合后续通过对应 CodeStable 子技能 backfill；当前均保留原位。

| 现有文件 | 推测内容类型 | 建议归入 CodeStable | 置信度 | 本次处理 |
|---|---|---|---|---|
| `docs/architecture.md` | 系统架构入口 | `.codestable/architecture/ARCHITECTURE.md` 或拆分为 `architecture/system-*.md` | 高 | 已在 `ARCHITECTURE.md` 建索引，原文保留 |
| `docs/backend-guide.md` | 后端开发指南 / 约束 | `.codestable/architecture/backend-api.md` + `attention.md` 摘要 | 高 | 摘要写入 `attention.md`，原文保留 |
| `docs/frontend-guide.md` | 前端开发指南 / 约束 | `.codestable/architecture/frontend-web.md` + `attention.md` 摘要 | 高 | 摘要写入 `attention.md`，原文保留 |
| `docs/deployment.md` | 部署与本地环境指南 | `.codestable/architecture/infra-deployment.md` 或外部 guide 保留 | 高 | 原文保留 |
| `docs/security-audit.md` | 安全审计跟踪 | `.codestable/issues/` 或 `compound/*-decision/security-*` 后续拆分 | 高 | 原文保留 |
| `docs/deploy-control-plane.md` | Deploy 模块架构 | `.codestable/architecture/deploy-control-plane.md` | 高 | 已在架构入口索引，原文保留 |
| `docs/design-system.md` | UI 设计系统 | `.codestable/architecture/ui-design-system.md` | 高 | 原文保留 |
| `docs/troubleshooting.md` | 运维排障指南 | 外部 guide 保留；关键坑可用 `cs-learn` 沉淀 | 高 | 原文保留 |
| `docs/n9e-integration-review.md` | N9E 集成评审 | `.codestable/architecture/monitoring-n9e.md` 或历史 feature acceptance | 高 | 原文保留 |

## 中置信度映射候选

这些文件可归入 CodeStable，但需要 owner 确认目标层级后再迁移。

| 现有文件 | 推测内容类型 | 候选位置 | 置信度 | 本次处理 |
|---|---|---|---|---|
| `docs/dingtalk-build-deploy-v1-scope.md` | DingTalk 构建部署范围声明 | `requirements/dingtalk-build-deploy.md` 或 `roadmap/dingtalk-build-deploy/` | 中 | 保留原位，待确认 |
| `docs/dingtalk-build-deploy-v1.1-plan.md` | Direct 模式规划 | `roadmap/dingtalk-direct-mode/` | 中 | 保留原位，待确认 |
| `docs/dingtalk-hermes-api-contract.md` | Agent/Hermes API 契约 | `architecture/integration-hermes-agent.md` 或 external API guide | 中 | 保留原位，待确认 |
| `docs/dingtalk-oa-template.md` | 钉钉 OA 审批方案 | `architecture/integration-dingtalk-oa.md` 或 roadmap 子项 | 中 | 保留原位，待确认 |
| `docs/dingtalk-userid-bootstrap.md` | 操作指南 | external user guide 保留；关键前置可进 `attention.md` | 中 | 保留原位，待确认 |
| `docs/api-changelog.md` | API 变更日志 | external changelog 保留；重大契约拆到 architecture | 中 | 保留原位，待确认 |
| `progress.md` | CMDB 2.0 历史进度 | `compound/*-explore-cmdb-history.md` 或历史归档保留 | 中 | 保留原位，待确认 |
| `docs/full-repo-review.md` | 全仓审计报告 | `compound/*-explore-security-review.md` 或 issues 拆分 | 中 | 保留原位，待确认 |

## 低置信度 / 建议保留原位

| 现有文件 | 原因 | 本次处理 |
|---|---|---|
| `README.md` | 项目对外入口，不建议迁入 CodeStable | 保留原位 |
| `CLAUDE.md` / `AGENTS.md` | Agent/工程约束入口，CodeStable 不兼容读取为注意事项入口；关键约束已摘入 `attention.md` | 保留原位 |
| `docker/README.md` | 已标注过时，由 `docs/deployment.md` 替代 | 保留原位 |
| `docs/specs/**` | 旧 spec-driven 模板/示例，与 CodeStable 工作流重叠；是否保留需 owner 决策 | 保留原位 |
| `docs/archive/**` | 历史交接/临时方案/旧版文档，适合保留归档或按需拆成 learning/explore | 保留原位 |
| `docs/superpowers/plans/**` | 旧规划产物，是否迁入 roadmap 需 owner 决策 | 保留原位 |
| `docs/issue-labels.md` | Issue 标签体系，可能仍是外部协作指南 | 保留原位 |

## 后续建议

1. 用 `cs-arch backfill` 从 `docs/architecture.md`、`docs/backend-guide.md`、`docs/frontend-guide.md`、`docs/deploy-control-plane.md` 生成 CodeStable 架构现状文档。
2. 用 `cs-req backfill` 或 `cs-roadmap new/update` 处理 DingTalk 构建部署相关文档。
3. 用 `cs-learn` 把 `docs/archive/**` 中仍有价值的踩坑和经验沉淀为 `compound/*-learning-*`。
4. 明确 `docs/specs/**` 与 CodeStable 新流程的关系：保留历史参考、迁入、或废弃。

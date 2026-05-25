# AutoOps 文档索引

本目录保存 AutoOps 的当前架构、开发约束、部署说明和专题设计。根目录 `CLAUDE.md` 只作为 agent harness 入口；具体规则以本目录中的专题文档为准。

## 先读哪一篇

| 目标 | 文档 | 说明 |
|---|---|---|
| 理解系统整体 | [architecture.md](architecture.md) | 技术基线、请求链路、开发 / 生产拓扑、监控和 GitOps 约束 |
| 写后端代码 | [backend-guide.md](backend-guide.md) | 模块布局、路由、RBAC、配置、监控兼容、测试命令 |
| 写前端代码 | [frontend-guide.md](frontend-guide.md) | API 封装、路由、Vuex、localStorage、devServer proxy、页面 checklist |
| 调整 UI | [design-system.md](design-system.md) | Element Plus、颜色、排版、间距、组件交互约定 |
| 部署和环境 | [deployment.md](deployment.md) | Docker Compose、Kubernetes + ArgoCD、Secret、GitOps 工作树、外部监控 |
| 部署控制台 | [deploy-control-plane.md](deploy-control-plane.md) | Direct / GitOps 发布模型和 pukka-gitops 工作树分工 |
| 安全相关改动 | [security-audit.md](security-audit.md) | 高危 / 中危问题状态、修复优先级和完整审计来源 |
| 排查问题 | [troubleshooting.md](troubleshooting.md) | Docker、数据库、前端、Kubernetes、N9E 常见问题 |
| N9E 集成 | [n9e-integration-review.md](n9e-integration-review.md) | N9E 同步、数据模型、端点、已验证问题和后续关注项 |
| 巡检功能意图 | [intent/inspection.md](intent/inspection.md) | Host 巡检的目标、成功标准、约束和非目标 |
| 巡检技术设计 | [TECH-inspection.md](TECH-inspection.md) | 巡检模块架构、数据模型、API、报告、通知、清理策略 |
| 巡检实施计划 | [PLAN-inspection.md](PLAN-inspection.md) | 分阶段任务、验收标准、风险和开放问题 |
| 规格驱动开发 | [specs/README.md](specs/README.md) | 何时写 PRODUCT / TECH spec、模板和验证规则 |

## 文档分层

```text
CLAUDE.md                 # 根级 agent harness 入口：硬规则、任务路由、命令
AGENTS.md                 # 其他 agent harness 的薄入口，避免重复规则
README.md                 # 项目介绍和面向人的快速入口

docs/
  README.md               # 本索引
  architecture.md         # 当前系统地图
  backend-guide.md        # 后端开发规则
  frontend-guide.md       # 前端开发规则
  deployment.md           # 环境与部署
  security-audit.md       # 安全审计跟踪
  troubleshooting.md      # 排障手册
  specs/                  # 规格驱动开发模板
  intent/                 # 确认后的产品意图
  decisions/              # ADR 和长期决策
  archive/                # 历史材料，只作背景
```

## 写文档的原则

- **源码优先**：行为、接口和字段必须能在当前代码中验证；不能验证的内容标为待确认，不写成事实。
- **少复制**：同一规则只放在一个权威位置，其他文件用链接引用。
- **少而准**：优先维护根级 `CLAUDE.md` 和专题文档，不为每个目录铺设 agent 文件。
- **写边界**：明确什么适用、什么不适用、什么时候要读下一篇。
- **写验证**：涉及实现的文档应给出可运行的验证命令或检查路径。
- **精准更新**：只改与当前任务有关的段落；不要借文档调整顺手重写无关内容。
- **归档不删史**：过时但有历史价值的材料放入 `docs/archive/`，不要继续从根入口链接为主线文档。

## 新增或修改文档时

1. 判断是否已有专题文档可更新，优先更新现有文档。
2. 跨模块、数据库、安全或部署行为变更，先看 [specs/README.md](specs/README.md)。
3. 架构决策长期有效时，写入 `docs/decisions/`，不要只写在任务计划里。
4. 修改文档索引时，确认链接存在。
5. 不要为了局部目录创建新的 `AGENTS.md` / `CLAUDE.md`，除非该目录有长期、独立、必须自动加载的规则。

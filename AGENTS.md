# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**AutoOps** — Go + Vue 3 构建的统一运维管理平台。管理 CMDB 资产、Kubernetes 集群、任务自动化、监控告警（N9E/FlashDuty）和配置中心。

| 层 | 技术 |
|----|------|
| Backend | Go 1.25 / Gin / GORM / PostgreSQL 17（模块名 `dodevops-api`） |
| Frontend | Vue 3.5 / Element Plus 2.10 / Vuex / Vue Router |
| Infra | PostgreSQL 17.4, Valkey 9（Redis 协议兼容）, Prometheus, Pushgateway |

## Documentation — 按场景阅读

| 文档 | 何时阅读 |
|------|---------|
| [docs/architecture.md](docs/architecture.md) | 理解系统架构、请求链路、模块边界、启动序列 |
| [docs/backend-guide.md](docs/backend-guide.md) | 后端开发：新增模型/路由/中间件/错误码/测试 |
| [docs/frontend-guide.md](docs/frontend-guide.md) | 前端开发：新增页面/API 调用/路由/状态管理 |
| [docs/deployment.md](docs/deployment.md) | Docker 环境搭建、配置说明、seed 数据导入 |
| [docs/security-audit.md](docs/security-audit.md) | 涉及安全相关修改前必读（9 个高危漏洞跟踪） |
| [docs/design-system.md](docs/design-system.md) | UI 样式/颜色/排版/间距修改 |
| [docs/troubleshooting.md](docs/troubleshooting.md) | 开发或部署遇到报错时 |
| [docs/n9e-integration-review.md](docs/n9e-integration-review.md) | N9E 监控集成开发 |
| [docs/deploy-control-plane.md](docs/deploy-control-plane.md) | Deploy 模块、Direct/GitOps、审批链路开发 |
| [docs/dingtalk-oa-template.md](docs/dingtalk-oa-template.md) | 钉钉 OA 审批模板设计与权限说明 |

## Commands

### Docker（推荐全栈开发）
```bash
cd docker
docker compose up -d                     # 启动 6 个服务
docker compose logs -f devops-api        # 查看 API 日志
docker compose down                      # 停止
```

### Backend
```bash
cd api
go run main.go -c config.yaml            # 启动 API（端口 8000）
go build -o devops-api .                 # 编译
go test ./...                            # 全量测试
go test -v -run TestName ./path/to/pkg/  # 单个测试
make swagger                             # 重新生成 Swagger 文档
make fmt && make lint                    # 格式化 + 检查
```

### Frontend
```bash
cd web
npm install
npm run serve                            # 开发服务器（端口 8080）
npm run build                            # 生产构建
npm run lint                             # ESLint
```

## Key Constraints

1. **数据库是 PostgreSQL 17**，不是 MySQL。缓存是 **Valkey 9**，不是 Redis
2. 新 GORM 模型**必须**注册到 `api/pkg/db/migrate.go`，否则不会自动建表
3. 路由注册在 `router/{module}/{module}.go`，在 `router/router.go` 中调用
4. 错误码分配：通用 400-434，CI 440-456，项目 460-465，**下一个可用段 470+**（定义在 `api/common/constant/constant.go`）
5. RBAC 权限码格式 `module:sub:action`，通过 `middleware.RbacMiddleware("code")` 逐路由应用
6. JWT 生产环境**必须**设置 `JWT_SECRET` 环境变量（否则 panic）
7. 配置加载用 `gopkg.in/yaml.v2`（不是 Viper），启动参数 `-c config.yaml`
8. 前端 API URL **不需要**写 `/api/v1` 前缀 — request.js 拦截器自动添加
9. 前端 localStorage 命名空间实际 key 是 `"undefined"`（VUE_APP_NAME_SPACE 构建时未注入）
10. 前端 `vue.config.js` proxy target 硬编码为旧 IP，本地开发需修改

## Architecture Quick Reference

```
HTTP → Recovery → CORS → Logger
  ├─ [公开] /captcha, /login, /healthz, /monitor/agent/heartbeat
  └─ [JWT] → AuthMiddleware → LogMiddleware → AuditMiddleware → [RBAC per route]
       → Controller → Service → DAO → PostgreSQL
```

模块布局：`api/api/{module}/controller/`, `service/`, `dao/`, `model/`
模型约定：Models + DTOs + VOs 同文件，参考 `api/api/cmdb/model/ciType.go`

## Active Context

CMDB 2.0 升级已完成 Phase 0-8（详见 `progress.md`）。新功能开发时，以 CI 模型文件（`api/api/cmdb/model/ciType.go`、`dao/ciType.go`、`controller/ciType.go`）作为参考实现。

## Developer Skills

This project uses executable skills in `.agents/skills/` to capture team knowledge and provide repeatable workflows. These skills serve as **bus-factor protection** for a small team (≤3 people).

### Available Skills

| Skill | Purpose | When to Use |
|-------|---------|-------------|
| `deploy-flow` | Deploy through AutoOps pipeline | Releasing, hotfixing, onboarding new projects |
| `db-migration` | Add/modify GORM models and migrations | Creating tables, adding columns, schema changes |
| `cmdb-change` | Modify CMDB assets or CI types | New asset types, attribute changes, lifecycle updates |
| `incident-recovery` | Recover from production failures | Service down, deployment failures, DB issues |
| `spec-driven-implementation` | Drive spec-first workflow for features | Starting significant features, agent-driven implementation |
| `write-product-spec` | Write PRODUCT.md with behavior invariants | Defining feature behavior before implementation |
| `write-tech-spec` | Write TECH.md grounded in codebase context | Implementation planning for cross-module changes |
| `review-pr` | Structured PR review for correctness and security | Reviewing PRs, pre-merge quality check |
| `go-modern-practices` | Modern Go 1.25+ idioms and conventions | Writing/refactoring Go code, PR review reference |

### Using Skills

Skills are markdown files with structured instructions. Both human developers and AI agents can follow them.

To use a skill:
1. Read the skill file: `.agents/skills/<skill-name>/SKILL.md`
2. Follow the workflow section step by step
3. Complete the verification checklist before finishing

### Adding New Skills

Create a new skill when:
- A workflow is repeated ≥3 times
- The workflow is complex and easy to forget steps
- The workflow involves risk (production deploy, DB changes)
- A new team member would struggle without guidance

Skill format:
```markdown
---
name: skill-name
description: One-line purpose
---

# Skill Title

## When to Use

## Prerequisites

## Workflow

### Step 1: ...
### Step 2: ...

## Verification Checklist
- [ ] ...
```

## Development Commands

### Pull Request Workflow

- **ALWAYS** run `./scripts/presubmit` (or `make presubmit`) before opening a PR or pushing updates to an existing PR branch. It must pass completely — if `go fmt`, `go vet`, `go test`, `npm run lint`, or `npm run build` fail, fix all issues before proceeding.

### Testing Requirements

- **Bug fixes** must include a regression test that would have caught the bug.
- **Algorithmic or non-trivial logic** requires unit tests.
- **User-facing flows** should have end-to-end coverage whenever the behavior can be exercised that way.
- If a change genuinely cannot be tested automatically, document the reason in the PR description.

### Quick Start
```bash
make bootstrap    # Check environment
make dev-check    # Start services + smoke test
make fmt          # Format Go code
make lint         # Run all linters
make test         # Run all tests
make help         # Show all commands
```

### Manual Setup
```bash
# Check prerequisites
./scripts/bootstrap

# Start services
cd docker && docker compose up -d postgres valkey

# Smoke test
./scripts/smoke-test

# Run backend
cd api && go run .

# Run frontend
cd web && npm run dev
```

## Legacy Docs Status

| 文件 | 状态 | 说明 |
|------|------|------|
| `DEVELOPMENT.md` | ⚠️ 过时 | 仍引用 MySQL/Redis，仅 P1-P7 路线图段有参考价值 |
| `handoff_context.md` | ⚠️ 过时 | Phase 2-4 标记为"待接手"但已完成，仅 Phase 0-1 背景有历史价值 |
| `progress.md` | ✅ 权威 | Phase 0-8 完整进度记录，API 端点清单，错误码分配 |
| `docker/README.md` | ⚠️ 过时 | 仍引用 MySQL/Redis，由 `docs/deployment.md` 替代 |
| `.impeccable.md` | ✅ 有效 | 设计系统完整版，`docs/design-system.md` 为精简版 |
| `docs/full-repo-review.md` | ✅ 有效 | 原始安全审计，`docs/security-audit.md` 为跟踪版 |


<claude-mem-context>
# Memory Context

# [AutoOps] recent context, 2026-05-06 1:36pm GMT+8

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 50 obs (16,369t read) | 547,567t work | 97% savings

### Apr 29, 2026
S5 $ralplan — Plan for Spring Boot demo with no-DB, GitLab push, then full CI/CD pipeline via bot (Jenkins → Harbor → AutoOps → K8s) (Apr 29, 9:02 AM)
S3 User asked how to talk/interact with the system to eventually deploy a demo app with external access via NodePort (Apr 29, 9:02 AM)
S4 User asked about deployment workflow for a demo app with external NodePort access via AutoOps (Apr 29, 9:02 AM)
S6 用户发现已存在的 GitLab 远程仓库 `git@gayhub.seeingtv.com:ipaas/java-demo.git`，询问如何处理本地 demo 和远程仓库的关系，然后推到 GitLab 用于测试 AutoOps 部署流程。 (Apr 29, 9:13 AM)
101 10:25a 🔵 Deployment selector mismatch root cause identified: old web and valkey use single-label selectors
102 " 🔵 autoops-api new pod fails because sshSecretName="" still creates empty-named Secret volume
103 " 🔵 autoops-postgres fails: secret "autoops-runtime-secrets" not found
104 " 🔵 autoops-api deployment still has gitops-ssh volume with hardcoded autoops-gitops-ssh secret name
106 10:26a 🔵 Helm template selectorLabels helper defined; web and valkey use it with component suffix
107 10:27a 🔵 autoops-runtime Secret exists but with wrong name: "autoops-runtime" vs configured "autoops-runtime-secrets"
108 " ⚖️ Shift from deployment to end-to-end testing of AutoOps workflow
110 10:37a 🔵 Exploring service mesh and ingress topology for end-to-end testing
109 10:44a ⚖️ Deployment confirmation and completion persistence finalized
111 10:46a 🔵 Service mesh topology mapped for AutoOps and platform services
113 " ⚖️ Shift to demo project creation for end-to-end deployment testing
114 " 🔵 Exploring AutoOps deploy API endpoints for bot interaction
115 " 🔵 AutoOps deploy module file structure mapped
112 " 🔵 AutoOps and Harbor externally accessible via Envoy Gateway, Jenkins returns 403
116 10:49a 🔵 AutoOps deploy module API surface fully mapped for agent/bot interaction
117 " 🔵 AutoOps agent build-deploy flow designed for Spring Boot projects via DingTalk
118 10:51a 🔵 ~/Code directory inventoried — no demo project exists yet for GitLab push testing
119 11:09a 🔵 User reports "network busy" errors from DingTalk bot for opsclaw/hermes-agents
S7 用户通过钉钉机器人与 opsclaw hermes-agents 对话得到"网络繁忙"回复，要求排查 hermes-agents 服务状态。之前已完成 java-demo 项目分析和 Jenkinsfile 修改准备推送。 (Apr 29, 11:36 AM)
### Apr 30, 2026
138 12:35a 🔵 AutoOps deploy pipeline has no dedicated retry/reset endpoint for failed pipeline runs
137 " 🔵 AutoOps Deploy Module API Exploration Requested
145 " 🔵 AutoOps deploy integration API endpoint search yielded weak evidence
136 " 🔵 AutoOps deploy integration API endpoint structure explored
139 12:38a 🔵 AutoOps Deploy Module Agent API Endpoints Fully Mapped
140 " 🔵 Agent Outbox Event Model for Reliable Hermes Delivery
141 " 🔵 CreateAgentBuildDeployRequest Profile Resolution Flow
142 " 🔵 CreateAgentProjectOnboardBuildDeployRequest Auto-Onboarding Flow
143 " 🔵 Deploy Execution Status Lifecycle and Gate Checks
144 12:40a 🔵 Session resumed with no observable work
146 " 🔵 AutoOps deployment investigation underway for java-demo
149 " 🔵 AutoOps deployment pre-validation: git branches and connectivity confirmed
150 " 🔴 Split Harbor push host from image reference host in Jenkinsfile
147 12:41a 🔵 Jenkins-to-Harbor internal connectivity confirmed
148 " 🔵 AutoOps deploy integration Agent API endpoints mapped and documented
151 12:42a ✅ Ralph mode iteration 5: Jenkinsfile fix committed and pushed
152 " 🔵 AutoOps pipeline state reset via SQL for DR20260429150308
153 " ✅ AutoOps database reset complete for java-demo deployment
154 " 🔵 Poll loop started to monitor AutoOps pipeline re-execution
155 " 🔵 AutoOps pipeline re-triggered and in "building" stage
156 " 🔵 Jenkins build not yet started after 90+ seconds in queue
157 " 🔵 Jenkins agent pod spun up for java-demo build
158 12:46a 🔵 Jenkins build #28 is running — prior observation was wrong
159 " 🔵 Jenkins build #28 actively resolving Maven dependencies
160 " 🔵 AutoOps DB build_number stuck at 0 despite active Jenkins build #28
161 " 🔵 Hermes opsclaw session and deployment skill documentation inspected
162 12:48a 🔵 Hermes session transcript reveals java-demo deployment flow and behavioral discrepancy
163 12:49a 🔵 Jenkins build #28 completed with BUILD SUCCESS
164 12:51a 🔵 Maven build succeeded in 5 min 13 sec with zero test failures
165 " 🔵 AutoOps DB not updating after Jenkins build completion for 2+ minutes
166 12:53a 🟣 Jenkinsfile fix verified: Jib pushing to Harbor via internal cluster DNS
167 " 🔴 Jenkinsfile fix confirmed: Jib pushing to Harbor via internal cluster DNS

Access 548k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>
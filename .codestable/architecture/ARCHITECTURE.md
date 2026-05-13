# AutoOps 架构总入口

> 状态：初始地图（Deploy 集成现状已回填）
> 创建日期：2026-05-12

## 1. 项目简介

AutoOps 是 Go + Vue 3 构建的统一运维管理平台，用于管理 CMDB 资产、Kubernetes 集群、任务自动化、监控告警（N9E / FlashDuty）和配置中心。

## 2. 核心概念 / 术语表

待 `cs-arch backfill` 从现有文档和代码补齐。

## 3. 子系统 / 模块索引

现有架构与开发入口文档暂保留在原位：

- `docs/architecture.md` — 当前系统架构、请求链路、模块边界、启动序列
- `docs/backend-guide.md` — 后端开发约束与模块布局
- `docs/frontend-guide.md` — 前端页面、API、路由与状态管理约定
- `docs/deployment.md` — Docker 环境、配置与 seed 数据
- `docs/security-audit.md` — 安全审计跟踪
- `docs/deploy-control-plane.md` — Deploy 控制平面设计
- [`deploy-dingtalk-autoops-e2e.md`](deploy-dingtalk-autoops-e2e.md) — 钉钉群触发 AutoOps 构建部署的当前生产验证路径

## 4. 关键架构决定

- Hermes 只负责把钉钉群需求转成 AutoOps Agent 请求；审批、构建、部署、状态和通知由 AutoOps 统一执行和记录。
- Direct NodePort 的当前访问地址来自 Kubernetes 节点 IP + NodePort；`nodeport_access_host` 尚未被 Go 配置消费。
- 写权限部署凭证使用独立 `deployerAccess`，不扩展只读 `clusterAccess`。

## 5. 已知约束 / 硬边界

- PostgreSQL 17 / Valkey 9 是当前基础设施约束。
- 新 GORM 模型必须注册到 `api/pkg/db/migrate.go`。
- 前端 API URL 不手写 `/api/v1` 前缀。
- 生产环境必须设置 `JWT_SECRET`。
- 钉钉部署 E2E 证据不得记录真实 token、密码、PAT、webhook、kubeconfig 或 Secret value。

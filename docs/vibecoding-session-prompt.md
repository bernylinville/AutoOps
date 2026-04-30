# AutoOps Deploy 控制平面 — 新环境接手 Prompt

> 用途：在新环境、新会话或新的编码代理中快速恢复上下文，用于继续推进 AutoOps Deploy 控制平面。

## 使用前先阅读

- `docs/vibecoding-handoff.md`
- `docs/deploy-control-plane.md`
- `docs/dingtalk-oa-template.md`
- `docs/dingtalk-userid-bootstrap.md`
- `progress.md`

## 推荐 Prompt（可直接复制）

```text
当前项目是 /home/kchou/Code/AutoOps，参考 GitOps 仓库是 /home/kchou/Code/pukka-gitops。

先不要从零设计，也不要重做已经完成的功能。先阅读以下文档并基于真实代码继续执行：
- docs/vibecoding-handoff.md
- docs/deploy-control-plane.md
- docs/dingtalk-oa-template.md
- docs/dingtalk-userid-bootstrap.md
- progress.md

当前真实状态：
- AutoOps deploy 控制平面主体已经落地
- direct mode / gitops mode 主链已实现
- 钉钉 OA client、审批同步、审批通过后自动执行已实现
- 结果通知链路已实现到数据库记录、API 查询、前端弹窗
- Hermes skill skeleton 已落地
- GitOps delete / rollback API 与 reconcile 命令已落地

当前主要未完成项：
1. 用真实钉钉 userId 打通 OA 审批实例
2. 用真实 deploy_bot webhook 验证结果回群
3. 在真实 Hermes 会话里验证确认流和下单链路
4. 如有需要，再做 GitOps destructive rollback 的真实集群验收

硬性约束：
1. 同一资源只能有一个 owner，不能 GitOps/direct 双主控制
2. 所有 direct mode 资源都必须带 managed-by / owner-system / deploy-mode / request-id / ttl 标签与注解
3. 不绕过审批
4. 不让 Hermes 直接操作 Git 或 Kubernetes
5. 优先复用现有 deploy、RBAC、审计、scheduler、configcenter 模块
6. 遇到 GitOps 边界、权限边界、审批边界不明确时暂停并说明问题

执行方式：
- 一次只做一个小任务
- 动手前先读对应文件
- 每完成一个任务先自检，再继续下一步
- 不要擅自扩大范围

建议优先顺序：
1. 检查 sys_admin.dingtalk_user_id 是否已填真实值
2. 验证钉钉 OA 审批实例能否真实发起
3. 验证审批通过后自动执行
4. 配置 deploy_bot 并验证通知回群
5. 最后验证 Hermes skill 的真实会话链路

本地验证命令：
- cd api && mise exec go@1.25.0 -- go test ./...
- cd web && mise exec node@24.14.1 -- npm run lint
- AGENT_TOKEN=change-me-agent-token python3 scripts/deploy/hermes-contract-test.py
- cd api && mise exec go@1.25.0 -- go run . -c /tmp/autoops-reconcile-config.yaml --reconcile-gitops

注意：
- 2026-04-17 本机运行中的 devops-api 已通过静态编译二进制热替换为最新代码
- docker build 可能因拉取 docker.io/library/alpine:3.23 超时失败
- 如果需要重建容器，优先确认网络或使用同样的静态二进制替换方案
```

## 额外说明

- 如果新环境没有本地数据库或钉钉配置，优先恢复环境，不要直接跑破坏性脚本。
- 如果要验证真实 OA 或真实群聊消息，先确认外部配置已经齐备，再执行脚本。

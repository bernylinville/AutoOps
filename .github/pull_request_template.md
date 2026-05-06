## Description
<!-- 描述本次变更。如包含 UI 改动，请添加截图/录屏并说明验证路径。 -->

## Linked Issue / Spec
<!-- 提交前确认：小修/单文件 bugfix 可不写 spec；跨模块、DB、RBAC、安全、部署流程变更应关联 docs/specs/<id>/。 -->
- [ ] 已关联 Issue / 工单，或说明无需关联的原因
- [ ] 如为较大或高风险变更，已添加/更新 `docs/specs/<id>/PRODUCT.md` 和 `docs/specs/<id>/TECH.md`
- [ ] 如使用 readiness label，Issue 已标记 `ready-to-spec` 或 `ready-to-implement`

## Review Checklist（人类审查）
- [ ] **DB Migration**：如有新增/修改 GORM 模型，已注册到 `api/pkg/db/migrate.go`，或在 migration allowlist 中写明非迁移原因
- [ ] **RBAC**：如有新增 API 端点，已添加权限码并注册到路由
- [ ] **API URL**：前端调用未硬编码 `/api/v1` 前缀（request.js 拦截器自动添加）
- [ ] **安全**：如涉及 auth/data/secret/logging 变更，已检查潜在安全问题
- [ ] **文档**：如用户可见行为变更，已更新相关文档或说明无需更新

## Agent / AI Usage
- [ ] AutoOps Agent Mode — 本次 PR 由 AI Agent 生成或辅助
- [ ] 我已人工复核 AI 生成代码的范围、凭据泄露风险、迁移注册、RBAC 和测试覆盖

## Verification
<!-- 如何测试这次变更？添加了哪些自动化测试？如果没有添加新测试，请说明理由。 -->
- [ ] Backend: `cd api && go test ./... -v -count=1`
- [ ] Frontend: `cd web && npm run lint && npm run build`
- [ ] Migration checker（如 model/schema 变更）: `./scripts/check-migrations`
- [ ] UI 变更已附截图/录屏，或说明无需视觉证据
- [ ] Bug fix 已添加回归测试，或说明无法自动化覆盖的原因

## Changelog Entries
<!--
使用以下前缀（不带 `{{}}`）。如不需要 changelog，删除对应行：

* NEW-FEATURE: 相对大的新功能
* IMPROVEMENT: 现有功能改进
* BUG-FIX: 已知 bug 或回归修复
* DOCS: 文档变更
-->

CHANGELOG-NEW-FEATURE: {{text goes here...}}
CHANGELOG-IMPROVEMENT: {{text goes here...}}
CHANGELOG-BUG-FIX: {{text goes here...}}
CHANGELOG-DOCS: {{text goes here...}}

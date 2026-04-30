## Description
<!-- 描述本次变更。如包含 UI 改动，请添加设计同事 review。 -->

## Review Checklist (人类审查)
<!-- 提交前请确认以下事项 -->
- [ ] **DB Migration**：如有新增/修改 GORM 模型，已注册到 `api/pkg/db/migrate.go`
- [ ] **RBAC**：如有新增 API 端点，已添加权限码并注册到路由
- [ ] **API URL**：前端调用未硬编码 `/api/v1` 前缀（request.js 拦截器自动添加）
- [ ] **本地测试**：`go test ./...` 和 `npm run lint` 已通过
- [ ] **安全**：如涉及 auth/data 变更，已检查潜在安全问题
- [ ] **文档**：如用户可见行为变更，已更新相关文档

## Testing
<!--
如何测试这次变更？添加了哪些自动化测试？如果没有添加新测试，请说明理由。
-->

## Agent Mode
- [ ] AutoOps Agent Mode — 本次 PR 由 AI Agent 生成

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

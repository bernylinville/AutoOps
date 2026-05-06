# Security Audit & Remediation

> 涉及安全相关代码修改前必读。审计日期：2026-03-23。

## 高危问题（H1-H9）

| ID | 问题 | 位置 | 修复状态 |
|----|------|------|---------|
| H1 | 仅 JWT 验证无 RBAC 鉴权（部分路由） | router/router.go | ⬜ 待确认 |
| H2 | JWT Secret / AES Key 硬编码 + 日志泄露 | pkg/jwt/jwt.go, common/util | ⬜ 待确认 |
| H3 | 密码方案：无盐 MD5 + 前端明文传输 | system/service, login 流程 | ⬜ 待确认 |
| H4 | SSH 凭据明文存储 | configcenter/model | ⬜ 待确认 |
| H5 | WebSocket/SSE 认证绕过（token 在 URL query） | middleware/authMiddleware.go | ✅ 已接受（WebSocket/SSE 行业标准做法，JWT 24h 过期，生产 HTTPS） |
| H6 | XSS 注入（v-html + dangerouslyUseHTMLString） | 前端多处 | ✅ 已缓解（全部 v-html 经 highlight.js 转义输出） |
| H7 | K8s Secret data 明文返回 | k8s/controller | ✅ 已修复（convertToSecretModel Data=nil + Spec DeepCopy 遮蔽） |
| H8 | 远程命令注入（SSH installDir 拼接） | tool/service | ✅ 已缓解（shell 元字符 + 路径遍历校验 + 绝对路径强制） |
| H9 | SQL 控制台无 RBAC 保护 | cmdb/controller/sql | ✅ 已修复（全部 SQL 路由已注册 RbacMiddleware） |

## 中危问题（M1-M9）

| ID | 问题 | 修复状态 |
|----|------|---------|
| M1 | Controller 包级变量竞态 | ⬜ 待确认 |
| M2 | SQL bind 双绑定 + TODO stub 返回 | ⬜ 待确认 |
| M3 | SQL 端点 ID/Name 不一致 | ⬜ 待确认 |
| M4 | HTTP 状态码使用不一致 | ⬜ 待确认 |
| M5 | SSH Terminal goroutine 泄露 | ⬜ 待确认 |
| M6 | element-plus Message import 错误 | ⬜ 待确认 |
| M7 | AutoMigrate 缺少 N9E 模型 | ✅ 已修复（Phase 6） |
| M8 | Redis 健康检查密码不匹配 | ✅ 已修复（Valkey 迁移时） |
| M9 | GetDB() Ping panic | ⬜ 待确认 |

## 修复优先级

| 优先级 | 问题 | 建议时间 |
|--------|------|---------|
| **P0 紧急** | H2 硬编码密钥, H3 MD5 密码（用户暂缓） | 按需 |
| **P1 高优** | H5 WebSocket 认证（已接受）, H6 XSS（已缓解）, H9 SQL 控制台（已修复） | 已完成 |
| **P2 中优** | H1 RBAC 覆盖, H4 凭据存储, H7 Secret（已修复）, M1-M5 | 1-2 周 |
| **P3 低优** | H8 命令注入（已缓解），M6-M9 | 按需 |

## 来源

完整审计报告：[docs/full-repo-review.md](../docs/full-repo-review.md)

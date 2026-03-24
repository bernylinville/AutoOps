# AutoOps 全仓代码审查报告

> Codex Review — 2026-03-23

---

## 一、高危 (9 项)

### H1 — 后端无 RBAC 授权 🔴
所有接口只校验 JWT 是否有效，不检查权限码。任意登录用户可调用凭据解密、SSH、K8s 管理等高危接口。
- `router/router.go:102` — jwtGroup 下无权限中间件
- `router/configCenter/configCenter.go:31`
- `router/cmdb/cmdb.go:36`
- 前端权限只是按钮隐藏：`permission/Authority.js:17`

**修复方案**：新增 RBAC 中间件，从 `sys_role_menu.value` 读权限码，在路由组级别拦截。

---

### H2 — 密钥/凭据硬编码 + 日志泄露 🔴
| 问题 | 位置 |
|------|------|
| JWT Secret 硬编码 | `pkg/jwt/jwt.go:23` |
| AES Key 硬编码 + 打印到 stderr | `common/util/encryption.go:18,25` |
| 鉴权失败日志泄露 JWT Secret | `middleware/authMiddleware.go:93` |
| 明文配置提交到仓库 | `config.yaml:18,53`, `docker/.env:34` |
| Agent 心跳密钥硬编码 | `common/agent/agent.go:318` |

**修复方案**：密钥迁移到环境变量/Vault；config.yaml 加入 .gitignore；日志脱敏。

---

### H3 — 密码方案：无盐 MD5 + 前端泄露 🔴
- `encryption.go:116` — 无盐 MD5
- 登录成功返回完整 SysAdmin 对象含 password 字段
- 前端写入 localStorage：`Login.vue:92`

**修复方案**：改为 bcrypt；DTO 层排除 password 字段；清理 localStorage 存储内容。

---

### H4 — SSH 凭据明文存储 + 无权限返回 🔴
- `ecsAuth.go:15,16` — SSH 密码/私钥明文存储
- 列表/详情接口直接返回凭据：`ecsAuth.go:37,57,143,165`
- 结合 H1（无 RBAC），任意用户可获取所有主机登录凭据

**修复方案**：数据库存储加密；返回时脱敏；结合 H1 做权限拦截。

---

### H5 — WebSocket/SSE 鉴权旁路 + Token URL 泄露 🔴
- SSE 日志流 token 为空仍返回日志：`taskansible.go:339,345`
- WebSocket token 为空直接放行：`websocket.go:35,62`
- SSH WebSocket 跨域全放行：`cmdbHostSSH.go:83`
- Token 在 URL query string 中，会进浏览器历史/代理日志

**修复方案**：WebSocket/SSE 入口强制校验 token；token 改用 header 或一次性 ticket。

---

### H6 — XSS 注入 (v-html + dangerouslyUseHTMLString) 🔴
- 账号解密：`accountauth.vue:350,358` — dangerouslyUseHTMLString 拼接
- 任务日志：`LogDialog.vue:9,42` — v-html 渲染未转义日志
- Ansible 日志：`AnsibleLogDialog.vue:80,534` — v-html

**修复方案**：日志内容 HTML 转义后渲染；解密结果用文本节点而非 HTML。

---

### H7 — K8s Secret 完整泄露 🔴
- `k8sconfig.go:296,309` — convertToSecretModel 返回 secret.Data 原始数据
- `k8sconfig.go:496,507` — GetSecretDetail 挂完整 secret 对象
- 注释写"隐藏敏感数据"但代码未实现

**修复方案**：只返回 key 名列表，不返回 value。

---

### H8 — 远程命令注入 (SSH installDir) 🔴
- `serviceDeploy.go:423,466,478` — installDir 直接拼入 shell 命令
- `sshUtil.go:314,329` — session.Run() 执行拼接字符串
- 用户可控输入 → 远程代码执行

**修复方案**：installDir 白名单校验 + shellescape 转义。

---

### H9 — SQL 控制台暴露 🔴
- `/cmdb/sql/*` 接口开放数据库操作权
- 只传 databaseName 时默认 AccountID=1 建连
- 结合 H1（无 RBAC）= 任意用户操作数据库

**修复方案**：严格 RBAC + 审计日志；限制可执行 SQL 类型。

---

## 二、中危 (9 项)

### M1 — Controller 包级全局变量竞态
请求体绑定到全局 var，并发请求互相覆盖。
- `sysMenu.go:14,26`, `sysPost.go:11,22`, `sysDept.go:11`, `cmdbGroup.go:12`
- **修复**：改为函数内局部变量

### M2 — SQL 执行器双重 Bind + TODO 伪成功
- `cmdbSQLRecord.go:160` — 外层 Bind 后内层再 Bind → EOF
- Insert/Delete 是 TODO 但返回成功
- **修复**：一次 Bind + 传参；TODO 改为 501

### M3 — SQL 接口 databaseId/Name 不一致
只传 ID 时仍使用空 Name 做 DSN 拼接。
- **修复**：按 ID 反查 Name

### M4 — HTTP 语义不一致
后端所有错误返回 HTTP 200，前端只处理 401/406。NOAUTH=403 不触发登出。
- **修复**：统一使用 HTTP 状态码或前端增加 403 处理

### M5 — SSH Terminal goroutine 泄漏
`select {}` 永久阻塞，defer 不执行，连接资源不回收。
- **修复**：使用 context 或 channel 监听连接关闭

### M6 — element-plus Message 导入错误
`request.js:7` 导入不存在的 Message，运行时错误提示失效。
- **修复**：改为 `ElMessage`

### M7 — AutoMigrate 缺 N9E 模型
空库启动时 AutoMigrate 不含 N9E 表 → 缺表。
- **修复**：在 migrate.go 添加 N9E 模型

### M8 — Redis 健康检查密码不匹配
`docker-compose.yml` healthcheck 写死 `zhangfan@123`，实际用 `${REDIS_PASSWORD}`。
- **修复**：healthcheck 改用 `${REDIS_PASSWORD}`

### M9 — GetDB() Ping 失败 panic
每次获取连接都 Ping，抖动直接 panic 崩进程。
- **修复**：移除 Ping 或改为 log+return error

---

## 三、低危 (3 项)

### L1 — go test 编译失败
- `agent.go:549` — `%%v` 格式串
- `jwt.go:92,98` — `string(0)` 写法

### L2 — 角色分配异步写库吞错误
先返回 200 再 go 写库，失败被吞。

### L3 — 登录菜单 N+1 查询
先查一级菜单，再逐个查子菜单。

---

## 四、修复优先级建议

| 优先级 | 项目 | 预估工时 | 说明 |
|--------|------|----------|------|
| P0 紧急 | H2 密钥硬编码 | 2h | 只需迁移配置，不改业务逻辑 |
| P0 紧急 | H3 密码方案 | 3h | 改 bcrypt + DTO 脱敏 |
| P0 紧急 | H5 WS/SSE 鉴权 | 2h | 加 token 校验 |
| P0 紧急 | H6 XSS | 1h | v-html → 转义 |
| P1 高 | H1 RBAC 中间件 | 8h | 核心架构变更 |
| P1 高 | H4 凭据加密 | 4h | 需要数据迁移 |
| P1 高 | H8 命令注入 | 2h | shellescape |
| P1 高 | M1 全局变量 | 1h | 简单机械修改 |
| P2 中 | H7 K8s Secret | 1h | 返回值过滤 |
| P2 中 | H9 SQL 控制台 | 依赖 H1 | RBAC 先行 |
| P2 中 | M2-M9 | 各 0.5-2h | 逐步处理 |
| P3 低 | L1-L3 | 各 0.5h | 不影响运行 |

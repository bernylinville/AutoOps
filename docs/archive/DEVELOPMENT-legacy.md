# AutoOps 开发上下文 & 后续规划

> **目的**: 本文档包含项目全量已实现功能的技术细节 + 7 项后续开发计划，作为任何 AI 编程工具的上下文输入，确保切换工具后能无缝继续开发。
>
> **最后更新**: 2026-03-24 · **Git HEAD**: `89fa842` · **分支**: `main`

---

## 一、项目概况

| 指标 | 数值 |
|------|------|
| 后端 Go 文件 | 154 |
| 前端 Vue 页面 | 112 |
| API 路由注册 | 404 行 |
| RBAC 权限码 | 195 条 (menu_type=3) |
| 数据库表 | 40+ |
| Git 提交 (本轮) | 7 commits, +7473/-664 |

### 技术栈

```
后端: Go 1.25 · Gin · GORM · JWT · robfig/cron · golang.org/x/crypto (bcrypt)
前端: Vue 3.5 · Element Plus 2.10 · ECharts · xterm.js · axios
数据库: MySQL 8.0 · Redis 7.2
监控: Prometheus · Pushgateway · N9E 夜莺
部署: Docker Compose · Alpine 3.23 · Nginx 1.28
```

### 项目结构

```
AutoOps/
├── api/                          # Go 后端 (端口 8000)
│   ├── api/                      # 业务模块
│   │   ├── cmdb/                 # CMDB (controller/dao/model)
│   │   │   └── controller/
│   │   │       ├── cmdbHost.go           # 主机 CRUD + 分页 + sourceType 筛选
│   │   │       ├── cmdbHostSSH.go        # SSH 终端 + SCP 文件传输
│   │   │       ├── cmdbGroup.go          # 主机分组
│   │   │       ├── cmdbDatabase.go       # 5 类数据库管理
│   │   │       ├── cmdbSQLRecord.go      # SQL 在线执行 (SELECT/UPDATE 白名单)
│   │   │       └── cmdbSwitch.go         # 网络设备 SNMP
│   │   ├── k8s/                  # Kubernetes 管理
│   │   │   └── controller/
│   │   │       ├── k8sCluster.go         # 多集群 CRUD + kubeconfig
│   │   │       ├── k8sNode.go            # 节点标签/污点/封锁/驱逐
│   │   │       ├── k8sDeployment.go      # 伸缩/重启/回滚/YAML
│   │   │       ├── k8sPod.go             # Pod 日志/删除
│   │   │       ├── k8sterminal.go        # Pod WebSocket 终端
│   │   │       ├── k8sService.go         # Service CRUD
│   │   │       ├── k8sIngress.go         # Ingress CRUD
│   │   │       ├── k8sNamespace.go       # 命名空间 + ResourceQuota
│   │   │       ├── k8sSecret.go          # Secret 加密存储
│   │   │       ├── k8sConfigMap.go       # ConfigMap 管理
│   │   │       └── k8sStorage.go         # PV/PVC/StorageClass
│   │   ├── n9e/                  # N9E 夜莺监控 + 告警通知
│   │   │   ├── model/
│   │   │   │   ├── n9e.go                # N9EConfig/BusiGroup/DataSource/SyncLog
│   │   │   │   └── alertModel.go         # AlertRule/AlertEvent/NotifyChannel
│   │   │   ├── dao/
│   │   │   │   ├── n9e.go                # N9E 配置/业务组/数据源/同步日志 DAO
│   │   │   │   └── alertDao.go           # 告警规则/事件/渠道 CRUD + 统计
│   │   │   ├── service/
│   │   │   │   ├── client.go             # N9E HTTP 客户端 (业务组/主机/数据源 API)
│   │   │   │   ├── sync.go               # FullSync 全量同步 (业务组→主机→数据源)
│   │   │   │   └── notifier.go           # 通知分发器 (企业微信/钉钉/邮件)
│   │   │   └── controller/
│   │   │       ├── n9e.go                # 配置/同步/总览/业务组/数据源/PromQL API
│   │   │       └── alertController.go    # 规则/事件/渠道 CRUD + Webhook 接收
│   │   ├── configcenter/         # 配置中心
│   │   │   └── controller/
│   │   │       ├── ecsAuth.go            # 主机凭据 (SSH 密钥/密码)
│   │   │       ├── accountAuth.go        # 通用账号 AES 加密
│   │   │       ├── keyManage.go          # 云密钥管理 (AccessKey)
│   │   │       └── syncSchedule.go       # 云同步定时配置
│   │   ├── task/                 # 任务中心
│   │   │   └── controller/
│   │   │       ├── tasktemplage.go       # Shell/Python 脚本模板
│   │   │       ├── taskwork.go           # 任务作业执行
│   │   │       └── taskansible.go        # Ansible Playbook 任务
│   │   ├── monitor/              # 监控 Agent
│   │   │   └── controller/
│   │   │       ├── agent.go              # Agent 部署/卸载/心跳
│   │   │       └── domainMonitor.go      # 域名 SSL 监控
│   │   ├── tool/                 # 运维工具
│   │   │   └── controller/
│   │   │       ├── tool.go               # 工具市场 (MySQL/Redis/Jenkins...)
│   │   │       └── serviceDeploy.go      # 服务部署管理
│   │   ├── app/                  # 服务管理
│   │   │   └── controller/
│   │   │       ├── application.go        # 应用 CRUD + 多环境
│   │   │       └── quickDeploy.go        # Jenkins 快速发布
│   │   └── system/               # 系统管理
│   │       └── controller/
│   │           ├── sysAdmin.go           # 用户 CRUD (bcrypt)
│   │           ├── sysRole.go            # 角色管理 + 权限分配
│   │           ├── sysMenu.go            # 三级菜单 (页面/路由/按钮)
│   │           └── sysDept.go            # 部门管理
│   ├── middleware/
│   │   ├── authMiddleware.go             # JWT 认证
│   │   ├── rbacMiddleware.go             # RBAC 权限校验
│   │   ├── rbacCache.go                  # sync.Map 权限缓存 (5min TTL)
│   │   └── logMiddleware.go              # 操作日志自动记录
│   ├── router/
│   │   ├── router.go                     # 主路由 (公开 + JWT + RBAC 三层)
│   │   ├── cmdb/cmdb.go                  # CMDB 路由
│   │   ├── k8s/k8s.go                    # K8s 路由 (40+ RBAC 保护)
│   │   ├── n9e/n9e.go                    # N9E + 告警路由 + Webhook
│   │   ├── configCenter/configCenter.go  # 配置中心路由 (30+ RBAC)
│   │   ├── system/system.go              # 系统管理 + 审计日志路由
│   │   ├── tool/tool.go                  # 工具路由
│   │   ├── monitor/monitor.go            # Agent 路由
│   │   ├── task/task.go                  # 任务路由
│   │   └── app/app.go                    # 服务管理路由
│   ├── scheduler/
│   │   ├── manager.go                    # 调度器管理器 (单例)
│   │   ├── syncScheduler.go              # 云厂商定时同步
│   │   └── n9eSyncScheduler.go           # N9E Cron 定时同步
│   ├── pkg/
│   │   ├── jwt/jwt.go                    # JWT 生成/解析
│   │   └── db/
│   │       ├── db.go                     # GORM 初始化
│   │       └── migrate.go               # AutoMigrate 模型注册
│   └── config.yaml                       # 应用配置
├── web/src/                      # Vue 3 前端
│   ├── views/
│   │   ├── cmdb/                         # 主机/数据库/网络设备/SQL
│   │   ├── K8s/                          # 集群/节点/Pod/Service/Ingress
│   │   ├── monitor/                      # 告警规则/事件/N9E总览/数据源/同步日志
│   │   ├── config/                       # 凭据/密钥管理
│   │   ├── task/                         # 任务模板/作业/Ansible
│   │   ├── system/                       # 用户/角色/菜单/部门/N9E配置
│   │   ├── tool/                         # 工具市场/Agent
│   │   ├── app/                          # 应用/快速发布
│   │   └── dashboard/                    # 仪表盘统计
│   ├── api/                              # API 请求封装
│   │   ├── cmdb.js                       # CMDB API
│   │   ├── n9e.js                        # N9E + 告警 API
│   │   └── ...                           # 其他模块 API
│   ├── router/
│   │   ├── system.js                     # 系统/监控/告警路由
│   │   └── ...                           # 其他模块路由
│   └── utils/request.js                  # axios 封装 + JWT 拦截器
└── docker/
    ├── docker-compose.yml                # 6 服务编排
    ├── api/Dockerfile                    # Go 多阶段构建 + 非 root
    └── web/Dockerfile                    # Node 构建 + Nginx
```

---

## 二、数据库核心表

### CMDB
| 表名 | 说明 |
|------|------|
| `cmdb_host` | 主机 (ip/hostname/sourceType/groupId/status) |
| `cmdb_group` | 主机分组 |
| `cmdb_sql` | 数据库连接配置 |
| `cmdb_sql_record` | SQL 执行记录 |

### K8s
| 表名 | 说明 |
|------|------|
| `kube_cluster` | K8s 集群 (kubeconfig/apiServer) |

### N9E 监控
| 表名 | 说明 |
|------|------|
| `n9e_config` | N9E 连接配置 (endpoint/token/syncCron/enabled) |
| `n9e_busi_group` | 业务组 (同步自 N9E) |
| `n9e_datasource` | 数据源 (Prometheus URL) |
| `n9e_sync_log` | 同步日志 (status/duration/triggerBy) |
| `n9e_alert_rule` | 告警规则 (name/severity/matchLabels/notifyChannels) |
| `n9e_alert_event` | 告警事件 (alertName/status/labels/notifyStatus) |
| `n9e_notify_channel` | 通知渠道 (type:wechat/dingtalk/email + config JSON) |

### 配置中心
| 表名 | 说明 |
|------|------|
| `ecs_auth` | 主机凭据 (SSH 密钥/密码) |
| `account_auth` | 通用账号 (AES 加密) |
| `key_manage` | 云密钥 (AccessKey/Secret) |
| `sync_schedule` | 云同步定时配置 |

### 系统
| 表名 | 说明 |
|------|------|
| `sys_admin` | 用户 (bcrypt 密码) |
| `sys_role` | 角色 (id=1 超级管理员) |
| `sys_menu` | 三级菜单 (menu_type: 1=目录, 2=页面, 3=权限按钮) |
| `sys_role_menu` | 角色-权限关联 |
| `sys_admin_role` | 用户-角色关联 |
| `sys_operation_log` | 操作日志 |
| `sys_login_info` | 登录日志 |

---

## 三、安全机制

| 机制 | 实现位置 | 说明 |
|------|---------|------|
| JWT 认证 | `middleware/authMiddleware.go` | 所有 API 默认需 JWT |
| RBAC 权限 | `middleware/rbacMiddleware.go` | 从 `sys_menu.value` 匹配权限码 |
| RBAC 缓存 | `middleware/rbacCache.go` | `sync.Map` + 5 分钟 TTL |
| 操作审计 | `middleware/logMiddleware.go` | 非 GET 请求自动记录 |
| 密码加密 | `sysAdmin.go` | bcrypt hash |
| 凭据加密 | `encryption.go` | AES-256 + 环境变量密钥 |
| SQL 白名单 | `cmdbSQLRecord.go:ExecuteUpdate` | 仅允许 UPDATE |
| Webhook 鉴权 | `alertController.go` | `X-Webhook-Token` header |
| 容器安全 | `docker/api/Dockerfile` | 非 root 用户 `devops` |
| 端口安全 | `docker-compose.yml` | MySQL/Redis/Prometheus 绑定 `127.0.0.1` |
| 健康检查 | `router.go` | `/healthz` + `/readyz` 公开端点 |

### RBAC 权限码命名规范

```
{模块}:{子模块}:{操作}

示例:
cmdb:ecs:add          # CMDB 创建主机
k8s:cluster:delete    # K8s 删除集群
config:key:decrypt    # 配置中心 解密密钥
monitor:alert:edit    # 监控 编辑告警规则
tool:deploy:create    # 工具 创建部署
base:log:clean        # 系统 清空日志
```

---

## 四、关键 API 端点

### 公开 (无需认证)
```
POST /api/v1/captcha                    # 验证码
POST /api/v1/login                      # 登录
GET  /api/v1/healthz                    # 健康检查
GET  /api/v1/readyz                     # 就绪检查
POST /api/v1/monitor/agent/heartbeat    # Agent 心跳
POST /api/v1/n9e/alert/webhook          # 告警 Webhook (token 校验)
```

### N9E 监控 (JWT)
```
GET/POST  /api/v1/n9e/config            # N9E 连接配置
POST      /api/v1/n9e/sync              # 触发同步
GET       /api/v1/n9e/overview           # CMDB 总览统计
GET       /api/v1/n9e/busi-groups        # 业务组列表
GET       /api/v1/n9e/datasources        # 数据源列表
POST      /api/v1/n9e/datasources/:id/check  # 数据源健康检查
GET       /api/v1/n9e/query              # PromQL 查询
```

### 告警通知 (JWT + RBAC)
```
GET/POST/PUT/DELETE  /api/v1/n9e/alert/rules       # 告警规则 CRUD
GET                  /api/v1/n9e/alert/events       # 告警事件列表
GET                  /api/v1/n9e/alert/events/stats  # 事件统计
GET/POST/PUT/DELETE  /api/v1/n9e/alert/channels     # 通知渠道 CRUD
POST                 /api/v1/n9e/alert/channels/:id/test  # 测试通知
```

---

## 五、后续规划 — 7 项开发方案

### P1: N9E 实际环境联调 (预计 2 天)

**目标**: 对接真实夜莺 N9E 环境，验证 PromQL 查询和告警推送

**实现要点**:
- 验证 `n9e/service/client.go` 对真实 N9E API 的兼容性
- 实现 PromQL 查询代理，前端展示时序图表 (ECharts line)
- 配置 N9E Alertmanager → AutoOps Webhook 推送链路
- 前端 `N9eMonitor.vue` 接入真实数据展示

**涉及文件**:
- `api/api/n9e/service/client.go` — 可能需适配 N9E v6/v7 API 差异
- `api/api/n9e/controller/n9e.go` — QueryPromQL() 增加时间序列返回格式
- `web/src/views/monitor/N9eMonitor.vue` — ECharts 时序图
- `api/config.yaml` → 增加 N9E 连接参数

---

### P2: Windows 主机管理 + 远程桌面 (预计 5 天)

**目标**: 支持 Windows 主机管理，通过 Guacamole 实现 Web RDP

**实现要点**:
- `cmdb_host` 新增 `os_type` 字段 (linux/windows)
- 后端新增 WinRM 连接器 (Go WinRM 库)
- 集成 Apache Guacamole (Docker 容器)，实现 Web RDP
- 前端主机列表增加 OS 类型图标 + RDP 终端按钮

**新建文件**:
- `api/api/cmdb/controller/cmdbHostWinRM.go`
- `web/src/views/cmdb/cmdbRDP.vue`
- `docker/guacamole/` — Guacamole Docker 配置

**数据库变更**:
```sql
ALTER TABLE cmdb_host ADD COLUMN os_type VARCHAR(20) DEFAULT 'linux';
ALTER TABLE cmdb_host ADD COLUMN rdp_port INT DEFAULT 3389;
```

---

### P3: K8s HPA 自动扩缩容 (预计 3 天)

**目标**: K8s HPA 配置管理 + 扩缩容事件可视化

**实现要点**:
- 后端调用 K8s `autoscaling/v2` API 实现 HPA CRUD
- 前端 HPA 配置表单 (目标 CPU/内存/自定义指标)
- 扩缩容事件时间线展示 (ECharts)
- 与现有 Deployment 管理联动 (HPA 列关联)

**新建文件**:
- `api/api/k8s/controller/k8sHPA.go`
- `web/src/views/K8s/workloads/K8S-hpa.vue`
- 路由: `k8s.go` 增加 `/k8s/hpa/*` 路由

---

### P4: SQL 工单系统 (预计 5 天)

**目标**: DBA 审批流程的 SQL 工单，类似 Archery

**实现要点**:
- 工单模型: `sql_ticket` (applicant → reviewer → executor 三态)
- 审批流: 提交 → DBA 审批 → 自动执行 → 结果回填
- SQL 语法检查 (基础 parser)
- 执行结果 + 影响行数记录
- 前端工单列表 + 详情 + 审批操作

**新建文件**:
- `api/api/cmdb/model/sqlTicket.go`
- `api/api/cmdb/dao/sqlTicketDao.go`
- `api/api/cmdb/controller/sqlTicket.go`
- `web/src/views/cmdb/SQLTicket.vue`
- `web/src/views/cmdb/SQLTicketDetail.vue`

**数据库**:
```sql
CREATE TABLE sql_ticket (
  id INT PRIMARY KEY AUTO_INCREMENT,
  title VARCHAR(200),
  db_id INT,               -- 关联 cmdb_sql
  sql_content TEXT,
  sql_type VARCHAR(20),    -- DDL/DML/DQL
  applicant_id INT,
  reviewer_id INT,
  status VARCHAR(20),      -- pending/approved/rejected/executed/failed
  affect_rows INT,
  execute_result TEXT,
  create_time DATETIME,
  review_time DATETIME,
  execute_time DATETIME
);
```

---

### P5: 运维工单系统 (预计 5 天)

**目标**: 通用运维变更工单 (非 SQL)

**实现要点**:
- 工单类型: 发布上线/配置变更/权限申请/故障处理
- 多级审批流 (申请 → 审批 → 执行 → 验证 → 关闭)
- 与任务中心联动 (工单触发 Ansible 任务)
- 工单看板 (Kanban 视图)

**新建文件**:
- `api/api/ticket/` — 新模块 (model/dao/controller)
- `web/src/views/ticket/TicketList.vue`
- `web/src/views/ticket/TicketDetail.vue`
- `web/src/views/ticket/TicketKanban.vue`

---

### P6: 运维知识库 (预计 3 天)

**目标**: Markdown 运维知识文档管理

**实现要点**:
- 文档 CRUD + 目录树结构
- Markdown 编辑器 (md-editor-v3)
- 全文搜索 (MySQL FULLTEXT 或 ES)
- 文档版本历史
- 标签分类

**新建文件**:
- `api/api/knowledge/` — 新模块
- `web/src/views/knowledge/KnowledgeList.vue`
- `web/src/views/knowledge/KnowledgeEditor.vue`

---

### P7: AI 大模型分析 (AIOps) (预计 5 天)

**目标**: AI 辅助运维分析

**实现要点**:
- 集成 LLM API (OpenAI/DeepSeek/通义千问)
- 日志分析: 粘贴日志 → AI 给出故障原因建议
- 告警聚合: 多条告警 → AI 自动关联分析根因
- 运维问答: 基于知识库的 RAG 问答
- 配置文件: `config.yaml` 增加 AI 模型配置

**新建文件**:
- `api/api/ai/` — 新模块 (service/controller)
- `web/src/views/ai/AiAssistant.vue`
- `web/src/views/ai/LogAnalyzer.vue`

---

## 六、开发约定

### 后端

```
命名: camelCase 变量, PascalCase 类型/函数
模块目录: api/api/{module}/controller|dao|model|service
路由注册: api/router/{module}/{module}.go → RegisterXxxRoutes()
自动建表: api/pkg/db/migrate.go → models 切片
RBAC 格式: middleware.RbacMiddleware("{module}:{sub}:{action}")
错误返回: result.Failed(c, httpCode, msg) / result.Success(c, data)
分页返回: result.SuccessWithPage(c, list, total, page, pageSize)
```

### 前端

```
页面: web/src/views/{module}/XxxPage.vue
API:  web/src/api/{module}.js → request({url, method, params/data})
路由: web/src/router/system.js (添加 import + route 对象)
请求: import request from '@/utils/request'
通知: ElMessage.success/error/warning
图标: @element-plus/icons-vue
```

### 数据库

```
菜单: INSERT INTO sys_menu (parent_id, menu_name, value, menu_type, url, sort, create_time)
  menu_type: 1=目录, 2=页面路由, 3=权限按钮
权限: INSERT INTO sys_role_menu (role_id, menu_id) VALUES (1, @menuId)  -- admin
```

### Docker

```bash
docker compose build devops-api devops-web      # 构建
docker compose up -d devops-api devops-web       # 部署
docker compose logs -f devops-api                # 日志
docker exec devops-mysql mysql -uroot -pdevops@2025 devops  # 数据库
```

---

## 七、环境信息

| 项目 | 值 |
|------|--|
| 仓库地址 | `github.com/bernylinville/AutoOps` |
| 主分支 | `main` |
| API 端口 | `18000 (host) → 8000 (container)` |
| Web 端口 | `18088 (host) → 80 (container)` |
| MySQL | `127.0.0.1:13306`, root/devops@2025, db=autoops |
| Redis | `127.0.0.1:16379`, password=123456 |
| 默认账号 | admin / 123456 |
| Go 版本 | 1.25 |
| Node 版本 | 20 |
| Webhook Token | `webhook-notify-token-2024` |

# 天枢 AutoOps 运维管理系统

<p align="center">
  <b>统一平台 · 全栈治理 · 安全合规</b>
</p>

## 项目简介

天枢 AutoOps（枢=枢纽+中心）是一个基于 **Go 1.25 + Vue 3 + Element Plus** 开发的企业级运维自动化平台。打破 CI/CD、监控、CMDB、K8s、工单等系统孤岛，数据互通、策略统一，将常用运维工具（夜莺 N9E、Jenkins、JumpServer、Kuboard、Archery 等）集成在统一平台中，实现运维自动化。

![架构图](assets/jg.png)

## 技术栈

| 层级 | 技术 |
|------|------|
| **后端** | Go 1.25 · Gin · GORM · JWT · RBAC · Cron |
| **前端** | Vue 3.5 · Element Plus 2.10 · ECharts · xterm.js |
| **数据库** | PostgreSQL 17.4 (JSONB/GIN/Recursive CTE) · Valkey 9.0 (BSD) |
| **监控** | Prometheus · Pushgateway · N9E 夜莺 |
| **部署** | Docker Compose · Alpine 3.23 · Nginx 1.28 |
| **安全** | bcrypt 密码 · AES 加密 · RBAC 中间件 · 非 root 容器 |

---

## 功能清单

### 🖥 CMDB 资产管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 主机管理 | ✅ | 支持阿里/腾讯/百度/华为/跳板机，827+ 主机 |
| 主机终端 (WebTerminal) | ✅ | 基于 xterm.js + WebSocket，实时 SSH |
| 数据库管理 | ✅ | MySQL / PostgreSQL / Redis / ES / MongoDB 五类 |
| SQL 在线执行 | ✅ | SELECT/UPDATE 在线执行 + SQL 类型白名单校验 |
| 数据来源筛选 | ✅ | N9E 同步 / 手动录入 / 云厂商 三种来源标签 |
| 文件上传/下载 | ✅ | SCP 文件传输到远程主机 |
| 批量操作 | ✅ | 批量命令执行 / 文件分发 |

### 🔷 动态 CI 模型系统 (CMDB 2.0)

| 功能 | 状态 | 说明 |
|------|------|------|
| 动态 CI 类型管理 | ✅ | 自定义 CI 类型 + 动态 JSONB 属性，表单自动渲染 |
| 内置 CI 模板 | ✅ | 6 种预置类型（服务器/数据库/网络设备/中间件/存储/负载均衡），共 33 个属性 |
| CI 实例管理 | ✅ | CRUD + JSONB 动态查询 + 批量导入导出 |
| CI 关系拓扑 | ✅ | PostgreSQL `WITH RECURSIVE` CTE + ECharts Force-Graph 可视化 |
| 项目维度管理 | ✅ | 项目 → 主机/数据库/应用 三维度资产关联 + 资产统计仪表盘 |
| 资产变更日志 | ✅ | 字段级变更审计（变更前值 / 变更后值 / 操作人 / 时间） |
| 主机生命周期 | ✅ | 10 状态机（采购→入库→待上线→上线→…→报废）+ 批量变更 |
| 网络设备管理 | ✅ | SNMP v2c 巡检 + TCP 三端口探测 + 连续失败告警 |
| 到期预警通知 | ✅ | 每日 09:00 扫描 30 天内到期主机 + 钉钉 Webhook 推送 |

### ☸ Kubernetes 集群管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 多集群管理 | ✅ | 注册/同步/编辑/删除集群 |
| 节点管理 | ✅ | 监控/标签/污点/封锁/驱逐 |
| 工作负载管理 | ✅ | Deployment 伸缩/重启/回滚/YAML编辑 |
| Pod 管理 | ✅ | 日志查看 / 终端连接 / 删除 |
| Service & Ingress | ✅ | 网络资源 CRUD + YAML |
| 命名空间管理 | ✅ | 资源配额 + LimitRange 配置 |
| ConfigMap & Secret | ✅ | 配置管理 + 密钥安全存储 |
| 存储管理 | ✅ | PV / PVC / StorageClass |

### 📡 监控中心

| 功能 | 状态 | 说明 |
|------|------|------|
| N9E 夜莺集成 | ✅ | 自动同步业务组 + 主机 + 数据源 |
| CMDB 总览 | ✅ | 主机统计 + 数据来源饼图 + 30s 自动刷新 |
| 数据源健康检查 | ✅ | Prometheus 连通性检测 + 全量批量检查 |
| 定时同步调度器 | ✅ | Cron 配置化 + 自动写入同步日志 |
| 同步日志 | ✅ | 可展开明细 + 状态/触发方式筛选 |
| 告警规则管理 | ✅ | 规则 CRUD + 严重级别 + 来源分类 |
| 告警事件接收 | ✅ | Alertmanager 兼容 Webhook + 规则匹配 |
| 通知渠道 | ✅ | 企业微信 / 钉钉 / 邮件 + 测试发送 |
| 域名监控 | ✅ | SSL 证书到期 + DNS 巡检 |
| 故障管理 | ✅ | 故障记录 + 状态跟踪 |

### 🚀 服务管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 应用管理 | ✅ | 多环境配置 + 版本管理 |
| Jenkins 快速发布 | ✅ | 可视化触发 Jenkins Pipeline |
| 工单审批发布 | ✅ | 应用上线 + 脚本上线审批流程 |

### 📋 任务中心

| 功能 | 状态 | 说明 |
|------|------|------|
| 任务模板 | ✅ | Shell / Python 脚本模板 |
| 任务作业 | ✅ | 定时执行 + 手动触发 |
| Ansible Playbook | ✅ | 可视化 Ansible 任务管理 + 日志回放 |

### 🔧 运维工具

| 功能 | 状态 | 说明 |
|------|------|------|
| 服务市场 | ✅ | MySQL / Redis / PostgreSQL / Jenkins / GitLab 等一键部署 |
| Agent 管理 | ✅ | 监控 Agent 远程部署/卸载/重启 |
| 部署管理 | ✅ | 服务部署记录 + 卸载 |

### ⚙ 配置中心

| 功能 | 状态 | 说明 |
|------|------|------|
| 主机凭据 (ECS) | ✅ | SSH 密钥/密码管理 |
| 通用凭据 | ✅ | 第三方账号 AES 加密存储 + 解密授权 |
| 云密钥管理 | ✅ | 阿里/腾讯/华为 AccessKey 管理 + 同步 |
| 同步配置 | ✅ | 云厂商定时同步调度 |

### 🔒 安全与审计

| 功能 | 状态 | 说明 |
|------|------|------|
| RBAC 权限控制 | ✅ | 200+ 权限码 + 5min 缓存 + 细粒度路由保护 |
| 菜单-路由-按钮权限 | ✅ | 三级权限模型 (menu_type 1/2/3) |
| 操作日志审计 | ✅ | 自动记录用户/IP/URL/Method |
| 登录日志审计 | ✅ | 登录记录 + 批量清理 |
| 数据库操作审计 | ✅ | SQL 执行日志 + 类型白名单 |
| 会话录制 | ✅ | 终端操作录像回放 |
| CI 变更审计 | ✅ | 字段级变更追踪（主机/数据库/CI实例） |

### 🏠 系统管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 用户管理 | ✅ | bcrypt 密码 + 角色分配事务化 |
| 角色管理 | ✅ | 超级管理员 + 自定义角色 |
| 菜单管理 | ✅ | 动态菜单 + 批量查询优化 |
| 部门管理 | ✅ | 组织架构树 |
| N9E 配置 | ✅ | 夜莺连接配置 + Cron 热更新 |

---

## 安全特性

- ✅ JWT + AES 密钥从环境变量读取，不硬编码
- ✅ RBAC 中间件保护 200+ 敏感路由
- ✅ SQL 执行类型白名单（仅允许 SELECT/UPDATE）
- ✅ WebSocket/SSE 强制 JWT 鉴权
- ✅ 命令注入参数校验
- ✅ SSH 凭据 API 响应脱敏
- ✅ Docker 基础设施端口绑定 `127.0.0.1`
- ✅ API 容器以非 root 用户运行
- ✅ `/healthz` + `/readyz` 无鉴权健康检查端点
- ✅ Webhook token 校验
- ✅ CI 变更审计日志（不可篡改）

---

## 快速部署

### Docker Compose 一键部署

```bash
# 1. 克隆仓库
git clone https://github.com/bernylinville/AutoOps.git
cd AutoOps/docker

# 2. 配置环境变量（可选，修改端口/密码等）
cp .env.example .env
vi .env

# 3. 配置 API（可选，修改钉钉 Webhook 等）
cp api/config.yaml.example api/config.yaml
vi api/config.yaml

# 4. 启动所有服务
docker compose up -d

# 5. 查看服务状态
docker compose ps

# 6. 访问系统
# Web 前端: http://localhost:18088
# API 后端: http://localhost:18000
# 默认账号: admin / 123456
```

### 服务组件

| 服务 | 端口 | 说明 |
|------|------|------|
| devops-web | 18088 → 80 | Nginx 前端 |
| devops-api | 18000 → 8000 | Go API |
| devops-postgres | 127.0.0.1:15432 → 5432 | PostgreSQL 17.4 |
| devops-valkey | 127.0.0.1:16379 → 6379 | Valkey 9.0 (Redis 兼容) |
| devops-prometheus | 127.0.0.1:19090 → 9090 | Prometheus |
| devops-pushgateway | 127.0.0.1:19091 → 9091 | Pushgateway |

> 💡 PostgreSQL/Valkey/Prometheus 端口绑定 `127.0.0.1`，仅本机访问，生产环境安全。

详细部署文档: [docker/README.md](docker/README.md)

---

## 本地开发环境

### 前置条件

| 工具 | 版本 | 说明 |
|------|------|------|
| [mise](https://mise.jdx.dev/) | latest | 多版本运行时管理 (替代 asdf/nvm/gvm) |
| Go | 1.24+ | `mise install go@1.24` |
| Node.js | 22+ | `mise install node@22` |
| Docker / Docker Compose | 27+ | 容器运行环境 |

### 首次启动

```bash
# 1. 安装运行时
mise install

# 2. 启动数据库和基础服务（不构建镜像）
cd docker && docker compose up -d postgres valkey prometheus pushgateway

# 3. 启动后端
cd api
cp config.yaml.example config.yaml
go mod download
go run .

# 4. 启动前端
cd web
npm install
npm run dev

# 5. 访问 http://localhost:5173（Vite 开发服务器）
```

### 数据库备份与恢复

```bash
# 备份 PostgreSQL
docker compose exec postgres pg_dump -U devops autoops > backup.sql

# 恢复 PostgreSQL
docker compose exec -T postgres psql -U devops autoops < backup.sql
```

---

## 测试环境

🌐 https://autoops.com.cn/  
📺 [B 站安装视频教程](https://www.bilibili.com/video/BV179Wxz1Ez6/?vd_source=37f81c1b36b3818cbad621bcbe5c3e49)

```
账号: test
密码: 123456
```

---

## 项目结构

```
AutoOps/
├── api/                        # Go 后端
│   ├── api/                    # 业务逻辑 (controller/dao/model/service)
│   │   ├── cmdb/               # CMDB 资产管理 + 动态 CI + 项目 + 拓扑 + 网络设备
│   │   ├── k8s/                # Kubernetes 管理
│   │   ├── n9e/                # N9E 监控 + 告警通知
│   │   ├── configcenter/       # 配置中心
│   │   ├── task/               # 任务中心
│   │   ├── monitor/            # 监控 Agent
│   │   ├── tool/               # 运维工具
│   │   ├── app/                # 服务管理 + 应用管理
│   │   ├── dashboard/          # 业务大盘
│   │   ├── flashduty/          # FlashDuty 告警集成
│   │   └── system/             # 系统管理
│   ├── middleware/              # JWT / RBAC / 日志中间件
│   ├── router/                 # 路由注册
│   ├── scheduler/              # 定时同步调度器 + 到期预警
│   ├── pkg/                    # 公共包 (JWT/DB/Migration)
│   └── config.yaml             # 配置文件
├── web/                        # Vue 3 前端
│   └── src/
│       ├── views/              # 页面组件 (含 CI/拓扑/变更日志/网络设备/项目)
│       ├── api/                # API 请求封装
│       ├── router/             # 前端路由
│       └── utils/              # 工具函数
└── docker/                     # Docker 部署
    ├── docker-compose.yml      # 6 服务编排
    ├── .env                    # 环境变量（端口/密码等）
    ├── api/Dockerfile          # Go 多阶段构建 + 非 root
    ├── web/Dockerfile          # Node 构建 + Nginx
    ├── postgres/               # PG 数据 + 初始化脚本
    └── valkey/                 # Valkey 缓存数据
```

---

<details>
<summary><b>🔍 点击展开优势对比详情</b></summary>

### DevOps 运维管理系统优势

✅ **轻量级** — 单体应用，部署简单，资源占用少  
✅ **全栈运维** — 同时管理传统主机和 K8s 集群，无需多套系统  
✅ **开箱即用** — 内置 CMDB、任务调度、SQL 审计等企业级功能  
✅ **安全合规** — RBAC 200+ 权限码 + 操作审计 + 非 root 容器  
✅ **监控集成** — N9E 夜莺深度集成 + 告警通知 (微信/钉钉/邮件)  
✅ **动态 CI** — 自定义资产模型 + JSONB 动态属性 + 关系拓扑图  
✅ **二次开发友好** — Go 语言，代码结构清晰，易于定制  
✅ **成本低** — 无商业授权费用，适合中小企业  

### 对比总结

```
传统运维平台痛点：
├─ 🔴 工具碎片化 → 多系统切换，数据孤岛
├─ 🔴 流程不闭环 → 发布、审批、审计分离
├─ 🔴 资产模型固定 → 难以适配异构设备
├─ 🔴 云原生支持弱 → 难以适配容器化架构
└─ 🔴 安全审计缺失 → 缺乏 RBAC 和操作追溯

AutoOps 优势：
├─ 🟢 一体化设计 → 统一平台，数据打通
├─ 🟢 动态 CI 模型 → 自定义资产类型 + JSONB 属性
├─ 🟢 云原生支持 → K8s 全要素深度集成
├─ 🟢 安全合规 → RBAC + 审计 + 非 root 部署
├─ 🟢 告警通知 → Webhook + 微信/钉钉/邮件
├─ 🟢 N9E 集成 → 夜莺监控数据自动同步
└─ 🟢 资产生命周期 → 10 状态机 + 变更审计 + 到期预警
```

### 业界对比

| 功能领域 | AutoOps | 蓝鲸 CMDB | VeOps | NetBox | iTop |
|----------|:---:|:---:|:---:|:---:|:---:|
| 动态 CI 模型 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 项目维度管理 | ✅ | ✅ | ❌ | ❌ | ❌ |
| CI 关系拓扑 | ✅ | ✅ | ✅ | ✅ | ✅ |
| 资产生命周期 | ✅ | ✅ | ❌ | ❌ | ✅ |
| 变更审计日志 | ✅ | ✅ | ❌ | ✅ | ✅ |
| 网络设备巡检 | ✅ | ❌ | ❌ | ✅ | ❌ |
| K8s 深度管理 | ✅ | ✅ | ❌ | ❌ | ❌ |
| SSH 终端 | ✅ | ✅ | ❌ | ❌ | ❌ |
| SQL 执行+审计 | ✅ | ❌ | ❌ | ❌ | ❌ |
| 监控集成 | ✅ | ✅ | ❌ | ❌ | ❌ |

### 适用场景

| 场景 | 推荐方案 |
|------|---------|
| 中小企业混合环境 (VM + K8s) | **AutoOps** |
| 快速上线，资源有限 | **AutoOps** |
| 需要传统运维 + 容器化双轨并行 | **AutoOps** |
| 纯 K8s 大规模部署 | KubeSphere / Rancher |

</details>

---

## 后续规划

| 功能 | 优先级 | 状态 |
|------|:---:|------|
| 巡检系统集成 (inspection-tool 评估引擎) | P1 | 📋 规划中 |
| ITSM 工单 (变更管理 + 钉钉企业内部应用) | P1 | 📋 规划中 |
| 虚拟化集成 (VMware vCenter / PVE) | P1 | 📋 规划中 |
| 多云覆盖 (火山云/天翼云/华为云) | P1 | 📋 规划中 |
| IPAM 地址管理 (网段/IP/VLAN + 冲突检测) | P2 | 📋 规划中 |
| 机柜可视化 (U 位立面图 + 拖拽放置) | P2 | 📋 规划中 |
| 资产统计报表 (多维 Dashboard + 导出) | P2 | 📋 规划中 |
| 运维知识库 (Markdown + PG 全文搜索) | P2 | 📋 规划中 |
| Windows 主机管理 + 远程桌面 | P2 | 📋 规划中 |
| Open API (API Key 鉴权 + Webhook 推送) | P3 | 📋 规划中 |
| AIOps (CI 拓扑告警收敛 + pgvector) | P3 | 📋 规划中 |

---

## 感谢以下同学对本项目提供的打赏

<p align="center">
  <img src="assets/zanzhu/1.png" width="120" />
  <img src="assets/zanzhu/4.png" width="120" />
  <img src="assets/zanzhu/5.png" width="120" />
  <img src="assets/zanzhu/6.png" width="120" />
  <img src="assets/zanzhu/7.png" width="120" />
  <img src="assets/zanzhu/8.png" width="120" />
  <img src="assets/zanzhu/9.png" width="120" />
  <img src="assets/zanzhu/10.png" width="120" />
  <img src="assets/zanzhu/11.png" width="120" />
  <img src="assets/zanzhu/12.png" width="120" />
  <img src="assets/zanzhu/13.png" width="120" />
  <img src="assets/zanzhu/14.png" width="120" />
</p>

## 联系作者

## 技术交流+社区

<img src="assets/zf.jpg" width="300" />

#### 加群技术交流

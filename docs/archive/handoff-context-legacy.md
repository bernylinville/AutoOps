# AutoOps CMDB 平台升级 —— 开发交接文档

> **文档生成时间**: 2026-04-03
> **项目背景**: AutoOps 是一个统一运维管理平台。目前正在进行 CMDB（配置管理数据库）模块的核心层升级，用以支持更加灵活的动态 CI 模型、项目维度管理以及拓扑可视化等功能，并已完成底层基础设施更换。

---

## 1. 实现目标 (Implementation Goals)

1. **底层架构现代化**：将数据库从 MySQL 8.0 迁移至 PostgreSQL 17.4（利用其强大的 JSONB 和递归查询能力），并将缓存从 Redis 迁移至开源的 Valkey 8.1。
2. **灵活的资产模型**：废弃死板的单表模型，引入基于 `JSONB` 的**动态 CI (配置项) 模型**系统。管理员可自定义 CI 类型和属性，无需修改底层 Schema。
3. **多维资产视角**：建立“项目 (Project) - 应用 (Application)”维度的资产管理，便于从业务视角统计和管理资源。
4. **关系与拓扑可视化**：建立 CI 之间的关联关系，并通过前端 ECharts Graph 实现资源链路的拓扑可视化展示与影响分析。
5. **完整的生命周期跟踪**：为所有核心资产加入规范的状态流转（采购→入库→运行→退服等）与维保到期提醒。

---

## 2. 方案规划 (Solution Planning)

系统升级划分为 **5 个核心长线阶段 (Phases)**，技术栈要求与方案如下：

- **系统技术栈**：
  - 后端：Go 1.24+ / Gin / GORM (gorm.io/driver/postgres, gorm.io/datatypes)
  - 前端：Vue 3.5 / Element Plus 2.10 / JS (原 Vben Admin 架构已调整)
  - 存储层：PostgreSQL 17.4, Valkey 8.1

- **P0 阶段规划路线**：
  - **Phase 0: 基础设施迁移** (完成数据库与缓存替换，迁移近 1000 条原有资产数据)。
  - **Phase 1: 动态 CI 模型系统** (`CIType`, `CITypeAttribute`, `CIInstance`, `CIRelation` 表结构与全套 CRUD 及动态表单展示)。
  - **Phase 2: 项目维度资产管理** (新增项目/应用模型，关联原有的主机、数据库资源)。
  - **Phase 3: CI 关系与拓扑层** (利用 PG 的 `WITH RECURSIVE` 递归查询生成拓扑树，Vue 前端 ECharts 渲染)。
  - **Phase 4: 资产生命周期与预警** (定时任务Cron，钉钉推送，变更日志 `CIChangeLog` 表)。
  - **Phase 5: 网络设备管理后端增强** (特定设备的独立管理后端与页面)。

---

## 3. 当前完成情况 (Current Status)

✅ **Phase 0 (基础设施迁移) — 100% 完成**
- PostgreSQL 17 容器 (`devops-postgres` 端口 15432) 和 Valkey 容已就绪并替换。
- 业务大盘旧有数据（827 台主机、监控配置等）已全部成功平滑迁移至 PG，并经过数据一致性与仪表盘面板验证。
- 后端 GORM 驱动全量替换完成并自动建表成功。

✅ **Phase 1 (动态 CI 模型系统) — 100% 完成**
- **后端**：借助 `gorm.io/datatypes` 实现 4 个核心 GORM 模型的 JSONB 存储；完成 15 个 REST API 接口；内置 6 大内置模型（服务器、数据库、中间件、网络设备、存储、负载均衡）并成功 Seed (包含 33 个总属性)。
- **前端**：新增 `CIManage.vue`，实现动态的 Tabs 切换渲染、根据 `showInList` 动态生成的表格列、基于 `dataType` 动态拉起的表单（Input/Select/Switch 等）。
- **菜单权限**：`sys_menu` 表数据已修复并赋权，“资产管理 -> CI模型” 正常展示且无 Bug。

---

## 4. 任务规划 (Next Task Planning)

接下来的开发将主要围绕 **Phase 2** 及后续展开。这是新 AI 助手接手的待办清单：

### ⏳ 待接手任务 1：Phase 2 - 项目维度管理 (Project Dimension Management)
- [ ] 创建 `Project` 与 `Application` 后端 GORM 模型。
- [ ] 在原有的 `CmdbHost` 和 `CmdbSQL` 表中扩展绑定 `ProjectID` 字段。
- [ ] 新增相关的 Controller、DAO 接口（例如：获取项目关联的资产统计信息）。
- [ ] 开发前端管理页面：`ProjectList.vue` (项目列表展示)、`ProjectDetail.vue` (业务全景监控)。

### ⏳ 待接手任务 2：Phase 3 - CI 拓扑可视化 (CI Topology)
- [ ] 后端：编写针对 `ci_relation` 表的深层向上/向下递归查询 API 接口。
- [ ] 前端：集成 ECharts，开发 `CITopology.vue`，实现基于节点类型的样式组分类显示和链路渲染。

### ⏳ 待接手任务 3：Phase 4 - 资产生命周期与预警
- [ ] 扩展主机状态枚举为全生命周期类型。
- [ ] 构建 `CIChangeLog` 表以追踪属性修改与状态流转历史。

---

## 5. 给新工具的启动提示词 (Prompt)

请将下方提示词及本文档一起发送给接手的 Vibe Coding 工具：

```text
角色设定：你是一个资深的 Go 全栈 AI 专家，当前正在接手【AutoOps 统一运维管理平台】的迭代开发。

背景上下文：
前任开发已将系统从 MySQL 切换至 PostgreSQL 17，用 JSONB 落地了一套完整的“动态 CI 系统”，并且实现了基础组件与菜单（参考交接文档中 Phase 0 和 Phase 1）。

你的任务：
请详细阅读附带的《交接文档》。你接下来要直接进入 【Phase 2: 项目维度管理】 的开发工作。
目前系统根目录位于: /home/kchou/Code/ops/AutoOps (或具体的用户工作区)。

第一步要求：
在不破坏现有的 CI 模型代码与 PG 数据库连接的前提下，请先检索分析当前的 `api/api/cmdb/model` 目录，输出对于 Phase 2 中 `Project` 与 `Application` 模型的代码设计思路以及向现有表的迁移策略，并等待我的确认。在你分析前也可以随时翻阅已经完成的 `ciType.go` 的写法风格。
```

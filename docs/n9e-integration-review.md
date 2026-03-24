# AutoOps N9E (夜莺监控) 集成 — Code Review 文档

## 项目背景

在 AutoOps 运维管理系统中引入 N9E (Nightingale) 夜莺监控能力，实现：
- N9E 连接配置与测试
- 自动同步 N9E 的业务组、主机、数据源到 AutoOps CMDB
- 定时同步 (Cron 调度)
- 同步日志记录
- CMDB 总览仪表盘
- Stale (失联) 资产检测

技术栈：Go (Gin) 后端 + Vue.js (Element Plus) 前端 + MySQL + Docker

---

## 一、新增/修改文件清单

### 后端 (Go)

#### 新增文件

| 文件 | 说明 |
|------|------|
| `api/api/n9e/model/n9e.go` | N9E 数据模型：N9EConfig, N9EBusiGroup, N9EDataSource, N9ESyncLog, DTO |
| `api/api/n9e/model/types.go` | N9E API 响应类型：TargetData, BusiGroupData, DatasourceData, SyncResult |
| `api/api/n9e/dao/n9e.go` | DAO 层：配置/业务组/数据源/同步日志 CRUD, FindOrCreateCmdbGroup, MarkHostsStale |
| `api/api/n9e/service/client.go` | N9E API 客户端：GetBusiGroups, GetTargets(分页), GetDatasourceBriefs, QueryPromQL |
| `api/api/n9e/service/sync.go` | 同步服务：FullSync (业务组+主机+数据源), stale 检测, 日志记录, sync.Mutex 防重入 |
| `api/api/n9e/controller/n9e.go` | 控制器：GetConfig(脱敏), SaveConfig(跳过脱敏Token), TestConnection(从DB读真Token), TriggerSync |
| `api/api/n9e/controller/overview.go` | 总览控制器：GetOverview (聚合统计) |
| `api/router/n9e/n9e.go` | N9E 路由注册 (11 个端点) |
| `api/scheduler/n9eSyncScheduler.go` | N9E 定时同步调度器 (robfig/cron + SkipIfStillRunning) |

#### 修改文件

| 文件 | 变更 |
|------|------|
| `api/api/cmdb/model/cmdbHost.go` | CmdbHost 新增 `SourceType`, `N9EID`, `N9EIdent` 字段；CmdbHostVo 同步新增 |
| `api/api/cmdb/service/cmdbHost.go` | 8 个查询方法的 VO 构建补充 SourceType/N9EID/N9EIdent 映射 |
| `api/router/router.go` | 导入并注册 N9E 路由模块 |
| `api/scheduler/manager.go` | 集成 N9ESyncScheduler, 新增 ReloadN9ECron() |
| `api/sql/n9e_migration.sql` | DDL: n9e_config, n9e_sync_log, n9e_busi_group, n9e_datasource; ALTER cmdb_host; 菜单数据 |
| `api/sql/autoops.sql` | 基线 SQL 更新：N9E 4 张表 + cmdb_host 字段定义与迁移脚本一致 |

### 前端 (Vue)

#### 新增文件

| 文件 | 说明 |
|------|------|
| `web/src/views/system/N9eConfig.vue` | N9E 配置页：连接表单 + 测试 + 手动同步 + 同步统计 |
| `web/src/views/monitor/N9eMonitor.vue` | N9E 监控页：统计卡片(含失联) + 筛选 + 多页加载主机列表 |
| `web/src/views/monitor/N9eDatasource.vue` | 数据源页：列表 + PromQL 查询 + ECharts 图表 |
| `web/src/views/monitor/N9eOverview.vue` | CMDB 总览：统计卡片 + 来源饼图 + 同步状态 |

#### 修改文件

| 文件 | 变更 |
|------|------|
| `web/src/api/n9e.js` | 11 个 API 方法封装 |
| `web/src/router/system.js` | 4 条新路由 |

---

## 二、API 端点

```
GET    /api/v1/n9e/config            # 获取 N9E 配置 (Token 脱敏返回)
POST   /api/v1/n9e/config            # 保存 N9E 配置 (脱敏 Token 不覆盖)
POST   /api/v1/n9e/test-connection   # 测试 N9E 连接 (脱敏 Token 从 DB 读取真实值)
POST   /api/v1/n9e/sync              # 手动触发同步 (Mutex 防重入)
GET    /api/v1/n9e/sync/status       # 获取同步状态
GET    /api/v1/n9e/sync/logs         # 查询同步历史日志
GET    /api/v1/n9e/overview          # CMDB 总览统计
GET    /api/v1/n9e/busi-groups       # 获取 N9E 业务组列表
GET    /api/v1/n9e/datasources       # 获取数据源列表
GET    /api/v1/n9e/query             # PromQL 代理查询
```

所有端点在 JWT 鉴权组下。

---

## 三、已验证结果 (2026-03-23)

连接 N9E 实例 `http://120.26.87.44:17000` 测试：

- ✅ 配置保存 + 测试连接成功 ("连接成功！N9E 服务器响应正常")
- ✅ 脱敏 Token 保存不会覆盖真实凭据
- ✅ 全量同步：**72 业务组 / 827 主机 / 1 数据源**
- ✅ 同步日志写入 `n9e_sync_log`
- ✅ 业务组自动映射到 CMDB 分组 (group_id=71, 非硬编码 1)
- ✅ N9E 监控页正确显示 827 主机 + 数据来源 "N9E" 标签
- ✅ CMDB 总览页正确显示统计卡片 + 来源饼图
- ✅ 防重入同步 (sync.Mutex + SkipIfStillRunning) 工作正常

---

## 四、Codex Review Bug 修复清单

| # | 问题 | 修复方式 | 修改文件 | 状态 |
|---|------|----------|----------|------|
| F1 | `autoops.sql` 基线表结构与代码不一致 | 重写 N9E 表定义 + 补充 `n9e_sync_log` | `autoops.sql` | ✅ |
| F2 | Token 脱敏值回写覆盖真实凭据 | DAO 检测 `****` 跳过更新；TestConnection 从 DB 读真实 Token | `dao/n9e.go`, `controller/n9e.go` | ✅ |
| F3 | N9eMonitor 页面数据拿不到 | CmdbHostVo 补充 3 字段 (8 方法)；前端多页加载 | `cmdbHost.go`, `N9eMonitor.vue` | ✅ |
| F4 | 同步无防重入 | `sync.Mutex.TryLock()` + 单例 SyncService | `sync.go`, `n9eSyncScheduler.go` | ✅ |
| F5 | GetTargets 仅取第一页 | 循环分页 pageSize=5000 | `client.go` | ✅ |

### F2 Token 脱敏修复详情

**问题**：GetConfig 返回脱敏 Token（`750a****e9f0`），前端直接回传 SaveConfig/TestConnection，导致真实 Token 被覆盖或连接 401。

**修复**：
```go
// dao/n9e.go - SaveN9EConfig
if !strings.Contains(dto.Token, "****") {
    updates["token"] = dto.Token  // 仅真实 Token 才更新
}

// controller/n9e.go - TestConnection
if strings.Contains(token, "****") {
    config, _ := dao.GetN9EConfig()
    token = config.Token  // 从 DB 读真实 Token
}
```

### F3 N9eMonitor 修复详情

**问题**：前端 `pageSize: 10000` 超出后端 max=100 限制 → 返回空；CmdbHostVo 缺少 sourceType → 过滤失败。

**修复**：
- 后端 8 个方法全量补充 `SourceType`, `N9EID`, `N9EIdent` 映射
- 前端改为 `while` 循环 pageSize=100 分页加载

### F4 并发安全修复详情

**修复**：
- `SyncService` 改为单例 + `sync.Mutex`
- `FullSync` 使用 `TryLock()` 非阻塞防重入
- Cron 注册 `SkipIfStillRunning` 包装
- `FindOrCreateCmdbGroup` 事务性 upsert

---

## 五、仍需关注项

- `service/sync.go` syncHosts 循环：单条失败跳过继续，建议增加错误汇总
- Stale 检测逻辑：首次同步时不应标记任何 stale
- ECharts 全量引入可考虑按需引入减小包体积
- N9E 同步增量模式 (当前为全量同步)

---

## 六、后续开发需求

### Phase 2 — 监控告警集成
- [ ] N9E 告警规则同步与展示
- [ ] FlashDuty 告警对接
- [ ] 告警通知 (企业微信/钉钉/邮件)

### Phase 3 — 深度资产管理
- [ ] 主机详情页 (基础信息 + N9E 监控指标图表)
- [ ] 资产变更审计日志
- [ ] 资产标签/分类管理
- [ ] 批量操作 (分组移动、状态变更)

### Phase 4 — 自动化运维
- [ ] 基于 CMDB 的 Ansible 自动化集成
- [ ] 主机初始化模板
- [ ] 巡检任务自动化

---

## 七、数据库 Schema

### 新增表

```sql
-- N9E 连接配置 (单行配置表)
n9e_config (id, endpoint, token, timeout, sync_cron, enabled, last_sync_time, last_sync_result, create_time, update_time)

-- N9E 同步日志
n9e_sync_log (id, sync_type, status, result_json, error_msg, duration_ms, trigger_by, created_at)

-- N9E 业务组 (去重: n9e_group_id)
n9e_busi_group (id, n9e_group_id, name, create_time, update_time)

-- N9E 数据源 (去重: n9e_source_id)
n9e_datasource (id, n9e_source_id, name, plugin_type, category, url, status, create_time, update_time)
```

### 扩展字段

```sql
ALTER TABLE cmdb_host ADD COLUMN source_type varchar(20) DEFAULT 'manual';
ALTER TABLE cmdb_host ADD COLUMN n9e_id bigint DEFAULT NULL;
ALTER TABLE cmdb_host ADD COLUMN n9e_ident varchar(128) DEFAULT NULL;
```

### Status 值映射

| 值 | 含义 |
|----|------|
| 1 | 认证成功 (在线) |
| 2 | 未认证 |
| 3 | 认证失败 |
| 4 | Stale (失联 - N9E 同步后未出现) |

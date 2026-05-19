# AutoOps 巡检功能 — 技术规格 (TECH.md)

**对应意图文档**: [docs/intent/inspection.md](../intent/inspection.md)  
**编写日期**: 2026-05-19  
**状态**: Draft

---

## 1. 总体架构

### 1.1 模块划分

新增 `api/api/inspection/` 模块，遵循 AutoOps 现有约定：

```
api/api/inspection/
  controller/       # HTTP 控制器
  service/          # 业务逻辑
  dao/              # 数据访问
  model/            # GORM 模型 + DTO + VO
  scheduler/        # 定时调度器
  report/           # 报告生成（复用/适配 inspection-tool）

router/inspection/  # 路由注册
```

### 1.2 与 inspection-tool 的集成方式

**不直接 import CLI 入口**（Cobra/Viper/`os.Exit` 不兼容），而是**抽取核心库**：

| 来源 | 抽取内容 | 目标位置 |
|------|---------|---------|
| `internal/model/` | Alert, HostResult, InspectionResult, MetricValue, HostStatus, AlertLevel 等 | `api/api/inspection/model/itmodel/`（inspection-tool model 适配层） |
| `internal/service/` | Collector（采集）、Evaluator（评估）、Inspector（编排）的核心逻辑 | `api/api/inspection/service/engine/` |
| `internal/client/n9e` | N9E 主机元信息获取 | 复用 AutoOps 已有的 `api/api/n9e/service/client.go` |
| `internal/client/vm` | VictoriaMetrics PromQL 查询 | **移植** inspection-tool 的 VM client 核心到 `api/api/inspection/service/engine/vmclient/`。AutoOps 现有 `QueryPromQL` 是 `query_range`，巡检需要 **instant query** + **label matcher 注入** + **按 ident 分组解析** |
| `internal/report/excel` | Excel 报告生成 | `api/api/inspection/report/excel.go`（适配，修复多业务组文件名冲突） |

**抽取原则**：
- 移除 Cobra/Viper 依赖，配置通过函数参数传入
- 移除 `zerolog`，改用 AutoOps 的 `pkg/log`
- 移除 `os.Exit`，错误通过 `error` 返回
- 接受 `context.Context`，支持取消和超时
- 输出结构化数据（`InspectionResult`），不直接写文件

### 1.3 依赖关系

```
inspection/controller
    -> inspection/service
        -> inspection/dao
        -> inspection/service/engine (抽取的巡检核心)
        -> n9e/service (N9E 客户端)
        -> deploy/service/notifier (钉钉通知)
        -> inspection/report/excel
```

---

## 2. 数据模型

### 2.1 实体关系

```
+----------------------------------+
| inspection_task                  |  任务定义（按业务组）
| - id PK                          |
| - n9e_group_id                   |  <-- 关联 n9e_busi_group.n9e_group_id
| - n9e_group_name                 |  <-- 业务组名称快照（改名后仍可按旧名称过滤 VM）
| - name                           |
| - enabled                        |
| - cron                           |  <-- 默认 "CRON_TZ=Asia/Shanghai 0 10 * * *"
| - target_query                   |  <-- VM label filter，如 busigroup="生产环境"
| - notify_webhook_url             |  <-- 钉钉 Webhook（GET 脱敏返回）
| - notify_secret                  |  <-- 钉钉 Secret（GET 不返回）
| - notify_on_warning              |  <-- warning 是否通知（默认 true）
| - notify_on_critical             |  <-- critical 是否通知（默认 true）
| - notify_on_failure              |  <-- 采集失败/无主机是否通知（默认 true）
| - created_at / updated_at        |
+----------------------------------+
         | 1 : N
         v
+----------------------------------+
| inspection_run                   |  单次运行记录
| - id PK                          |
| - task_id FK                     |
| - trigger_type                   |  <-- "cron" | "manual"
| - triggered_by                   |  <-- 手动触发时的用户 ID
| - status                         |  <-- "pending" | "running" | "success" | "partial" | "failed"
| - started_at                     |
| - ended_at                       |
| - duration_ms                    |
| - total_hosts                    |
| - normal_hosts                   |
| - warning_hosts                  |
| - critical_hosts                 |
| - failed_hosts                   |
| - total_alerts                   |
| - config_snapshot JSON           |  <-- 运行时的配置快照（阈值等）
| - error_message                  |
| - created_at                     |
+----------------------------------+
         | 1 : N
         v
+----------------------------------+
| inspection_target_result         |  主机级结果
| - id PK                          |
| - run_id FK                      |
| - hostname                       |
| - ident                          |  <-- N9E ident
| - ip                             |
| - os                             |
| - status                         |  <-- "normal" | "warning" | "critical" | "failed"
| - error                          |  <-- 采集失败原因
| - metrics JSON                   |  <-- 结构化指标数据
| - boot_time                      |
| - collected_at                   |
| - created_at                     |
+----------------------------------+
         | 1 : N
         v
+----------------------------------+
| inspection_alert                 |  异常明细
| - id PK                          |
| - run_id FK                      |
| - target_result_id FK            |
| - hostname                       |
| - metric_name                    |
| - metric_display_name            |
| - current_value                  |
| - warning_threshold              |
| - critical_threshold             |
| - level                          |  <-- "warning" | "critical"
| - message                        |
| - labels JSON                    |
| - created_at                     |
+----------------------------------+
         | 1 : 0..1
         v
+----------------------------------+
| inspection_report_artifact       |  报告文件
| - id PK                          |
| - run_id FK                      |
| - file_path                      |  <-- /data/inspection/{run_id}/report.xlsx
| - file_size                      |
| - format                         |  <-- "excel"
| - status                         |  <-- "pending" | "success" | "failed"
| - error_message                  |
| - created_at                     |
| - expires_at                     |  <-- created_at + 30 天
+----------------------------------+
```

### 2.2 GORM 模型定义

参考 `api/api/cmdb/model/ciType.go` 的 Models + DTOs + VOs 同文件约定。

```go
// model/task.go
package model

type InspectionTask struct {
    ID               uint       `gorm:"column:id;primaryKey;NOT NULL" json:"id"`
    N9EGroupID       int64      `gorm:"column:n9e_group_id;uniqueIndex:idx_task_group;NOT NULL;comment:'N9E 业务组 ID'" json:"n9eGroupId"`
    N9EGroupName     string     `gorm:"column:n9e_group_name;type:varchar(200);comment:'业务组名称快照'" json:"n9eGroupName"`
    Name             string     `gorm:"column:name;type:varchar(200);NOT NULL;comment:'任务名称'" json:"name"`
    Enabled          bool       `gorm:"column:enabled;default:true;comment:'是否启用'" json:"enabled"`
    Cron             string     `gorm:"column:cron;type:varchar(100);default:'CRON_TZ=Asia/Shanghai 0 10 * * *';comment:'Cron 表达式'" json:"cron"`
    TargetQuery      string     `gorm:"column:target_query;type:varchar(500);comment:'VM label filter，如 busigroup=生产环境'" json:"targetQuery"`
    NotifyWebhookURL string     `gorm:"column:notify_webhook_url;type:varchar(500);comment:'钉钉 Webhook URL'" json:"notifyWebhookUrl"`
    NotifySecret     string     `gorm:"column:notify_secret;type:varchar(200);comment:'钉钉 Secret'" json:"-"` // GET 不返回
    NotifyOnWarning  bool       `gorm:"column:notify_on_warning;default:true;comment:'Warning 是否通知'" json:"notifyOnWarning"`
    NotifyOnCritical bool       `gorm:"column:notify_on_critical;default:true;comment:'Critical 是否通知'" json:"notifyOnCritical"`
    NotifyOnFailure  bool       `gorm:"column:notify_on_failure;default:true;comment:'失败是否通知'" json:"notifyOnFailure"`
    CreateTime       util.HTime `gorm:"column:create_time;NOT NULL" json:"createTime"`
    UpdateTime       util.HTime `gorm:"column:update_time;NOT NULL" json:"updateTime"`
}

// TaskVO —— GET 接口返回的脱敏视图
type TaskVO struct {
    InspectionTask
    NotifyWebhookURL string `json:"notifyWebhookUrl"` // 脱敏: https://oapi.dingtalk.com/robot/send?access_token=***
    NotifySecret     string `json:"notifySecret"`     // 始终返回空字符串
}

// UpdateTaskDto —— PUT 接口入参
// notifyWebhookUrl / notifySecret 传空字符串表示保留原值
type UpdateTaskDto struct {
    Enabled          *bool   `json:"enabled,omitempty"`
    Cron             string  `json:"cron,omitempty"`
    NotifyWebhookURL string  `json:"notifyWebhookUrl,omitempty"`
    NotifySecret     string  `json:"notifySecret,omitempty"`
    NotifyOnWarning  *bool   `json:"notifyOnWarning,omitempty"`
    NotifyOnCritical *bool   `json:"notifyOnCritical,omitempty"`
    NotifyOnFailure  *bool   `json:"notifyOnFailure,omitempty"`
}

// ... 其余模型详见实现阶段
```

### 2.3 索引设计

| 表 | 索引 | 用途 |
|---|---|---|
| `inspection_task` | `unique(n9e_group_id)` | 每个业务组唯一任务 |
| `inspection_run` | `(task_id, created_at DESC)` | 按任务查询运行历史 |
| `inspection_run` | `(status, created_at)` | 清理任务扫描 |
| `inspection_target_result` | `(run_id, status)` | 按运行查询结果 |
| `inspection_alert` | `(run_id, level)` | 按运行查询异常 |
| `inspection_report_artifact` | `(expires_at)` | 清理任务扫描 |
| `inspection_notification` | `(run_id, created_at)` | 按运行查询通知记录 |

### 2.5 通知记录表

```go
type InspectionNotification struct {
    ID         uint       `gorm:"column:id;primaryKey" json:"id"`
    RunID      uint       `gorm:"column:run_id;index;NOT NULL" json:"runId"`
    Channel    string     `gorm:"column:channel;type:varchar(20);default:'dingtalk'" json:"channel"`
    Payload    string     `gorm:"column:payload;type:text;comment:'通知内容摘要'" json:"payload"`
    Status     string     `gorm:"column:status;type:varchar(20);default:'pending'" json:"status"` // pending/sent/failed/skipped
    ErrorMsg   string     `gorm:"column:error_msg;type:text" json:"errorMsg"`
    SentAt     *time.Time `gorm:"column:sent_at" json:"sentAt"`
    CreatedAt  util.HTime `gorm:"column:created_at" json:"createdAt"`
}
```

### 2.4 唯一约束（PostgreSQL）

```sql
-- 防止同一任务同一天自动任务重复运行
CREATE UNIQUE INDEX idx_run_unique_daily
ON inspection_run (task_id, run_date)
WHERE trigger_type = 'cron';
```

GORM 中通过 `run_date` 字段 + `BeforeCreate` 钩子自动设置：
```go
type InspectionRun struct {
    // ... 其他字段
    RunDate string `gorm:"column:run_date;type:date;comment:'运行日期'" json:"-"`
}

func (r *InspectionRun) BeforeCreate(tx *gorm.DB) error {
    if r.RunDate == "" {
        r.RunDate = time.Now().Format("2006-01-02")
    }
    return nil
}
```

幂等插入策略：
```go
// 尝试直接 INSERT，依赖唯一约束冲突作为幂等点
err := db.Create(run).Error
if errors.Is(err, gorm.ErrDuplicatedKey) {
    return fmt.Errorf("task %d already running today", taskID)
}
```

---

## 3. API 接口设计

### 3.1 路由注册

在 `router/inspection/inspection.go` 中注册，在 `router/router.go` 的 JWT 组中加入：

```go
inspection.RegisterInspectionRoutes(jwtGroup)
```

### 3.2 接口清单

| Method | Path | RBAC | 说明 |
|--------|------|------|------|
| GET | `/inspection/tasks` | — | 任务列表 |
| GET | `/inspection/tasks/:id` | — | 任务详情（密钥脱敏） |
| PUT | `/inspection/tasks/:id` | `inspection:task:edit` | 更新任务配置（空字符串保留原密钥） |
| POST | `/inspection/tasks/:id/trigger` | `inspection:task:trigger` | 手动触发巡检 |
| GET | `/inspection/runs` | — | 运行历史列表（支持筛选/分页） |
| GET | `/inspection/runs/:id` | — | 运行详情（含汇总统计） |
| GET | `/inspection/runs/:id/results` | — | 主机级结果列表（支持筛选/分页） |
| GET | `/inspection/runs/:id/alerts` | — | 异常明细列表（支持筛选/分页） |
| GET | `/inspection/runs/:id/report` | — | 下载 Excel 报告（JWT + 审计） |
| GET | `/inspection/overview` | — | 巡检概览（今日统计、最近异常） |

### 3.3 Query 参数

**GET /inspection/runs**

| 参数 | 类型 | 说明 |
|------|------|------|
| `taskId` | int | 按任务筛选 |
| `n9eGroupId` | int | 按业务组筛选 |
| `status` | string | 状态筛选：`pending/running/success/partial/failed` |
| `triggerType` | string | `cron/manual` |
| `dateFrom` | string | 日期范围起 `2006-01-02` |
| `dateTo` | string | 日期范围止 `2006-01-02` |
| `page` | int | 页码，默认 1 |
| `pageSize` | int | 每页条数，默认 20，最大 100 |

**GET /inspection/runs/:id/results**

| 参数 | 类型 | 说明 |
|------|------|------|
| `status` | string | `normal/warning/critical/failed` |
| `hostname` | string | 主机名模糊匹配 |
| `page` | int | 页码，默认 1 |
| `pageSize` | int | 每页条数，默认 20 |

**GET /inspection/runs/:id/alerts**

| 参数 | 类型 | 说明 |
|------|------|------|
| `level` | string | `warning/critical` |
| `hostname` | string | 主机名模糊匹配 |
| `page` | int | 页码，默认 1 |
| `pageSize` | int | 每页条数，默认 20 |

### 3.3 关键接口详情

**PUT /inspection/tasks/:id** — 更新任务配置

```json
{
  "enabled": true,
  "cron": "0 10 * * *",
  "notifyWebhookUrl": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
  "notifySecret": "SECxxx",
  "notifyOnWarning": true,
  "notifyOnCritical": true,
  "notifyOnFailure": true
}
```

**GET /inspection/runs/:id** — 运行详情

```json
{
  "id": 42,
  "taskId": 3,
  "triggerType": "cron",
  "status": "success",
  "startedAt": "2026-05-19T10:00:00+08:00",
  "endedAt": "2026-05-19T10:02:30+08:00",
  "durationMs": 150000,
  "summary": {
    "totalHosts": 15,
    "normalHosts": 12,
    "warningHosts": 2,
    "criticalHosts": 0,
    "failedHosts": 1,
    "totalAlerts": 3
  },
  "report": {
    "format": "excel",
    "status": "success",
    "downloadUrl": "/api/v1/inspection/runs/42/report"
  }
}
```

**GET /inspection/runs/:id/results** — 主机结果列表

```json
{
  "list": [
    {
      "hostname": "prod-web-01",
      "ident": "prod-web-01",
      "ip": "10.0.1.101",
      "os": "CentOS 7.9",
      "status": "warning",
      "metrics": {
        "cpu_usage": {"value": 75.2, "status": "warning", "formatted": "75.2%"},
        "memory_usage": {"value": 45.0, "status": "normal", "formatted": "45.0%"},
        "disk_usage_max": {"value": 82.1, "status": "critical", "formatted": "82.1%"}
      },
      "alerts": [
        {"metricName": "cpu_usage", "level": "warning", "message": "CPU 利用率 警告: 75.2% (阈值: 70.0%)"},
        {"metricName": "disk_usage_max", "level": "critical", "message": "磁盘利用率 严重: 82.1% (阈值: 90.0%)"}
      ]
    }
  ],
  "total": 15
}
```

---

## 4. 核心流程

### 4.1 巡检引擎流程

```
+-----------+     +------------------+     +------------------+
| Inspector | --> | Collector        | --> | Evaluator        |
| Run(ctx)  |     | CollectAll(ctx)  |     | EvaluateAll()    |
+-----------+     +------------------+     +------------------+
     |                   |                        |
     v                   v                        v
  创建 run     1. 从 N9E 获取主机列表      评估阈值生成告警
  记录状态     2. 并发查询 VM 指标         汇总统计
  执行采集     3. 标记采集失败主机
  评估阈值
  生成报告
  发送通知
  更新状态
```

### 4.2 定时调度

引入 `github.com/robfig/cron/v3`，显式指定时区：

```go
loc, _ := time.LoadLocation("Asia/Shanghai")
scheduler := cron.New(cron.WithLocation(loc))
```

**调度策略**：
- 每个启用的 `inspection_task` 注册一个 cron job
- Cron 表达式存储在任务配置中，默认 `"CRON_TZ=Asia/Shanghai 0 10 * * *"`（北京时间 10 点）
- 服务启动时加载所有启用任务并注册调度
- 任务配置变更时重新注册

**N9E 同步后自动创建任务**：
- N9E 全量同步完成后，遍历 `n9e_busi_group` 表
- 对不存在对应 `inspection_task` 的业务组，自动创建默认任务（enabled=false，待人工启用）
- 业务组删除时：对应任务标记为 `enabled=false`（不删除，保留历史记录）
- 业务组改名时：`n9e_group_name` 快照不自动更新，但提供手动刷新接口

### 4.3 并发控制

```
全局信号量: maxConcurrent = 5

定时触发时:
  for each task:
    获取信号量 (阻塞等待)
    go runInspection(task) {
      defer 释放信号量
      // 执行巡检
    }
```

**跳过重入**：同一任务若已有运行中状态，新触发直接跳过并记录日志。

**超时控制**：
- 单次巡检全局超时：10 分钟
- 单个指标查询超时：继承 N9E 客户端配置（默认 30s）

### 4.4 防重复运行

**自动任务幂等**：依赖 PostgreSQL partial unique index `idx_run_unique_daily(task_id, run_date) WHERE trigger_type='cron'`。

```go
// 幂等插入：尝试直接 INSERT，唯一约束冲突即表示今日已运行
run := &InspectionRun{
    TaskID:      taskID,
    TriggerType: "cron",
    Status:      "running",
    RunDate:     time.Now().Format("2006-01-02"),
}
if err := db.Create(run).Error; errors.Is(err, gorm.ErrDuplicatedKey) {
    return fmt.Errorf("task %d already running today: %w", taskID, err)
}
```

**手动触发**：每次独立运行，不限制重复。

**运行中状态保护**：`status='running'` 的 run 在更新为终态前，不允许新的 cron 触发（已由唯一约束保证）。

---

## 5. 报告生成

### 5.1 流程

1. 巡检完成后，调用 `report.GenerateExcel(runID, result)`
2. 生成文件到 `/data/inspection/{run_id}/report.xlsx`
3. 更新 `inspection_report_artifact` 记录
4. 下载时通过 `run_id` 查找文件路径，走 JWT 鉴权后流式返回

### 5.2 文件名规则

修复 CLI 的多业务组冲突问题：

```
{output_dir}/{run_id}/inspection_report_{run_id}_{date}.xlsx
```

### 5.3 存储位置

- **独立目录**：`/data/inspection/`（Docker 挂载独立 named volume，不走 `/api/v1/upload` 静态目录）
- 报告下载必须通过 `GET /inspection/runs/:id/report` 接口，JWT 鉴权 + 审计后流式返回
- 文件路径校验：确保 `file_path` 在 `/data/inspection/` 根目录内，防止路径遍历

---

## 6. 通知机制

### 6.1 通知触发条件

| 场景 | 条件 | 默认行为 |
|------|------|---------|
| Critical 异常 | `critical_hosts > 0` | 必通知 |
| Warning 异常 | `warning_hosts > 0 && notify_on_warning` | 默认通知 |
| 采集失败 | `failed_hosts > 0 && notify_on_failure` | 默认通知 |
| 无可用主机 | `total_hosts == 0` | 默认通知 |
| 巡检失败 | `status == 'failed'` | 默认通知 |

### 6.2 钉钉消息格式

复用 `deploy/service/notifier.go` 的 `dingtalkbot` 客户端。

```markdown
### 巡检报告 — 生产环境业务组

- **任务**: 生产环境 (ID: 3)
- **触发**: 定时调度 (2026-05-19 10:00)
- **耗时**: 2分30秒
- **主机**: 15 台 (正常 12, 警告 2, 严重 0, 失败 1)
- **异常**: 3 条

**异常汇总**:
- prod-web-01: CPU 利用率 警告 (75.2%)
- prod-web-01: 磁盘利用率 严重 (82.1%)
- prod-db-02: 内存利用率 警告 (78.5%)

[查看详情](https://autoops.example.com/inspection/runs/42)
[下载报告](https://autoops.example.com/api/v1/inspection/runs/42/report)
```

### 6.3 抑制策略

- **同一任务连续异常不抑制**：每次异常都通知，避免漏报
- **通知失败不重试**：记录失败日志，不阻塞主流程

---

## 7. 清理策略

### 7.1 定时清理任务

每天凌晨 3 点执行：

```sql
-- 1. 删除 30 天前的运行记录（级联删除 target_result, alert, report_artifact）
DELETE FROM inspection_run WHERE created_at < NOW() - INTERVAL '30 days';

-- 2. 删除 30 天前的报告文件
-- 扫描 inspection_report_artifact，删除文件系统中对应路径
```

### 7.2 级联删除

GORM 配置 `ON DELETE CASCADE`：
- `inspection_run` 删除 → 级联删除 `inspection_target_result`, `inspection_alert`, `inspection_report_artifact`

---

## 8. 错误码分配

| 错误码 | 含义 |
|--------|------|
| 471 | INSPECTION_TASK_NOT_FOUND |
| 472 | INSPECTION_TASK_UPDATE_FAILED |
| 473 | INSPECTION_RUN_NOT_FOUND |
| 474 | INSPECTION_TRIGGER_FAILED |
| 475 | INSPECTION_REPORT_NOT_FOUND |
| 476 | INSPECTION_REPORT_GENERATE_FAILED |
| 477 | INSPECTION_ALREADY_RUNNING |

---

## 9. 前端页面设计

### 9.1 页面清单

| 页面 | 路径 | 功能 |
|------|------|------|
| 巡检概览 | `/inspection/overview` | 今日统计、最近异常、快速入口 |
| 任务管理 | `/inspection/tasks` | 任务列表、启用/禁用、配置通知 |
| 运行历史 | `/inspection/runs` | 运行记录列表、状态筛选 |
| 运行详情 | `/inspection/runs/:id` | 主机结果、异常明细、报告下载 |

### 9.2 关键交互

- **任务列表**：显示业务组名称、上次运行时间、状态、启用开关
- **手动触发**：按钮点击后显示执行中状态，轮询直到完成
- **运行详情**：Tab 切换「概览 / 主机列表 / 异常列表」
- **报告下载**：点击下载按钮，走 JWT 鉴权流式下载 Excel

---

## 10. 实现顺序

### Phase 0: Engine Contract Spike（必须先做）
**目标**：用一个业务组跑通 Host 巡检，确认 VM/N9E 过滤、指标解析、超时和错误语义。

1. 在 `service/engine/` 下移植 inspection-tool 的 Collector + Evaluator + Inspector
2. 实现 `vmclient.QueryInstantWithFilter()`（instant query + label matcher + ident 分组）
3. 对接 AutoOps N9E 客户端获取主机列表，按业务组名称过滤
4. 跑通一次完整巡检，产出结构化 `InspectionResult` 和 Excel 报告
5. **验收标准**：能对一个业务组执行巡检，Excel 报告格式正确，无 DB 依赖

### Phase 1: 基础框架
1. 创建 `api/api/inspection/model/` 数据模型（含 `inspection_notification`）
2. 注册模型到 `pkg/db/migrate.go`
3. 创建 `router/inspection/` 路由骨架
4. 实现 `inspection_task` CRUD API（密钥脱敏、保留原值逻辑）
5. 实现 N9E 同步后自动创建/停用任务
6. 配置 `run_date` + PostgreSQL partial unique index

### Phase 2: 巡检引擎固化
1. 将 Phase 0 的 engine 集成到 service 层
2. 实现运行记录持久化（run/target_result/alert）
3. 实现并发控制（信号量 + 跳过重入）

### Phase 3: 调度与执行
1. 实现定时调度器（cron，Asia/Shanghai 时区）
2. 实现手动触发 API
3. 实现幂等插入防重复

### Phase 4: 报告与通知
1. 适配 Excel 报告生成（修复文件名冲突）
2. 实现报告下载 API（独立目录 + JWT + 审计）
3. 实现钉钉通知（含 `inspection_notification` 落库）

### Phase 5: 前端与收尾
1. 前端页面开发
2. 清理任务实现（30 天级联删除）
3. 集成测试

---

## 11. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| inspection-tool 核心代码耦合 Cobra/Viper | 抽取困难 | 只抽取 `internal/service/` 和 `internal/model/` 的纯逻辑，移除 CLI 依赖 |
| VM 客户端不能简单复用现有 QueryPromQL | 高 | 新增 `QueryInstantWithFilter` 接口，移植 inspection-tool VM client 核心（instant query + label matcher + ident 分组） |
| VM 查询并发压垮 VictoriaMetrics | 高 | 信号量限制并发（5），单指标查询有超时和重试 |
| 72 个业务组同时启动 | 高 | 全局信号量 + 单任务跳过重入 |
| 钉钉密钥泄露 | 高 | GET 接口脱敏返回 webhook_url（token 部分掩码），secret 不返回；PUT 空字符串保留原值 |
| Excel 报告文件冲突 | 中 | 文件名包含 `run_id`，按 `run_id` 分目录存储 |
| 大量历史数据膨胀 | 中 | 30 天自动清理 + 级联删除 |
| N9E 业务组变更导致任务失效 | 低 | 任务按 `n9e_group_id` 关联，ID 不变则任务有效；同步后自动创建/停用任务 |

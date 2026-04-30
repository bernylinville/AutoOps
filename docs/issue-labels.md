# AutoOps Issue 标签体系

基于 Warp 项目标签策略，为 AutoOps 建立标准化标签体系。

## 状态流标签

| 标签 | 颜色 | 含义 |
|---|---|---|
| `needs-info` | `fbca04` | 需要更多信息才能处理 |
| `triaged` | `5319e7` | 已初步分类 |
| `ready-to-spec` | `1d76db` | 问题已理解，需要写 spec |
| `ready-to-implement` | `0e8a16` | 设计已定，可直接实现 |

## 类型标签

| 标签 | 颜色 | 含义 |
|---|---|---|
| `bug` | `d73a4a` | Bug |
| `enhancement` | `a2eeef` | 功能增强 |
| `documentation` | `0075ca` | 文档 |

## 领域标签 (area:)

| 标签 | 颜色 | 负责人 | 说明 |
|---|---|---|---|
| `area:cmdb` | `0052cc` | 待定 | CMDB 资产模块 |
| `area:deploy` | `0e8a16` | 待定 | 发布部署模块 |
| `area:monitor` | `e36209` | 待定 | 监控告警 (N9E/FlashDuty) |
| `area:task` | `1d76db` | 待定 | 任务自动化 |
| `area:auth` | `5319e7` | 待定 | 认证授权 |
| `area:frontend` | `7dc4e4` | 待定 | 前端通用 |
| `area:infra` | `6e7681` | 待定 | 基础设施 / Docker / K8s |
| `area:agent` | `c6e2d6` | 待定 | AI Agent / 钉钉机器人 |

## 复现度标签

| 标签 | 颜色 | 含义 |
|---|---|---|
| `repro:high` | `b60205` | 高复现度 |
| `repro:medium` | `fbca04` | 中等复现度 |
| `repro:low` | `d4c5f9` | 低复现度 |

## 创建命令

在 GitHub 仓库 Settings > Labels 中创建，或使用 GitHub CLI：

```bash
# 状态流
gh label create "needs-info" --color "fbca04" --description "需要更多信息"
gh label create "triaged" --color "5319e7" --description "已初步分类"
gh label create "ready-to-spec" --color "1d76db" --description "需要写 spec"
gh label create "ready-to-implement" --color "0e8a16" --description "可直接实现"

# 类型
gh label create "bug" --color "d73a4a" --description "Bug"
gh label create "enhancement" --color "a2eeef" --description "功能增强"

# 领域
gh label create "area:cmdb" --color "0052cc" --description "CMDB 模块"
gh label create "area:deploy" --color "0e8a16" --description "发布部署模块"
gh label create "area:monitor" --color "e36209" --description "监控告警"
gh label create "area:task" --color "1d76db" --description "任务自动化"
gh label create "area:auth" --color "5319e7" --description "认证授权"
gh label create "area:frontend" --color "7dc4e4" --description "前端"
gh label create "area:infra" --color "6e7681" --description "基础设施"
gh label create "area:agent" --color "c6e2d6" --description "AI Agent"
```

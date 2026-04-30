# 运维知识库 (Knowledge Base) — Tech Spec

## Context

### 当前系统
- PostgreSQL 17.4 已部署，支持全文搜索和 GIN 索引
- 前端使用 Vue 3 + Element Plus，已有 Markdown 渲染组件
- RBAC 权限系统已就绪（200+ 权限码）
- 文件上传系统已存在（如需要扩展）

### 相关文件
- `api/pkg/db/migrate.go` — 模型注册入口
- `api/middleware/rbac.go` — RBAC 中间件
- `web/src/components/` — 前端组件目录
- `api/api/system/model/` — 参考现有模型定义风格

## Proposed Changes

### 1. 数据库模型

```go
// api/api/knowledge/model/knowledgeDoc.go
package model

import "gorm.io/gorm"

type KnowledgeDoc struct {
    gorm.Model
    Title       string `json:"title" gorm:"size:128;not null;index"`
    Content     string `json:"content" gorm:"type:text;not null"`
    CategoryID  uint   `json:"category_id" gorm:"not null;index"`
    AuthorID    uint   `json:"author_id" gorm:"not null;index"`
    Tags        string `json:"tags" gorm:"size:256"` // JSON array
    Version     int    `json:"version" gorm:"default:1"` // 乐观锁
    SearchVector string `json:"-" gorm:"type:tsvector"` // PG 全文搜索
}

type KnowledgeCategory struct {
    gorm.Model
    Name        string `json:"name" gorm:"size:64;not null;unique"`
    Code        string `json:"code" gorm:"size:64;not null;unique"`
    IsBuiltin   bool   `json:"is_builtin" gorm:"default:false"`
    SortOrder   int    `json:"sort_order" gorm:"default:0"`
}
```

**Migration:** 注册到 `api/pkg/db/migrate.go`

**GIN 索引:**
```sql
CREATE INDEX idx_knowledge_docs_search ON knowledge_docs USING GIN(search_vector);
```

### 2. 全文搜索实现

使用 PostgreSQL `tsvector` + `tsquery`：

```go
// 创建/更新时生成 tsvector
func (d *KnowledgeDoc) BeforeSave(tx *gorm.DB) error {
    d.SearchVector = gorm.Expr("to_tsvector('chinese', ? || ' ' || ?)", d.Title, d.Content)
    return nil
}

// 搜索查询
func SearchDocs(keyword string) ([]KnowledgeDoc, error) {
    var docs []KnowledgeDoc
    err := db.Where("search_vector @@ plainto_tsquery('chinese', ?)", keyword).
        Order("ts_rank(search_vector, plainto_tsquery('chinese', ?)) DESC", keyword).
        Find(&docs).Error
    return docs, err
}
```

**注意：** PostgreSQL 默认不支持中文分词，需要额外安装 `zhparser` 或使用 `pg_bigm`。
备选方案：使用 `to_tsvector('simple', ...)` + `LIKE` 辅助（牺牲部分精度）。

### 3. API 设计

```
GET    /api/v1/knowledge/docs              # 列表（分页+搜索+分类筛选）
POST   /api/v1/knowledge/docs              # 创建
GET    /api/v1/knowledge/docs/:id          # 详情
PUT    /api/v1/knowledge/docs/:id          # 更新（乐观锁校验）
DELETE /api/v1/knowledge/docs/:id          # 软删除
GET    /api/v1/knowledge/categories        # 分类列表
POST   /api/v1/knowledge/categories        # 创建分类（管理员）
```

### 4. 前端页面

```
web/src/views/Knowledge/
├── KnowledgeList.vue      # 文档列表+搜索
├── KnowledgeDetail.vue    # 文档详情+渲染
├── KnowledgeEdit.vue      # 创建/编辑
└── components/
    ├── KnowledgeSearch.vue    # 搜索框+结果
    ├── KnowledgeSidebar.vue   # 分类树
    └── MarkdownPreview.vue    # Markdown 渲染（复用现有或新建）
```

### 5. 权限设计

| 操作 | 权限码 | 说明 |
|------|--------|------|
| 查看文档 | `knowledge:doc:view` | 所有登录用户 |
| 创建文档 | `knowledge:doc:create` | 运维人员及以上 |
| 编辑文档 | `knowledge:doc:edit` | 创建者或管理员 |
| 删除文档 | `knowledge:doc:delete` | 创建者或管理员 |
| 管理分类 | `knowledge:category:manage` | 管理员 |

## Testing and Validation

### 功能测试
- [ ] 创建文档并验证 PG 全文搜索索引生成
- [ ] 搜索关键词验证结果排序和高亮
- [ ] 分类筛选与搜索组合使用
- [ ] 编辑时乐观锁冲突处理
- [ ] 软删除后管理员恢复

### 性能测试
- [ ] 1000 篇文档全文搜索 < 500ms
- [ ] 列表页分页加载 < 200ms

### 集成测试
- [ ] 端到端：创建 → 搜索 → 编辑 → 删除流程

## Risks and Mitigations

| 风险 | 影响 | 缓解措施 |
|------|------|---------|
| 中文分词不支持 | 搜索精度下降 | 使用 `simple` 配置 + `pg_bigm` 扩展 |
| Markdown 渲染 XSS | 安全漏洞 | 使用 `marked.js` + DOMPurify 前端净化 |
| 大量文档性能下降 | 搜索变慢 | 添加分页限制 + 索引优化 |

## Follow-ups

- 文档版本历史（Phase 2）
- 文档导出 PDF/Markdown
- 热门文档统计和推荐算法优化

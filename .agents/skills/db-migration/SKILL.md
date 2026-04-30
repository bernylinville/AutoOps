---
name: db-migration
description: Add or modify a GORM model and register it for automatic migration. Use when creating new tables, adding columns, or changing database schema.
---

# db-migration

Add or modify a GORM model and ensure it is registered for automatic PostgreSQL migration.

## When to Use

- Creating a new database table
- Adding/modifying/removing columns on existing table
- Adding indexes or constraints
- Changing model relationships

## Prerequisites

1. PostgreSQL 17 is running (via Docker Compose or local)
2. You know the module where the model belongs (e.g., `cmdb`, `deploy`, `app`)
3. You understand whether this is a new model or modification to existing model

## Workflow

### 1. Create or Update the Model

New model goes to: `api/api/{module}/model/{ModelName}.go`

```go
package model

import "gorm.io/gorm"

type YourModel struct {
    gorm.Model
    Name        string `json:"name" gorm:"size:128;not null;comment:名称"`
    Description string `json:"description" gorm:"size:512;comment:描述"`
    Status      int    `json:"status" gorm:"default:1;comment:状态 1-启用 2-禁用"`
    // Add more fields as needed
}

// TableName overrides default table name
tableName := "your_models"
```

**Model Conventions:**
- Use `gorm.Model` for standard id/created_at/updated_at/deleted_at
- Add `json` tags for all exported fields
- Add `gorm:"comment:..."` for documentation
- Use `size:N` for strings (PostgreSQL performance)
- Use `default:X` for non-null fields with defaults

### 2. Register Migration

**CRITICAL**: All new models MUST be registered in `api/pkg/db/migrate.go`.

Edit `api/pkg/db/migrate.go`:

```go
func Migrate(db *gorm.DB) error {
    return db.AutoMigrate(
        // ... existing models ...
        &cmdbModel.YourModel{},  // <-- ADD HERE
    )
}
```

### 3. Add DTOs and VOs (if API-facing)

In the same model file or `api/api/{module}/dto/`:

```go
// CreateRequest for POST
type YourModelCreateRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
}

// UpdateRequest for PUT
type YourModelUpdateRequest struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Status      int    `json:"status"`
}

// Response for GET
type YourModelResponse struct {
    ID          uint      `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Status      int       `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
}
```

### 4. Verify Migration Locally

```bash
cd api

# Start PostgreSQL if not running
docker compose up -d postgres

# Run application (will auto-migrate on startup)
go run main.go -c config.yaml

# Verify table exists
psql -h localhost -U devops -d autoops -c "\dt"
psql -h localhost -U devops -d autoops -c "\d your_models"
```

### 5. Write Tests

```go
// api/api/{module}/service/your_model_test.go
func TestYourModel_Create(t *testing.T) {
    // Test creation logic
}

func TestYourModel_Validation(t *testing.T) {
    // Test validation rules
}
```

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Table not created | Model not registered in migrate.go | Add to `api/pkg/db/migrate.go` |
| Column type wrong | Missing gorm tag | Add `gorm:"type:varchar(128)"` etc. |
| Migration panics | Existing data incompatible | Use GORM migrator for complex changes |
| Index not created | Missing index tag | Add `gorm:"index"` or `gorm:"uniqueIndex"` |

## Verification Checklist

- [ ] Model file created in correct module directory
- [ ] All fields have `json` and `gorm` tags
- [ ] Model registered in `api/pkg/db/migrate.go`
- [ ] DTOs/VOs created for API operations (if needed)
- [ ] Local migration verified (table exists with correct schema)
- [ ] Tests written and passing
- [ ] No sensitive data in model fields (use encryption for secrets)

## Related

- `deploy-flow` — If deploying app with new migrations to production
- `cmdb-change` — If modifying CMDB-related models

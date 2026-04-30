---
name: cmdb-change
description: Modify CMDB assets, CI types, or CI instances. Use when adding new asset types, changing CI models, or updating CMDB business logic.
---

# cmdb-change

Modify CMDB (Configuration Management Database) assets, CI types, or CI instances.

## When to Use

- Adding a new CI type (asset category)
- Modifying CI type attributes (add/remove/change fields)
- Updating CI instance business logic
- Changing asset lifecycle states or transitions
- Modifying CI relationship topology

## Prerequisites

1. Understand CI model architecture:
   - `CiType` — Dynamic CI type definition (name, attributes as JSONB)
   - `CiInstance` — Actual asset instance with dynamic attributes
   - `CiTypeAttribute` — Attribute schema for a CI type
2. Know whether change affects type definition or instance logic
3. Consider data migration for existing instances

## Workflow

### 1. Identify Change Scope

**Type A: New CI Type**
→ Add to `api/api/cmdb/model/ciType.go`

**Type B: Modify Existing CI Type Attributes**
→ Update `CiTypeAttribute` definitions
→ Consider migration for existing instances

**Type C: Instance Logic Change**
→ Modify service layer in `api/api/cmdb/service/`

**Type D: Lifecycle/State Change**
→ Update state machine in `api/api/cmdb/service/lifecycle.go`

### 2. Implement Type Definition Changes

For new CI types or attribute changes:

```go
// api/api/cmdb/model/ciType.go

// Built-in template (optional)
var NewCiTypeTemplate = CiType{
    Name:        "负载均衡",
    Code:        "loadbalancer",
    Description: "负载均衡设备",
    Icon:        "el-icon-s-data",
    Attributes: []CiTypeAttribute{
        {
            Name:     "品牌",
            Code:     "brand",
            Type:     "select",
            Required: true,
            Options:  []string{"F5", "A10", "Citrix", "自研"},
        },
        {
            Name:     "VIP数量",
            Code:     "vip_count",
            Type:     "number",
            Required: false,
        },
    },
}
```

**Attribute Types:**
- `text` — Single line text
- `textarea` — Multi-line text
- `number` — Integer
- `select` — Dropdown (requires Options)
- `multiselect` — Multi-select
- `date` — Date picker
- `datetime` — DateTime picker
- `switch` — Boolean toggle
- `password` — Encrypted storage

### 3. Update Instance Logic

If changing instance CRUD or validation:

```go
// api/api/cmdb/service/ciInstance.go

func (s *CiInstanceService) Create(req *model.CiInstanceCreateRequest) error {
    // 1. Validate CI type exists
    // 2. Validate required attributes
    // 3. Validate attribute types match schema
    // 4. Create instance with JSONB attributes
    // 5. Record audit log
}
```

### 4. Handle Data Migration

If modifying existing type attributes:

```sql
-- Example: Add default value for new required attribute
UPDATE ci_instances
SET attributes = attributes || '{"new_field": "default_value"}'::jsonb
WHERE ci_type_id = 'xxx' AND attributes->>'new_field' IS NULL;
```

Or use GORM migration in `api/pkg/db/migrate.go`.

### 5. Update Frontend Forms

If CI type has frontend form rendering:

```javascript
// web/src/views/cmdb/components/CiTypeForm.vue
// Dynamic form already renders based on attribute schema
// Usually no changes needed if using dynamic JSONB forms
```

### 6. Verify Changes

```bash
cd api
go test ./api/cmdb/... -v

# Test specific scenarios:
# 1. Create CI type with new attributes
# 2. Create instance with all required fields
# 3. Update instance (partial update)
# 4. Validate attribute type constraints
# 5. Check audit log records
```

## Common Issues

| Issue | Cause | Fix |
|-------|-------|-----|
| Form not rendering | Missing attribute type mapping | Add type to frontend form renderer |
| Validation fails | Attribute type mismatch | Check `Type` field matches input data |
| Instance search broken | Missing GIN index on JSONB | Ensure `attributes` column has GIN index |
| Audit log missing | Forgot to call audit function | Add `auditService.Record()` calls |

## Verification Checklist

- [ ] CI type definition updated (model file)
- [ ] New attributes have correct types and validation
- [ ] Instance CRUD logic handles changes
- [ ] Data migration script written (if needed)
- [ ] Frontend dynamic form renders correctly
- [ ] Tests cover new attributes and validation
- [ ] Audit log records changes
- [ ] API documentation updated (Swagger comments)

## Related

- `db-migration` — If adding new tables for CMDB
- `deploy-flow` — If deploying CMDB changes

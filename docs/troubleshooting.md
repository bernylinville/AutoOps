# Troubleshooting

开发或部署遇到问题时阅读。

## Docker 环境

### 网络子网冲突

**症状**：`docker compose up` 报错 `Pool overlaps with other one on this address space`。

**原因**：`docker-compose.yml` 中 subnet（如 `172.20.0.0/16`）被其他 Docker 网络占用。

**排查**：

```bash
docker network inspect $(docker network ls -q) 2>/dev/null | \
  python3 -c "import sys,json; [print(f'{n[\"Name\"]}: {n[\"IPAM\"][\"Config\"][0][\"Subnet\"]}') for n in json.load(sys.stdin) if n.get('IPAM',{}).get('Config')]"
```

**修复**：修改 `docker-compose.yml` 中 subnet 为未占用段（如 `172.22.0.0/16`）。

### API 健康检查 unhealthy

**症状**：`docker compose ps` 显示 devops-api 状态为 unhealthy，但 `curl localhost:18000/api/v1/healthz` 返回正常。

**原因**：`wget --spider` 发送 HEAD 请求，Gin 路由未注册 HEAD 方法。

**修复**：healthcheck 改用 `curl -sf`：

```yaml
healthcheck:
  test: ["CMD", "curl", "-sf", "http://127.0.0.1:8000/api/v1/healthz"]
```

### 镜像构建超时

首次 `docker compose up --build` 需下载 Go 依赖和 npm 包，通常需要 5–10 分钟。构建层已缓存，后续构建较快。

## 数据库

### seed_data.sql 导入 cmdb_group 报错

**症状**：`column "remark" of relation "cmdb_group" does not exist`。

**原因**：`cmdb_group` 表已移除 `remark` 和 `update_time` 字段，但 `seed_data.sql` 未更新。

**修复**：其他表数据已正常导入。`cmdb_group` 需单独用精简语句导入：

```sql
INSERT INTO cmdb_group (id, parent_id, name, create_time) VALUES
(1, 0, '默认业务组', '2025-07-10 11:02:07'),
...
ON CONFLICT (id) DO NOTHING;
```

### 生产环境 PostgreSQL 连接失败

**症状**：API Pod 日志报 `dial tcp: lookup autoops-postgres on ...` 或 `connection refused`。

**排查**：

```bash
kubectl -n autoops get svc autoops-postgres
kubectl -n autoops get pods -l app.kubernetes.io/component=postgres
kubectl -n autoops logs -l app.kubernetes.io/component=api --tail=50 | grep -i "db\|postgres"
```

**常见原因**：

- PostgreSQL Pod 未就绪（`pg_isready` probe 失败）
- `autoops-runtime` Secret 中 `POSTGRES_PASSWORD` 与实际密码不一致
- PVC 满：`kubectl -n autoops exec deploy/autoops-postgres -- df -h /var/lib/postgresql/data`

## 前端

### 登录后仍跳回 login 页

**症状**：输入正确账号密码，API 返回 200，但页面仍停留在 `/login`。

**原因**：前端 `storage.js` 使用 `process.env.VUE_APP_NAME_SPACE` 作为 localStorage key，但构建时该变量未注入，实际 key 为字符串 `"undefined"`。直接设置 `localStorage.setItem('devops-api', ...)` 无效。

**修复**：需在 `"undefined"` key 下存储数据，或修复 `.env` 文件位置（从 `src/.env` 移到项目根目录并改用 `KEY=value` 格式）。

### devServer proxy 连接失败

**症状**：`npm run serve` 后 API 请求返回 502 或 504。

**原因**：`vue.config.js` 中 proxy target 硬编码为 `http://192.168.1.156:5700`。

**修复**：修改为本地后端地址（如 `http://localhost:8000`）。

## K8s 生产环境

### ArgoCD Sync 失败

**症状**：ArgoCD UI 显示 autoops Application OutOfSync 或 SyncError。

**排查**：

```bash
kubectl -n argocd get app autoops
kubectl -n autoops get pods
kubectl -n autoops get pvc
kubectl -n autoops describe deploy autoops-api | tail -30
```

**常见原因**：

- 镜像 tag 不存在于镜像仓库
- PVC `autoops-gitops-working-tree` 未提前创建（生产使用 existingClaim）
- Secret `autoops-runtime` 未创建或 key 名不匹配

### Gateway 不可达

**症状**：访问 `http://10.0.17.206` 无响应或返回 502。

**排查**：

```bash
kubectl -n autoops get gateway,httproute
kubectl -n envoy-gateway-system get pods
kubectl -n autoops get svc autoops-web
```

**常见原因**：

- Envoy Gateway Pod 未就绪
- MetalLB 未分配 `10.0.17.206`
- `autoops-web` Service 无健康 Pod

## 监控集成

### N9E 集成问题

参考 [docs/n9e-integration-review.md](n9e-integration-review.md) 中的 F1–F5 修复记录：

- F1：SQL schema 不匹配
- F2：Token 脱敏覆盖
- F3：CmdbHostVo 缺少字段
- F4：并发同步无保护（已用 `sync.Mutex` 修复）
- F5：GetTargets 分页遍历

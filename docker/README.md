# DevOps 运维管理系统 Docker 部署

本目录提供了 DevOps 运维管理系统的 Docker Compose 部署方案，所有配置文件和数据已持久化到本地。

## 目录结构

```
docker/
├── docker-compose.yml          # Docker Compose 编排文件
├── .env                        # 环境变量配置
├── .env.example                # 环境变量模板
├── devops-start.sh             # 一键启动脚本
├── devops-stop.sh              # 一键停止脚本
├── import-dev-data.sh          # 导入 seed 数据脚本
├── api/                        # 后端配置
│   ├── config.yaml             # API 配置文件
│   ├── Dockerfile              # API 镜像构建文件
│   ├── templates/              # Excel 模板
│   └── ssh_keys/               # SSH 密钥
├── web/                        # 前端配置
│   ├── Dockerfile              # Web 镜像构建文件
│   └── devops.conf             # Nginx 配置文件
├── postgres/                   # PostgreSQL 初始化脚本
│   └── init/                   # 首次启动执行的 SQL
├── valkey/                     # Valkey 配置
│   └── valkey.conf
├── prometheus/                 # Prometheus 配置
│   └── prometheus.yml
└── pushgateway/                # Pushgateway 配置
```

## 服务列表

| 服务名 | 容器名 | 端口映射 | 说明 |
|--------|--------|----------|------|
| postgres | devops-postgres | 15432:5432 | PostgreSQL 17.4 数据库 |
| valkey | devops-valkey | 16379:6379 | Valkey 9.0 缓存 (Redis 协议兼容) |
| pushgateway | devops-pushgateway | 19091:9091 | Pushgateway 指标推送 |
| prometheus | devops-prometheus | 19090:9090 | Prometheus 监控 |
| devops-api | devops-api | 18000:8000 | DevOps API 后端 (本地构建) |
| devops-web | devops-web | 18088:80 | DevOps Web 前端 (本地构建) |

## 部署前准备

### 1. 配置文件准备

```bash
# 复制环境变量模板（首次部署）
cp .env.example .env
# 按需修改端口、密码

# 后端配置
cp api/config.yaml.example api/config.yaml
# 按需修改 db.password、redis.password 等
```

两个配置文件的密码必须一致：`.env` 的 `POSTGRES_PASSWORD` = `api/config.yaml` 的 `db.password`；`REDIS_PASSWORD` = `redis.password`。

### 2. 构建并启动所有服务

```bash
docker compose up -d --build
```

首次构建需要 5-10 分钟（下载依赖、编译 Go 和前端）。

### 3. 导入 seed 数据（首次部署）

```bash
./import-dev-data.sh --force
# 默认账号: admin / 123456
```

## 访问地址

- **Web 前端**: http://127.0.0.1:18088 (admin/123456)
- **API 后端**: http://127.0.0.1:18000
- **Prometheus**: http://127.0.0.1:19090
- **Pushgateway**: http://127.0.0.1:19091

## 服务管理

### 查看服务状态

```bash
docker compose ps
```

### 查看服务日志

```bash
docker compose logs -f                    # 所有服务
docker compose logs -f devops-api         # 指定服务
```

### 停止服务

```bash
docker compose stop
```

### 重启服务

```bash
docker compose restart
```

### 停止并删除容器

```bash
docker compose down
```

### 停止并删除容器及数据卷（危险操作）

```bash
docker compose down -v
```

## 日常更新（代码变更后）

```bash
# 重建 API 和 Web 镜像（基础服务无需重建）
docker compose down devops-api devops-web
docker compose up -d --build devops-api devops-web

# 等待健康检查通过，验证服务
curl -sf http://127.0.0.1:18000/api/v1/healthz
curl -sf -o /dev/null -w "%{http_code}\n" http://127.0.0.1:18088/
```

**注意**：基础服务（postgres、valkey、prometheus、pushgateway）镜像从 registry 拉取，通常不需要更新。只有 `devops-api` 和 `devops-web` 通过本地 `build:` 构建，代码变更后需要重建。

如果 `.env` 或 `api/config.yaml` 有改动，需要全量重启：

```bash
docker compose down && docker compose up -d --build
```

## 数据持久化

以下 Docker volume 已持久化，停止容器后数据不会丢失：

| Volume | 说明 |
|--------|------|
| autoops-postgres-data | PostgreSQL 数据 |
| autoops-valkey-data | Valkey 数据 |
| autoops-prometheus-data | Prometheus 数据 |
| autoops-pushgateway-data | Pushgateway 数据 |
| autoops-api-log | API 日志 |
| autoops-api-upload | 上传文件 |
| autoops-api-inspection | 巡检报告 |

## 网络配置

所有服务运行在独立的 Docker 网络 `devops-network` 中，使用子网 `172.20.0.0/16`。

服务之间通过容器名通信：
- API 连接 PostgreSQL: `postgres:5432`
- API 连接 Valkey: `valkey:6379`
- Prometheus 采集 Pushgateway: `pushgateway:9091`
- Web 代理 API: `devops-api:8000`

## 备份与恢复

### 备份数据库

```bash
docker compose exec -T postgres pg_dump -U devops autoops > backup_$(date +%Y%m%d).sql
```

### 恢复数据库

```bash
docker compose exec -T postgres psql -U devops autoops < backup.sql
```

## 故障排查

### 容器启动失败

```bash
docker compose logs <service-name>
docker compose ps
```

### 数据库连接失败

```bash
docker compose exec postgres pg_isready -U devops -d autoops
docker compose exec postgres psql -U devops -d autoops -c "SELECT 1"
```

### API 无法连接数据库

```bash
docker compose exec devops-api ping postgres
```

### 重置数据库

```bash
docker compose down
docker volume rm autoops-postgres-data
docker compose up -d --build   # 会自动初始化数据库
./import-dev-data.sh --force   # 重新导入 seed 数据
```

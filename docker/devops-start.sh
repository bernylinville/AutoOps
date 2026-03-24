#!/bin/bash

# DevOps 服务启动脚本 v4.0 (本地构建版)
# 用法: ./devops-start.sh <ip> <web_port> [api_port] [mysql_port] [redis_port]
# 示例: ./devops-start.sh 10.0.7.225 18088

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认值
API_PORT=${3:-18000}
MYSQL_PORT=${4:-13306}
REDIS_PORT=${5:-16379}
PROMETHEUS_PORT=19090
PUSHGATEWAY_PORT=19091

# 参数验证
if [ $# -lt 2 ]; then
    echo -e "${RED}错误: 参数不足${NC}"
    echo "用法: $0 <ip> <web_port> [api_port] [mysql_port] [redis_port]"
    echo ""
    echo "参数说明:"
    echo "  ip           - 服务器IP地址或域名 (例如: 10.0.7.225, localhost)"
    echo "  web_port     - 前端访问端口 (例如: 18088)"
    echo "  api_port     - API后端端口 (可选, 默认: 18000)"
    echo "  mysql_port   - MySQL端口 (可选, 默认: 13306)"
    echo "  redis_port   - Redis端口 (可选, 默认: 16379)"
    echo ""
    echo "示例:"
    echo "  $0 10.0.7.225 18088"
    echo "  $0 10.0.7.225 18088 18000 13306 16379"
    exit 1
fi

SERVER_IP=$1
WEB_PORT=$2

# 验证IP地址格式
if ! [[ "$SERVER_IP" =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]] && ! [[ "$SERVER_IP" =~ ^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$ ]]; then
    if [ "$SERVER_IP" != "localhost" ]; then
        echo -e "${RED}错误: IP地址或域名格式不正确${NC}"
        exit 1
    fi
fi

# 验证端口号
for port in $WEB_PORT $API_PORT $MYSQL_PORT $REDIS_PORT; do
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo -e "${RED}错误: 端口号无效: $port${NC}"
        exit 1
    fi
done

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}DevOps 服务启动脚本 v4.0 (本地构建版)${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "配置信息:"
echo "  服务器IP:     $SERVER_IP"
echo "  前端端口:     $WEB_PORT"
echo "  API端口:      $API_PORT"
echo "  MySQL端口:    $MYSQL_PORT"
echo "  Redis端口:    $REDIS_PORT"
echo "  Prometheus:   $PROMETHEUS_PORT"
echo "  Pushgateway:  $PUSHGATEWAY_PORT"
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检测 docker compose 命令
detect_docker_compose() {
    if docker compose version &> /dev/null 2>&1; then
        echo "docker compose"
    elif command -v docker-compose &> /dev/null; then
        echo "docker-compose"
    else
        echo ""
    fi
}

DOCKER_COMPOSE_CMD=$(detect_docker_compose)
if [ -z "$DOCKER_COMPOSE_CMD" ]; then
    echo -e "${RED}错误: 找不到 docker compose 或 docker-compose 命令${NC}"
    exit 1
fi

# 检查必要的文件
if [ ! -f "docker-compose.yml" ]; then
    echo -e "${RED}错误: 找不到 docker-compose.yml${NC}"
    exit 1
fi

if [ ! -f ".env" ]; then
    echo -e "${RED}错误: 找不到 .env 文件${NC}"
    exit 1
fi

if [ ! -f "api/config.yaml" ]; then
    echo -e "${RED}错误: 找不到 api/config.yaml${NC}"
    exit 1
fi

echo -e "${YELLOW}步骤 1: 更新 .env 文件...${NC}"
sed -i.bak "s|^WEB_PORT=.*|WEB_PORT=$WEB_PORT|" .env
sed -i.bak "s|^API_PORT=.*|API_PORT=$API_PORT|" .env
sed -i.bak "s|^MYSQL_PORT=.*|MYSQL_PORT=$MYSQL_PORT|" .env
sed -i.bak "s|^REDIS_PORT=.*|REDIS_PORT=$REDIS_PORT|" .env
sed -i.bak "s|^PROMETHEUS_PORT=.*|PROMETHEUS_PORT=$PROMETHEUS_PORT|" .env
sed -i.bak "s|^PUSHGATEWAY_PORT=.*|PUSHGATEWAY_PORT=$PUSHGATEWAY_PORT|" .env
sed -i.bak "s|^IMAGE_HOST=.*|IMAGE_HOST=http://$SERVER_IP:$WEB_PORT|" .env
rm -f .env.bak
echo -e "${GREEN}✓ .env 文件已更新${NC}"

echo -e "${YELLOW}步骤 2: 更新 api/config.yaml 文件...${NC}"
# 使用通用正则替换，支持重复运行（幂等）
sed -i.bak "s|url: \"http://[^\"]*:9090\"|url: \"http://$SERVER_IP:$PROMETHEUS_PORT\"|" api/config.yaml
sed -i.bak "s|url: \"http://[^\"]*:9091\"|url: \"http://$SERVER_IP:$PUSHGATEWAY_PORT\"|" api/config.yaml
sed -i.bak "s|heartbeat_server_url: \"http://[^\"]*:8000|heartbeat_server_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
sed -i.bak "s|heartbeat_server_url: \"http://[^\"]*:18000|heartbeat_server_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
sed -i.bak "s|installer_base_url: \"http://[^\"]*:8000|installer_base_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
sed -i.bak "s|installer_base_url: \"http://[^\"]*:18000|installer_base_url: \"http://$SERVER_IP:$API_PORT|" api/config.yaml
sed -i.bak "s|pushgateway_url: \"http://[^\"]*:9091\"|pushgateway_url: \"http://$SERVER_IP:$PUSHGATEWAY_PORT\"|" api/config.yaml
rm -f api/config.yaml.bak
echo -e "${GREEN}✓ api/config.yaml 文件已更新${NC}"

echo -e "${YELLOW}步骤 3: 停止现有服务...${NC}"
$DOCKER_COMPOSE_CMD down 2>/dev/null || true
echo -e "${GREEN}✓ 现有服务已停止${NC}"

echo -e "${YELLOW}步骤 4: 构建镜像并启动服务...${NC}"
$DOCKER_COMPOSE_CMD up -d --build

# 等待服务启动
echo -e "${YELLOW}等待服务启动...${NC}"
sleep 15

# 检查服务状态
echo -e "${YELLOW}步骤 5: 检查服务状态...${NC}"
SERVICES=("devops-mysql" "devops-redis" "devops-pushgateway" "devops-prometheus" "devops-api" "devops-web")
ALL_HEALTHY=true

for service in "${SERVICES[@]}"; do
    if docker ps --filter "name=$service" --filter "status=running" | grep -q "$service"; then
        echo -e "${GREEN}✓ $service 运行中${NC}"
    else
        echo -e "${RED}✗ $service 未运行${NC}"
        ALL_HEALTHY=false
    fi
done

echo ""
echo -e "${GREEN}========================================${NC}"
if [ "$ALL_HEALTHY" = true ]; then
    echo -e "${GREEN}✓ 所有服务启动成功!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "访问地址:"
    echo -e "  前端:       ${GREEN}http://$SERVER_IP:$WEB_PORT${NC}  (admin/123456)"
    echo -e "  API:        ${GREEN}http://$SERVER_IP:$API_PORT${NC}"
    echo -e "  Prometheus: ${GREEN}http://$SERVER_IP:$PROMETHEUS_PORT${NC}"
    echo -e "  Pushgateway:${GREEN}http://$SERVER_IP:$PUSHGATEWAY_PORT${NC}"
    echo ""
    echo "数据库连接:"
    echo -e "  MySQL:      ${GREEN}$SERVER_IP:$MYSQL_PORT${NC}"
    echo -e "  Redis:      ${GREEN}$SERVER_IP:$REDIS_PORT${NC}"
else
    echo -e "${RED}✗ 部分服务启动失败,请检查日志${NC}"
    echo -e "${RED}========================================${NC}"
    echo ""
    echo "查看日志:"
    echo "  $DOCKER_COMPOSE_CMD logs -f"
    exit 1
fi

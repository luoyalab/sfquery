#!/bin/bash
set -e

# 顺丰查询服务一键部署脚本
DOMAIN="${DOMAIN:-xunma56.com}"
DOCKER="docker"
COMPOSE="docker-compose"

# 检查是否需要 sudo
if ! docker ps >/dev/null 2>&1; then
    DOCKER="sudo docker"
    COMPOSE="sudo docker-compose"
    echo "[提示] 使用 sudo 运行 docker"
fi

echo "[部署] 顺丰查询服务 -> $DOMAIN"

# 检查依赖
command -v docker >/dev/null 2>&1 || { echo "错误: Docker 未安装"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "错误: docker-compose 未安装"; exit 1; }

# 替换域名
sed -i "s/server_name .*/server_name $DOMAIN www.$DOMAIN;/" nginx.conf

# 部署（幂等）
$COMPOSE down 2>/dev/null || true
$COMPOSE up --build -d

# 等待就绪
echo "[等待] 服务启动中..."
sleep 2
for i in {1..30}; do
    curl -s http://localhost/health >/dev/null 2>&1 && break
    sleep 1
done

echo "[完成] 服务已部署: http://$DOMAIN"
echo "[日志] $COMPOSE logs -f"

#!/bin/bash
# 修复权限并重新部署

echo "[修复] 1. 将当前用户加入 docker 组..."
sudo usermod -aG docker $USER

echo "[修复] 2. 清理失败的镜像..."
sudo docker rmi certbot/certbot:latest 2>/dev/null || true

echo "[修复] 3. 创建证书目录..."
sudo mkdir -p certbot/conf certbot/www
sudo chown -R $USER:$USER certbot/

echo "[修复] 4. 重新拉取镜像..."
sudo docker pull certbot/certbot

echo "[修复] 5. 重新部署（使用 sudo）..."
sudo docker-compose down 2>/dev/null || true
sudo docker-compose up --build -d

echo "[完成] 请退出并重新登录 SSH，使 docker 组生效"
echo "或者运行: newgrp docker"

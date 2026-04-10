#!/bin/bash
# 初始化 Let's Encrypt 证书（首次运行）

domain="${DOMAIN:-xunma56.com}"
email="${EMAIL:-admin@$domain}"

echo "[SSL] 为 $domain 申请证书..."

# 创建目录
mkdir -p certbot/conf certbot/www

# 临时启动 nginx 用于验证
docker-compose up -d nginx
sleep 2

# 申请证书
docker run -it --rm \
  -v "$(pwd)/certbot/conf:/etc/letsencrypt" \
  -v "$(pwd)/certbot/www:/var/www/certbot" \
  certbot/certbot certonly \
  --webroot -w /var/www/certbot \
  -d "$domain" -d "www.$domain" \
  --email "$email" \
  --agree-tos \
  --no-eff-email

# 重启 nginx 加载证书
docker-compose restart nginx

echo "[完成] HTTPS 已启用: https://$domain"

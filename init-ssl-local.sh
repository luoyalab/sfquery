#!/bin/bash
# 本地部署模式申请 SSL 证书

domain="${DOMAIN:-xunma56.com}"
email="${EMAIL:-admin@$domain}"
APP_NAME="sfquery"

echo "[SSL] 为 $domain 申请证书..."

# 安装 certbot（如未安装）
if ! command -v certbot > /dev/null 2>&1; then
    echo "[安装] certbot..."
    sudo apt update
    sudo apt install -y certbot python3-certbot-nginx
fi

# 申请证书并自动配置 nginx
sudo certbot --nginx -d "$domain" -d "www.$domain" --email "$email" --agree-tos --no-eff-email

echo "[完成] HTTPS 已启用: https://$domain"
echo "[自动续期] certbot 已配置 systemd 定时任务"

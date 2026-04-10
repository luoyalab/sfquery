#!/bin/bash
set -e

DOMAIN="${DOMAIN:-xunma56.com}"
APP_NAME="sfquery"
APP_DIR="/opt/$APP_NAME"

echo "[部署] $APP_NAME -> $DOMAIN"

# 检查依赖
command -v go >/dev/null 2>&1 || { echo "错误: Go 未安装"; exit 1; }
command -v nginx >/dev/null 2>&1 || { echo "错误: Nginx 未安装 (apt install nginx)"; exit 1; }

# 编译
echo "[编译] 构建应用..."
go build -ldflags="-s -w" -o $APP_NAME ./cmd/server

# 创建目录
sudo mkdir -p $APP_DIR
sudo cp $APP_NAME $APP_DIR/

# 创建 systemd 服务
echo "[服务] 创建 systemd 服务..."
sudo tee /etc/systemd/system/$APP_NAME.service << EOF
[Unit]
Description=SF Query Service
After=network.target

[Service]
Type=simple
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/$APP_NAME -addr :8080
Restart=always
RestartSec=5
User=www-data

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable $APP_NAME
sudo systemctl restart $APP_NAME

# 配置 nginx
echo "[Nginx] 配置反向代理..."
sudo tee /etc/nginx/sites-available/$APP_NAME << EOF
server {
    listen 80;
    server_name $DOMAIN www.$DOMAIN;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/$APP_NAME /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default

# 测试并重载 nginx
sudo nginx -t && sudo systemctl reload nginx

echo "[完成] 服务已部署: http://$DOMAIN"
echo "[管理] systemctl {start|stop|restart|status} $APP_NAME"

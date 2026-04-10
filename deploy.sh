#!/bin/bash
set -e

DOMAIN="${DOMAIN:-xunma56.com}"
EMAIL="${EMAIL:-admin@$DOMAIN}"
APP_NAME="sfquery"
APP_DIR="/opt/$APP_NAME"

log() { echo "[$(date '+%H:%M:%S')] $1"; }

# 安装依赖
install_deps() {
    log "检查依赖..."

    if ! command -v go >/dev/null 2>&1; then
        log "安装 Go..."
        curl -fsSL https://mirrors.aliyun.com/golang/go1.21.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xzf -
        echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
        export PATH=$PATH:/usr/local/go/bin
    fi

    if ! command -v nginx >/dev/null 2>&1; then
        log "安装 Nginx..."
        apt-get update
        apt-get install -y nginx curl
    fi
}

# 编译应用
build() {
    log "编译应用..."
    go build -ldflags="-s -w" -o $APP_NAME ./cmd/server
    sudo mkdir -p $APP_DIR
    sudo cp $APP_NAME $APP_DIR/
}

# 配置 systemd
setup_service() {
    log "配置服务..."
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

[Install]
WantedBy=multi-user.target
EOF
    sudo systemctl daemon-reload
    sudo systemctl enable $APP_NAME
    sudo systemctl restart $APP_NAME
}

# 配置 nginx
setup_nginx() {
    log "配置 Nginx..."
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
    }
}
EOF
    sudo ln -sf /etc/nginx/sites-available/$APP_NAME /etc/nginx/sites-enabled/
    sudo rm -f /etc/nginx/sites-enabled/default
    sudo nginx -t && sudo systemctl reload nginx
}

# 配置 SSL
setup_ssl() {
    log "配置 HTTPS..."

    apt-get install -y certbot python3-certbot-nginx

    sudo certbot --nginx -d "$DOMAIN" -d "www.$DOMAIN" \
        --email "$EMAIL" --agree-tos --no-eff-email || true

    # 自动续期已内置在 certbot
}

# 主流程
main() {
    log "开始部署 $APP_NAME -> $DOMAIN"

    install_deps
    build
    setup_service
    setup_nginx

    # 如果传了 --ssl 参数或环境变量 SSL=1，则配置证书
    if [ "${1:-}" = "--ssl" ] || [ "${SSL:-}" = "1" ]; then
        setup_ssl
        log "完成: https://$DOMAIN"
    else
        log "完成: http://$DOMAIN"
        log "如需 HTTPS，运行: $0 --ssl"
    fi

    log "管理: systemctl {stop|restart|status} $APP_NAME"
}

main "$@"

#!/bin/bash
# ============================================================
# TokenHub SSL 证书申请 + 自动续期配置
# 使用: sudo bash deploy-ssl.sh [域名]
# 默认域名: tokenhub.com
# ============================================================
set -e

DOMAIN="${1:-tokenhub.com}"
EMAIL="${2:-admin@${DOMAIN}}"
INSTALL_DIR="/opt/tokenhub"

echo "=========================================="
echo "  TokenHub SSL 证书配置"
echo "  域名: ${DOMAIN}"
echo "  邮箱: ${EMAIL}"
echo "=========================================="

# ===== 1. 检查 DNS 解析 =====
echo "[1/5] 检查 DNS 解析..."
SERVER_IP=$(curl -s ifconfig.me)
DOMAIN_IP=$(dig +short ${DOMAIN} | tail -1)

if [ -z "$DOMAIN_IP" ]; then
    echo "❌ DNS 未解析！请先将 ${DOMAIN} 指向 ${SERVER_IP}"
    echo "   添加 A 记录: ${DOMAIN} → ${SERVER_IP}"
    echo "   添加 A 记录: www.${DOMAIN} → ${SERVER_IP}"
    echo ""
    echo "   DNS 配置示例（以 Cloudflare 为例）:"
    echo "   类型: A    名称: @     内容: ${SERVER_IP}    代理: DNS only"
    echo "   类型: A    名称: www  内容: ${SERVER_IP}    代理: DNS only"
    echo ""
    read -p "DNS 已配置好？按回车继续或 Ctrl+C 退出..."
    DOMAIN_IP=$(dig +short ${DOMAIN} | tail -1)
fi

if [ "$DOMAIN_IP" != "$SERVER_IP" ]; then
    echo "⚠️  DNS 指向 ${DOMAIN_IP}，本机 IP 为 ${SERVER_IP}"
    echo "   如果使用 CDN/代理（如 Cloudflare），这可能是正常的"
    read -p "继续？[y/N] " confirm
    [ "$confirm" = "y" ] || exit 1
fi

echo "✅ DNS 解析正常: ${DOMAIN} → ${DOMAIN_IP}"

# ===== 2. 安装 Certbot =====
echo "[2/5] 安装 Certbot..."
if ! command -v certbot &> /dev/null; then
    if command -v apt &> /dev/null; then
        apt update
        apt install -y certbot python3-certbot-nginx
    elif command -v yum &> /dev/null; then
        yum install -y certbot python3-certbot-nginx
    fi
fi

# ===== 3. 临时允许 HTTP 80 端口（验证用） =====
echo "[3/5] 确保 Nginx 监听 80 端口..."
# 检查防火墙
if command -v ufw &> /dev/null; then
    ufw allow 80/tcp 2>/dev/null || true
    ufw allow 443/tcp 2>/dev/null || true
elif command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-service=http 2>/dev/null || true
    firewall-cmd --permanent --add-service=https 2>/dev/null || true
    firewall-cmd --reload 2>/dev/null || true
fi

# 确保 Nginx 配置中有 80 端口的 server block
NGINX_CONF="/etc/nginx/conf.d/${DOMAIN}.conf"
if [ ! -f "$NGINX_CONF" ]; then
    NGINX_CONF="/etc/nginx/sites-available/${DOMAIN}"
fi

# ===== 4. 申请 SSL 证书 =====
echo "[4/5] 申请 Let's Encrypt SSL 证书..."
echo "   使用 webroot 方式验证..."

# 创建 webroot 验证目录
mkdir -p /var/www/tokenhub/frontend/.well-known

# 确保 nginx 配置中有 .well-known 的 location
if ! grep -q "\.well-known" "$NGINX_CONF" 2>/dev/null; then
    # 临时在 80 端口的 server block 里添加验证路径
    sed -i '/listen 80;/a\
\n    # Certbot 验证路径\n    location /.well-known/acme-challenge/ {\n        root /var/www/tokenhub/frontend;\n    }' "$NGINX_CONF" 2>/dev/null || true
    nginx -t 2>/dev/null && systemctl reload nginx
fi

# 申请证书
certbot certonly \
    --webroot \
    --webroot-path=/var/www/tokenhub/frontend \
    --domain ${DOMAIN} \
    --domain www.${DOMAIN} \
    --email ${EMAIL} \
    --agree-tos \
    --non-interactive \
    --rsa-key-size 4096

echo "✅ SSL 证书申请成功！"

# ===== 5. 配置 Nginx 启用 SSL =====
echo "[5/5] 配置 Nginx 启用 HTTPS..."

# 使用 certbot 自动配置
certbot --nginx \
    --domain ${DOMAIN} \
    --domain www.${DOMAIN} \
    --non-interactive \
    --redirect

# 或者手动替换配置（如果 certbot --nginx 不生效）
if ! grep -q "ssl_certificate" "$NGINX_CONF"; then
    echo "手动配置 SSL..."

    # 备份
    cp "$NGINX_CONF" "${NGINX_CONF}.bak"

    # 生成新的 Nginx 配置
    cat > "$NGINX_CONF" <<NGINXEOF
# HTTP → HTTPS 重定向
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};

    # Certbot 验证
    location /.well-known/acme-challenge/ {
        root /var/www/tokenhub/frontend;
    }

    location / {
        return 301 https://\$server_name\$request_uri;
    }
}

# HTTPS 主配置
server {
    listen 443 ssl http2;
    server_name ${DOMAIN} www.${DOMAIN};

    # SSL 证书
    ssl_certificate     /etc/letsencrypt/live/${DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${DOMAIN}/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers on;
    ssl_session_cache   shared:SSL:10m;
    ssl_session_timeout 10m;

    # HSTS
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Gzip
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript;
    gzip_min_length 1000;

    client_max_body_size 10m;

    # 前端
    root /var/www/tokenhub/frontend/dist;
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # API 代理
    location /v1/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
        proxy_buffering off;
    }

    location /admin/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /auth/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    # 公开端点
    location /public/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }

    location /health {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
    }

    location /metrics {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
        allow 127.0.0.1;
        deny all;
    }

    # WebSocket
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_read_timeout 86400;
    }

    location /uploads/ {
        alias /var/www/tokenhub/uploads/;
        expires 7d;
    }

    location ~ /\. {
        deny all;
    }
}
NGINXEOF
fi

nginx -t && systemctl reload nginx

# ===== 自动续期 =====
echo ""
echo "配置自动续期..."
# Certbot 自动添加了 cron/systemd timer，验证一下
if systemctl list-timers | grep -q certbot; then
    echo "✅ Certbot 自动续期已配置 (systemd timer)"
else
    # 添加 cron 任务
    echo "0 3 * * * root certbot renew --quiet --post-hook 'systemctl reload nginx'" > /etc/cron.d/certbot-renew
    echo "✅ Certbot 自动续期已配置 (cron: 每天凌晨3点检查)"
fi

# ===== 验证 =====
echo ""
echo "=========================================="
echo "  SSL 证书配置完成！"
echo ""
echo "  HTTPS 地址: https://${DOMAIN}"
echo "  证书路径: /etc/letsencrypt/live/${DOMAIN}/"
echo "  自动续期: 已启用"
echo ""
echo "  测试命令:"
echo "    curl -sI https://${DOMAIN}/health"
echo "    echo | openssl s_client -connect ${DOMAIN}:443 2>/dev/null | openssl x509 -noout -dates"
echo ""
echo "  手动续期测试:"
echo "    sudo certbot renew --dry-run"
echo "=========================================="

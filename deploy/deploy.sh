#!/bin/bash
# TokenHub 一键部署脚本
# 适用于 Ubuntu 22.04 / Debian 12 / CentOS 8+
# 使用: sudo bash deploy.sh

set -e

# ===== 配置 =====
DOMAIN="${1:-tokenhub.com}"
ADMIN_EMAIL="${2:-admin@tokenhub.com}"
ADMIN_PASSWORD="${3:-Admin@2026}"
DB_PASSWORD="$(openssl rand -hex 16)"
JWT_SECRET="$(openssl rand -hex 32)"
INSTALL_DIR="/opt/tokenhub"

echo "=========================================="
echo "  TokenHub 部署脚本"
echo "  域名: $DOMAIN"
echo "  管理员: $ADMIN_EMAIL"
echo "=========================================="

# ===== 1. 系统依赖 =====
echo "[1/8] 安装系统依赖..."
if command -v apt &> /dev/null; then
    apt update && apt install -y curl wget git nginx postgresql postgresql-contrib redis-server certbot python3-certbot-nginx
elif command -v yum &> /dev/null; then
    yum install -y curl wget git nginx postgresql-server redis certbot python3-certbot-nginx
    postgresql-setup initdb
    systemctl enable postgresql
    systemctl start postgresql
fi

# ===== 2. 配置 PostgreSQL =====
echo "[2/8] 配置 PostgreSQL..."
sudo -u postgres psql -c "CREATE USER tokenhub WITH PASSWORD '${DB_PASSWORD}';" 2>/dev/null || true
sudo -u postgres psql -c "CREATE DATABASE tokenhub OWNER tokenhub;" 2>/dev/null || true
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE tokenhub TO tokenhub;" 2>/dev/null || true

# 初始化数据库
echo "[2.1] 初始化数据库表结构..."
sudo -u postgres psql -d tokenhub -f ${INSTALL_DIR}/backend/scripts/init_db.sql

# ===== 3. 配置 Redis =====
echo "[3/8] 配置 Redis..."
systemctl enable redis-server 2>/dev/null || systemctl enable redis
systemctl start redis-server 2>/dev/null || systemctl start redis

# ===== 4. 编译后端 =====
echo "[4/8] 编译后端..."
if ! command -v go &> /dev/null; then
    wget -q https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
    rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
fi
cd ${INSTALL_DIR}/backend
go mod download
CGO_ENABLED=0 GOOS=linux go build -o tokenhub-server ./cmd/server/

# ===== 5. 编译前端 =====
echo "[5/8] 编译前端..."
if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt install -y nodejs
fi
cd ${INSTALL_DIR}/frontend
npm install
VITE_API_BASE_URL="" npm run build

# ===== 6. 配置应用 =====
echo "[6/8] 配置应用..."
mkdir -p /var/www/tokenhub/frontend
mkdir -p /var/www/tokenhub/uploads/avatars
cp -r ${INSTALL_DIR}/frontend/dist/* /var/www/tokenhub/frontend/
cp ${INSTALL_DIR}/backend/tokenhub-server /usr/local/bin/

# 写入配置文件
cat > ${INSTALL_DIR}/configs/config.yaml <<EOF
server:
  port: 8080
  mode: release
  read_timeout: 30s
  write_timeout: 120s
  graceful_wait: 15s

database:
  host: 127.0.0.1
  port: 5432
  user: tokenhub
  password: ${DB_PASSWORD}
  dbname: tokenhub
  max_idle_conns: 20
  max_open_conns: 200
  log_level: warn

redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0
  pool_size: 200
  min_idle_conns: 20

jwt:
  secret: "${JWT_SECRET}"
  expire: 24h
  issuer: tokenhub
  refresh_exp: 168h

gateway:
  max_retries: 3
  timeout: 60s
  stream_timeout: 300s
  max_concurrent: 2000
  rate_limit_per_sec: 100

billing:
  default_currency: CNY
  min_balance: 100
  token_granularity: 1

wallet:
  deposit_addresses:
    ethereum: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"
    bsc: "0x742d35Cc6634C0532925a3b844Bc9e7595f2bD18"
    tron: "TYDzsWUZbMF6n3qG1GmwVsQ1v9Xd9gKb0S"
  chain_monitor:
    poll_interval: 10s
    batch_size: 100

payment:
  alipay:
    enabled: false
    app_id: ""
    private_key: ""
    public_key: ""
    notify_url: "https://${DOMAIN}/auth/callback/alipay"
    is_sandbox: true
  wechat_pay:
    enabled: false
    app_id: ""
    mch_id: ""
    api_key: ""
    cert_path: ""
    notify_url: "https://${DOMAIN}/auth/callback/wechat"
    is_sandbox: true
  stripe:
    enabled: false
    publishable_key: ""
    secret_key: ""
    webhook_secret: ""
    notify_url: "https://${DOMAIN}/auth/callback/stripe"

verification:
  sms:
    provider: aliyun
    aliyun:
      access_key_id: ""
      access_key_secret: ""
      sign_name: "TokenHub"
      template_code: ""
    code_length: 6
    expire_minutes: 5
    cooldown_seconds: 60
    max_send_per_hour: 5
    max_verify_attempts: 5
  email:
    provider: smtp
    smtp:
      host: "smtp.${DOMAIN}"
      port: 465
      username: ""
      password: ""
      from_address: "noreply@${DOMAIN}"
      from_name: "TokenHub"
    code_length: 6
    expire_minutes: 5
    cooldown_seconds: 60

monitor:
  enable_metrics: true
  metrics_path: /metrics
  report_interval: 10s
  alert_webhook: ""

security:
  enable_waf: true
  enable_desensitize: true
  blocked_ips: []
  max_request_body: 10485760

i18n:
  default_locale: zh-CN
  supported_locales:
    - zh-CN
    - en-US
    - ja-JP
    - ko-KR
EOF

# 环境变量
cat > /etc/tokenhub.env <<EOF
CONFIG_PATH=${INSTALL_DIR}/configs/config.yaml
ADMIN_EMAIL=${ADMIN_EMAIL}
ADMIN_PASSWORD=${ADMIN_PASSWORD}
UPLOAD_DIR=/var/www/tokenhub/uploads
EOF

# ===== 7. Systemd 服务 =====
echo "[7/8] 配置 systemd 服务..."
cat > /etc/systemd/system/tokenhub.service <<EOF
[Unit]
Description=TokenHub Server
After=network.target postgresql.service redis.service
Requires=postgresql.service redis.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=/etc/tokenhub.env
ExecStart=/usr/local/bin/tokenhub-server
Restart=always
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable tokenhub
systemctl start tokenhub

# ===== 8. Nginx 配置 =====
echo "[8/8] 配置 Nginx..."
sed "s/{{DOMAIN}}/${DOMAIN}/g" ${INSTALL_DIR}/deploy/nginx/tokenhub.conf > /etc/nginx/sites-available/tokenhub
ln -sf /etc/nginx/sites-available/tokenhub /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

# SSL 证书（Let's Encrypt）
echo ""
echo "=========================================="
echo "  部署完成！"
echo ""
echo "  后端服务: http://localhost:8080"
echo "  前端地址: https://${DOMAIN}"
echo "  管理员邮箱: ${ADMIN_EMAIL}"
echo "  数据库密码: ${DB_PASSWORD}"
echo "  JWT密钥: ${JWT_SECRET}"
echo ""
echo "  下一步操作:"
echo "  1. 配置 DNS 将 ${DOMAIN} 指向本服务器 IP"
echo "  2. 运行 SSL 证书申请:"
echo "     sudo certbot --nginx -d ${DOMAIN}"
echo "  3. 配置支付渠道密钥:"
echo "     vi ${INSTALL_DIR}/configs/config.yaml"
echo "  4. 重启服务:"
echo "     sudo systemctl restart tokenhub"
echo ""
echo "  凭据已保存到 /etc/tokenhub.env"
echo "=========================================="

#!/bin/bash
# ============================================================
# TokenHub 一键自动安装脚本
# 支持: CentOS / Alibaba Cloud Linux / Rocky / Ubuntu / Debian
# 功能: 安装依赖(Go/Node/PostgreSQL/Redis) → 克隆代码 → 编译后端
#       → 构建前端(自动 swap 防 OOM) → 配置 nginx/systemd → 启动
# 用法:
#   bash install.sh --domain tokenshopai.com
#   bash install.sh --domain example.com --db-pass "StrongPass123" \
#                   --admin-email admin@example.com --admin-pass "admin123456"
# ============================================================
set -e

# ---------- 颜色 ----------
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info(){ echo -e "${BLUE}[INFO]${NC} $1"; }
ok(){ echo -e "${GREEN}[ OK ]${NC} $1"; }
warn(){ echo -e "${YELLOW}[WARN]${NC} $1"; }
err(){ echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
step(){ echo -e "\n${BLUE}==== $1 ====${NC}"; }

# ---------- 默认参数 ----------
DOMAIN="tokenshopai.com"
BASE="/root/tokenhub"
REPO="https://github.com/mei651250-eng/tokenshopai.git"
DB_PASS="tokenhub_secret"
ADMIN_EMAIL="admin@tokenhub.com"
ADMIN_PASS="admin123456"
GO_VERSION="1.22.5"
NODE_VERSION="20.15.1"

# ---------- 解析参数 ----------
while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2;;
    --base) BASE="$2"; shift 2;;
    --repo) REPO="$2"; shift 2;;
    --db-pass) DB_PASS="$2"; shift 2;;
    --admin-email) ADMIN_EMAIL="$2"; shift 2;;
    --admin-pass) ADMIN_PASS="$2"; shift 2;;
    -h|--help) sed -n '2,12p' "$0"; exit 0;;
    *) warn "未知参数: $1"; shift;;
  esac
done

FRONTEND_URL="https://${DOMAIN}"

# ---------- 权限检查 ----------
[ "$(id -u)" -eq 0 ] || err "请使用 root 用户运行此脚本"

# ---------- 检测包管理器 ----------
if command -v dnf >/dev/null 2>&1; then PKG="dnf";
elif command -v yum >/dev/null 2>&1; then PKG="yum";
elif command -v apt-get >/dev/null 2>&1; then PKG="apt";
else err "不支持的 Linux 发行版（未找到 yum/dnf/apt）"; fi
info "检测到包管理器: $PKG"

# ============================================================
step "1/11 确保交换分区 (防前端构建 OOM)"
# ============================================================
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
if [ "$TOTAL_MEM" -lt 3500 ]; then
  if ! swapon --show | grep -q '/swap_tokenhub'; then
    info "物理内存 ${TOTAL_MEM}MB < 3.5GB，创建 4GB swap..."
    fallocate -l 4G /swap_tokenhub 2>/dev/null || dd if=/dev/zero of=/swap_tokenhub bs=1M count=4096
    chmod 600 /swap_tokenhub
    mkswap /swap_tokenhub
    swapon /swap_tokenhub
    grep -q '/swap_tokenhub' /etc/fstab || echo '/swap_tokenhub swap swap defaults 0 0' >> /etc/fstab
    ok "swap 已启用"
  else
    info "swap 已存在，跳过"
  fi
else
  info "物理内存充足 (${TOTAL_MEM}MB)，无需 swap"
fi

# ============================================================
step "2/11 安装系统依赖"
# ============================================================
if [ "$PKG" = "apt" ]; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -y
  apt-get install -y -q git curl wget unzip tar ca-certificates gnupg lsb-release
  # PostgreSQL + Redis
  apt-get install -y -q postgresql postgresql-contrib redis-server
else
  $PKG install -y epel-release 2>/dev/null || true
  $PKG install -y git curl wget unzip tar
  # PostgreSQL 15
  if ! command -v psql >/dev/null 2>&1; then
    $PKG install -y postgresql-server postgresql-contrib 2>/dev/null || \
    ( rpm -Uvh https://download.postgresql.org/pub/repos/yum/reporpms/EL-$(rpm -E %rhel)-x86_64/pgdg-redhat-repo-latest.noarch.rpm 2>/dev/null && $PKG install -y postgresql15-server postgresql15 )
  fi
  # Redis
  command -v redis-server >/dev/null 2>&1 || $PKG install -y redis
fi
ok "系统依赖已安装"

# ============================================================
step "3/11 初始化 PostgreSQL 与 Redis"
# ============================================================
# 启动 PostgreSQL
if command -v pg_ctlcluster >/dev/null 2>&1; then
  pg_ctlcluster $(ls /etc/postgresql)/main start 2>/dev/null || true
elif [ -f /usr/pgsql-*/bin/postgresql-*.setup ]; then
  POSTGRES_BIN=$(ls -d /usr/pgsql-* | head -1)/bin
  if [ ! -f /var/lib/pgsql/data/PG_VERSION ]; then
    $POSTGRES_BIN/postgresql-$(ls /usr/pgsql-* | head -1 | grep -oP '\d+$')-setup initdb 2>/dev/null || \
    sudo -u postgres $POSTGRES_BIN/initdb -D /var/lib/pgsql/data
  fi
fi
systemctl enable postgresql redis 2>/dev/null || true
systemctl start postgresql redis || systemctl start redis-server 2>/dev/null || true
sleep 3

# 创建数据库与用户
if command -v psql >/dev/null 2>&1; then
  sudo -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='tokenhub'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE USER tokenhub WITH PASSWORD '$DB_PASS' SUPERUSER;"
  sudo -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname='tokenhub'" | grep -q 1 || \
    sudo -u postgres psql -c "CREATE DATABASE tokenhub OWNER tokenhub;"
  ok "PostgreSQL 数据库 tokenhub 已就绪"
fi

# 配置 Redis 监听 localhost
if [ -f /etc/redis.conf ]; then
  sed -i 's/^bind 127.0.0.1 -::1/bind 127.0.0.1/' /etc/redis.conf 2>/dev/null
  systemctl restart redis 2>/dev/null || systemctl restart redis-server 2>/dev/null || true
fi
ok "Redis 已启动"

# ============================================================
step "4/11 安装 Go ${GO_VERSION}"
# ============================================================
if ! command -v go >/dev/null 2>&1 || [ "$(go version | grep -oP '\d+\.\d+')" \< "1.21" ]; then
  info "下载 Go..."
  wget -q "https://golang.google.cn/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz || \
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tar.gz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ok "Go 已安装: $(/usr/local/go/bin/go version)"
else
  info "Go 已安装: $(go version)"
fi

# ============================================================
step "5/11 安装 Node.js ${NODE_VERSION}"
# ============================================================
if ! command -v node >/dev/null 2>&1 || [ "$(node -v | grep -oP '\d+')" -lt 18 ]; then
  info "下载 Node.js 二进制..."
  wget -q "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" -O /tmp/node.tar.xz || \
  wget -q "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" -O /tmp/node.tar.xz
  tar -xf /tmp/node.tar.xz -C /usr/local --strip-components=1
  ok "Node.js 已安装: $(node -v)"
else
  info "Node.js 已安装: $(node -v)"
fi
# 配置 npm 国内镜像
npm config set registry https://registry.npmmirror.com 2>/dev/null || true

# ============================================================
step "6/11 获取代码"
# ============================================================
if [ -d "$BASE/.git" ]; then
  info "代码已存在，执行 git pull..."
  cd "$BASE" && git pull
else
  info "克隆仓库 $REPO → $BASE"
  git clone "$REPO" "$BASE"
fi
ok "代码就绪: $BASE"

# ============================================================
step "7/11 生成密钥与环境变量"
# ============================================================
JWT_SECRET=$(openssl rand -hex 32)
KEYVAULT_MASTER_KEY=$(openssl rand -hex 32)
mkdir -p /etc/systemd/system/tokenhub.service.d
cat > /etc/systemd/system/tokenhub.service.d/env.conf <<EOF
[Service]
Environment="FRONTEND_URL=${FRONTEND_URL}"
Environment="JWT_SECRET=${JWT_SECRET}"
Environment="KEYVAULT_MASTER_KEY=${KEYVAULT_MASTER_KEY}"
Environment="ADMIN_EMAIL=${ADMIN_EMAIL}"
Environment="ADMIN_PASSWORD=${ADMIN_PASS}"
Environment="DB_PASSWORD=${DB_PASS}"
Environment="GIN_MODE=release"
EOF
ok "环境变量已写入 /etc/systemd/system/tokenhub.service.d/env.conf"

# ============================================================
step "8/11 安装 Go 依赖并编译后端"
# ============================================================
cd "$BASE/backend"
export GOPROXY="https://goproxy.cn,direct"
export GOTOOLCHAIN="local"
export PATH=$PATH:/usr/local/go/bin
info "下载 Go 模块并编译 (这可能需要几分钟)..."
go mod download 2>&1 | tail -2 || true
go build -o tokenhub ./cmd/server/ && ok "后端编译完成" || err "后端编译失败"

# ============================================================
step "9/11 构建前端 (swap + 内存限制 防 OOM)"
# ============================================================
cd "$BASE/frontend"
export NODE_OPTIONS="--max-old-space-size=2048"
info "安装 npm 依赖..."
npm install --no-audit --no-fund 2>&1 | tail -3 || warn "npm install 有警告，继续..."
info "构建前端 (--minify false 以降低内存占用)..."
npx vite build --minify false 2>&1 | tail -5 && ok "前端构建完成" || err "前端构建失败"

# ============================================================
step "10/11 部署前端与 Nginx 配置"
# ============================================================
mkdir -p /var/www/tokenhub/frontend
rm -rf /var/www/tokenhub/frontend/dist
cp -r "$BASE/frontend/dist" /var/www/tokenhub/frontend/dist
ok "前端已部署到 /var/www/tokenhub/frontend/dist"

# 生成 nginx 配置 (Cloudflare Flexible 模式，仅监听 80)
cat > /etc/nginx/conf.d/tokenhub.conf <<EOF
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};

    client_max_body_size 10m;
    root /var/www/tokenhub/frontend/dist;
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)\$ {
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

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
    location /user/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    location /distributor/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
    location /public/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }
    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_read_timeout 86400;
    }
    location /health {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
    }
    location ~ /\. { deny all; }
}
EOF
# 移除默认配置避免冲突
rm -f /etc/nginx/sites-enabled/default /etc/nginx/conf.d/default.conf 2>/dev/null
nginx -t && systemctl reload nginx && ok "Nginx 配置已加载" || warn "Nginx 测试失败，请检查配置"

# ============================================================
step "11/11 安装服务并启动"
# ============================================================
cp "$BASE/deploy/tokenhub.service" /etc/systemd/system/tokenhub.service
systemctl daemon-reload
systemctl enable tokenhub
systemctl restart tokenhub
sleep 3

# 验证
if systemctl is-active --quiet tokenhub; then
  ok "TokenHub 后端服务运行中"
else
  err "TokenHub 服务未启动，请查看: journalctl -u tokenhub -n 50"
fi

echo ""
echo -e "${GREEN}============================================${NC}"
echo -e "${GREEN}  🎉 TokenHub 安装完成!${NC}"
echo -e "${GREEN}============================================${NC}"
echo -e "  访问地址:   ${BLUE}http://${DOMAIN}${NC}  (或 https://${DOMAIN} 经 Cloudflare)"
echo -e "  管理后台:   ${BLUE}http://${DOMAIN}/#/admin/dashboard${NC}"
echo -e "  用户端:     ${BLUE}http://${DOMAIN}/#/home${NC}"
echo -e "  分销商端:   ${BLUE}http://${DOMAIN}/#/distributor${NC}"
echo -e "  登录页:     ${BLUE}http://${DOMAIN}/#/login${NC}"
echo ""
echo -e "  管理员账号: ${YELLOW}${ADMIN_EMAIL}${NC}"
echo -e "  管理员密码: ${YELLOW}${ADMIN_PASS}${NC}"
echo ""
echo -e "  数据库:     ${YELLOW}PostgreSQL tokenhub / ${DB_PASS}${NC}"
echo -e "  JWT 密钥:   已自动生成并写入 systemd 环境"
echo ""
echo -e "  后续更新:   ${BLUE}cd $BASE && bash deploy/update.sh${NC}"
echo -e "${GREEN}============================================${NC}"

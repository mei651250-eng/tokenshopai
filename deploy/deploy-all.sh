#!/bin/bash
# ============================================================
# TokenHub 一键全量部署脚本
# 拉取代码 → 编译后端 → 构建前端(防OOM) → 部署 → 重启服务 → 验证
# ============================================================
set -e

BASE="/root/tokenhub"
GOPROXY="https://goproxy.cn,direct"
GOTOOLCHAIN="local"
FRONTEND_DEST="/var/www/tokenhub/frontend/dist"

echo "=========================================="
echo " TokenHub 一键全量部署"
echo " $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="

# 0. 确保 swap (小内存机器防前端构建 OOM)
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
if [ "$TOTAL_MEM" -lt 3500 ] && ! swapon --show | grep -q '/swap_tokenhub'; then
  echo "[0/5] 启用 4GB swap 防止前端构建 OOM..."
  fallocate -l 4G /swap_tokenhub 2>/dev/null || dd if=/dev/zero of=/swap_tokenhub bs=1M count=4096
  chmod 600 /swap_tokenhub && mkswap /swap_tokenhub && swapon /swap_tokenhub
  grep -q '/swap_tokenhub' /etc/fstab || echo '/swap_tokenhub swap swap defaults 0 0' >> /etc/fstab
fi

# 1. 拉取
echo "[1/5] 拉取最新代码..."
cd $BASE && git pull

# 2. 编译后端
echo "[2/5] 编译后端..."
cd $BASE/backend
export GOPROXY=$GOPROXY
export GOTOOLCHAIN=$GOTOOLCHAIN
export PATH=$PATH:/usr/local/go/bin
go build -o tokenhub ./cmd/server/
echo "  后端编译完成"

# 3. 构建前端
echo "[3/5] 构建前端..."
cd $BASE/frontend
export NODE_OPTIONS="--max-old-space-size=2048"
npm install --no-audit --no-fund 2>&1 | tail -2 || true
npx vite build --minify false 2>&1 | tail -5
echo "  前端构建完成"

# 4. 部署
echo "[4/5] 部署..."
rm -rf $FRONTEND_DEST
mkdir -p /var/www/tokenhub/frontend
cp -r $BASE/frontend/dist $FRONTEND_DEST
chmod -R 755 $FRONTEND_DEST
echo "  前端部署完成"

# 重启后端
systemctl restart tokenhub && sleep 1
systemctl reload nginx 2>/dev/null && sleep 1 || true
echo "  服务重启完成"

# 5. 验证
echo ""
echo "=========================================="
echo " 验证部署..."
echo "=========================================="

HEALTH=$(curl -s http://localhost:8080/health)
echo "  后端: $HEALTH"

if systemctl is-active --quiet tokenhub; then
  echo "  服务状态: ✅ 运行中"
else
  echo "  服务状态: ❌ 未运行 (journalctl -u tokenhub -n 50)"
fi

echo ""
echo "=========================================="
echo "  部署完成! 请按 Ctrl+Shift+R 强制刷新"
echo "  Cloudflare → Caching → Purge Everything (如需)"
echo "=========================================="

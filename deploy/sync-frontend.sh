#!/bin/bash
# ============================================================
# TokenHub 前端同步部署脚本
# 将本地修改的前端文件同步到服务器并重新构建
#
# 使用方法 (在本地 Windows PowerShell 中):
#   bash deploy/sync-frontend.sh [服务器IP或域名]
#
# 默认服务器: root@tokenshopai.com
# ============================================================

set -e

REMOTE_HOST="${1:-root@tokenshopai.com}"
REMOTE_DIR="/root/tokenhub/frontend"
DIST_DIR="/var/www/tokenhub/frontend/dist"

echo "=========================================="
echo "  TokenHub 前端同步部署"
echo "  目标服务器: ${REMOTE_HOST}"
echo "=========================================="

# ===== 1. SCP 传输文件 =====
echo ""
echo "[1/3] 传输前端源文件到服务器..."

# 传输 API 配置
scp frontend/src/api/index.ts ${REMOTE_HOST}:${REMOTE_DIR}/src/api/index.ts
echo "  ✅ api/index.ts"

# 传输 ModelListView.vue (厂商分组 + 一键导入)
scp frontend/src/views/models/ModelListView.vue ${REMOTE_HOST}:${REMOTE_DIR}/src/views/models/ModelListView.vue
echo "  ✅ views/models/ModelListView.vue"

# 传输 ModelDetailView.vue (API base 修复)
scp frontend/src/views/models/ModelDetailView.vue ${REMOTE_HOST}:${REMOTE_DIR}/src/views/models/ModelDetailView.vue
echo "  ✅ views/models/ModelDetailView.vue"

# ===== 2. 在服务器上重新构建 =====
echo ""
echo "[2/3] 在服务器上重新构建前端..."
ssh ${REMOTE_HOST} << 'REMOTE_SCRIPT'
set -e
cd /root/tokenhub/frontend

echo "  安装依赖..."
npm install --prefer-offline 2>/dev/null || npm install

echo "  构建前端 (VITE_API_BASE_URL=\"\")..."
VITE_API_BASE_URL="" npm run build

echo "  检查构建产物..."
if [ -d dist ]; then
  echo "  ✅ 构建成功，dist 目录已生成"
  # 检查是否还有 localhost:8080 引用
  COUNT=$(grep -rl "localhost:8080" dist/assets/*.js 2>/dev/null | wc -l)
  if [ "$COUNT" -gt 0 ]; then
    echo "  ⚠️  发现残留的 localhost:8080 引用，正在修复..."
    sed -i 's|http://localhost:8080||g' dist/assets/*.js
    echo "  ✅ 已修复"
  else
    echo "  ✅ 无 localhost:8080 残留引用"
  fi
else
  echo "  ❌ 构建失败，dist 目录不存在"
  exit 1
fi
REMOTE_SCRIPT

# ===== 3. 部署到 Nginx =====
echo ""
echo "[3/3] 部署到 Nginx..."
ssh ${REMOTE_HOST} << 'REMOTE_SCRIPT'
set -e
DIST_DIR="/var/www/tokenhub/frontend/dist"

echo "  备份旧文件..."
[ -d ${DIST_DIR} ] && mv ${DIST_DIR} ${DIST_DIR}.bak.$(date +%Y%m%d%H%M%S) 2>/dev/null || true

echo "  复制新文件..."
mkdir -p ${DIST_DIR}
\cp -r /root/tokenhub/frontend/dist/* ${DIST_DIR}/

echo "  设置权限..."
chown -R www-data:www-data ${DIST_DIR} 2>/dev/null || chown -R nginx:nginx ${DIST_DIR} 2>/dev/null || true

echo "  验证部署..."
FILE_COUNT=$(find ${DIST_DIR}/assets -name "*.js" 2>/dev/null | wc -l)
echo "  ✅ 已部署 ${FILE_COUNT} 个 JS 文件"

# 检查厂商分组功能是否包含
if grep -q "厂商一键导入" ${DIST_DIR}/assets/*.js 2>/dev/null; then
  echo "  ✅ 厂商一键导入功能已包含"
else
  echo "  ⚠️  未检测到厂商一键导入功能，请检查构建"
fi

if grep -q "providerLabels" ${DIST_DIR}/assets/*.js 2>/dev/null; then
  echo "  ✅ 厂商分组标签已包含"
else
  echo "  ⚠️  未检测到厂商分组标签"
fi
REMOTE_SCRIPT

echo ""
echo "=========================================="
echo "  ✅ 前端同步部署完成！"
echo ""
echo "  请在浏览器中按 Ctrl+F5 强制刷新页面"
echo "  如果还不行，去 Cloudflare 面板清除缓存:"
echo "  Caching → Purge Everything"
echo "=========================================="

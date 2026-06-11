#!/bin/bash
# 修复前端 API 地址问题
# 在服务器上以 root 身份运行: bash fix-frontend-api.sh

set -e

INSTALL_DIR="/root/tokenhub"
FRONTEND_DIR="${INSTALL_DIR}/frontend"
DIST_DIR="/var/www/tokenhub/frontend/dist"

echo "===== 修复前端 API baseURL ====="

# 1. 修改源文件
echo "[1/4] 修改 api/index.ts ..."
cd ${FRONTEND_DIR}/src/api
sed -i "s|baseURL: import.meta.env.VITE_API_BASE_URL || (window.Capacitor ? 'http://localhost:8080' : ''),|baseURL: '',|g" index.ts
# 兼容已经修改过的文件
sed -i "s|baseURL: import.meta.env.VITE_API_BASE_URL || '',|baseURL: '',|g" index.ts

echo "[2/4] 修改 ModelDetailView.vue ..."
cd ${FRONTEND_DIR}/src/views/models
sed -i "s|import.meta.env.VITE_API_BASE_URL || 'https://api.example.com'|''|g" ModelDetailView.vue

# 2. 重新构建
echo "[3/4] 重新构建前端 ..."
cd ${FRONTEND_DIR}
VITE_API_BASE_URL="" npm run build

# 3. 部署
echo "[4/4] 部署到 Nginx ..."
cp -r ${FRONTEND_DIR}/dist/* ${DIST_DIR}/

echo ""
echo "===== 修复完成 ====="
echo "请在浏览器中按 Ctrl+F5 强制刷新页面，然后尝试登录"
echo "如果还不行，去 Cloudflare 面板清除缓存: Caching -> Purge Everything"

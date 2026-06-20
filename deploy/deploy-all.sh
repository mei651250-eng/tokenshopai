#!/bin/bash
# ============================================================
# TokenHub 一键全量部署脚本
# 拉取代码 → 编译后端 → 构建前端 → 部署 → 重启服务 → 验证
# ============================================================
set -e

BASE="/root/tokenhub"
GOPROXY="https://goproxy.cn,direct"
GOTOOLCHAIN="local"

echo "=========================================="
echo " TokenHub 一键全量部署"
echo " $(date '+%Y-%m-%d %H:%M:%S')"
echo "=========================================="

# 1. 拉取
echo "[1/4] 拉取最新代码..."
cd $BASE && git pull

# 2. 编译后端
echo "[2/4] 编译后端..."
cd $BASE/backend
export GOPROXY=$GOPROXY
export GOTOOLCHAIN=$GOTOOLCHAIN
/usr/bin/go build -o tokenhub ./cmd/server/
echo "  后端编译完成"

# 3. 构建前端
echo "[3/4] 构建前端..."
cd $BASE/frontend
npm run build 2>&1 | tail -5
echo "  前端构建完成"

# 4. 部署
echo "[4/4] 部署..."
# 部署前端
rm -rf /usr/share/nginx/html/assets /usr/share/nginx/html/icons 2>/dev/null
\cp -rf $BASE/frontend/dist/* /usr/share/nginx/html/
chmod -R 755 /usr/share/nginx/html/
echo "  前端部署完成"

# 重启后端
systemctl restart tokenhub && sleep 1
systemctl restart nginx && sleep 1
echo "  服务重启完成"

# 5. 验证
echo ""
echo "=========================================="
echo " 验证部署..."
echo "=========================================="

# 验证页面版本
LOCAL_VER=$(grep -oP 'assets/index-[^.]+' $BASE/frontend/dist/index.html | head -1)
DEPLOYED_VER=$(grep -oP 'assets/index-[^.]+' /usr/share/nginx/html/index.html | head -1)
echo "  本地构建: $LOCAL_VER"
echo "  已部署:   $DEPLOYED_VER"
if [ "$LOCAL_VER" == "$DEPLOYED_VER" ]; then
  echo "  ✅ 版本一致"
else
  echo "  ❌ 版本不一致!"
fi

# 验证后端健康
echo ""
HEALTH=$(curl -s http://localhost:8080/health)
echo "  后端: $HEALTH"

# 展示标题(用于判断是否最新版)
TITLE=$(grep -oP '<title>[^<]+' /usr/share/nginx/html/index.html)
echo "  页面标题: $TITLE"

echo ""
echo "=========================================="
echo "  部署完成! 请按 Ctrl+Shift+R 强制刷新"
echo "  如仍有缓存问题:"
echo "  Cloudflare → Caching → Purge Everything"
echo "  或在浏览器开发者工具中 Disable cache"
echo "=========================================="

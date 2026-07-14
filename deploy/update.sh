#!/bin/bash
# ============================================================
# TokenHub 一键更新脚本
# 拉取代码 → 编译后端 → 构建前端(自动 swap 防 OOM) → 部署 → 重启
# 用法: cd /root/tokenhub && bash deploy/update.sh
#       SKIP_PULL=1 bash deploy/update.sh   # 跳过 git pull
# ============================================================
set -e

BASE="/root/tokenhub"
RED='\033[0;31m'; GREEN='\033[0;32m'; BLUE='\033[0;34m'; NC='\033[0m'
step(){ echo -e "\n${BLUE}==== $1 ====${NC}"; }

# 1. 确保 swap (小内存机器防止前端构建 OOM)
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
if [ "$TOTAL_MEM" -lt 3500 ] && ! swapon --show | grep -q '/swap_tokenhub'; then
  step "确保 swap 分区 (6G)"
  # 用 dd 创建非稀疏文件：fallocate 在部分云盘生成空洞，swapon 会静默失败
  rm -f /swap_tokenhub
  dd if=/dev/zero of=/swap_tokenhub bs=1M count=6144 status=none
  chmod 600 /swap_tokenhub
  mkswap /swap_tokenhub
  swapon /swap_tokenhub || { echo -e "${RED}错误: swapon 失败，前端构建可能 OOM。${NC}"; exit 1; }
  grep -q '/swap_tokenhub' /etc/fstab || echo '/swap_tokenhub swap swap defaults 0 0' >> /etc/fstab
  echo -e "${GREEN}swap 已启用${NC}"
fi

# 2. 拉取代码
step "拉取最新代码"
cd "$BASE"
if [ -n "$SKIP_PULL" ]; then
  echo "已设置 SKIP_PULL，跳过 git pull（使用当前已有代码）"
elif git pull; then
  echo -e "${GREEN}[ OK ] 已拉取最新代码${NC}"
else
  echo -e "${RED}警告: git pull 失败 (网络/DNS 问题)，将使用当前已有代码继续。${NC}"
fi

# 3. 编译后端
step "编译后端"
cd "$BASE/backend"
export GOPROXY="https://goproxy.cn,direct"
export GOTOOLCHAIN="local"
export PATH=$PATH:/usr/local/go/bin
go build -o tokenhub ./cmd/server/ && echo -e "${GREEN}[ OK ] 后端编译完成${NC}"

# 4. 构建前端 (限制堆内存，避免 OOM)
step "构建前端"
cd "$BASE/frontend"
export NODE_OPTIONS="--max-old-space-size=4096"
npm install --no-audit --no-fund 2>&1 | tail -3 || true
# 注意: 不要加 | tail 吞掉错误；必须 --minify false 降低内存占用
npx vite build --minify false
if [ ! -f dist/index.html ]; then
  echo -e "${RED}错误: 构建未生成 dist/index.html，中止部署（否则前端将 403）。${NC}"
  exit 1
fi
echo -e "${GREEN}[ OK ] 前端构建完成${NC}"

# 5. 部署前端
step "部署前端"
rm -rf /var/www/tokenhub/frontend/dist
cp -r "$BASE/frontend/dist" /var/www/tokenhub/frontend/dist

# 6. 重启服务
step "重启服务"
systemctl restart tokenhub
systemctl reload nginx 2>/dev/null || true
sleep 2

# 7. 验证
step "验证"
HEALTH=""
for i in 1 2 3 4 5 6; do
  HEALTH=$(curl -s --max-time 5 http://localhost:8080/health || echo "")
  if echo "$HEALTH" | grep -q '"status":"healthy"'; then break; fi
  echo "  等待后端就绪... ($i/6)"
  sleep 3
done
echo "  后端健康: $HEALTH"
if systemctl is-active --quiet tokenhub; then
  echo -e "${GREEN}  服务运行中 ✓${NC}"
else
  echo -e "${RED}  服务未运行，请检查: journalctl -u tokenhub -n 50${NC}"
fi

echo -e "\n${GREEN}更新完成! 请 Ctrl+Shift+R 强制刷新浏览器。${NC}"

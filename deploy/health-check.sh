#!/bin/bash
# TokenHub 服务健康检查脚本
# 用途：验证 tokenshopai.com 的 HTTPS 跳转、API 健康状态
# 用法：bash deploy/health-check.sh

set -euo pipefail

DOMAIN="tokenshopai.com"
BASE_URL="https://${DOMAIN}"
PASS=0
FAIL=0

green() { echo -e "\033[32m$1\033[0m"; }
red()   { echo -e "\033[31m$1\033[0m"; }
bold()  { echo -e "\033[1m$1\033[0m"; }

print_result() {
  local name="$1" status="$2" detail="$3"
  if [ "$status" = "PASS" ]; then
    PASS=$((PASS + 1))
    echo -e "$(green "[PASS]") $name"
    [ -n "$detail" ] && echo "       $detail"
  else
    FAIL=$((FAIL + 1))
    echo -e "$(red "[FAIL]") $name"
    [ -n "$detail" ] && echo "       $(red "$detail")"
  fi
}

bold "===== TokenHub 服务健康检查 ====="
echo ""

# ========== 测试 1: HTTP → HTTPS 301 跳转 ==========
bold "[测试 1] HTTP → HTTPS 自动跳转"

HTTP_RESP=$(curl -sI -o /tmp/th_http_header.txt -w "%{http_code}" "http://${DOMAIN}" 2>/dev/null || true)
HTTP_CODE=$(cat /tmp/th_http_header.txt 2>/dev/null | head -1 | awk '{print $2}' || echo "$HTTP_RESP")

# 重新获取完整响应头用于检查 Location
curl -sI -o /tmp/th_http_header_full.txt "http://${DOMAIN}" 2>/dev/null || true
HTTP_CODE_CHECK=$(head -1 /tmp/th_http_header_full.txt | awk '{print $2}')
LOCATION=$(grep -i "^Location:" /tmp/th_http_header_full.txt | tr -d '\r' | sed 's/^Location: //i')

rm -f /tmp/th_http_header.txt /tmp/th_http_header_full.txt

if [ -z "$HTTP_CODE_CHECK" ]; then
  print_result "HTTP 状态码" "FAIL" "无法获取 HTTP 响应，域名可能未解析或服务未运行"
elif [ "$HTTP_CODE_CHECK" != "301" ]; then
  print_result "HTTP 状态码" "FAIL" "期望 301，实际返回 $HTTP_CODE_CHECK"
else
  print_result "HTTP 状态码" "PASS" "返回 301 Moved Permanently"
fi

EXPECTED_LOCATION="https://${DOMAIN}/"
if [ "$LOCATION" = "$EXPECTED_LOCATION" ]; then
  print_result "Location 头" "PASS" "正确跳转到 $LOCATION"
elif [ -n "$LOCATION" ]; then
  print_result "Location 头" "FAIL" "期望 $EXPECTED_LOCATION，实际为 $LOCATION"
else
  print_result "Location 头" "FAIL" "未获取到 Location 头"
fi

echo ""

# ========== 测试 2: /health 接口健康检查 ==========
bold "[测试 2] API 健康检查 — ${BASE_URL}/health"

HEALTH_JSON=$(curl -s --connect-timeout 5 --max-time 10 "${BASE_URL}/health" 2>/dev/null || true)

if [ -z "$HEALTH_JSON" ]; then
  print_result "API 响应" "FAIL" "无法连接到 ${BASE_URL}/health"
  echo ""
  bold "===== 检查结果: $(red "$FAIL 项失败") / $((PASS + FAIL)) 项 ====="
  exit 1
fi

# 检查 status
OVERALL_STATUS=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('status',''))" 2>/dev/null || echo "")
if [ "$OVERALL_STATUS" = "healthy" ]; then
  print_result "总体状态" "PASS" "status: healthy"
else
  print_result "总体状态" "FAIL" "期望 healthy，实际为 $OVERALL_STATUS"
fi

# 检查 database.status
DB_STATUS=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('database',{}).get('status',''))" 2>/dev/null || echo "")
DB_OPEN=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('database',{}).get('open_connections',''))" 2>/dev/null || echo "")
DB_IDLE=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('database',{}).get('idle_connections',''))" 2>/dev/null || echo "")
DB_INUSE=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('database',{}).get('in_use',''))" 2>/dev/null || echo "")

if [ "$DB_STATUS" = "ok" ]; then
  print_result "数据库" "PASS" "连接数: ${DB_OPEN} (空闲 ${DB_IDLE}, 使用中 ${DB_INUSE})"
else
  print_result "数据库" "FAIL" "期望 ok，实际为 $DB_STATUS"
fi

# 检查 redis.status
REDIS_STATUS=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('redis',{}).get('status',''))" 2>/dev/null || echo "")
REDIS_LATENCY=$(echo "$HEALTH_JSON" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('checks',{}).get('redis',{}).get('latency',''))" 2>/dev/null || echo "")

if [ "$REDIS_STATUS" = "ok" ]; then
  print_result "Redis" "PASS" "延迟: ${REDIS_LATENCY:-N/A}"
else
  print_result "Redis" "FAIL" "期望 ok，实际为 $REDIS_STATUS"
fi

echo ""
bold "===== 检查结果: $([ "$FAIL" -eq 0 ] && green "全部通过") || $([ "$FAIL" -gt 0 ] && echo "$(red "$FAIL 项失败")") / $((PASS + FAIL)) 项 ====="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi

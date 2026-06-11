#!/bin/bash
# ============================================================
# TokenHub 日志收集 + 告警配置
# 部署: Prometheus + Grafana + AlertManager
# 使用: sudo bash deploy-monitoring.sh [域名]
# 默认域名: tokenhub.com
# ============================================================
set -e

DOMAIN="${1:-tokenshopai.com}"
INSTALL_DIR="/root/tokenhub"
MONITOR_DIR="/root/tokenhub/monitoring"

echo "=========================================="
echo "  TokenHub 监控告警部署"
echo "  域名: ${DOMAIN}"
echo "  组件: Prometheus + Grafana + AlertManager"
echo "=========================================="

# ===== 1. 创建监控目录 =====
echo "[1/6] 创建监控目录..."
mkdir -p ${MONITOR_DIR}/{prometheus,grafana,alertmanager}

# ===== 2. Prometheus 配置 =====
echo "[2/6] 配置 Prometheus..."
cat > ${MONITOR_DIR}/prometheus/prometheus.yml <<EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

# 告警规则文件
rule_files:
  - "alerts/*.yml"

# AlertManager 配置
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']

# 采集目标
scrape_configs:
  # TokenHub 后端指标
  - job_name: 'tokenhub'
    metrics_path: '/metrics'
    static_configs:
      - targets: ['localhost:8080']
        labels:
          service: 'tokenhub-api'

  # Node Exporter (系统指标)
  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
        labels:
          service: 'tokenhub-server'

  # PostgreSQL 指标
  - job_name: 'postgresql'
    static_configs:
      - targets: ['localhost:9187']
        labels:
          service: 'tokenhub-db'

  # Redis 指标
  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']
        labels:
          service: 'tokenhub-cache'

  # Nginx 指标
  - job_name: 'nginx'
    static_configs:
      - targets: ['localhost:9113']
        labels:
          service: 'tokenhub-proxy'

  # Prometheus 自身
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF

# ===== 告警规则 =====
mkdir -p ${MONITOR_DIR}/prometheus/alerts
cat > ${MONITOR_DIR}/prometheus/alerts/tokenhub.yml <<'EOF'
groups:
  - name: tokenhub
    rules:
      # 服务不可用
      - alert: TokenHubServiceDown
        expr: up{job="tokenhub"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "TokenHub API 服务不可用"
          description: "TokenHub API 已停止响应超过 1 分钟"

      # 高错误率
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API 错误率超过 5%"
          description: "最近 5 分钟 API 5xx 错误率过高"

      # 高延迟
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "API P99 延迟超过 5 秒"
          description: "当前 P99 延迟 {{ $value }} 秒"

      # 余额不足
      - alert: LowBalance
        expr: tokenhub_user_balance_cents < 100
        for: 10m
        labels:
          severity: info
        annotations:
          summary: "用户余额不足 1 元"

      # 磁盘空间不足
      - alert: DiskSpaceLow
        expr: node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} < 0.15
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "磁盘空间不足 15%"
          description: "磁盘剩余空间过低"

      # 内存不足
      - alert: MemoryLow
        expr: node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes < 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "可用内存不足 10%"

      # PostgreSQL 连接数过高
      - alert: PGHighConnections
        expr: pg_stat_activity_count > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "PostgreSQL 活跃连接数过高: {{ $value }}"

      # Redis 内存过高
      - alert: RedisHighMemory
        expr: redis_memory_used_bytes / redis_memory_max_bytes > 0.85
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Redis 内存使用超过 85%"
EOF

# ===== 3. AlertManager 配置 =====
echo "[3/6] 配置 AlertManager..."
cat > ${MONITOR_DIR}/alertmanager/alertmanager.yml <<EOF
global:
  resolve_timeout: 5m

# 告警路由
route:
  group_by: ['alertname', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'default'
  routes:
    # 严重告警 → 立即通知
    - match:
        severity: critical
      receiver: 'critical'
      repeat_interval: 1h
    # 警告 → 工作时间通知
    - match:
        severity: warning
      receiver: 'warning'

receivers:
  - name: 'default'
    webhook_configs:
      - url: 'http://localhost:8080/admin/notifications/alert'
        send_resolved: true

  - name: 'critical'
    # 邮件通知（需要邮件服务配置）
    email_configs:
      - to: 'admin@${DOMAIN}'
        from: 'alertmanager@${DOMAIN}'
        smarthost: 'smtp.${DOMAIN}:465'
        auth_username: 'alertmanager@${DOMAIN}'
        auth_password: 'CHANGE_ME'
        headers:
          Subject: '[严重] TokenHub 告警: {{ .GroupLabels.alertname }}'
    # Webhook (企业微信/钉钉/飞书)
    webhook_configs:
      - url: 'http://localhost:8080/admin/notifications/alert'
        send_resolved: true

  - name: 'warning'
    email_configs:
      - to: 'admin@${DOMAIN}'
        from: 'alertmanager@${DOMAIN}'
        smarthost: 'smtp.${DOMAIN}:465'
        auth_username: 'alertmanager@${DOMAIN}'
        auth_password: 'CHANGE_ME'
        headers:
          Subject: '[警告] TokenHub 告警: {{ .GroupLabels.alertname }}'
EOF

# ===== 4. Grafana 配置 =====
echo "[4/6] 配置 Grafana..."
cat > ${MONITOR_DIR}/grafana/grafana.ini <<EOF
[server]
domain = ${DOMAIN}
root_url = https://${DOMAIN}/grafana/

[security]
admin_user = admin
admin_password = TokenHub@2026

[auth.anonymous]
enabled = false

[paths]
data = /var/lib/grafana
logs = /var/log/grafana
plugins = /var/lib/grafana/plugins

[log]
mode = file
level = info
EOF

# Grafana Dashboard JSON (TokenHub 概览)
mkdir -p ${MONITOR_DIR}/grafana/dashboards
cat > ${MONITOR_DIR}/grafana/dashboards/tokenhub-overview.json <<'DASHBOARD'
{
  "dashboard": {
    "title": "TokenHub 概览",
    "tags": ["tokenhub"],
    "timezone": "browser",
    "panels": [
      {
        "title": "QPS",
        "type": "stat",
        "gridPos": {"h": 4, "w": 6, "x": 0, "y": 0},
        "targets": [{"expr": "sum(rate(http_requests_total[1m]))", "legendFormat": "QPS"}]
      },
      {
        "title": "P99 延迟",
        "type": "stat",
        "gridPos": {"h": 4, "w": 6, "x": 6, "y": 0},
        "targets": [{"expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))", "legendFormat": "P99"}]
      },
      {
        "title": "成功率",
        "type": "stat",
        "gridPos": {"h": 4, "w": 6, "x": 12, "y": 0},
        "targets": [{"expr": "sum(rate(http_requests_total{status=~\"2..\"}[5m])) / sum(rate(http_requests_total[5m])) * 100", "legendFormat": "成功率%"}]
      },
      {
        "title": "活跃用户",
        "type": "stat",
        "gridPos": {"h": 4, "w": 6, "x": 18, "y": 0},
        "targets": [{"expr": "tokenhub_active_users", "legendFormat": "活跃用户"}]
      },
      {
        "title": "请求趋势",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 4},
        "targets": [{"expr": "sum(rate(http_requests_total[1m])) by (status)", "legendFormat": "{{status}}"}]
      },
      {
        "title": "Token 消耗",
        "type": "timeseries",
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 4},
        "targets": [{"expr": "sum(rate(tokenhub_tokens_total[5m])) by (model)", "legendFormat": "{{model}}"}]
      },
      {
        "title": "模型调用分布",
        "type": "piechart",
        "gridPos": {"h": 8, "w": 8, "x": 0, "y": 12},
        "targets": [{"expr": "sum(rate(http_requests_total[1h])) by (model)", "legendFormat": "{{model}}"}]
      },
      {
        "title": "系统资源",
        "type": "gauge",
        "gridPos": {"h": 8, "w": 8, "x": 8, "y": 12},
        "targets": [
          {"expr": "100 - (node_cpu_seconds_total{mode=\"idle\"} * 100)", "legendFormat": "CPU%"},
          {"expr": "(1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100", "legendFormat": "MEM%"}
        ]
      },
      {
        "title": "数据库连接",
        "type": "stat",
        "gridPos": {"h": 8, "w": 8, "x": 16, "y": 12},
        "targets": [{"expr": "pg_stat_activity_count", "legendFormat": "活跃连接"}]
      }
    ]
  }
}
DASHBOARD

# ===== 5. Docker Compose 启动监控栈 =====
echo "[5/6] 生成 Docker Compose 配置..."
cat > ${MONITOR_DIR}/docker-compose.yml <<EOF
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:v2.51.0
    container_name: tokenhub-prometheus
    restart: always
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus:/etc/prometheus
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=30d'
    network_mode: host

  alertmanager:
    image: prom/alertmanager:v0.27.0
    container_name: tokenhub-alertmanager
    restart: always
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager:/etc/alertmanager
    command:
      - '--config.file=/etc/alertmanager/alertmanager.yml'
    network_mode: host

  grafana:
    image: grafana/grafana:10.4.1
    container_name: tokenhub-grafana
    restart: always
    ports:
      - "3000:3000"
    volumes:
      - ./grafana:/etc/grafana
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=TokenHub@2026
      - GF_SERVER_ROOT_URL=https://${DOMAIN}/grafana/
    network_mode: host

  node-exporter:
    image: prom/node-exporter:v1.7.0
    container_name: tokenhub-node-exporter
    restart: always
    ports:
      - "9100:9100"
    volumes:
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
      - /:/rootfs:ro
    command:
      - '--path.procfs=/host/proc'
      - '--path.sysfs=/host/sys'
      - '--path.rootfs=/rootfs'
    network_mode: host

  postgres-exporter:
    image: prometheuscommunity/postgres-exporter:v0.15.0
    container_name: tokenhub-pg-exporter
    restart: always
    ports:
      - "9187:9187"
    environment:
      - DATA_SOURCE_NAME=postgresql://tokenhub:tokenhub_secret@localhost:5432/tokenhub?sslmode=disable
    network_mode: host

  redis-exporter:
    image: oliver006/redis_exporter:v1.59.0
    container_name: tokenhub-redis-exporter
    restart: always
    ports:
      - "9121:9121"
    environment:
      - REDIS_ADDR=localhost:6379
    network_mode: host

volumes:
  prometheus-data:
  grafana-data:
EOF

# ===== 6. 启动监控栈 =====
echo "[6/6] 启动监控服务..."

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "安装 Docker..."
    curl -fsSL https://get.docker.com | sh
    systemctl enable docker
    systemctl start docker
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo "安装 Docker Compose..."
    if command -v apt &> /dev/null; then
        apt install -y docker-compose-plugin
    elif command -v yum &> /dev/null; then
        yum install -y docker-compose-plugin
    fi
fi

cd ${MONITOR_DIR}

# 使用 docker compose v2 或 docker-compose v1
if docker compose version &> /dev/null; then
    docker compose up -d
else
    docker-compose up -d
fi

echo "等待服务启动..."
sleep 10

# 验证
echo ""
echo "验证服务状态..."
curl -s http://localhost:9090/-/healthy > /dev/null && echo "✅ Prometheus 运行正常" || echo "❌ Prometheus 启动失败"
curl -s http://localhost:9093/-/healthy > /dev/null && echo "✅ AlertManager 运行正常" || echo "❌ AlertManager 启动失败"
curl -s http://localhost:3000/api/health > /dev/null && echo "✅ Grafana 运行正常" || echo "❌ Grafana 启动失败"

# ===== 配置 Nginx 代理 Grafana =====
echo ""
echo "配置 Nginx 代理监控面板..."
NGINX_CONF="/etc/nginx/conf.d/tokenhub.conf"
[ ! -f "$NGINX_CONF" ] && NGINX_CONF="/etc/nginx/conf.d/${DOMAIN}.conf"
[ ! -f "$NGINX_CONF" ] && NGINX_CONF="/etc/nginx/sites-available/${DOMAIN}"

if [ -f "$NGINX_CONF" ] && ! grep -q "/grafana/" "$NGINX_CONF"; then
    # 在 Nginx 配置中添加 Grafana 代理
    sed -i "/location ~ \/\\\./i\\
    # Grafana 监控面板\n    location /grafana/ {\n        proxy_pass http://127.0.0.1:3000/;\n        proxy_set_header Host \\\$host;\n        proxy_set_header X-Real-IP \\\$remote_addr;\n        proxy_set_header X-Forwarded-For \\\$proxy_add_x_forwarded_for;\n        proxy_set_header X-Forwarded-Proto \\\$scheme;\n    }\n" "$NGINX_CONF"

    nginx -t 2>/dev/null && systemctl reload nginx
    echo "✅ Grafana 已通过 Nginx 代理"
else
    echo "⚠️  请手动配置 Nginx 代理 Grafana"
fi

# ===== Systemd 服务（可选，不用 Docker） =====
cat > /etc/systemd/system/tokenhub-monitoring.service <<EOF
[Unit]
Description=TokenHub Monitoring Stack
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=${MONITOR_DIR}
ExecStart=/usr/bin/docker compose -f ${MONITOR_DIR}/docker-compose.yml up -d
ExecStop=/usr/bin/docker compose -f ${MONITOR_DIR}/docker-compose.yml down

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

echo ""
echo "=========================================="
echo "  监控告警部署完成！"
echo ""
echo "  Prometheus:   http://localhost:9090"
echo "  AlertManager: http://localhost:9093"
echo "  Grafana:       http://localhost:3000"
echo "                 https://${DOMAIN}/grafana/"
echo ""
echo "  Grafana 登录:"
echo "    用户名: admin"
echo "    密码: TokenHub@2026"
echo ""
echo "  配置文件:"
echo "    Prometheus:   ${MONITOR_DIR}/prometheus/prometheus.yml"
echo "    告警规则:     ${MONITOR_DIR}/prometheus/alerts/tokenhub.yml"
echo "    AlertManager: ${MONITOR_DIR}/alertmanager/alertmanager.yml"
echo ""
echo "  ⚠️  请修改 AlertManager 邮件配置:"
echo "    vi ${MONITOR_DIR}/alertmanager/alertmanager.yml"
echo ""
echo "  添加企业微信/钉钉 Webhook:"
echo "    编辑 alertmanager.yml 的 webhook_configs.url"
echo ""
echo "  查看告警:"
echo "    http://localhost:9093/#/alerts"
echo ""
echo "  管理命令:"
echo "    cd ${MONITOR_DIR} && docker compose up -d    # 启动"
echo "    cd ${MONITOR_DIR} && docker compose down      # 停止"
echo "    docker logs tokenhub-prometheus               # 日志"
echo "=========================================="

# TokenHub 服务器部署完整指南

## 目录
1. [服务器准备](#1-服务器准备)
2. [域名配置](#2-域名配置)
3. [一键部署](#3-一键部署)
4. [SSL证书](#4-ssl证书)
5. [支付渠道配置](#5-支付渠道配置)
6. [运维管理](#6-运维管理)

---

## 1. 服务器准备

### 推荐配置

| 规模 | CPU | 内存 | 硬盘 | 带宽 | 月费参考 |
|------|-----|------|------|------|---------|
| 初期 | 2核 | 4GB | 50GB SSD | 5Mbps | ¥200-400 |
| 中期 | 4核 | 8GB | 100GB SSD | 10Mbps | ¥500-800 |
| 大型 | 8核 | 16GB | 200GB SSD | 20Mbps | ¥1000-2000 |

### 推荐云服务商
- 腾讯云 CVM（国内用户首选）
- 阿里云 ECS
- AWS EC2（海外用户）
- Vultr / DigitalOcean（性价比高）

### 操作系统
- Ubuntu 22.04 LTS（推荐）
- Debian 12
- CentOS Stream 8+

### 安全组/防火墙开放端口
| 端口 | 协议 | 用途 |
|------|------|------|
| 22 | TCP | SSH |
| 80 | TCP | HTTP |
| 443 | TCP | HTTPS |
| 5432 | TCP | PostgreSQL（仅内网） |
| 6379 | TCP | Redis（仅内网） |
| 8080 | TCP | 后端API（仅内网，Nginx代理） |

---

## 2. 域名配置

### 2.1 购买域名
- 阿里云万网 / 腾讯云DNSPod / Namecheap / Cloudflare
- 建议选择 `.com` / `.ai` / `.io` 后缀

### 2.2 DNS 解析
登录域名管理后台，添加 A 记录：

```
类型    主机记录    记录值
A       @          你的服务器IP
A       www        你的服务器IP
A       api        你的服务器IP
```

### 2.3 ICP 备案（国内服务器必须）
- 如果服务器在中国大陆，域名必须完成 ICP 备案
- 腾讯云/阿里云提供免费备案服务
- 备案周期约 7-20 天
- 备案期间网站不可访问，但可开发调试

---

## 3. 一键部署

### 3.1 上传代码到服务器

```bash
# 方式1: Git 克隆
git clone https://github.com/yourname/tokenhub.git /opt/tokenhub

# 方式2: SCP 上传
scp -r ./tokenhub root@your-server-ip:/opt/tokenhub
```

### 3.2 执行部署脚本

```bash
sudo bash /opt/tokenhub/deploy/deploy.sh tokenhub.com admin@tokenhub.com
```

脚本会自动完成：
- 安装 PostgreSQL、Redis、Nginx、Go、Node.js
- 创建数据库并初始化表结构
- 编译后端 Go 二进制
- 编译前端 Vue 项目
- 配置 systemd 服务（开机自启、自动重启）
- 配置 Nginx 反向代理

### 3.3 Docker Compose 部署（可选）

```bash
cd /opt/tokenhub
docker-compose -f deploy/docker/docker-compose.yml up -d
```

---

## 4. SSL 证书

### 4.1 Let's Encrypt 免费证书

```bash
sudo certbot --nginx -d tokenhub.com -d www.tokenhub.com
```

### 4.2 自动续期

```bash
# 测试续期
sudo certbot renew --dry-run

# Certbot 会自动添加 cron 任务续期
# 查看续期定时器
sudo systemctl list-timers | grep certbot
```

### 4.3 商业证书（EV/OV）

如果需要更高信任等级，可购买商业 SSL 证书：
- DigiCert / GlobalSign / Sectigo
- EV 证书显示企业名称，提升用户信任
- 价格 ¥1000-5000/年

---

## 5. 支付渠道配置

### 5.1 支付宝商户接入

1. 访问 https://open.alipay.com
2. 创建应用 → 获取 AppID
3. 配置应用公钥/私钥
4. 设置回调地址: `https://tokenhub.com/auth/callback/alipay`
5. 编辑 `configs/config.yaml` 填入密钥

### 5.2 微信支付商户接入

1. 访问 https://pay.weixin.qq.com
2. 申请商户号 (MchID)
3. 设置 API 密钥
4. 下载商户证书
5. 配置回调地址: `https://tokenhub.com/auth/callback/wechat`

### 5.3 Stripe 接入（海外收款）

1. 访问 https://dashboard.stripe.com
2. 获取 Publishable Key 和 Secret Key
3. 设置 Webhook Secret
4. 回调地址: `https://tokenhub.com/auth/callback/stripe`

### 5.4 重启服务使配置生效

```bash
sudo systemctl restart tokenhub
```

---

## 6. 运维管理

### 6.1 常用命令

```bash
# 查看服务状态
sudo systemctl status tokenhub

# 查看实时日志
sudo journalctl -u tokenhub -f

# 重启服务
sudo systemctl restart tokenhub

# 重新加载 Nginx
sudo nginx -s reload

# 查看数据库连接数
sudo -u postgres psql -d tokenhub -c "SELECT count(*) FROM pg_stat_activity;"

# 查看 Redis 内存使用
redis-cli info memory | grep used_memory_human
```

### 6.2 数据库备份

```bash
# 手动备份
sudo -u postgres pg_dump tokenhub > /backup/tokenhub_$(date +%Y%m%d).sql

# 自动每日备份（添加到 crontab）
echo "0 2 * * * sudo -u postgres pg_dump tokenhub | gzip > /backup/tokenhub_\$(date +\%Y\%m\%d).sql.gz" | sudo crontab -
```

### 6.3 监控告警

```bash
# 安装 Prometheus + Grafana（可选）
sudo apt install prometheus grafana-server

# 或使用云监控
# 腾讯云云监控 / 阿里云云监控
```

### 6.4 更新部署

```bash
cd /opt/tokenhub

# 拉取最新代码
git pull origin main

# 重新编译后端
cd backend && go build -o tokenhub-server ./cmd/server/

# 重新编译前端
cd ../frontend && npm install && npm run build

# 更新静态文件
cp -r dist/* /var/www/tokenhub/frontend/

# 重启后端
sudo systemctl restart tokenhub
```

### 6.5 余额预警脚本

在 crontab 中添加余额检查：
```bash
# 每小时检查用户余额，低于阈值发送通知
0 * * * * curl -s http://localhost:8080/admin/report/summary -H "Authorization: Bearer YOUR_ADMIN_TOKEN" | python3 /opt/tokenhub/scripts/balance_alert.py
```

---

## 常见问题

### Q: 前端页面空白
检查 Nginx 配置中 `root` 指向的目录是否有 `dist/` 文件夹，以及 Vue Router 的 `base` 配置。

### Q: API 返回 401
确认 `Authorization: Bearer <token>` 头是否正确传递。检查 JWT 过期时间。

### Q: WebSocket 连接失败
确认 Nginx 配置中 `/ws/` 的 `proxy_set_header Upgrade` 已正确设置。

### Q: 数据库连接失败
检查 `pg_hba.conf` 是否允许本地连接：
```
local   tokenhub   tokenhub   md5
host    tokenhub   tokenhub   127.0.0.1/32  md5
```

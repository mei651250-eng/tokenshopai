# ============================================
# TokenHub 一键部署脚本 (Windows Server)
# 通过 git clone 拉取代码，自动编译部署
# 以管理员身份运行: powershell -ExecutionPolicy Bypass -File .\deploy-full.ps1
# ============================================

$ErrorActionPreference = "Stop"
$ProjectDir = "C:\tokenhub"
$RepoUrl = "https://github.com/mei651250-eng/tokenshopai.git"

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  TokenHub AI API Gateway 一键部署" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan

# ---------- 1. 安装 Chocolatey ----------
Write-Host "`n[1/8] 安装 Chocolatey 包管理器..." -ForegroundColor Yellow
if (!(Get-Command choco -ErrorAction SilentlyContinue)) {
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
    refreshenv 2>$null; $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
Write-Host "Chocolatey 已就绪" -ForegroundColor Green

# ---------- 2. 安装 Git ----------
Write-Host "`n[2/8] 安装 Git..." -ForegroundColor Yellow
if (!(Get-Command git -ErrorAction SilentlyContinue)) {
    choco install git -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
Write-Host "Git 已就绪" -ForegroundColor Green

# ---------- 3. 克隆项目代码 ----------
Write-Host "`n[3/8] 克隆项目代码..." -ForegroundColor Yellow
if (Test-Path $ProjectDir) {
    Write-Host "目录已存在，拉取最新代码..." -ForegroundColor Yellow
    cd $ProjectDir; git pull
} else {
    git clone $RepoUrl $ProjectDir
}
cd $ProjectDir
Write-Host "代码已就绪" -ForegroundColor Green

# ---------- 4. 安装 Go ----------
Write-Host "`n[4/8] 安装 Go 1.22..." -ForegroundColor Yellow
if (!(Get-Command go -ErrorAction SilentlyContinue)) {
    choco install golang -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
Write-Host "Go 已就绪: $(go version)" -ForegroundColor Green

# ---------- 5. 安装 Node.js ----------
Write-Host "`n[5/8] 安装 Node.js 20..." -ForegroundColor Yellow
if (!(Get-Command node -ErrorAction SilentlyContinue)) {
    choco install nodejs-lts -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
Write-Host "Node.js 已就绪: $(node --version)" -ForegroundColor Green

# ---------- 6. 安装 PostgreSQL ----------
Write-Host "`n[6/8] 安装 PostgreSQL 16..." -ForegroundColor Yellow
if (!(Get-Command psql -ErrorAction SilentlyContinue)) {
    $pgPwd = "TokenHub2026!"
    choco install postgresql16 --params '/Password:TokenHub2026! /Port:5432' -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
Write-Host "PostgreSQL 已就绪" -ForegroundColor Green

# ---------- 7. 安装 Redis ----------
Write-Host "`n[7/8] 安装 Redis..." -ForegroundColor Yellow
if (!(Get-Command redis-server -ErrorAction SilentlyContinue)) {
    choco install redis-64 -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}
# 启动 Redis
$redisProc = Get-Process redis-server -ErrorAction SilentlyContinue
if (!$redisProc) {
    Start-Process redis-server -WindowStyle Hidden
    Start-Sleep -Seconds 2
}
Write-Host "Redis 已就绪" -ForegroundColor Green

# ---------- 8. 初始化数据库 ----------
Write-Host "`n[8/8] 初始化数据库..." -ForegroundColor Yellow
$env:PGPASSWORD = "TokenHub2026!"
$psqlPath = Get-Command psql -ErrorAction SilentlyContinue
if ($psqlPath) {
    # 创建数据库
    & psql -U postgres -c "SELECT 1 FROM pg_database WHERE datname='tokenhub'" 2>$null | Out-Null
    $dbExists = & psql -U postgres -t -c "SELECT 1 FROM pg_database WHERE datname='tokenhub'" 2>$null
    if ($dbExists -match "1") {
        Write-Host "数据库 tokenhub 已存在" -ForegroundColor Yellow
    } else {
        & psql -U postgres -c "CREATE DATABASE tokenhub;"
        Write-Host "数据库 tokenhub 创建成功" -ForegroundColor Green
    }
} else {
    Write-Host "psql 未找到，跳过数据库初始化" -ForegroundColor Yellow
}

# ---------- 编译后端 ----------
Write-Host "`n=== 编译后端 ===" -ForegroundColor Cyan
cd "$ProjectDir\backend"
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go mod tidy
go build -o tokenhub-server.exe ./cmd/server/
if (Test-Path "tokenhub-server.exe") {
    Write-Host "后端编译成功: tokenhub-server.exe" -ForegroundColor Green
} else {
    Write-Host "后端编译失败！" -ForegroundColor Red
    exit 1
}

# ---------- 编译前端 ----------
Write-Host "`n=== 编译前端 ===" -ForegroundColor Cyan
cd "$ProjectDir\frontend"
npm install --registry https://registry.npmmirror.com
npm run build
if (Test-Path "dist") {
    Write-Host "前端编译成功" -ForegroundColor Green
} else {
    Write-Host "前端编译失败！" -ForegroundColor Red
    exit 1
}

# ---------- 配置后端 ----------
Write-Host "`n=== 配置后端 ===" -ForegroundColor Cyan
cd "$ProjectDir\configs"
if (Test-Path "config.yaml") {
    $config = Get-Content "config.yaml" -Raw
    # 更新数据库密码
    $config = $config -replace 'password:.*', "password: TokenHub2026!"
    $config = $config -replace 'host: localhost', "host: 127.0.0.1"
    Set-Content "config.yaml" $config
    Write-Host "配置文件已更新" -ForegroundColor Green
}

# ---------- 注册为 Windows 服务 ----------
Write-Host "`n=== 注册 Windows 服务 ===" -ForegroundColor Cyan

# 使用 NSSM 注册服务
if (!(Get-Command nssm -ErrorAction SilentlyContinue)) {
    choco install nssm -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# 后端服务
$backendExists = & nssm get tokenhub-backend DisplayName 2>&1
if ($backendExists -match "tokenhub-backend") {
    Write-Host "后端服务已注册" -ForegroundColor Yellow
} else {
    & nssm install tokenhub-backend "$ProjectDir\backend\tokenhub-server.exe"
    & nssm set tokenhub-backend AppDirectory "$ProjectDir\backend"
    & nssm set tokenhub-backend Start SERVICE_AUTO_START
    & nssm set tokenhub-backend AppEnvironment "GIN_MODE=release"
    Write-Host "后端服务已注册" -ForegroundColor Green
}
& nssm start tokenhub-backend 2>$null

# Redis 服务
$redisExists = & nssm get tokenhub-redis DisplayName 2>&1
if ($redisExists -match "tokenhub-redis") {
    Write-Host "Redis 服务已注册" -ForegroundColor Yellow
} else {
    $redisPath = (Get-Command redis-server).Source
    & nssm install tokenhub-redis $redisPath
    & nssm set tokenhub-redis Start SERVICE_AUTO_START
    Write-Host "Redis 服务已注册" -ForegroundColor Green
}
& nssm start tokenhub-redis 2>$null

# ---------- 安装 Nginx ----------
Write-Host "`n=== 安装 Nginx ===" -ForegroundColor Cyan
if (!(Test-Path "C:\nginx")) {
    [Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-WebRequest -Uri "https://nginx.org/download/nginx-1.24.0.zip" -OutFile "C:\nginx.zip" -UseBasicParsing
    Expand-Archive -Path "C:\nginx.zip" -DestinationPath "C:\" -Force
    Rename-Item "C:\nginx-1.24.0" "C:\nginx" -ErrorAction SilentlyContinue
    Remove-Item "C:\nginx.zip" -ErrorAction SilentlyContinue
}

# 写入 Nginx 配置
$nginxConf = @"
worker_processes  1;

events {
    worker_connections  1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout  65;

    server {
        listen       80;
        server_name  _;

        # 前端静态文件
        location / {
            root   $ProjectDir\frontend\dist;
            index  index.html;
            try_files `$uri `$uri/ /index.html;
        }

        # 后端 API 代理
        location /api/ {
            proxy_pass http://127.0.0.1:8080/api/;
            proxy_set_header Host `$host;
            proxy_set_header X-Real-IP `$remote_addr;
            proxy_set_header X-Forwarded-For `$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto `$scheme;
        }

        # 健康检查
        location /health {
            proxy_pass http://127.0.0.1:8080/health;
        }
    }
}
"@
Set-Content "C:\nginx\conf\nginx.conf" $nginxConf -Encoding UTF8

# Nginx 服务
$nginxExists = & nssm get tokenhub-nginx DisplayName 2>&1
if ($nginxExists -match "tokenhub-nginx") {
    & nssm restart tokenhub-nginx
} else {
    & nssm install tokenhub-nginx "C:\nginx\nginx.exe"
    & nssm set tokenhub-nginx AppDirectory "C:\nginx"
    & nssm set tokenhub-nginx Start SERVICE_AUTO_START
    & nssm start tokenhub-nginx
}

Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "  部署完成！" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""
Write-Host "前端访问: http://47.98.169.227" -ForegroundColor Cyan
Write-Host "API 地址: http://47.98.169.227/api/" -ForegroundColor Cyan
Write-Host "健康检查: http://47.98.169.227/health" -ForegroundColor Cyan
Write-Host ""
Write-Host "管理员账号: admin@tokenhub.com / Admin@2026" -ForegroundColor Yellow
Write-Host ""
Write-Host "服务管理命令:" -ForegroundColor Yellow
Write-Host "  nssm status tokenhub-backend" -ForegroundColor White
Write-Host "  nssm restart tokenhub-backend" -ForegroundColor White
Write-Host "  nssm status tokenhub-nginx" -ForegroundColor White
Write-Host "  nssm restart tokenhub-nginx" -ForegroundColor White
Write-Host ""
Write-Host "日志位置:" -ForegroundColor Yellow
Write-Host "  后端: C:\tokenhub\backend\tokenhub-server.log" -ForegroundColor White
Write-Host "  Nginx: C:\nginx\logs\" -ForegroundColor White

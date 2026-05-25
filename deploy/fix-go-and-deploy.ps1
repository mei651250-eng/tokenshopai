# =============================================
# TokenHub 部署 - 修复Go安装 + 继续部署
# 在 ECS 管理员 PowerShell 中粘贴运行
# =============================================

$ErrorActionPreference = "Continue"
$ProjectDir = "C:\tokenhub"

Write-Host "=== 修复 Go 安装 ===" -ForegroundColor Cyan

# 清理失败的 choco 安装
choco uninstall golang -y 2>$null | Out-Null

# 用阿里云镜像下载 Go 1.22.10
Write-Host "从国内镜像下载 Go 1.22.10..." -ForegroundColor Yellow
[Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072

$goZip = "C:\go1.22.10.zip"
$downloaded = $false

# 尝试阿里云镜像
try {
    Write-Host "尝试阿里云镜像..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri "https://mirrors.aliyun.com/golang/go1.22.10.windows-amd64.zip" -OutFile $goZip -UseBasicParsing -TimeoutSec 120
    $downloaded = $true
} catch {
    Write-Host "阿里云镜像失败，尝试 golang.google.cn..." -ForegroundColor Yellow
}

# 尝试 golang 中国镜像
if (!$downloaded) {
    try {
        Invoke-WebRequest -Uri "https://golang.google.cn/dl/go1.22.10.windows-amd64.zip" -OutFile $goZip -UseBasicParsing -TimeoutSec 120
        $downloaded = $true
    } catch {
        Write-Host "golang.google.cn 也失败，尝试 go.dev 镜像..." -ForegroundColor Yellow
    }
}

# 尝试 studygolang 镜像
if (!$downloaded) {
    try {
        Invoke-WebRequest -Uri "https://dl.golang.google.cn/go1.22.10.windows-amd64.zip" -OutFile $goZip -UseBasicParsing -TimeoutSec 120
        $downloaded = $true
    } catch {
        Write-Host "所有镜像都失败！" -ForegroundColor Red
    }
}

if (!$downloaded) {
    Write-Host "无法下载 Go，请手动下载 go1.22.10.windows-amd64.zip 放到 C:\ 后重新运行" -ForegroundColor Red
    exit 1
}

# 解压
Write-Host "解压 Go..." -ForegroundColor Yellow
if (Test-Path "C:\Go") { Remove-Item -Recurse -Force "C:\Go" }
Expand-Archive -Path $goZip -DestinationPath "C:\" -Force
Remove-Item $goZip -ErrorAction SilentlyContinue

# 设置环境变量
$goBin = "C:\Go\bin"
$currentPath = [System.Environment]::GetEnvironmentVariable("Path","Machine")
if ($currentPath -notlike "*$goBin*") {
    [System.Environment]::SetEnvironmentVariable("Path", "$currentPath;$goBin", "Machine")
}
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
$env:GOPROXY = "https://goproxy.cn,direct"
[System.Environment]::SetEnvironmentVariable("GOPROXY", "https://goproxy.cn,direct", "Machine")

# 验证 Go
$goVer = & C:\Go\bin\go.exe version 2>&1
Write-Host "Go 安装成功: $goVer" -ForegroundColor Green

# ============================================
# 继续部署 - 编译后端
# ============================================
Write-Host "`n=== 编译后端 ===" -ForegroundColor Cyan
cd "$ProjectDir\backend"
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOPROXY = "https://goproxy.cn,direct"
C:\Go\bin\go.exe mod tidy
C:\Go\bin\go.exe build -o tokenhub-server.exe ./cmd/server/
if (Test-Path "tokenhub-server.exe") {
    Write-Host "后端编译成功" -ForegroundColor Green
} else {
    Write-Host "后端编译失败！" -ForegroundColor Red
    exit 1
}

# ============================================
# 编译前端
# ============================================
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

# ============================================
# 更新配置
# ============================================
Write-Host "`n=== 配置后端 ===" -ForegroundColor Cyan
$configFile = "$ProjectDir\configs\config.yaml"
if (Test-Path $configFile) {
    $config = Get-Content $configFile -Raw
    $config = $config -replace 'password:.*', "password: TokenHub2026!"
    $config = $config -replace 'host: localhost', "host: 127.0.0.1"
    Set-Content $configFile $config -Encoding UTF8
    Write-Host "配置已更新" -ForegroundColor Green
}

# ============================================
# 注册服务 (NSSM)
# ============================================
Write-Host "`n=== 注册服务 ===" -ForegroundColor Cyan
if (!(Get-Command nssm -ErrorAction SilentlyContinue)) {
    choco install nssm -y --no-progress
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# 后端服务
$svc = & nssm get tokenhub-backend DisplayName 2>&1
if ($svc -notmatch "tokenhub-backend") {
    & nssm install tokenhub-backend "$ProjectDir\backend\tokenhub-server.exe"
    & nssm set tokenhub-backend AppDirectory "$ProjectDir\backend"
    & nssm set tokenhub-backend Start SERVICE_AUTO_START
    & nssm set tokenhub-backend AppEnvironment "GIN_MODE=release"
    Write-Host "后端服务已注册" -ForegroundColor Green
}
& nssm start tokenhub-backend 2>$null

# Redis 服务
$svc = & nssm get tokenhub-redis DisplayName 2>&1
if ($svc -notmatch "tokenhub-redis") {
    $redisPath = (Get-Command redis-server -ErrorAction SilentlyContinue).Source
    if ($redisPath) {
        & nssm install tokenhub-redis $redisPath
        & nssm set tokenhub-redis Start SERVICE_AUTO_START
        Write-Host "Redis 服务已注册" -ForegroundColor Green
    }
}
& nssm start tokenhub-redis 2>$null

# ============================================
# 安装 Nginx
# ============================================
Write-Host "`n=== 安装 Nginx ===" -ForegroundColor Cyan
if (!(Test-Path "C:\nginx")) {
    [Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
    Invoke-WebRequest -Uri "https://nginx.org/download/nginx-1.24.0.zip" -OutFile "C:\nginx.zip" -UseBasicParsing
    Expand-Archive -Path "C:\nginx.zip" -DestinationPath "C:\" -Force
    Rename-Item "C:\nginx-1.24.0" "C:\nginx" -ErrorAction SilentlyContinue
    Remove-Item "C:\nginx.zip" -ErrorAction SilentlyContinue
}

# 写 Nginx 配置文件
$nginxDistPath = "$ProjectDir\frontend\dist"
$nginxConfFile = "C:\nginx\conf\nginx.conf"
$nginxContent = @"
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

        location / {
            root   $nginxDistPath;
            index  index.html;
            try_files `$uri `$uri/ /index.html;
        }

        location /api/ {
            proxy_pass http://127.0.0.1:8080/api/;
            proxy_set_header Host `$host;
            proxy_set_header X-Real-IP `$remote_addr;
            proxy_set_header X-Forwarded-For `$proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto `$scheme;
        }

        location /health {
            proxy_pass http://127.0.0.1:8080/health;
        }
    }
}
"@
Set-Content $nginxConfFile $nginxContent -Encoding UTF8

$svc = & nssm get tokenhub-nginx DisplayName 2>&1
if ($svc -notmatch "tokenhub-nginx") {
    & nssm install tokenhub-nginx "C:\nginx\nginx.exe"
    & nssm set tokenhub-nginx AppDirectory "C:\nginx"
    & nssm set tokenhub-nginx Start SERVICE_AUTO_START
    & nssm start tokenhub-nginx
} else {
    & nssm restart tokenhub-nginx
}

# ============================================
# 开放防火墙端口
# ============================================
Write-Host "`n=== 开放防火墙端口 ===" -ForegroundColor Cyan
New-NetFirewallRule -Name "TokenHub-HTTP" -DisplayName "TokenHub HTTP" -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 80 -ErrorAction SilentlyContinue
New-NetFirewallRule -Name "TokenHub-API" -DisplayName "TokenHub API" -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 8080 -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "  部署完成！" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""
Write-Host "前端: http://47.98.169.227" -ForegroundColor Cyan
Write-Host "API:  http://47.98.169.227/api/" -ForegroundColor Cyan
Write-Host "健康: http://47.98.169.227/health" -ForegroundColor Cyan
Write-Host ""
Write-Host "管理员: admin@tokenhub.com / Admin@2026" -ForegroundColor Yellow
Write-Host ""
Write-Host "!! 重要: 请在阿里云安全组开放 80 和 8080 端口 !!" -ForegroundColor Red

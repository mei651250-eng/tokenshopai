# ============================================
# ECS 初始化脚本 - 安装 OpenSSH Server
# 在 ECS 上以管理员 PowerShell 运行此脚本
# ============================================

Write-Host "=== 安装 OpenSSH Server ===" -ForegroundColor Cyan

# 安装 OpenSSH Server
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0

# 启动并设置自启
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic

# 防火墙放行 22 端口
New-NetFirewallRule -Name sshd -DisplayName 'OpenSSH Server' -Enabled True -Direction Inbound -Protocol TCP -Action Allow -LocalPort 22

# 允许密码认证
$sshdConfig = "C:\ProgramData\ssh\sshd_config"
if (Test-Path $sshdConfig) {
    (Get-Content $sshdConfig) -replace '#PasswordAuthentication yes','PasswordAuthentication yes' -replace '#PubkeyAuthentication yes','PubkeyAuthentication yes' | Set-Content $sshdConfig
    Restart-Service sshd
}

# 检测系统版本
Write-Host ""
Write-Host "=== 系统信息 ===" -ForegroundColor Cyan
Get-ComputerInfo | Select-Object OsName, OsVersion, OsArchitecture, CsTotalPhysicalMemory | Format-List

Write-Host ""
Write-Host "=== SSH 安装完成 ===" -ForegroundColor Green
Write-Host "SSH 端口: 22" -ForegroundColor Yellow
Write-Host "请确保阿里云安全组已开放 22 端口！" -ForegroundColor Yellow
Write-Host ""
Write-Host "现在可以让部署工具通过 SSH 连接了" -ForegroundColor Green

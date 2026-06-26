@echo off
cd /d %~dp0frontend
echo Building frontend...
call npx vite build
echo Build complete.
pause

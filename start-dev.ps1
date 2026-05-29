# AIRA 本地开发 — 一键启动脚本
# 用法：PowerShell 中执行 .\start-dev.ps1
#
# 启动三个进程，每个在独立窗口：
#   1. 后端 API server      (:3001)
#   2. 前端 Next.js dev     (:3000)
#   3. Worker               (无端口；负责处理 ingest 上传清洗、
#                            legacy 题目预处理。没启它 /upload 上传
#                            的任务会卡在 pending 永不动)

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "  AIRA 本地开发环境启动" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan

# 启动后端
Write-Host "`n[1/3] 启动后端 (Go)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\back'; go run ." -WindowStyle Normal

# 等后端先启动，让端口先占住
Start-Sleep -Seconds 3

# 启动 Worker（ingest 清洗 / 题目预处理）
Write-Host "[2/3] 启动 Worker (Go)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\back'; go run ./cmd/worker" -WindowStyle Normal

# 启动前端
Write-Host "[3/3] 启动前端 (Next.js)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\aira-web-4'; npm run dev:web" -WindowStyle Normal

Write-Host "`n======================================" -ForegroundColor Green
Write-Host "  前端: http://localhost:3000" -ForegroundColor Green
Write-Host "  后端: http://localhost:3001" -ForegroundColor Green
Write-Host "  Worker: 后台运行，无端口" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host "`n关闭时请关闭三个弹出的 PowerShell 窗口" -ForegroundColor Gray
Write-Host "或一次性清掉 3000/3001 端口占用：" -ForegroundColor Gray
Write-Host "  Get-NetTCPConnection -LocalPort 3000,3001 -State Listen -EA SilentlyContinue | ForEach-Object { Stop-Process -Id `$_.OwningProcess -Force }" -ForegroundColor DarkGray

# AIRA 本地开发 — 一键启动脚本
# 用法：PowerShell 中执行 .\start-dev.ps1

Write-Host "======================================" -ForegroundColor Cyan
Write-Host "  AIRA 本地开发环境启动" -ForegroundColor Cyan
Write-Host "======================================" -ForegroundColor Cyan

# 启动后端
Write-Host "`n[1/2] 启动后端 (Go)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\back'; go run ." -WindowStyle Normal

# 等后端先启动
Start-Sleep -Seconds 3

# 启动前端
Write-Host "[2/2] 启动前端 (Next.js)..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\aira-web-4'; npm run dev:web" -WindowStyle Normal

Write-Host "`n======================================" -ForegroundColor Green
Write-Host "  前端: http://localhost:3000" -ForegroundColor Green
Write-Host "  后端: http://localhost:3001" -ForegroundColor Green
Write-Host "======================================" -ForegroundColor Green
Write-Host "`n关闭时请关闭两个弹出的 PowerShell 窗口" -ForegroundColor Gray

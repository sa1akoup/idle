# 从源码启动本地开发环境
$ErrorActionPreference = "Stop"

Write-Host "启动后端 :8081 ..."
Start-Process -FilePath "go" -ArgumentList "run","." -WindowStyle Hidden -WorkingDirectory "D:/idle/backend"
Start-Sleep -Seconds 3
try { Invoke-RestMethod -Uri "http://localhost:8081/api/health" | Out-Null; Write-Host "后端就绪 http://localhost:8081" } catch { Write-Host "后端启动失败 $_" -ForegroundColor Red; exit 1 }

Write-Host "启动前端 :5173 ..."
Start-Process -FilePath "npm" -ArgumentList "run","dev" -WindowStyle Hidden -WorkingDirectory "D:/idle/frontend"
Start-Sleep -Seconds 4
Write-Host "前端就绪 http://localhost:5173"
Write-Host "服务已在后台运行。"

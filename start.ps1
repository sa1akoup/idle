# 从源码启动本地开发环境。
#
# 用法：
#   .\start.ps1           启动后端(:8081)与前端(:5173)，记录 PID
#   .\start.ps1 -Stop     停止本脚本启动的前后端
#
# 停止时优先验证 PID 文件中的入口进程和启动时间，再递归处理其子进程。
# 端口只用于检查和提示，不会因为端口命中就强制结束未知进程。
param(
    [switch]$Stop
)

$ErrorActionPreference = "Stop"

$ProjectRoot  = (Resolve-Path -LiteralPath $PSScriptRoot).Path
$ScriptPath   = (Resolve-Path -LiteralPath $PSCommandPath).Path
$BackendDir   = Join-Path $ProjectRoot "backend"
$FrontendDir  = Join-Path $ProjectRoot "frontend"
$BackendPort  = 8081
$FrontendPort = 5173
$PidDir       = Join-Path $ProjectRoot ".run"
$PidFile      = Join-Path $PidDir "dev.pid"

# 按端口反查正在监听该端口的进程 PID，仅用于就绪检查和停止后的告警。
# go run . 会再派生一个真正的监听进程（孙进程），入口进程树会一并记录和处理。
function Get-PidByPort([int]$port) {
    $line = netstat -ano -p tcp 2>$null | Select-String (":$port\s") | Select-String "LISTENING" | Select-Object -First 1
    if (-not $line) { return $null }
    if ($line.Line -match '(\d+)\s*$') { return [int]$Matches[1] }
    return $null
}

function Get-ProcessTreeIds([int]$rootPid) {
    $ids = [System.Collections.Generic.List[int]]::new()
    $pending = [System.Collections.Generic.Queue[int]]::new()
    $pending.Enqueue($rootPid)

    while ($pending.Count -gt 0) {
        $currentPid = $pending.Dequeue()
        if ($ids.Contains($currentPid)) { continue }
        $ids.Add($currentPid)

        $children = @(Get-CimInstance Win32_Process -Filter "ParentProcessId = $currentPid" -ErrorAction SilentlyContinue)
        foreach ($child in $children) {
            $pending.Enqueue([int]$child.ProcessId)
        }
    }

    return $ids.ToArray()
}

# 仅对已经验证过的入口进程递归结束其全部子孙进程。
function Stop-ProcessTree([int]$rootPid) {
    if (-not $rootPid -or $rootPid -le 0) { return }
    $children = Get-CimInstance Win32_Process -Filter "ParentProcessId = $rootPid" -ErrorAction SilentlyContinue
    foreach ($c in $children) { Stop-ProcessTree ([int]$c.ProcessId) }
    Stop-Process -Id $rootPid -Force -ErrorAction SilentlyContinue
}

function Get-RecordedProcess($record) {
    if (-not $record -or -not $record.pid -or -not $record.startTime) { return $null }
    try {
        $process = Get-Process -Id ([int]$record.pid) -ErrorAction Stop
        $expectedStart = [DateTimeOffset]::Parse([string]$record.startTime).ToUniversalTime()
        $actualStart = [DateTimeOffset]::new($process.StartTime.ToUniversalTime())
        if ([math]::Abs(($actualStart - $expectedStart).TotalSeconds) -gt 2) { return $null }
        return $process
    } catch {
        return $null
    }
}

function New-ProcessRecord([int]$processId) {
    $process = Get-Process -Id $processId -ErrorAction Stop
    [pscustomobject]@{
        pid       = [int]$process.Id
        startTime = $process.StartTime.ToUniversalTime().ToString("o")
    }
}

function Stop-RecordedProcess($record) {
    $process = Get-RecordedProcess $record
    if (-not $process) { return @() }
    $treePids = Get-ProcessTreeIds ([int]$process.Id)
    Stop-ProcessTree ([int]$process.Id)
    return $treePids
}

function Save-RunState($state) {
    New-Item -ItemType Directory -Force -Path $PidDir | Out-Null
    $state | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath $PidFile -Encoding UTF8
}

function Read-RunState {
    if (-not (Test-Path -LiteralPath $PidFile)) { return $null }
    try {
        $state = Get-Content -LiteralPath $PidFile -Raw | ConvertFrom-Json
        if ([string]$state.scriptPath -ne $ScriptPath) {
            Write-Warning "PID 文件不属于当前 start.ps1，未停止其中记录的进程。"
            return $null
        }
        return $state
    } catch {
        Write-Warning "PID 文件格式无法验证，未按 PID 文件停止任何进程。"
        return $null
    }
}

function Test-PortOwnedByTree([int]$port, [int]$rootPid) {
    $ownerPid = Get-PidByPort $port
    if (-not $ownerPid) { return $false }
    return (Get-ProcessTreeIds $rootPid) -contains $ownerPid
}

function Wait-ForBackend([int]$rootPid, [int]$timeoutSeconds) {
    $deadline = (Get-Date).AddSeconds($timeoutSeconds)
    do {
        if (Test-PortOwnedByTree $BackendPort $rootPid) {
            try {
                Invoke-RestMethod -Uri "http://localhost:$BackendPort/api/health" -TimeoutSec 2 | Out-Null
                return $true
            } catch {
                # 后端进程已监听，但健康检查可能仍在初始化。
            }
        }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    return $false
}

function Wait-ForPortOwnedByTree([int]$port, [int]$rootPid, [int]$timeoutSeconds) {
    $deadline = (Get-Date).AddSeconds($timeoutSeconds)
    do {
        if (Test-PortOwnedByTree $port $rootPid) { return $true }
        Start-Sleep -Seconds 1
    } while ((Get-Date) -lt $deadline)
    return $false
}

function Wait-ForPortFree([int]$port, [int]$timeoutSeconds) {
    $deadline = (Get-Date).AddSeconds($timeoutSeconds)
    do {
        if (-not (Get-PidByPort $port)) { return $true }
        Start-Sleep -Milliseconds 200
    } while ((Get-Date) -lt $deadline)
    return $false
}

function Assert-PortAvailable([int]$port) {
    $ownerPid = Get-PidByPort $port
    if ($ownerPid) {
        throw "端口 $port 已被 PID $ownerPid 占用，未启动本项目服务。"
    }
}

if ($Stop) {
    Write-Host "正在停止 start.ps1 记录的服务 ..."
    $state = Read-RunState
    $stoppedPids = @()

    if ($state) {
        foreach ($record in @($state.backend, $state.frontend)) {
            if (-not $record) { continue }
            $treePids = Stop-RecordedProcess $record
            if ($treePids.Count -eq 0) {
                Write-Warning "跳过 PID $($record.pid)：进程不存在或启动时间不匹配。"
                continue
            }
            $stoppedPids += $treePids
        }
    }

    if (Test-Path -LiteralPath $PidFile) {
        Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
    }

    foreach ($port in @($BackendPort, $FrontendPort)) {
        $ownerPid = Get-PidByPort $port
        if (-not $ownerPid) { continue }
        if ($stoppedPids -contains $ownerPid) {
            if (-not (Wait-ForPortFree $port 3)) {
                Write-Warning "端口 $port 仍由本次记录的 PID $ownerPid 占用，请手动检查。"
            }
            continue
        }
        Write-Warning "端口 $port 由未验证的 PID $ownerPid 占用，未自动停止该进程。"
    }

    if ($stoppedPids.Count -eq 0) {
        Write-Host "未找到可验证的本项目进程；未知端口占用仅作提示。"
    } else {
        Write-Host "已停止记录进程树: $($stoppedPids -join ', ')"
    }
    return
}

# --- 启动 ---
$backend = $null
$frontend = $null
$state = $null

try {
    Assert-PortAvailable $BackendPort
    Assert-PortAvailable $FrontendPort

    Write-Host "启动后端 :$BackendPort ..."
    $backend = Start-Process -FilePath "go" -ArgumentList "run", "." -WorkingDirectory $BackendDir -PassThru -ErrorAction Stop
    $state = [pscustomobject]@{
        version    = 1
        scriptPath = $ScriptPath
        backend    = New-ProcessRecord ([int]$backend.Id)
        frontend   = $null
    }
    Save-RunState $state

    if (-not (Wait-ForBackend ([int]$backend.Id) 20)) {
        throw "后端未在 $BackendPort 通过健康检查。"
    }
    Write-Host "后端就绪 http://localhost:$BackendPort"

    Write-Host "启动前端 :$FrontendPort ..."
    $frontend = Start-Process -FilePath "npm" -ArgumentList "run", "dev" -WorkingDirectory $FrontendDir -PassThru -ErrorAction Stop
    $state.frontend = New-ProcessRecord ([int]$frontend.Id)
    Save-RunState $state

    if (-not (Wait-ForPortOwnedByTree $FrontendPort ([int]$frontend.Id) 20)) {
        throw "前端未在 $FrontendPort 启动，或端口已被非本项目进程占用。"
    }
    Write-Host "前端就绪 http://localhost:$FrontendPort"
} catch {
    $message = $_.Exception.Message
    if ($state -and $state.frontend) { [void](Stop-RecordedProcess $state.frontend) }
    if ($state -and $state.backend) { [void](Stop-RecordedProcess $state.backend) }
    if (Test-Path -LiteralPath $PidFile) {
        Remove-Item -LiteralPath $PidFile -Force -ErrorAction SilentlyContinue
    }
    Write-Host "启动失败：$message" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "服务已启动。停止请运行:  .\start.ps1 -Stop"

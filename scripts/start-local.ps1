param(
    [int]$WebPort = 3100,
    [int]$ApiPort = 8081,
    [switch]$WithBackend,
    [switch]$NoBrowser
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$logDir = Join-Path $projectRoot 'logs'
$binDir = Join-Path $projectRoot 'bin'
$statePath = Join-Path $logDir 'local-processes.json'

function Save-LocalProcess($Process, [string]$Executable, [string]$CommandMarker = '') {
    $script:started += [pscustomobject]@{
        pid = $Process.Id
        started_at = $Process.StartTime.ToUniversalTime().ToString('o')
        executable = $Executable
        command_marker = $CommandMarker
    }
    ConvertTo-Json -InputObject @($script:started) | Set-Content -LiteralPath $statePath -Encoding utf8
}

function Wait-Endpoint([string]$Url, [int]$Seconds = 60) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-WebRequest -Uri $Url -TimeoutSec 3 -UseBasicParsing
            if ($response.StatusCode -eq 200) { return }
        } catch { }
        Start-Sleep -Milliseconds 500
    }
    throw "Service did not become ready: $Url. See logs in $logDir"
}

if ($WebPort -eq $ApiPort -or $WebPort -lt 1024 -or $ApiPort -lt 1024 -or $WebPort -gt 65535 -or $ApiPort -gt 65535) {
    throw 'Choose two different ports between 1024 and 65535.'
}
$ports = @($WebPort)
if ($WithBackend) { $ports += $ApiPort }
foreach ($port in $ports) {
    $listener = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($listener) { throw "Port $port is occupied. Stop the previous local run or choose another port." }
}
if (Test-Path -LiteralPath $statePath) {
    throw 'A previous local run is recorded. Run scripts/stop-local.ps1 before starting again.'
}
$node = (Get-Command node -ErrorAction Stop).Source
$npm = (Get-Command npm.cmd -ErrorAction Stop).Source
New-Item -ItemType Directory -Path $logDir, $binDir -Force | Out-Null

Push-Location $projectRoot
try {
    if ($WithBackend) {
        $go = (Get-Command go -ErrorAction Stop).Source
        $backendExe = Join-Path $binDir 'fundlive-local.exe'
        & $go build -o $backendExe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw 'Backend build failed.' }
    }
    if (-not (Test-Path (Join-Path $projectRoot 'web/node_modules/next/dist/bin/next'))) {
        Push-Location (Join-Path $projectRoot 'web')
        try {
            & $npm ci
            if ($LASTEXITCODE -ne 0) { throw 'Frontend dependency installation failed.' }
        } finally { Pop-Location }
    }
    $envNames = @('STORAGE_MODE','PORT','AUTH_EMAIL_CODE_ENABLED','BACKEND_URL')
    $saved = @{}
    foreach ($key in $envNames) { $saved[$key] = [Environment]::GetEnvironmentVariable($key, 'Process') }
    $started = @()
    try {
        $env:STORAGE_MODE = 'memory'
        $env:PORT = "$ApiPort"
        $env:AUTH_EMAIL_CODE_ENABLED = 'false'
        $env:BACKEND_URL = "http://127.0.0.1:$ApiPort"
        if ($WithBackend) {
            $backend = Start-Process -FilePath $backendExe -WorkingDirectory $projectRoot -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $logDir 'local-api.log') -RedirectStandardError (Join-Path $logDir 'local-api-error.log')
            Save-LocalProcess $backend $backendExe
            Wait-Endpoint "http://127.0.0.1:$ApiPort/health"
        }
        $nextScript = Join-Path $projectRoot 'web/node_modules/next/dist/bin/next'
        $frontend = Start-Process -FilePath $node -ArgumentList @(('"{0}"' -f $nextScript),'dev','--webpack','--hostname','127.0.0.1','--port',"$WebPort") -WorkingDirectory (Join-Path $projectRoot 'web') -WindowStyle Hidden -PassThru -RedirectStandardOutput (Join-Path $logDir 'local-web.log') -RedirectStandardError (Join-Path $logDir 'local-web-error.log')
        Save-LocalProcess $frontend $node $nextScript
        Wait-Endpoint "http://127.0.0.1:$WebPort/research"
    } catch {
        & (Join-Path $PSScriptRoot 'stop-local.ps1')
        throw
    } finally {
        foreach ($key in $envNames) { [Environment]::SetEnvironmentVariable($key, $saved[$key], 'Process') }
    }
    $url = "http://127.0.0.1:$WebPort/research"
    Write-Host "FundLive: $url"
    Write-Host 'Stop: powershell -ExecutionPolicy Bypass -File scripts/stop-local.ps1'
    if (-not $NoBrowser) { Start-Process $url }
} finally { Pop-Location }

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$statePath = Join-Path $projectRoot 'logs/local-processes.json'
if (-not (Test-Path -LiteralPath $statePath)) {
    Write-Host 'No recorded local processes.'
    exit 0
}
$records = Get-Content -LiteralPath $statePath -Raw | ConvertFrom-Json
$processes = @(Get-CimInstance Win32_Process)
function Stop-LocalTree([int]$ProcessId) {
    foreach ($child in $processes | Where-Object { $_.ParentProcessId -eq $ProcessId }) {
        Stop-LocalTree $child.ProcessId
    }
    Stop-Process -Id $ProcessId -ErrorAction SilentlyContinue
}
foreach ($record in $records) {
    $processId = [int]$record.pid
    $process = $processes | Where-Object { $_.ProcessId -eq $processId }
    if (-not $process) { continue }
    $running = Get-Process -Id $processId -ErrorAction SilentlyContinue
    $sameStart = $running -and $running.StartTime.ToUniversalTime().ToString('o') -eq $record.started_at
    $sameExecutable = $process.ExecutablePath -eq $record.executable
    $sameCommand = -not $record.command_marker -or $process.CommandLine.Contains($record.command_marker)
    if ($sameStart -and $sameExecutable -and $sameCommand) {
        Stop-LocalTree $processId
    } else {
        Write-Warning "PID $processId is no longer a recorded FundLive process; skipped."
    }
}
Remove-Item -LiteralPath $statePath
Write-Host 'Local FundLive stopped.'

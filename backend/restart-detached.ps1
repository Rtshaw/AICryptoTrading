# Restarts the backend as a detached background process (survives this
# session/terminal closing) - the standing deployment method for this
# project when not using docker-compose. Stops whatever's listed in
# detached-server.pid (if any), rebuilds nothing itself (run `go build -o
# cryptotrading-server.exe ./cmd/server` first if source changed), then
# relaunches from ..\.env (plain KEY=value) and ..\.env.prompts (optional -
# heredoc-syntax AI prompt overrides, see .env.prompts.example) and starts
# the binary hidden, redirecting stdout/stderr to log files here.
#
# Usage: powershell -File .\restart-detached.ps1
# (run from backend/, or it cd's there itself)

Set-Location $PSScriptRoot

$pidFile = "detached-server.pid"
if (Test-Path $pidFile) {
    $oldPid = Get-Content $pidFile
    Stop-Process -Id $oldPid -Force -ErrorAction SilentlyContinue
    Start-Sleep -Milliseconds 500
}

# Heredoc-aware, UTF8-correct env-file loader. Handles both plain
# KEY=value lines (as in .env) and multi-line `KEY<<DELIM ... DELIM` blocks
# (as in .env.prompts) - Get-Content's default encoding mangles non-ASCII
# text and can desync line boundaries on multi-byte UTF-8, so -Encoding
# UTF8 is required here, not optional.
function Import-EnvFile($path) {
    if (-not (Test-Path $path)) { return }
    $lines = Get-Content $path -Encoding UTF8
    $i = 0
    while ($i -lt $lines.Count) {
        $line = $lines[$i]
        if ($line -match '^\s*#') { $i++; continue }
        if ($line -match '^\s*$') { $i++; continue }
        if ($line -match '^([A-Za-z_][A-Za-z0-9_]*)<<(\S+)$') {
            $name = $Matches[1]
            $delim = $Matches[2]
            $i++
            $valueLines = @()
            while ($i -lt $lines.Count -and $lines[$i] -ne $delim) {
                $valueLines += $lines[$i]
                $i++
            }
            $value = $valueLines -join "`n"
            [Environment]::SetEnvironmentVariable($name, $value, 'Process')
            $i++
            continue
        }
        if ($line -match '^([^=]+)=(.*)$') {
            [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], 'Process')
        }
        $i++
    }
}

Import-EnvFile "..\.env"
Import-EnvFile "..\.env.prompts"

$proc = Start-Process -FilePath ".\cryptotrading-server.exe" -WindowStyle Hidden `
    -RedirectStandardOutput "detached-server.out.log" -RedirectStandardError "detached-server.err.log" -PassThru
$proc.Id | Out-File -FilePath $pidFile -Encoding ascii
Start-Sleep -Seconds 3
Get-Process -Id $proc.Id -ErrorAction SilentlyContinue | Select-Object Id, ProcessName, StartTime

# Run the next pending sub-task in the current window
# Called by sub-cron every 10 minutes
# Falls back to gen-plan.ps1 if window has expired

$ErrorActionPreference = 'Stop'

$schedulerDir = 'D:\MiniMax Code\1Panel-edu-research\.scheduler'
$stateFile = Join-Path $schedulerDir 'state.json'
$repoRoot = 'D:\MiniMax Code\1Panel-edu-research'
$now = Get-Date

# Load state
if (-not (Test-Path $stateFile)) {
    # No plan yet — generate one
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $schedulerDir 'gen-plan.ps1') | Out-Null
}
$state = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json
$window = $state.current_window
$windowStart = Get-Date $window.window_start
$windowEnd = Get-Date $window.window_end

# If window expired, generate a new plan (anti-miss)
if ($now -ge $windowEnd) {
    Write-Host "[$(Get-Date -Format 'HH:mm')] window expired, regenerating plan"
    & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $schedulerDir 'gen-plan.ps1') | Out-Null
    $state = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json
    $window = $state.current_window
    $windowStart = Get-Date $window.window_start
    $windowEnd = Get-Date $window.window_end
}

# Find first pending or failed (retryable) sub-task
$next = $null
foreach ($t in $window.sub_tasks) {
    if ($t.status -in @('pending', 'failed')) {
        $next = $t
        break
    }
}

# No pending tasks in this window — silent
if ($null -eq $next) {
    Write-Host "<mavis-progress>silent: window $($window.window_id) all done ($($window.sub_tasks.Count)/$($window.sub_tasks.Count))</mavis-progress>"
    exit 0
}

# Mark running
$idx = [array]::IndexOf(@($window.sub_tasks), $next)
$state.current_window.sub_tasks[$idx].status = 'running'
$state.current_window.sub_tasks[$idx].started_at = $now.ToString('o')
$state | ConvertTo-Json -Depth 6 | Set-Content $stateFile -Encoding UTF8

# Run the task
$startTime = Get-Date
$taskId = $next.id
$taskName = $next.name
$logFile = Join-Path $schedulerDir "task-${taskId}.log"
Write-Host "[$(Get-Date -Format 'HH:mm:ss')] running: $taskId ($taskName)"

$success = $true
$errorMsg = ''
try {
    switch ($taskId) {
        'index-rebuild' {
            $modules = Get-ChildItem "$repoRoot\modules" -Directory | Sort-Object Name
            $totalMd = 0
            $totalHtml = 0
            $totalSize = 0
            $index = "# KB 索引`n`n生成时间: $(Get-Date -Format 'o')`n`n"
            $index += "| 模块 | HR | VA | 大小 |`n|---|---|---|---|\n"
            foreach ($m in $modules) {
                $hr = Get-Item "$($m.FullName)\HUMAN-READABLE.md" -ErrorAction SilentlyContinue
                $va = Get-Item "$($m.FullName)\visual-atlas.html" -ErrorAction SilentlyContinue
                if ($hr) { $totalMd++; $totalSize += $hr.Length }
                if ($va) { $totalHtml++; $totalSize += $va.Length }
                $hrSize = if ($hr) { $hr.Length } else { 0 }
                $vaSize = if ($va) { $va.Length } else { 0 }
                $index += "| $($m.Name) | $(if($hr){'OK'}else{'MISSING'}) | $(if($va){'OK'}else{'MISSING'}) | $((($hrSize + $vaSize)/1KB).ToString('0.0')) KB |`n"
            }
            $index += "`n**汇总**: $totalMd HR + $totalHtml VA = $((($totalSize)/1MB).ToString('0.2')) MB`n"
            $index | Set-Content "$repoRoot\KB-INDEX.md" -Encoding UTF8
            "Index rebuilt: $totalMd modules, $((($totalSize)/1MB).ToString('0.2')) MB"
        }
        'quality-check' {
            # Scan for common issues: typo, missing files, broken refs
            $issues = @()
            $modules = Get-ChildItem "$repoRoot\modules" -Directory | Sort-Object Name
            foreach ($m in $modules) {
                $hr = Get-Content "$($m.FullName)\HUMAN-READABLE.md" -Raw -ErrorAction SilentlyContinue
                if (-not $hr) { $issues += "$($m.Name): missing HUMAN-READABLE.md"; continue }
                $va = Test-Path "$($m.FullName)\visual-atlas.html"
                if (-not $va) { $issues += "$($m.Name): missing visual-atlas.html" }
                # Check for placeholders
                if ($hr -match 'TODO|TBD|FIXME|XXX') { $issues += "$($m.Name): has TODO/TBD/FIXME" }
            }
            if ($issues.Count -gt 0) {
                $report = "# Quality Check `n`n$(Get-Date -Format 'o')`n`n" + ($issues | ForEach-Object { "- $_" }) -join "`n"
                $report | Set-Content "$repoRoot\QUALITY-REPORT.md" -Encoding UTF8
                "Quality: $($issues.Count) issue(s) found"
            } else {
                "OK: no issues" | Set-Content "$repoRoot\QUALITY-REPORT.md" -Encoding UTF8
                "OK: all modules complete"
            }
        }
        'stats-report' {
            $stats = Get-ChildItem "$repoRoot\modules" -Recurse -File | Measure-Object Length -Sum
            $totalFiles = (Get-ChildItem "$repoRoot" -Recurse -File -Exclude '.git','.scheduler','daily-mgmt-state.json' | Measure-Object).Count
            $report = @"
# KB Stats

Generated: $(Get-Date -Format 'o')

- Total modules: $((Get-ChildItem "$repoRoot\modules" -Directory | Measure-Object).Count)
- Total files: $totalFiles
- modules/ size: $([math]::Round($stats.Sum / 1MB, 2)) MB
- Last commit: $((git -C $repoRoot log -1 --oneline 2>$null | Select-Object -First 1) -join '')
- GitHub: https://github.com/OpenUniversus/1Panel.edu
"@
            $report | Set-Content "$repoRoot\STATS.md" -Encoding UTF8
            "Stats: $totalFiles files, $([math]::Round($stats.Sum / 1MB, 2)) MB"
        }
        'git-sync' {
            & powershell -NoProfile -ExecutionPolicy Bypass -File (Join-Path $repoRoot 'daily-mgmt.ps1') 2>&1 | Out-Null
            "Git sync: completed (see DAILY-STATUS.md)"
        }
        'backup-snapshot' {
            $stamp = Get-Date -Format 'yyyyMMdd-HHmm'
            $backupDir = "$repoRoot\.backups"
            New-Item -Path $backupDir -ItemType Directory -Force | Out-Null
            $zipPath = "$backupDir\kb-snapshot-$stamp.zip"
            # Use .NET zip
            Add-Type -AssemblyName System.IO.Compression.FileSystem
            [System.IO.Compression.ZipFile]::CreateFromDirectory("$repoRoot\modules", $zipPath)
            # Keep last 10 backups
            Get-ChildItem "$backupDir\kb-snapshot-*.zip" | Sort-Object LastWriteTime -Descending | Select-Object -Skip 10 | Remove-Item -Force
            "Backup: $zipPath"
        }
        'audit-log' {
            $doneCount = ($window.sub_tasks | Where-Object { $_.status -eq 'done' }).Count
            $total = $window.sub_tasks.Count
            $audit = @"
# Window Audit

Window: $($window.window_id)
$doneCount / $total tasks done in this window

## Status

$($window.sub_tasks | ForEach-Object { "- [$($_.status)] $($_.id): $($_.name)" } -join "`n")
"@
            $audit | Set-Content "$repoRoot\WINDOW-AUDIT.md" -Encoding UTF8
            "Audit: $doneCount/$total done"
        }
        default {
            $success = $false
            $errorMsg = "unknown task: $taskId"
        }
    }
} catch {
    $success = $false
    $errorMsg = $_.Exception.Message
}

# Reload state to update
$state = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json
$idx = -1
for ($i = 0; $i -lt $state.current_window.sub_tasks.Count; $i++) {
    if ($state.current_window.sub_tasks[$i].id -eq $taskId) {
        $idx = $i
        break
    }
}
$endTime = Get-Date
$duration = [int](($endTime - $startTime).TotalSeconds)

if ($idx -ge 0) {
    $state.current_window.sub_tasks[$idx].finished_at = $endTime.ToString('o')
    $state.current_window.sub_tasks[$idx].duration_sec = $duration
    $state.current_window.sub_tasks[$idx].result = if ($success) { 'ok' } else { "failed: $errorMsg" }
    $state.current_window.sub_tasks[$idx].status = if ($success) { 'done' } else { 'failed' }
}
$state | ConvertTo-Json -Depth 6 | Set-Content $stateFile -Encoding UTF8

# Report
$status = if ($success) { 'done' } else { 'failed' }
$msg = if ($success) { "$taskId OK" } else { "$taskId FAILED" }
$durStr = $duration.ToString() + 's'
$ts = Get-Date -Format 'HH:mm:ss'
$hm = Get-Date -Format 'HH:mm'
Write-Host "[$ts] $status : $msg ($durStr)"

if ($success) {
    Write-Host "<mavis-progress>tick : $taskId done in $durStr - $hm</mavis-progress>"
} else {
    Write-Host "<mavis-progress>tick : $taskId FAILED : $errorMsg</mavis-progress>"
}

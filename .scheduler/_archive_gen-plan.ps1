# Generate a new 5-hour window plan
# Called by master cron at 0/5/10/15/20 hours
# Also called as fallback by run-next-task.ps1 if window expired

$ErrorActionPreference = 'Stop'

$schedulerDir = 'D:\MiniMax Code\1Panel-edu-research\.scheduler'
$stateFile = Join-Path $schedulerDir 'state.json'
$now = Get-Date

# 5-hour window: starting at 0/5/10/15/20 (round down to nearest 5h)
$hour = $now.Hour
$bucket = [Math]::Floor($hour / 5) * 5
$windowStart = Get-Date -Year $now.Year -Month $now.Month -Day $now.Day -Hour $bucket -Minute 0 -Second 0
$windowEnd = $windowStart.AddHours(5)

# 6 sub-tasks, evenly distributed across 5 hours (~50 min apart)
$tasks = @(
    @{ id = 'index-rebuild';   name = '重建 KB 索引';          offset_min = 0   }
    @{ id = 'quality-check';   name = '质量检查 (typo/断链)';  offset_min = 50  }
    @{ id = 'stats-report';    name = '统计报告 (大小/文件)';  offset_min = 100 }
    @{ id = 'git-sync';        name = 'GitHub 同步 (daily-mgmt)'; offset_min = 150 }
    @{ id = 'backup-snapshot'; name = 'KB 快照备份';          offset_min = 200 }
    @{ id = 'audit-log';       name = '窗口审计日志';          offset_min = 250 }
)

$subTasks = @()
foreach ($t in $tasks) {
    $dueAt = $windowStart.AddMinutes($t.offset_min)
    $subTasks += @{
        id          = $t.id
        name        = $t.name
        due_at      = $dueAt.ToString('o')
        status      = 'pending'  # pending / running / done / failed / skipped
        started_at  = $null
        finished_at = $null
        duration_sec = $null
        result      = $null
    }
}

# Load old state to preserve history
$oldState = $null
if (Test-Path $stateFile) {
    try { $oldState = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json } catch {}
}

$history = @()
if ($oldState -and $oldState.history) {
    $history = @($oldState.history)
}
# Mark previous window as completed
if ($oldState -and $oldState.current_window) {
    $prevWindow = $oldState.current_window
    $doneCount = ($prevWindow.sub_tasks | Where-Object { $_.status -eq 'done' }).Count
    $failCount = ($prevWindow.sub_tasks | Where-Object { $_.status -eq 'failed' }).Count
    $history += @{
        window_id    = $prevWindow.window_id
        completed_at = $now.ToString('o')
        tasks_done   = $doneCount
        tasks_failed = $failCount
    }
    # Keep last 20 history entries
    if ($history.Count -gt 20) {
        $history = $history[($history.Count - 20)..($history.Count - 1)]
    }
}

$state = @{
    schema_version = 1
    current_window = @{
        window_id    = "win-$($windowStart.ToString('yyyyMMdd-HHmm'))"
        window_start = $windowStart.ToString('o')
        window_end   = $windowEnd.ToString('o')
        generated_at = $now.ToString('o')
        sub_tasks    = $subTasks
    }
    history = $history
}

$state | ConvertTo-Json -Depth 6 | Set-Content $stateFile -Encoding UTF8
Write-Host "plan generated: $($state.current_window.window_id) ($($subTasks.Count) sub-tasks)"
Write-Host "  window: $($windowStart.ToString('HH:mm')) - $($windowEnd.ToString('HH:mm'))"
foreach ($t in $subTasks) {
    $dueStr = (Get-Date $t.due_at).ToString('HH:mm')
    Write-Host "  [$dueStr] $($t.id): $($t.name)"
}

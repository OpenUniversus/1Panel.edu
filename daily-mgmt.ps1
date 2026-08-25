# 1Panel.edu Daily Management
# Runs every 5 min via mavis cron
# - Detects file changes in local KB
# - Commits + pushes to GitHub
# - Updates DAILY-STATUS.md
# - Reports changes (silent on no-op)

$ErrorActionPreference = 'Stop'

$repoRoot = 'D:\MiniMax Code\1Panel-edu-research'
$localRoot = 'D:\MiniMax Code\1Panel\study'
$stateFile = Join-Path $repoRoot 'daily-mgmt-state.json'
$statusFile = Join-Path $repoRoot 'DAILY-STATUS.md'

# Load state (mtime manifest)
$state = @{ last_run = $null; files = @{} }
if (Test-Path $stateFile) {
    try { $state = Get-Content $stateFile -Raw -Encoding UTF8 | ConvertFrom-Json } catch {}
}

# Scan local KB files
$scanPaths = @(
    @{ src = "$localRoot\firewall-architecture.md"; dst = 'firewall-architecture.md' }
    @{ src = "$localRoot\modules\00-landscape.md"; dst = 'modules/00-landscape.md' }
    @{ src = "$localRoot\modules\00-KNOWLEDGE-BOOK.html"; dst = 'modules/00-KNOWLEDGE-BOOK.html' }
)
$modules = @('01-app-store','02-container','03-website','04-database','05-backup-snapshot','06-cronjob','07-alert','08-file','09-ai-agent','10-host-monitor','11-runtime-ai','12-security','13-frontend')
foreach ($m in $modules) {
    $scanPaths += @{ src = "$localRoot\modules\$m\HUMAN-READABLE.md"; dst = "modules/$m/HUMAN-READABLE.md" }
    $scanPaths += @{ src = "$localRoot\modules\$m\visual-atlas.html"; dst = "modules/$m/visual-atlas.html" }
}

# Compare with state, collect changes
$changes = @()
foreach ($p in $scanPaths) {
    if (-not (Test-Path $p.src)) { continue }
    $srcHash = (Get-FileHash $p.src -Algorithm SHA256).Hash
    $existing = $state.files.($p.dst)
    $existingHash = if ($existing) { $existing.hash } else { $null }
    if ($srcHash -ne $existingHash) {
        $changes += $p.dst
    }
}

# If no changes, silent exit
if ($changes.Count -eq 0) {
    Write-Host "<mavis-progress>silent: 1Panel.edu KB 无变化（$((Get-Date -Format 'HH:mm'))）</mavis-progress>"
    exit 0
}

# Changes detected — copy + commit + push + update status
Write-Host "[$(Get-Date -Format 'HH:mm:ss')] 检测到 $($changes.Count) 个文件变化"

foreach ($p in $scanPaths) {
    if ($changes -contains $p.dst) {
        $dst = Join-Path $repoRoot $p.dst
        New-Item -Path (Split-Path $dst) -ItemType Directory -Force | Out-Null
        Copy-Item -Path $p.src -Destination $dst -Force
    }
}

# Update state
$newState = @{ last_run = (Get-Date -Format 'o'); files = @{} }
foreach ($p in $scanPaths) {
    if (Test-Path $p.src) {
        $h = (Get-FileHash $p.src -Algorithm SHA256).Hash
        $newState.files.($p.dst) = @{ hash = $h; mtime = (Get-Item $p.src).LastWriteTime.ToString('o') }
    }
}
$newState | ConvertTo-Json -Depth 5 | Set-Content $stateFile -Encoding UTF8

# Git commit + push (via .bat to avoid PowerShell quote escaping hell)
$timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm'
$commitMsg = "chore(daily-mgmt): sync $($changes.Count) files at $timestamp"
Push-Location $repoRoot
try {
    $batPath = Join-Path $repoRoot 'run-add-commit.bat'
    $commitOutput = cmd.exe /c "`"$batPath`" `"$commitMsg`"" 2>&1
} finally { Pop-Location }

# Update DAILY-STATUS.md
$status = @"
# 1Panel.edu Daily Status

**最后更新**: $timestamp
**变更文件数**: $($changes.Count)

## 本次变更

$($changes | ForEach-Object { "- ``$_``" } | Out-String)

## 累计状态

- 13 个模块人话版 + 可视化图集
- 1 份总知识书 (00-KNOWLEDGE-BOOK.html, 357 KB)
- 1 份模块全景 (00-landscape.md)
- 1 份防火墙深度注解 (firewall-architecture.md)

## GitHub

- 仓库: https://github.com/OpenUniversus/1Panel.edu
- 分支: main
- 最后 commit: $commitMsg
"@
$status | Set-Content $statusFile -Encoding UTF8

# Report
Write-Host "<mavis-progress>tick done: $($changes.Count) files synced to 1Panel.edu: $($changes -join ', ')</mavis-progress>"

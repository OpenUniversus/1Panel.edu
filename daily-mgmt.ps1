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

# Git commit + push (via .bat to avoid PowerShell quote escaping hell)
$timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm'
$commitMsg = "chore(daily-mgmt): sync $($changes.Count) files at $timestamp"
Push-Location $repoRoot
$pushOk = $false
try {
    $batPath = Join-Path $repoRoot 'run-add-commit.bat'
    $commitOutput = cmd.exe /c "`"$batPath`" `"$commitMsg`"" 2>&1
    # If local is ahead of remote, push succeeded
    $localHead = (cmd.exe /c "git rev-parse HEAD" 2>&1 | Select-Object -Last 1).Trim()
    $remoteHead = (cmd.exe /c "git rev-parse origin/main" 2>&1 | Select-Object -Last 1).Trim()
    if ($localHead -eq $remoteHead) { $pushOk = $true }
} finally { Pop-Location }

if (-not $pushOk) {
    Write-Host "<mavis-progress>tick: changes detected but push failed — will retry next tick</mavis-progress>"
    exit 1
}

# Update state (only after successful push)
$newState = @{ last_run = (Get-Date -Format 'o'); files = @{} }
foreach ($p in $scanPaths) {
    if (Test-Path $p.src) {
        $h = (Get-FileHash $p.src -Algorithm SHA256).Hash
        $newState.files.($p.dst) = @{ hash = $h; mtime = (Get-Item $p.src).LastWriteTime.ToString('o') }
    }
}
$newState | ConvertTo-Json -Depth 5 | Set-Content $stateFile -Encoding UTF8

# Update DAILY-STATUS.md
$changesList = ($changes | ForEach-Object { "  - $_" }) -join "`n"
$status = "# 1Panel.edu Daily Status`n`n"
$status += "**最后更新**: $timestamp`n"
$status += "**变更文件数**: $($changes.Count)`n`n"
$status += "## 本次变更`n`n$changesList`n`n"
$status += "## 累计状态`n`n"
$status += "* 13 个模块人话版 + 可视化图集`n"
$status += "* 1 份总知识书 (00-KNOWLEDGE-BOOK.html, 357 KB)`n"
$status += "* 1 份模块全景 (00-landscape.md)`n"
$status += "* 1 份防火墙深度注解 (firewall-architecture.md)`n`n"
$status += "## GitHub`n`n"
$status += "* 仓库: https://github.com/OpenUniversus/1Panel.edu`n"
$status += "* 分支: main`n"
$status += "* 最后 commit: $commitMsg`n"
$status | Set-Content $statusFile -Encoding UTF8

# Report
Write-Host "<mavis-progress>tick done: $($changes.Count) files synced to 1Panel.edu: $($changes -join ', ')</mavis-progress>"

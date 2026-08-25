# Initialize daily-mgmt state with current file hashes
$repoRoot = 'D:\MiniMax Code\1Panel-edu-research'
$localRoot = 'D:\MiniMax Code\1Panel\study'
$stateFile = Join-Path $repoRoot 'daily-mgmt-state.json'

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

$state = @{ last_run = (Get-Date -Format 'o'); files = @{} }
foreach ($p in $scanPaths) {
    if (Test-Path $p.src) {
        $h = (Get-FileHash $p.src -Algorithm SHA256).Hash
        $state.files.($p.dst) = @{ hash = $h; mtime = (Get-Item $p.src).LastWriteTime.ToString('o') }
    }
}
$state | ConvertTo-Json -Depth 5 | Set-Content $stateFile -Encoding UTF8
Write-Host "State initialized: $stateFile ($((Get-Item $stateFile).Length) bytes)"
Write-Host "Files tracked: $($state.files.PSObject.Properties.Count)"

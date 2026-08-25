# 1Panel.edu Daily Status

**首次初始化**: 2026-08-25 09:12
**自动管理**: 每 5 分钟检测本地 KB 变化，自动 commit + push 到 GitHub

## 当前内容

13 个模块人话版 + 可视化图集：

* `modules/01-app-store/` — 一键装应用
* `modules/02-container/` — Docker 容器管理
* `modules/03-website/` — 网站 / Nginx
* `modules/04-database/` — 数据库管理（99 KB 深度注解）
* `modules/05-backup-snapshot/` — 备份与快照
* `modules/06-cronjob/` — 定时任务
* `modules/07-alert/` — 告警
* `modules/08-file/` — 文件管理
* `modules/09-ai-agent/` — AI Agent 平台（v2 核心新特性）
* `modules/10-host-monitor/` — 主机监控
* `modules/11-runtime-ai/` — AI 模型运行时
* `modules/12-security/` — 安全套件
* `modules/13-frontend/` — 前端

总知识书 / 全景 / 防火墙：

* `modules/00-KNOWLEDGE-BOOK.html` — 5 章 / 357 KB
* `modules/00-landscape.md` — 13 模块全景地图
* `firewall-architecture.md` — 防火墙 v2 深度注解

## GitHub

* 仓库: https://github.com/OpenUniversus/1Panel.edu
* 分支: main
* 最后 commit: 05b2493 test-chron
* 本地 / 远端同步：是

## 自动管理 cron

* 名称: `1panel-edu-daily-mgmt`
* 调度: `*/5 * * * *`（每 5 分钟）
* 状态: active
* 脚本: `daily-mgmt.ps1`
* 状态文件: `daily-mgmt-state.json`（29 个文件 SHA-256 哈希 + mtime）
* 输出: silent（无变化）/ tick done（有变化）

## 工作流

1. 你改本地 `D:\MiniMax Code\1Panel\study\modules\XX\HUMAN-READABLE.md`
2. 5 分钟内 daily-mgmt 检测到 SHA-256 变化
3. copy 到 `D:\MiniMax Code\1Panel-edu-research\modules\XX\`
4. `git add -A && git commit && git push origin main`
5. 远端 https://github.com/OpenUniversus/1Panel.edu 同步更新

push 失败时下一 tick 自动重试（state 不会"假装同步"）。

# 1Panel.edu Scheduler — SPEC v2

> 重设计版。Phase 1+2+3 全量接受。飞书 wiki 同步已搁置, 不在本 spec。

---

## 1. 架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│  monitor-daemon.py  (watchdog, 1 process)                        │
│  └─ 每 60s 检查 scheduler-daemon.py 是否存活, 挂了就 restart     │
└──────────────────────┬──────────────────────────────────────────┘
                       │ supervise
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  scheduler-daemon.py  (主循环, 1 process)                        │
│  ├─ master tick @ 0/5/10/15/20 :00: 跑 gen-plan + run-next-task  │
│  └─ sub tick @ every :15/:30/:45/:00: 跑 run-next-task           │
└──────────────────────┬──────────────────────────────────────────┘
                       │ call
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  gen-plan.py        生成 5h 窗口 plan, 9 task × 30min 间隔       │
│  run-next-task.py   跑下一个 pending/failed task                  │
│  run-master.py      链式触发: gen-plan + run-next-task           │
└──────────────────────┬──────────────────────────────────────────┘
                       │ call
                       ▼
┌─────────────────────────────────────────────────────────────────┐
│  9 task handlers (注册在 run-next-task.py 内)                    │
│  index-rebuild, quality-check, upstream-poll, module-coverage,  │
│  health-score, stats-report, git-sync, backup-snapshot,          │
│  daily-summary                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**核心不变原则**:
- 100% Python, 0 LLM token
- 状态机: pending → running → done/failed (in state.json)
- Anti-re-run: 只跑 pending/failed
- Anti-miss: window_expired → 自动 regenerate
- Crash-safe: 每 tick re-load state

---

## 2. Schedule

### 2.1 时间分布

- **Master tick** @ 0/5/10/15/20 :00 (= 5h 桶起点)
  - 跑 `gen-plan.py` + `run-next-task.py`
  - 桶内首 task (index-rebuild) 通常立即跑
- **Sub tick** @ :15, :30, :45 (3 次/小时, 15 min 间隔)
  - 跑 `run-next-task.py`
  - 选下一个 pending/failed task

> 之前 sub tick 10 min 太密, 改 15 min 足够覆盖 30 min 间隔的 task。

### 2.1.1 Task-driven 循环 (v2.1)

当 9 task 全部 done 后, 下一次 sub tick 检测 `all_done` → **force regen** 立即开新窗口, 不等 5h 桶结束。

- 之前: 9 task done 后 idle 4.4h 等 5h 桶边界
- 现在: 9 task done 后立即重置 9 task pending, 跑下一轮
- 5h 桶退化为"参考周期" (build_plan 仍按 bucket hour 算)
- 实际节奏: 9 task / ~2.25h, 1 天 ~10.7 轮, 永不 idle
- 适用场景: 任务全部幂等 (build index / push / 备份), 多跑无害

### 2.2 5h 窗口内 9 task 分布 (30 min 间隔)

| # | Offset (min) | task id | 名称 | 类别 | 状态 |
|---|---|---|---|---|---|
| 1 | 0   | index-rebuild    | 重建 KB 索引          | L1 内容 | 已有 |
| 2 | 30  | quality-check    | 质量检查 (placeholder+断链+typo) | L1 内容 | 扩展 |
| 3 | 60  | upstream-poll    | 1Panel 上游 commit 拉取与 diff | L1 内容 | 新 |
| 4 | 90  | module-coverage  | KB 模块 vs upstream 源文件覆盖度 | L1 内容 | 新 |
| 5 | 120 | health-score     | KB 健康分 (0-100) | L2 健康 | 新 |
| 6 | 150 | stats-report     | 统计报告 (大小/文件/commit) | L2 健康 | 已有 |
| 7 | 180 | git-sync         | 推送到 GitHub (纯 Python) | L3 同步 | 改 |
| 8 | 210 | backup-snapshot  | KB 快照 zip 备份 (留 10 份) | L3 同步 | 已有 |
| 9 | 240 | daily-summary    | 当日 5 窗口汇总 (周日扩为周报) | L4 周期 | 新 |

> 9 task × 30 min = 270 min, 5h 窗口 = 300 min, 留 30 min buffer。
> audit-log 功能合并进 daily-summary, 不单独 task。

### 2.3 周报触发

- daily-summary 内部检查当前时间是否周日
- 是周日 → 扩展输出, 写 `WEEKLY-REPORT.md`
- 不是周日 → 写 `DAILY-SUMMARY.md`

---

## 3. 异常处理

### 3.1 Lock File

- 路径: `.scheduler/lock`
- 内容: `{ "pid": <int>, "started_at": "<iso>", "task_id": "<id>" }`
- 生成: run-next-task 进入 running 状态时写
- 释放: task 完成 (done/failed) 时删
- 启动检查: lock 存在且 PID 还活着 + 任务跑 < 30 min → 跳过本次 tick
- 启动检查: lock 存在但 PID 已死 / 任务跑 > 30 min → 删除旧 lock, 重试
- 启动检查: lock 存在且 PID 还活着 + 任务跑 > 30 min → 强杀 PID, 删 lock, 重试

### 3.2 Circuit Breaker

- 跟踪位置: state.json 内的 `circuit_breaker` 字段
- 结构: `{ "<task_id>": { "consecutive_failures": <int>, "first_failure": "<iso>", "excluded": <bool> } }`
- 规则:
  - 任务失败 → 该 task consecutive_failures + 1
  - 任务成功 → 该 task consecutive_failures = 0
  - consecutive_failures >= 3 → excluded = true
  - excluded = true 的 task 在窗口内被跳过, 写 `BLACKLIST.md`
  - excluded = true 的 task 不会 retry, 直到下一个 5h 桶 (新窗口 reset)
- Reset: 每个 master tick 重置 circuit_breaker (新窗口 = 新机会)

### 3.3 Alert Log

- 路径: `ALERTS.md` (累积)
- 触发: 任一 task 状态变 failed
- 格式: append `[{timestamp}] {task_id} FAILED: {result}`
- Reset: master tick @ 00:00 跨日时清空 (新一天)

### 3.4 Health Score 扣分

- 单 task 失败: -5
- 单 task 连续失败: 额外 -5 (累计 -10/次)
- excluded (CB 触发): -20
- 窗口内 0 失败: +5 奖励
- 上游 commit > 7 天未同步: -10 (upstream-poll 报)
- 覆盖率 < 80%: -15 (module-coverage 报)

---

## 4. 文件布局

```
.scheduler/
├── SCHEDULER-SPEC.md     本文件
├── gen-plan.py           生成 5h 窗口 plan
├── run-next-task.py      跑下一个 pending/failed task
├── run-master.py         链式触发 gen-plan + run-next-task
├── scheduler-daemon.py   主循环 (master + sub tick)
├── monitor-daemon.py     watchdog: 监视 scheduler-daemon
├── state.json            当前窗口 + history + circuit_breaker
├── lock                  lock file (run-time, 临时)
├── BLACKLIST.md          excluded task 记录 (auto-gen)
├── ALERTS.md             失败 task 累积 (auto-gen, 每日 reset)
└── logs/
    ├── daemon.log        scheduler-daemon 主日志
    └── monitor.log       monitor-daemon watchdog 日志
```

### 4.1 Auto-generated 文件 (根目录)

- `KB-INDEX.md` — index-rebuild
- `QUALITY-REPORT.md` — quality-check
- `UPSTREAM-DIFF.md` — upstream-poll
- `COVERAGE-REPORT.md` — module-coverage
- `HEALTH-SCORE.md` — health-score
- `STATS.md` — stats-report (改名为 `KB-STATS.md` 避免和 root 旧 STATS.md 混淆, 或沿用 STATS.md)
- `DAILY-SUMMARY.md` — daily-summary (非周日)
- `WEEKLY-REPORT.md` — daily-summary (周日)
- `BLACKLIST.md` — circuit breaker
- `ALERTS.md` — 失败累积

---

## 5. 任务详细规格

### 5.1 index-rebuild (L1, 0min)

**输入**: `modules/` 目录
**输出**: `KB-INDEX.md`
**逻辑**: 扫每个模块子目录, 检查 `HUMAN-READABLE.md` + `visual-atlas.html` 存在, 算总大小
**耗时**: < 1s
**依赖**: 无

### 5.2 quality-check (L1, 30min, 扩展)

**输入**: `modules/` 目录
**输出**: `QUALITY-REPORT.md`
**逻辑** (4 项检查):
1. Placeholder 扫描: TODO/TBD/FIXME/XXX/HACK
2. Markdown 断链: `[text](path)` 内部引用, 验证文件存在
3. HTML 引用: `visual-atlas.html` 内 `<a href>` + `<img src>` 验证
4. 外链 dead check: `http(s)://` 链接, 简单 HEAD 请求 (timeout 3s, 失败标 dead)
   - 失败不算 quality-check 自己失败, 计入 dead count
**耗时**: < 30s (取决于外链数)
**依赖**: 无

### 5.3 upstream-poll (L1, 60min, 新)

**输入**: 1Panel upstream `dev-v2` 分支
**输出**: `UPSTREAM-DIFF.md`
**逻辑**:
1. `git ls-remote https://github.com/1Panel-dev/1Panel.git refs/heads/dev-v2` → 拿最新 commit
2. 读 `.upstream-state.json` (缓存上次拉的 commit + 时间)
3. 若 commit 变化 → diff commit message + 列出变更文件数, 写 UPSTREAM-DIFF.md
4. 若 7 天未变 → 仍写 UPSTREAM-DIFF.md, 标 `stale=7d+`
**耗时**: 2-5s (网络)
**依赖**: 无
**失败模式**: 网络不通 → 标 "network unreachable", 不算 quality-check 失败

### 5.4 module-coverage (L1, 90min, 新)

**输入**: 1Panel upstream `agent/` + `core/` 源文件
**输出**: `COVERAGE-REPORT.md`
**逻辑**:
1. 扫 1Panel upstream `agent/**/*.go` + `core/**/*.go`
2. 按业务领域分类 (database, container, firewall, ...)
3. 对 KB 13 模块, 列每个模块"覆盖了哪些源文件子集"
4. 算覆盖率 = 13 模块 HR.md 引用的源文件数 / upstream 总源文件数
**耗时**: 3-8s (要 clone or ls-remote tree)
**依赖**: upstream-poll 拿到的 commit 决定扫哪个 commit 的 tree
**失败模式**: 没 git access → 报 "无法访问 upstream tree", 覆盖率 = N/A

### 5.5 health-score (L2, 120min, 新)

**输入**: QUALITY-REPORT.md, COVERAGE-REPORT.md, UPSTREAM-DIFF.md, state.json
**输出**: `HEALTH-SCORE.md`
**逻辑**:
- 基础分: 100
- 扣分项 (见 §3.4)
- 输出: `{score} / 100, 等级: A/B/C/D/F`
**耗时**: < 1s
**依赖**: quality-check, upstream-poll, module-coverage 状态 (不强依赖, 取最近值)

### 5.6 stats-report (L2, 150min, 已有)

**输入**: 全 repo
**输出**: `STATS.md`
**逻辑**: 文件数 / 总大小 / 最后 commit / GitHub URL (沿用旧实现)
**耗时**: 1-3s

### 5.7 git-sync (L3, 180min, 改纯 Python)

**输入**: 暂存/未提交修改
**输出**: 推到 `origin/main`
**逻辑**:
1. `git status --porcelain` 检查是否有修改
2. 无修改 → 跳过 (return "no changes")
3. 有修改:
   a. `git add -A`
   b. `git commit -m "chore-daily-mgmt <timestamp>"` (chore-daily-mgmt prefix)
   c. `git push origin main`
   d. 验证 `git rev-parse HEAD` == `git rev-parse origin/main`
4. 失败回滚: `git commit --amend` 或 `git reset --soft HEAD~1`
**超时**: 120s
**失败模式**: 网络/push 失败 → 标 failed, 写 ALERTS.md

### 5.8 backup-snapshot (L3, 210min, 已有)

**输入**: `modules/`
**输出**: `.backups/kb-snapshot-<timestamp>.zip`
**逻辑**: zip modules/, 留最近 10 份 (沿用旧实现)
**耗时**: 2-5s

### 5.9 daily-summary (L4, 240min, 新)

**输入**: state.json (当日 history)
**输出**:
- 非周日: `DAILY-SUMMARY.md`
- 周日: `DAILY-SUMMARY.md` + `WEEKLY-REPORT.md`
**逻辑**:
1. 读 state.json 的 history
2. 过滤当日窗口 (0:00-23:59)
3. 统计: 总 task 数 / done 数 / failed 数 / 各 task 平均耗时
4. 周日额外: 取本周日-周六所有 history, 写周报
**耗时**: < 1s

---

## 6. 开机自启 (Phase 3, 不需 admin)

- **不**用 schtasks (需 admin)
- **不**用注册表 Run key (需 admin 改 HKLM, 但 HKCU 不用)
- **用** `shell:startup` 文件夹 (Start Menu 启动项, user 级, 不需 admin)
  - 路径: `%APPDATA%\Microsoft\Windows\Start Menu\Programs\Startup\`
  - 放一个 `1panel-scheduler.cmd` 启动 monitor-daemon.py
  - monitor-daemon.py 启动 scheduler-daemon.py
- **防多开**: monitor-daemon.py 启动时检查 PID 文件, 已存在则不启动

---

## 7. 实施计划 (按 user "Phase 1+2+3 一起" 拍板)

### Phase 1: 基础改造 (今天必做)
1. ✅ SPEC (本文件)
2. 改 gen-plan.py (9 task 30min 间隔)
3. 重写 run-next-task.py (lock + CB + alert + 改 git-sync 纯 Python)
4. 改 scheduler-daemon.py (sub tick 15min, 处理 lock)
5. 改 run-master.py (顺序触发, 错误处理)
6. 干跑验证 (gen-plan + run-next-task 模拟)

### Phase 2: 新 4 task + 扩展 (今天)
7. 扩展 quality-check
8. 实现 upstream-poll
9. 实现 module-coverage
10. 实现 health-score
11. 实现 daily-summary (含 weekly-report 触发)

### Phase 3: 收尾 (今天)
12. 写 monitor-daemon.py
13. 配开机自启 (Start Menu 启动)
14. 启动 daemon + watchdog, 观察 1 个 sub tick

### Phase 4: 验证 (今天)
15. 观察 1 个完整 5h 窗口 (or 加速模拟)
16. 确认所有 task 跑过, 输出产物正确

---

## 8. 风险与缓解

| 风险 | 缓解 |
|---|---|
| gen-plan.py 改 9 task 后旧 state.json 不兼容 | state.json schema_version 从 1 → 2, load 时检查, 不匹配则 re-init |
| 旧 daemon log 与新 log 格式冲突 | 用新 log file, 旧 daemon.log 备份到 daemon.log.v1 |
| Python 3.12 embeddable 缺 git CLI | 假设 PATH 有 git (user 已用 daily-mgmt 验证), 不行则报 failed |
| 1Panel upstream 网络不可达 | upstream-poll / module-coverage 标 N/A, 不算 failed |
| Lock 死锁 | 30 min 强杀, 记录异常 |
| Circuit breaker 误触发 | 3 次阈值, 跨窗口 reset, 写入 BLACKLIST.md 可见 |
| 启动项 monitor-daemon 反复重启 | PID file 锁, 已存在则 no-op |
| Watchdog 自身挂了 | 不递归 watchdog (避免循环), 接受偶发手动重启 |

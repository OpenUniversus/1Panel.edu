# 1Panel KB v2 — TASKS (56 个细分任务)

> 1Panel 知识库全面优化. 17 module × 4 步 = 68 task. 状态由 `.scheduler/gen-task-prompt.py` 维护.
> Docker (02-container) 已 FROZEN, 等 v3 拆分 strategy 再处理.

## 状态总览

| # | Module | 优先级 | 状态 | 备注 |
|---|---|---|---|---|
| 1 | 14-auth | P3 | **DONE** | 90.6 KB HR + 48.6 KB VA, Verifier 全 PASS |
| 2 | 02-container | P3 | **FROZEN** | Explore aborted (200 KB Go 太大), 等 v3 |
| 3 | 01-app-store | P3 | TODO | 下一跑 |
| 4 | 03-website | P3 | TODO | |
| 5 | 12-security (Nginx) | P2 | TODO | 拆自旧 12-security |
| 6 | 12-ssl (SSL) | P2 | TODO | 拆自旧 12-security |
| 7 | 06-cronjob | P2 | TODO | |
| 8 | 08-file | P2 | TODO | |
| 9 | 10-host-monitor | P1 | TODO | |
| 10 | 05-backup-snapshot | P1 | TODO | |
| 11 | 15-settings (新) | P1 | TODO | |
| 12 | 16-terminal (新) | P1 | TODO | |
| 13 | 04-database | - | TODO | KB 已有, 升级 v2 |
| 14 | 07-alert | - | TODO | KB 已有, 升级 v2 |
| 15 | 09-ai-agent | - | TODO | KB 已有, 升级 v2 |
| 16 | 11-runtime-ai | - | TODO | KB 已有, 升级 v2 |
| 17 | 13-frontend | - | TODO | KB 已有, 升级 v2 (Vue 3 + TS) |

## 推进顺序 (按优先级)

1. ⭐⭐⭐ 14-auth ✅ → 02-container ❄️ → 01-app-store ▶️ → 03-website
2. ⭐⭐ 12-security → 12-ssl → 06-cronjob → 08-file
3. ⭐ 10-host-monitor → 05-backup-snapshot → 15-settings → 16-terminal
4. - 04-database → 07-alert → 09-ai-agent → 11-runtime-ai → 13-frontend

## 每个 module 4 步流水线

每个 module 跑 4 步 (1 个 module = 1 个完整流水线 = 15-25 min):

| Step | 角色 | 描述 | 耗时 |
|---|---|---|---|
| 1. Worker | mavis `worker` | 1Panel Git 同步, 输出 SHA + 变更统计 | 5s |
| 2. Explore | mavis `explore` | 逐文件逐函数读源码, 输出 8 段讲解 | 5-10 min |
| 3. Coder | mavis `worker` | 生成 modules/<NN>-<module>/HR.md (≥80 KB) + VA.html (≥30 KB) | 5-10 min |
| 4. Verifier | mavis `verifier` | 12 项校验 (A-L), 0 误差才算 PASS | 1-2 min |

**跑法**:

```bash
# 列 56 task 状态
python .scheduler/gen-task-prompt.py --list

# 输出某个 module 的某步 prompt (可复制到 mavis task tool)
python .scheduler/gen-task-prompt.py 01-app-store worker
python .scheduler/gen-task-prompt.py 01-app-store explore
python .scheduler/gen-task-prompt.py 01-app-store coder
python .scheduler/gen-task-prompt.py 01-app-store verifier
```

## 节奏估算

- 1 module × 4 步 × ~6 min = 24 min
- 14 module × 24 min = 5.6 hour
- 1 module 1 验收 ("串行单功能验收" 原则)

## 验收标准 (1 module 跑完 4 步后)

- [ ] Worker 输出: SHA + 变更统计
- [ ] Explore 输出: 完整调用链 + **每个 func 8 段** + **每个 struct 字段** + ≥5 个类比
- [ ] Coder 输出: HR.md (≥80 KB) + VA.html (≥30 KB)
- [ ] Verifier 输出: PASS (A-L 全部 0 误差)
- [ ] git commit + push (daemon git-sync 自动推)
- [ ] KB-INDEX.md 自动更新 (daemon index-rebuild)

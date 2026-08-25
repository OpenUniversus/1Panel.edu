# 1Panel KB — 5 Agent 工作流 SPEC

> 知识库的核心不是写代码, 而是建立模块理解、调用链、数据库关系、API 文档.
> 本 spec 定义 5 个 Agent 角色 + 12 个模块的推进顺序.

---

## 1. 5 Agent 角色

| 角色 | mavis 落地 | 职责 | 触发 |
|---|---|---|---|
| **Worker** | `worker` (built-in) | 同步上游 / 跑硬任务 (git, API) | 每日 / 一次性 |
| **Explore** | `explore` (built-in) | 读源码, 输出调用链/DB 关系/模块说明 | 每个 module 1 次 |
| **Coder** | `worker` (built-in, 写文件) | 生成 Markdown + Mermaid 时序图 | 每个 module 1 次 |
| **Verifier** | `verifier` (built-in) | 对照源码校验文档 (函数名/行号/API/SQL) | 每日 |
| **Mavis** | `mavis` (root, 我) | 综合回答, 模块串联, 用户对接 | 持续 |

> 不创建 4 个 custom agent, 复用 mavis 4 个内置 role + 清晰 prompt.

---

## 2. 12 模块优先级 (1Panel 实际布局)

| 优先级 | 模块 | 1Panel 实际位置 | KB 现状 |
|---|---|---|---|
| ⭐⭐⭐ | Auth 登录认证 | `core/app/auth + core/app/service/auth.go + core/app/api/v2/auth.go` | **无** — 新建 |
| ⭐⭐⭐ | Docker 容器 | `agent/router/ro_container.go` | 02-container (部分) — 补充 |
| ⭐⭐⭐ | App 应用商店 | `agent/router/ro_app.go` | 01-app-store — 跑 Coder 重写 |
| ⭐⭐⭐ | Website 网站 | `agent/router/ro_website*.go` (8 files) | 03-website — 跑 Coder 重写 |
| ⭐⭐ | Nginx | `agent/router/ro_nginx.go` | 12-security (拆分) |
| ⭐⭐ | SSL 证书 | `agent/router/ro_website_ssl.go + ro_website_acme_account.go` | 12-security (拆分) |
| ⭐⭐ | CronJob | `agent/router/ro_cronjob.go` | 06-cronjob — 跑 Coder 重写 |
| ⭐⭐ | File 文件 | `agent/router/ro_file.go` | 08-file — 跑 Coder 重写 |
| ⭐ | Monitor 监控 | `agent/router/ro_host.go` | 10-host-monitor (部分) |
| ⭐ | Backup 备份 | `agent/router/ro_backup.go + core/app/service/backup.go` | 05-backup-snapshot (部分) |
| ⭐ | Settings 系统设置 | `agent/router/ro_setting.go + core/app/service/setting.go` | **无** — 新建 |
| ⭐ | Terminal WebSSH | `agent/router/ro_process.go + ro_toolbox.go` | **无** — 新建 |

### KB 13 vs User 12 差异

- **User 12 多**: Auth, Settings, Terminal (KB 完全没)
- **User 12 缺**: 04-database, 07-alert, 09-ai-agent, 11-runtime-ai, 13-frontend (KB 已有, user 未列优先级)
- **拆分**: 12-security → Nginx + SSL (2 个)

---

## 3. 每个 module 的 4 步流水线

```
┌─────────────────────────────────────────────────────────────┐
│  ┌─Worker─┐    ┌─Explore─┐    ┌─Coder──┐    ┌─Verifier┐  │
│  │ sync   │ -> │ analyze │ -> │ write  │ -> │ check  │   │
│  │ git    │    │ call    │    │ MD +   │    │ SHA    │     │
│  │ ls-    │    │ chain + │    │ Mer-   │    │ match  │     │
│  │ remote │    │ DB rel  │    | maid   │    │ PASS/  │     │
│  └────────┘    └─────────┘    └────────┘    │ FAIL   │     │
│                                              └────────┘     │
└─────────────────────────────────────────────────────────────┘
```

### 3.1 Worker: 源码同步

**作用**: 同步 1Panel v2 上游, 拉最新 commit.

**Prompt** (named: `1Panel Git 同步`):
```
同步本地 1Panel 仓库 (D:\MiniMax Code\1Panel) 到指定分支 dev-v2.
完成后输出:
1. 当前 Commit SHA
2. 新增文件数量
3. 修改文件列表
4. 删除文件列表

不要分析代码.
```

**落地**: `git -C <src> fetch origin dev-v2 && git -C <src> rev-parse origin/dev-v2 + diff --stat`

### 3.2 Explore: 源码分析

**作用**: 读 module 源码, 输出调用链 + DB 关系 + 模块说明.

**Prompt** (named: `解析 1Panel <module> 模块`):
```
阅读 1Panel <module> 模块源码.

输出:
- 模块职责
- 目录结构
- Router
- Handler
- Service
- Repository
- Model
- 调用链
- 数据库表
- Docker 相关依赖

不要修改代码, 只分析源码.

输出示例:

Router
  ↓
POST /api/v1/auth/login

Handler
  ↓
Login()

Service
  ↓
AuthService.Login()

Repository
  ↓
userRepo.GetByName()

Model
  ↓
users
```

### 3.3 Coder: 文档生成

**作用**: 把 Explore 输出转成 KB 模块 (Markdown + Mermaid 时序图).

**Prompt** (named: `生成 <module> 函数讲解`):
```
根据 Explore 的源码分析输出生成 Markdown.

写到 modules/<NN>-<module>/HUMAN-READABLE.md.

必须包含:

# <module> 模块

## 一句话作用
## 模块职责
## 目录结构
## 调用链 (Router → Handler → Service → Repository → Model)
## 数据库表
## 关键函数讲解 (每个函数 8 项)
  - 函数名
  - 一句话作用
  - 参数说明
  - 返回值
  - 执行流程
  - 调用者
  - 被调用函数
  - 涉及数据库
## Mermaid 时序图
## Docker 相关依赖

然后生成 modules/<NN>-<module>/visual-atlas.html (Mermaid 流程图, 可交互).

输出就是知识库页面.
```

### 3.4 Verifier: 质量校验

**作用**: 对照当前 Commit 校验 Coder 输出的文档, 防止 AI 幻觉.

**Prompt** (named: `校验 <module> 文档`):
```
对照当前 1Panel Commit 的源码验证 modules/<NN>-<module>/ 文档.

检查:
- 函数名是否存在 (grep)
- 行号是否正确
- API 路径是否一致
- SQL 表是否一致
- Mermaid 是否符合调用关系

输出 PASS 或 FAIL + 变更报告.
```

### 3.5 Mavis: 综合回答

**作用**: 接收用户问题, 组合 12 模块知识回答.

**Prompt** (用户视角):
```
用户: 1Panel 登录为什么经过这么多层?
Mavis: (从 14-auth 模块的 HUMAN-READABLE.md 调取) 
       1Panel 登录流程:
       [Router → Handler → Service → Repository → Model 完整调用链]
       [对应源码文件: line:line]
       [Mermaid 时序图]
       [为什么这么设计: 关注点分离, 1Panel 多端复用]
```

---

## 4. 推进顺序 (按 user 优先级)

```
⭐⭐⭐  Auth (新建)        → 跑完整 4 步
⭐⭐⭐  Docker             → 跑 Coder 重写 02-container
⭐⭐⭐  App                → 跑 Coder 重写 01-app-store
⭐⭐⭐  Website            → 跑 Coder 重写 03-website
⭐⭐   Nginx (拆 12-security)
⭐⭐   SSL   (拆 12-security)
⭐⭐   CronJob             → 跑 Coder 重写 06-cronjob
⭐⭐   File                → 跑 Coder 重写 08-file
⭐     Monitor             → 跑 Coder 重写 10-host-monitor
⭐     Backup              → 跑 Coder 重写 05-backup-snapshot
⭐     Settings (新建)
⭐     Terminal (新建)
```

**节奏**: 1 module/次, 4 sub-agent/次 (Worker → Explore → Coder → Verifier). user 验收后再下一个.

---

## 5. 跟现有 KB 维护 daemon 的关系

现有 7×24 自动化 daemon (v2.2) 跑 9 task:
- `index-rebuild` / `quality-check` / `upstream-poll` / `module-coverage` / `health-score` / `stats-report` / `git-sync` / `backup-snapshot` / `daily-summary`

**5 Agent 流水线跟 daemon 的关系**:
- Worker 同步 = daemon 里的 `upstream-poll` (每天跑 1 次) + 偶尔的 1Panel Git 同步 (写 commit)
- Verifier 校验 = daemon 里的 `quality-check` (每天跑, 简化版 — 全模块 placeholder 扫描, 不针对单 module 深度校验)
- 5 Agent 是 **人工触发, 模块级, 深度**; daemon 是 **自动触发, 整体级, 浅度**. 互补不冲突.

---

## 6. 不入库规则 (沿用)

- `.scheduler/*.py` — Python 源码不入仓
- `*.ps1 / *.bat / *.cmd / *.vbs` — 全部排除
- `.backups/` — 自动备份
- `_archive_*` — 历史版本

**入仓**: skeleton/ (Go) + modules/ (KB) + firewall-architecture.md + KB-INDEX/README/LICENSE.

---

## 7. 验收标准 (1 module 跑完整 4 步后)

- [ ] Worker 输出: SHA + 新增/修改/删除文件数
- [ ] Explore 输出: 完整调用链 (Router → Handler → Service → Repository → Model), DB 表, Docker 依赖
- [ ] Coder 输出: `modules/<NN>-<module>/HUMAN-READABLE.md` (≥ 50 KB 含 Mermaid) + `visual-atlas.html`
- [ ] Verifier 输出: PASS (无 hallucination)
- [ ] git commit + push (通过 daemon git-sync task)
- [ ] KB-INDEX.md 自动更新 (daemon 跑 index-rebuild)

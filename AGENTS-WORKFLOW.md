# 1Panel KB — 5 Agent 工作流 SPEC v2 (细到函数 + 非技术读者)

> 知识库的核心不是写代码, 而是建立模块理解、调用链、数据库关系、API 文档.
> **v2 升级**: 每个函数的讲解要细到 "函数级 / 方法级 / 类级", 让没做过产品、研发的人都能看懂.

---

## 1. 5 Agent 角色

| 角色 | mavis 落地 | 职责 | 触发 |
|---|---|---|---|
| **Worker** | `worker` (built-in) | 同步上游 / 跑硬任务 (git, API) | 每日 / 一次性 |
| **Explore** | `explore` (built-in) | **逐文件逐函数**读源码, 输出每个 func 详解 | 每个 module 1 次 |
| **Coder** | `worker` (built-in, 写文件) | 生成 Markdown + Mermaid 时序图 (**每个函数 8 段讲解**) | 每个 module 1 次 |
| **Verifier** | `verifier` (built-in) | 对照源码校验 (函数名/行号/API/SQL, 100% 必查) | 每日 / 每 module |
| **Mavis** | `mavis` (root, 我) | 综合回答, 模块串联, 用户对接 | 持续 |

> 不创建 4 个 custom agent, 复用 mavis 4 个内置 role + 清晰 prompt.

---

## 2. v2 升级基线 (vs v1)

| 维度 | v1 (旧) | v2 (新) |
|---|---|---|
| **讲解粒度** | 模块级 (做什么) | **函数级** (每个 func 单独讲解) |
| **方法/类** | 偶尔提 | **每个 method 必讲**, 每个 struct 必讲 |
| **目标读者** | 研发 (懂 Go) | **没做过产品、研发的人** (零基础能懂) |
| **行号引用** | 偶尔 | **每个 func 必须 file:line** |
| **代码片段** | 偶尔 | **关键逻辑必须有 5-15 行代码块 + 逐行解释** |
| **类比** | 没有 | **每个抽象概念必须有生活类比** (1Panel 登录像小区门禁) |
| **Jargon** | 自由用 | **每个技术名词首次出现必须括号解释** (GORM = Go ORM 框架) |

---

## 3. 每个 func 的 8 段讲解模板 (Coder 必用)

```
### func <Name>()  <file:line>

**一句话作用**: <普通人能听懂的 1 句话>

**参数说明**:
- `name` (string) — 用户名, 最大 32 字符
- `password` (string) — 密码原文 (从 HTTP body 拿)

**返回值**:
- `(*dto.UserLoginInfo, error)` — 成功返回用户信息, 失败返回 error

**执行流程** (5-8 步, 每步 1 句话 + 行号):
1. 校验请求体 (auth.go:33-36) — 看用户名密码格式对不对
2. 拿真实 IP (auth.go:38) — `common.GetRealClientIP(c)` 从 HTTP header 取
3. 检查 IP 锁定 (auth.go:39) — 5 分钟内失败 10 次就锁
4. 校验图形验证码 (auth.go:43-49) — 防止机器人刷
5. 解析安全入口码 (auth.go:51-61) — 用户访问 /abc123/login 而不是 /login
6. 调用 AuthProvider.Login (auth.go:63) — 真正的登录逻辑
7. 写 session cookie (auth.go:162-182) — 下发 psession + pcsrftoken
8. 写登录日志 (auth.go:65 + 529-541) — 异步写 login_logs 表

**调用者** (谁会调它):
- BaseApi.Login (auth.go:32) — HTTP POST /api/v2/core/auth/login 的 handler

**被调用函数** (它调谁):
- common.GetRealClientIP (common/util.go:XX) — 拿真实 IP
- global.IPTracker.IsLocked — 看 IP 是否锁了
- captcha.VerifyCode — 校验图形码
- xpack.AuthProvider.Login — 真正的登录 (社区版走 app/auth.Login)
- app/auth.Login (auth.go:24) — 核心登录实现
- generateSession (auth.go:162) — 写 session
- saveLoginLogs (auth.go:529) — 写登录日志

**涉及数据库**:
- SELECT * FROM settings WHERE key='UserName' (auth.go:25-32)
- SELECT * FROM settings WHERE key='PASSWORD_PRIVATE_KEY' (auth.go:33)
- SELECT * FROM settings WHERE key='Password' (auth.go:34-40)
- SELECT * FROM settings WHERE key='SecurityEntrance' (auth.go:41-47)
- INSERT INTO login_logs ... (auth.go:65, 异步)

**关键代码** (5-15 行, 加逐行解释):

```go
// auth.go:25-32 — 读用户名
userName, err := settingRepo.GetValueByKey("UserName")
if err != nil {
    return nil, err  // 读库失败直接返回
}
if userName == "" {
    return nil, errors.New("用户名未设置")  // 第一次安装 1Panel 之前
}
// 注意: 这里不直接比对 req.Name, 而是固定从 settings 读
// 1Panel 是单管理员系统, 用户名不存在多用户表里
```

**类比** (让没做过研发的人能懂):
> 这个函数像 "小区门禁":
> 1. 保安先看你的访客证格式对不对 (校验请求)
> 2. 看摄像头记不记得你 (IP 锁定检查)
> 3. 让你输入图形验证码 (防机器人)
> 4. 看你知不知道小区暗门 (安全入口码)
> 5. 真正的"门"是 AuthProvider.Login (跟保安队长核对身份)
> 6. 给你发门禁卡 (session cookie)
> 7. 把这次进门记到日志里 (登录日志)
```

---

## 4. 每个 struct / class 必讲 (Coder 必用)

```markdown
## type User struct

**一句话作用**: 表示 1Panel 系统里的"用户"(其实是单管理员).

**字段说明** (每个字段 1 行 + 通俗解释):
- `ID uint` — 用户的数据库 ID, 自增, 永远唯一. 像身份证号.
- `Name string` — 用户名, 默认 "admin". 像你的网名.
- `Password string` — **RSA 加密后的明文密码**. 不是哈希!
- `Salt string` — 盐值, 用于加密. 像调味料.
- `MFAStatus bool` — 是否开启两步验证 (TOTP). true = 开了.
- `MFASecret string` — TOTP 种子, 用于生成 6 位动态码. 像共享密钥.
- `CreatedAt time.Time` — 账号创建时间.
- `UpdatedAt time.Time` — 最后修改时间.

**方法** (每个 method 单独讲解, 套用 §3 8 段模板):
- `func (u *User) VerifyPassword(plaintext string) bool` — 校验明文密码 (auth.go:XXX)
- `func (u *User) GenerateTOTP() string` — 生成动态码 (auth.go:XXX)
- ...
```

---

## 5. 1Panel v2 源码布局参考 (1Panel 专用)

| 路径 | 内容 | 角色 |
|---|---|---|
| `core/router/ro_*.go` | 路由注册 | Router (入口) |
| `core/middleware/*.go` | 中间件 (Session/CSRF/Password) | Middleware (网关) |
| `core/app/api/v2/*.go` | HTTP handler | Handler |
| `core/app/service/*.go` | 业务逻辑 (AuthService 等) | Service |
| `core/app/repo/*.go` | 数据访问 (SettingRepo 等) | Repository |
| `core/app/model/*.go` | GORM 模型 (User/Setting 等) | Model |
| `core/app/dto/*.go` | HTTP 请求/响应结构 | DTO |
| `core/utils/*` | 工具 (encrypt/mfa/captcha/passkey) | Utility |
| `core/init/*` | 启动初始化 (session/db/router/auth) | Init |
| `agent/router/ro_*.go` | 节点端路由 (25+ 文件) | Agent Router |
| `agent/app/service/*` | 节点端业务 (container/file/nginx 等) | Agent Service |
| `agent/utils/*` | 节点端工具 (firewall/iptables/docker) | Agent Utility |

> 1Panel v2 **没有**独立 Repository 层 (DB 操作直接在 Service 里通过 repo 包). 这是为了简化.

---

## 6. 5 Agent 流水线 (每个 module 跑 1 次)

### 6.1 Worker: 源码同步 (5s)

**Prompt** (named: `1Panel Git 同步`):
```
同步本地 1Panel 仓库 (D:\MiniMax Code\1Panel) 到指定分支 dev-v2.
完成后输出: 当前 Commit SHA / 新增文件数 / 修改文件列表 / 删除文件列表.
不要分析代码.
```

### 6.2 Explore: 逐函数源码分析 (5-10 min)

**Prompt** (named: `解析 1Panel <module> 模块 - 逐函数版`):
```
阅读 1Panel <module> 模块源码. 范围:
- <具体文件路径列表>

**输出格式 (必须严格按这个)**:

# 1Panel <module> 模块解析 - 逐函数

## 0. 模块一句话作用
<1 句话讲模块角色, 用普通人能懂的话>

## 1. 目录结构
(tree, 标 file:行数)

## 2. 路由 / 入口列表
| Method Path | Handler | File:Line | 一句话作用 |

## 3. 中间件 / 过滤器
(每个 middleware: 用途 + 跳过路径 + file:line)

## 4. Handler 列表 (每个独立 8 段)
(对每个 handler func 单独一节, 8 段)

## 5. Service 列表 (每个独立 8 段)
(对每个 service func 单独一节, 8 段)

## 6. Repository / DAO 列表
(每个 method 8 段)

## 7. Model / DTO 列表
(每个 struct/class 字段说明 + 每个 method 8 段)

## 8. 工具 / 辅助函数
(每个 util func 8 段)

## 9. 关键文件清单 (按重要性)
(每个文件 1-2 句说明)

## 10. 行号索引 (精确)
(表格: func 名 → file:line)

**严格**:
- 每个 func 必须 8 段齐全
- file:line 实际 grep 找, 不编造
- 一句话作用要让非技术读者能懂
- 5-15 行代码片段 + 逐行解释, 至少 3 个关键 func
- 至少 5 个生活类比 (门禁/银行/快递/电话簿/...)
```

### 6.3 Coder: 文档生成 (5-10 min)

**Prompt** (named: `生成 <module> 函数讲解 - 细到函数级别`):
```
根据 Explore 输出生成 KB 文档.

写到 modules/<NN>-<module>/HUMAN-READABLE.md.

**v2 基线 (必达)**:
- 细到**函数级 / 方法级 / 类级**: 每个 func 单独一节, 8 段齐全 (§3 模板)
- 每个 struct/class 字段说明 (§4 模板)
- 行号引用: 每个 func 必须 file:line
- 代码片段: 关键 func 5-15 行 + 逐行解释
- 类比: 每个抽象概念 1 个生活类比
- Jargon 解释: 首次出现技术名词括号解释

**必含章节**:
1. 一句话作用
2. 模块职责
3. 目录结构 (带行数)
4. 调用链 (Router → Middleware → Handler → Service → Repository → Model)
5. 数据库表
6. **逐函数讲解** (按 §3 8 段模板, 每个 func 独立小节)
7. **类/结构体讲解** (按 §4 模板, 每个 struct 独立小节)
8. Mermaid 时序图
9. Docker 相关依赖
10. 跟其他模块的关系
11. 关键文件清单
12. 行号索引

visual-atlas.html 必须:
- 9+ sections
- 10+ Mermaid 图 (时序 + 流程图)
- 可在浏览器打开渲染

**严格**:
- 不修改 modules/01-13/ / .scheduler/ / README / .gitignore
- 不删除任何文件
- UTF-8 无 BOM
- HR.md ≥ 50 KB, VA.html ≥ 20 KB
```

### 6.4 Verifier: 质量校验 (1-2 min)

**Prompt** (named: `校验 <module> 文档 - 100% 必查`):
```
对照 1Panel 源码 (commit 2dea44a) 校验 modules/<NN>-<module>/ 文档.

**必查项 (100%, 不能跳过)**:
A. 函数名 (10 抽样) — grep 验证
B. API 路径 (5 抽样) — grep 验证
C. SQL 表 (3 抽样) — GORM 模型验证
D. 中间件 (5 抽样) — grep 验证
E. Mermaid 时序图 (visual-atlas.html) — 调用关系核对
F. 行号 (20 抽样) — Get-Content 验证
G. Docker 依赖 — import grep
H. 文件路径 (15 抽样) — Test-Path 验证
I. 设计点 (5 抽样) — 跟源码逻辑一致
J. Handler 覆盖度 (5 反向抽样) — 全部 handler 在 HR.md 都有
K. **新增**: 每个 func 8 段齐全 (purpose/params/returns/flow/callers/callees/db/code) — 抽样 5 个 func
L. **新增**: 每个 struct 字段说明 — 抽样 3 个 struct

**总体判定**:
- A-J 全部 PASS + K-L 通过 → 总体 PASS
- 任何 FAIL → 总体 FAIL, 列出 file:line + 文档章节
```

---

## 7. 12 模块优先级 + 推进顺序 (按 user spec)

| 优先级 | 模块 | 1Panel 位置 | 状态 |
|---|---|---|---|
| ⭐⭐⭐ | **Auth 登录认证** | `core/app/auth + service/auth + api/v2/auth.go` | ✅ **已完成** (v2 90.6 KB HR + 48.6 KB VA, Verifier PASS) |
| ⭐⭐⭐ | **Docker 容器** | `agent/router/ro_container.go + agent/app/service/` | ⏳ **下一跑** (5 Agent 4 步) |
| ⭐⭐⭐ | **App 应用商店** | `agent/router/ro_app.go` | pending |
| ⭐⭐⭐ | **Website 网站** | `agent/router/ro_website*.go` (8 files) | pending |
| ⭐⭐ | Nginx | `agent/router/ro_nginx.go` | pending |
| ⭐⭐ | SSL 证书 | `agent/router/ro_website_ssl.go + acme_account.go` | pending |
| ⭐⭐ | CronJob | `agent/router/ro_cronjob.go` | pending |
| ⭐⭐ | File | `agent/router/ro_file.go` | pending |
| ⭐ | Monitor 监控 | `agent/router/ro_host.go` | pending |
| ⭐ | Backup 备份 | `agent/router/ro_backup.go + core/service/backup.go` | pending |
| ⭐ | Settings 系统设置 | `agent/router/ro_setting.go + core/service/setting.go` | pending |
| ⭐ | Terminal WebSSH | `agent/router/ro_process.go + toolbox.go` | pending |

### 7.1 KB 已有但 user 未列优先级 (也要升级到 v2 基线)

| 现有 | 对应 user 12 | 升级策略 |
|---|---|---|
| 04-database | (未列) | 跑 5 Agent v2 |
| 07-alert | (未列) | 跑 5 Agent v2 |
| 09-ai-agent | (未列) | 跑 5 Agent v2 |
| 11-runtime-ai | (未列) | 跑 5 Agent v2 |
| 13-frontend | (未列) | 跑 5 Agent v2 (Vue 源码, 跟 Go 模式略不同) |

---

## 8. 推进节奏估算

| 阶段 | 模块数 | 4 步 × time | 总耗时 |
|---|---:|---|---|
| ⭐⭐⭐ 完成 | 1 | 4 × 5-10 min | 20-40 min (Auth ✅) |
| ⭐⭐⭐ 剩余 | 3 | 4 × 5-10 min | 60-120 min (Docker/App/Website) |
| ⭐⭐ 4 个 | 4 | 4 × 5-10 min | 80-160 min (Nginx/SSL/CronJob/File) |
| ⭐ 4 个 | 4 | 4 × 5-10 min | 80-160 min (Monitor/Backup/Settings/Terminal) |
| KB 已有 5 个 | 5 | 4 × 5-10 min | 100-200 min (database/alert/ai-agent/runtime-ai/frontend) |
| **总计** | **17** | 4 × 5-10 min | **340-680 min (5.7-11.3 hour)** |

> 1 module 1 验收 (按 user "串行单功能验收" 原则). 实际跑 ⭐⭐⭐ 剩余 3 个先看效果, 决定后续节奏.

---

## 9. 验收标准 (1 module 跑完 4 步后)

- [ ] Worker 输出: SHA + 新增/修改/删除文件数
- [ ] Explore 输出: 完整调用链 + **逐函数 8 段 + 逐 struct 字段** + 至少 5 个生活类比
- [ ] Coder 输出: `modules/<NN>-<module>/HUMAN-READABLE.md` (≥ 50 KB, **每个 func 8 段**, **每个 struct 字段说明**) + `visual-atlas.html` (≥ 20 KB, 9+ section, 10+ Mermaid)
- [ ] Verifier 输出: PASS (A-L 全部通过, 0 误差)
- [ ] git commit + push (通过 daemon git-sync task)
- [ ] KB-INDEX.md 自动更新 (daemon 跑 index-rebuild)

---

## 10. 不入库规则 (沿用)

- `.scheduler/*.py` — Python 源码不入仓
- `*.ps1 / *.bat / *.cmd / *.vbs` — 全部排除
- `.backups/` — 自动备份
- `_archive_*` — 历史版本

**入仓**: skeleton/ (Go) + modules/ (KB) + firewall-architecture.md + KB-INDEX/README/LICENSE.

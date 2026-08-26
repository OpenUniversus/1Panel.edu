# 1Panel 12-ssl（SSL 证书管理）模块 — 人类可读文档 v3

> 1Panel v2 节点端（agent）源码逐函数级讲解 · 适合零基础读者
> 配套可视化图集: `visual-atlas.html`（8+ Mermaid 图）
> 对应源码: `agent/router/ro_website_ssl.go` + `agent/router/ro_website_acme_account.go` + `agent/app/api/v2/website_ssl.go` + `agent/app/service/website_ssl.go` + `agent/utils/ssl/*`

---

## 1. 一句话作用

**12-ssl 模块就是 1Panel 的「证书管家」**：帮你把 Let's Encrypt 等 CA 颁发的 SSL 证书申请下来、上传已有的证书、按时自动续期、推送给绑定的网站用，让用户访问你的网站时浏览器地址栏出现那把绿色小锁 🔒。

类比：SSL 证书就像网站的"身份证 + 健康证"。身份证证明"这个域名是你的"，健康证证明"这张证书没过期"——12-ssl 就是帮你去办证、续证、存证、给网站佩戴证书的政府部门。

---

## 2. 模块职责

- **申请证书**：对接 Let's Encrypt / ZeroSSL / Google Trust Services / Buypass / FreeSSL / 自定义 ACME CA，通过 HTTP-01（让你服务器上跑 OpenResty 提供 `.well-known/acme-challenge` 文件）或 DNS-01（在你的 DNS 服务商添加 TXT 记录）两种方式证明域名是你的，自动签发免费证书。
- **管理证书**：把证书存到 SQLite 数据库 `website_ssls` 表里，记录主域名、SAN 多域名、过期时间、状态（init/ready/applying/applyError/error）等。
- **上传/导入证书**：支持 PEM 文本粘贴、服务器本地文件路径两种方式上传；也支持从 1Panel 主节点导入证书到工作节点（多节点 XPack 功能）。
- **自动续期**：通过 cron 任务每天扫一遍即将过期的证书（域名证书 < 30 天、IP 证书 < 72 小时），自动重新走 ACME 流程签发新证书。
- **重载 Nginx (OpenResty)**：证书签发成功后自动写一份到 OpenResty 的 `conf.d/ssl/`，然后 `nginx -s reload`，让网站立刻用新证书。
- **DNS 解析预检**：在申请 DNS-01 证书前，先查一下你的域名是否真的指向了当前服务器 IP（防止申请被 CA 拒绝）。
- **ACME 账户管理**：每个 CA（Let's Encrypt 等）对应一个 ACME 账户，记录邮箱 + 私钥 + 注册 URL + EAB 凭据（ZeroSSL/Google 需要），账户可代理访问。

---

## 3. 目录结构

```
agent/                                                          ← 1Panel 节点端
├── router/                                                     ← 路由层
│   ├── ro_website_ssl.go                  (1.1 KB / 30 行)     12 个 SSL 证书 API
│   └── ro_website_acme_account.go         (0.6 KB / 20 行)     4 个 ACME 账户 API
├── app/
│   ├── api/v2/                                                 ← HTTP handler 层
│   │   ├── website_ssl.go                 (11.7 KB / 380 行)   14 个 SSL handler
│   │   └── website_acme_account.go        (3.3 KB / 96 行)    4 个 ACME 账户 handler
│   ├── dto/
│   │   ├── request/website_ssl.go         (6.5 KB / 184 行)   SSL 请求体 (Create/Update/Upload/Apply/Push/Renew/...)
│   │   └── response/website_ssl.go        (0.8 KB / 34 行)    SSL 响应体 (WebsiteSSLDTO/WebsiteDNSRes/...)
│   ├── model/
│   │   ├── website_ssl.go                 (2 KB / 57 行)      WebsiteSSL GORM 模型 (主表)
│   │   └── website_acme_account.go        (0.6 KB / 19 行)    WebsiteAcmeAccount GORM 模型
│   ├── repo/
│   │   ├── website_ssl.go                 (3.8 KB / 128 行)   SSL 仓储 (Page/List/GetFirst/Create/Save/TryMarkApplying/...)
│   │   └── website_acme_account.go        (1.9 KB / 64 行)    ACME 账户仓储
│   ├── service/
│   │   ├── website_ssl.go                 (34.4 KB / 1090 行) SSL 业务逻辑（主线：obtainSSL 700 行）
│   │   └── website_acme_account.go        (3.5 KB / 100 行)   ACME 账户业务
│   └── task/                                                    (共用任务系统)
├── cron/job/                                                    ← 定时任务层
│   └── ssl.go                             (2.1 KB / 62 行)     每日证书续期 cron job
├── middleware/                                                  ← 中间件
│   └── certificate.go                     (1.3 KB / 56 行)     节点证书校验 (多节点 XPack)
└── utils/ssl/                                                   ← 工具层
    ├── acme.go                            (11.7 KB / 423 行)   ACME 客户端工厂 (NewRegisterClient/getCaDirURL/...)
    ├── client.go                          (7.4 KB / 166 行)   AcmeClient (UseDns/UseHTTP/ObtainSSL/ObtainIPSSL/RevokeSSL)
    ├── dns_provider.go                    (13.8 KB)            DNS 厂商 provider (Alidns/Tencent/Huawei/...)
    └── manual_client.go                   (13.3 KB)            手动 DNS 申请客户端 (DnsManual 流程)
```

总计 12 个核心 .go 文件（不含 dns_provider/manual_client） + 1 个 cron + 1 个 middleware + 1 个 model，加起来约 **67 KB 源码 / ~2400 行**。

---

## 4. 调用链

### 4.1 整体调用链（Router → Middleware → Handler → Service → Repository → Model）

```
┌──────────────────────────────────────────────────────────────────────┐
│  Browser / API client                                                 │
└────────────┬─────────────────────────────────────────────────────────┘
             │ HTTPS (TLS 1.3)
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Router: agent/router/ro_website_ssl.go  (12 路由)                   │
│    POST /websites/ssl/search    → PageWebsiteSSL                     │
│    POST /websites/ssl/list      → ListWebsiteSSL                     │
│    POST /websites/ssl           → CreateWebsiteSSL                   │
│    POST /websites/ssl/resolve   → GetDNSResolve                      │
│    POST /websites/ssl/del       → DeleteWebsiteSSL                   │
│    GET  /websites/ssl/:id       → GetWebsiteSSLById                  │
│    GET  /websites/ssl/website/:websiteId → GetWebsiteSSLByWebsiteId  │
│    POST /websites/ssl/update    → UpdateWebsiteSSL                   │
│    POST /websites/ssl/push      → PushWebsiteSSLToNode (XPack)       │
│    POST /websites/ssl/upload    → UploadWebsiteSSL                   │
│    POST /websites/ssl/upload/file → UploadSSLFile                    │
│    POST /websites/ssl/obtain    → ApplyWebsiteSSL                    │
│    POST /websites/ssl/download  → DownloadWebsiteSSL                 │
│    POST /websites/ssl/import    → ImportMasterSSL                    │
└────────────┬─────────────────────────────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Middleware: middleware/certificate.go                                │
│    Certificate() — Master 节点直接放行; Worker 节点校验节点证书 +   │
│    Proxy-Id 头 防止节点被中间人重放                                   │
└────────────┬─────────────────────────────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Handler: agent/app/api/v2/website_ssl.go                            │
│    PageWebsiteSSL/ListWebsiteSSL/CreateWebsiteSSL/ApplyWebsiteSSL   │
│    /DeleteWebsiteSSL/UpdateWebsiteSSL/UploadWebsiteSSL/...          │
│    职责: 解析 body → 调 service → 返回 JSON                          │
└────────────┬─────────────────────────────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Service: agent/app/service/website_ssl.go                           │
│    IWebsiteSSLService 接口 (14 方法):                                │
│      Page/GetSSL/Search/Create/GetDNSResolve/GetWebsiteSSL/Delete  │
│      /Update/Upload/PushToNode/ObtainSSL/AutoRenewSSL/SyncForRestart│
│      /DownloadFile/ImportMasterSSL                                   │
│    主线: obtainSSL() 触发 ACME 申请 + 写库 + 重载 Nginx + 续期      │
└────────────┬─────────────────────────────────────────────────────────┘
             │ 三个 repo:  websiteSSLRepo / websiteAcmeRepo / websiteDnsRepo
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Repository: agent/app/repo/website_ssl.go                           │
│    ISSLRepo 接口: Page/List/GetFirst/Create/Save/DeleteBy          │
│    /TryMarkApplying/SaveByMap/WithByDomain/WithByAcmeAccountId/...  │
│    GORM 操作 SQLite 表 `website_ssls` + 关联 Preload AcmeAccount    │
└────────────┬─────────────────────────────────────────────────────────┘
             │
             ▼
┌──────────────────────────────────────────────────────────────────────┐
│  Model: agent/app/model/website_ssl.go                               │
│    WebsiteSSL struct (GORM 模型, TableName = "website_ssls")        │
│    字段: PrimaryDomain/Pem/PrivateKey/Status/ExpireDate/AutoRenew  │
│    /AcmeAccountID/DnsAccountID/PushDir/PushNode/...                 │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.2 自动续期调用链（cron → service）

```
agent/init/runtime/   ← 启动时注册 cron job
        │
        │  每天 00:00 / 12:00 触发 (scheduler)
        ▼
agent/cron/job/ssl.go: Run()
        │
        │  1. repo.List() 拿所有证书
        │  2. service.SyncSystemSSL() 同步 1Panel 面板自己的证书
        │  3. 遍历每个 SSL:
        │     - 不是 AutoRenew / 是 manual / 在 applying 状态 → 跳过
        │     - 域名证书 < 30 天过期 → AutoRenewSSL(id)
        │     - IP 证书 < 72 小时过期 → AutoRenewSSL(id)
        │     - SelfSigned 类型 → 走 CA 模块的 ObtainSSL(Renew=true)
        ▼
agent/app/service/website_ssl.go: AutoRenewSSL(id)
        │
        │  → obtainSSL(id, autoRenew=true)
        │     - 走完整 ACME 流程: 拿账户 → DNS 验证 → 签发 → 写库
        │     - 续期时 logger 用 O_APPEND 模式打开日志文件
        │     - 加 banner: "==== [time] auto renew attempt ===="
        ▼
agent/app/service/website_ssl.go: obtainSSL() 内部 goroutine
        │
        │  → saveCertificateFile() 写 fullchain.pem + privkey.pem 到 OpenResty
        │  → opNginx("nginx -s reload") 触发 OpenResty 重载
        │  → reloadSystemSSL() 如果是面板证书则替换 server.crt + server.key
        │  → pushSSLToNode() 如果是 XPack 多节点则推证书到子节点
```

### 4.3 ACME 账户独立调用链

```
agent/router/ro_website_acme_account.go  (4 路由)
  POST /websites/acme/search
  POST /websites/acme                  → CreateWebsiteAcmeAccount
  POST /websites/acme/del              → DeleteWebsiteAcmeAccount
  POST /websites/acme/update           → UpdateWebsiteAcmeAccount
        │
        ▼
agent/app/service/website_acme_account.go
  Create() → ssl.NewRegisterClient()  ← 真正去 CA 注册账户 + 拿 EAB (ZeroSSL/Google)
  Delete() → 校验无关联证书才删
  Update() → 仅改 UseProxy 字段
        │
        ▼
agent/app/repo/website_acme_account.go  ← GORM 操作表 `website_acme_accounts`
```

---

## 5. 数据库表

### 5.1 `website_ssls`（证书主表）— 来自 `model/website_ssl.go`

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint（自增主键） | 证书记录 ID |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `primary_domain` | string | 主域名（如 `example.com`） |
| `private_key` | string (text) | PEM 格式的私钥，base64 编码 |
| `pem` | string (text) | 完整证书链 fullchain（站点证书 + 中间证书） |
| `domains` | string | 其他域名（SAN），逗号分隔（如 `www.example.com,api.example.com`） |
| `cert_url` | string | CA 返回的证书吊销 URL（用于吊销） |
| `type` | string | 颁发者 CN（CA 名字，如 `R3`/`R10`/`ZeroSSL ECC Domain Secure Site CA`） |
| `provider` | string | 申请方式: `dnsAccount`/`http`/`dnsManual`/`selfSigned`/`manual`/`fromMaster` |
| `organization` | string | 颁发者组织（如 `Let's Encrypt`/`ZeroSSL`） |
| `dns_account_id` | uint | DNS 账户 ID（外键到 `website_dns_accounts`） |
| `acme_account_id` | uint | ACME 账户 ID（外键到 `website_acme_accounts`） |
| `ca_id` | uint | 自签名 CA ID（仅 selfSigned 类型使用） |
| `auto_renew` | bool | 是否自动续期 |
| `expire_date` | datetime | 证书过期时间 |
| `start_date` | datetime | 证书生效时间 |
| `status` | string | `init` / `ready` / `applying` / `applyError` / `error` / `systemRestart` |
| `message` | string | 错误信息（申请失败时存详细原因） |
| `key_type` | string | 私钥类型: `EC256`/`EC384`/`RSA2048`/`RSA3072`/`RSA4096` |
| `push_dir` | bool | 是否推送证书到指定目录（OpenResty 之外额外存一份） |
| `dir` | string | 推送目录路径 |
| `description` | string | 用户备注 |
| `skip_dns` | bool | DNS 验证时是否跳过 DNS propagation check |
| `nameserver1` | string | 自定义 DNS 服务器 1（用于 DNS 查询） |
| `nameserver2` | string | 自定义 DNS 服务器 2 |
| `disable_cname` | bool | 禁用 CNAME 支持（DNS-01 时直接查权威而不是走 CNAME） |
| `exec_shell` | bool | 证书签发后是否执行用户脚本 |
| `shell` | string | 用户脚本内容 |
| `master_ssl_id` | uint | 从主节点导入的证书 ID（仅 `fromMaster` 类型） |
| `nodes` | string | 推送到哪些子节点 ID（逗号分隔） |
| `push_node` | bool | 是否启用多节点推送（XPack） |
| `private_key_path` | string | 本地文件路径（仅 upload type=local） |
| `cert_path` | string | 证书本地文件路径 |
| `is_ip` | bool | 是否是 IP 证书（IP cert 走短生命周期 14 天） |

**关联**（GORM Preload，不建外键约束）：
- `AcmeAccount` → `website_acme_accounts`（按 ID）
- `DnsAccount` → `website_dns_accounts`
- `Websites` → `websites`（通过 `website_ssl_id` 反向关联）

### 5.2 `website_acme_accounts`（ACME 账户表）— 来自 `model/website_acme_account.go`

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint | 账户 ID |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |
| `email` | string (not null) | 账户邮箱（CA 注册用，必填） |
| `url` | string (not null) | CA 返回的账户 URL（用于后续通信） |
| `private_key` | string (not null) | 账户私钥（PEM，ACME 协议签名用） |
| `type` | string (not null, default: letsencrypt) | `letsencrypt`/`zerossl`/`buypass`/`google`/`freessl`/`custom` |
| `eab_kid` | string | EAB Key ID（ZeroSSL/Google/FreeSSL 必填） |
| `eab_hmac_key` | string | EAB HMAC Key |
| `key_type` | string (default: RSA2048) | 账户私钥类型 |
| `use_proxy` | bool (default: false) | 是否走系统代理 |
| `ca_dir_url` | string | 自定义 CA ACME Directory URL（仅 `custom` 类型） |
| `use_eab` | bool | 自定义 CA 是否使用 EAB |

### 5.3 `website_dns_accounts`（DNS 账户表）

不在本模块代码内，但被本模块引用，用于 DNS-01 验证时调用对应 DNS 厂商的 API 添加 TXT 记录。表结构由 `model/website_dns.go` 定义。

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | uint | 账户 ID |
| `name` | string | 用户起的名字（如 "我的阿里云"） |
| `type` | string | DNS 厂商类型: `Aliyun`/`Tencent`/`Huawei`/`Cloudflare`/`GoDaddy`/`NameSilo`/`Namecheap`/`Dnspod`/... |
| `authorization` | string (JSON) | 各厂商的 API 凭据（access key/secret/token） |

---

## 6. 逐函数讲解（4 段模板）

> 选 6 个 service 主线函数 + 2 个 ACME 工具函数，按 v3 简化 4 段模板（purpose / params / flow / callees）讲。剩余函数用一句话索引列在第 12 节。

### 6.1 `func (w WebsiteSSLService) Create(create request.WebsiteSSLCreate) (request.WebsiteSSLCreate, error)`  —  `service/website_ssl.go:180`

**一句话作用**：用户填写完"主域名 + 申请方式 + ACME 账户"后，创建一个 SSL 证书申请记录，**如果申请方式不是手动 DNS（dnsManual），立刻在 goroutine 里启动 ACME 申请流程**。

**参数说明**：
- `create.PrimaryDomain` (string) — 主域名（必填，如 `example.com`）
- `create.OtherDomains` (string) — SAN 多域名，每行一个（可选）
- `create.Provider` (string) — 申请方式: `dnsAccount`/`http`/`dnsManual`/`selfSigned`/`manual`
- `create.AcmeAccountID` (uint) — 关联的 ACME 账户 ID（必填）
- `create.DnsAccountID` (uint) — DNS 账户 ID（仅 `dnsAccount` 方式必填）
- `create.AutoRenew` (bool) — 是否自动续期
- `create.KeyType` (string) — 私钥类型 EC256/RSA2048 等
- `create.PushDir` / `create.Dir` — 是否推送证书到指定目录
- `create.ExecShell` / `create.Shell` — 申请后是否执行用户脚本
- `create.Nameserver1` / `create.Nameserver2` — 自定义 DNS 服务器（用于 DNS 查询）
- `create.IsIp` (bool) — 是否是 IP 证书

**执行流程**（5 步 + 行号）：
1. 校验两个 DNS 服务器地址（如果是 IP 格式）— `service/website_ssl.go:181-186`
2. 加载 ACME 账户（必填项），构造 `WebsiteSSL` 模型，`Status = SSLInit` — `:188-210`
3. 校验 `OtherDomains` 格式（每个域名都必须是合法格式），如果是 `http` 方式不允许有通配符 `*` — `:221-241`
4. 如果 `dnsAccount` 方式，加载 DNS 账户；写库到 `website_ssls` 表；创建空日志文件 — `:243-263`
5. 启动后台 goroutine 调 `obtainSSL(websiteSSL.ID, false)` 真正去 CA 申请证书（dnsManual 类型跳过这一步等用户手动添加 DNS 记录后再触发） — `:264-272`

**被调用函数**（它调谁）：
- `common.IsValidIP` (`utils/common/util.go`) — 校验 IP 格式
- `common.IsValidDomain` (`utils/common/util.go`) — 校验域名格式
- `buserr.New` / `buserr.WithName` (`buserr/buserr.go`) — 国际化错误码
- `files.NewFileOp().Stat` — 检查目录是否存在
- `setSSLPushConfig` (`:291`) — 设置多节点推送标志（Master + XPack 才生效）
- `websiteAcmeRepo.GetFirst` — 查 ACME 账户
- `websiteDnsRepo.GetFirst` — 查 DNS 账户
- `websiteSSLRepo.Create` — 写库
- `w.ObtainSSL` → `w.obtainSSL` (`:419-421` → `:427-616`) — 真正走 ACME 申请流程（在 goroutine 里跑）

---

### 6.2 `func (w WebsiteSSLService) obtainSSL(id uint, autoRenew bool) error`  —  `service/website_ssl.go:427`

**一句话作用**：**SSL 申请的核心引擎**——加载证书记录 + ACME 账户、抢占"申请中"状态防并发、根据 `Provider` 类型走 HTTP-01 或 DNS-01 申请、签发成功后写库 + 重载 Nginx + 推节点。

**参数说明**：
- `id` (uint) — 证书记录 ID
- `autoRenew` (bool) — 是否是自动续期触发（影响日志模式：续期用 `O_APPEND` 并加 banner）

**执行流程**（8 步 + 行号）：
1. 加载 `WebsiteSSL` 记录；如果已经在 `SSLApply`（申请中）状态直接返回 `InExecuting` 错误 — `:439-445`
2. 加载 `AcmeAccount`；合并域名（主域名 + `Domains` 逗号分割）— `:446-453`
3. 根据 `Provider` 类型准备参数：`dnsAccount` 加载 DNS 账户；`http` 加载 OpenResty 安装路径并校验不能有通配符 — `:454-476`
4. 调 `websiteSSLRepo.TryMarkApplying(id)` 原子地把 `Status = SSLApply`，失败说明别处正在申请 — `:477-483`
5. 创建日志文件 + logger；goroutine 里跑主流程：
   - 私钥：调 `ssl.GetPrivateKeyByType` 拿 `crypto.Signer`（如果数据库里没存就新生成）— `:509-513`
   - 通过 `withLegoLoggerTimeout` 切到 lego 库的 logger（统一日志格式），超时 1 小时 — `:514-535`
   - 调 `newWebsiteSSLLegoClient` 拿 lego `AcmeClient`（如果账户 URL 缺失就调 `NewRegisterClient` 注册新账户）— `:515` + `service/website_ssl.go:70-95`
   - 选 challenge 类型：`dnsAccount` → `client.UseDns`；`http` → `client.UseHTTP(httpRoot)` — `:519-528`
   - 调 `client.ObtainSSL` / `client.ObtainIPSSL` 真正去 CA 拿证书（`isIP` 走 IP cert 流程）— `:529-533`
6. 拿到 `certificate.Resource`（含 `PrivateKey` + `Certificate`）后：
   - 写回 `WebsiteSSL.PrivateKey` / `Pem` / `CertURL` — `:555-557`
   - `parseCertificatePEM` 解析证书得 `ExpireDate` / `StartDate` / `Type` / `Organization` — `:558-568`
   - `Status = SSLReady` — `:569`
   - 调 `saveCertificateFile` 写证书到 OpenResty 的 `conf.d/ssl/<domain>/` — `:571`
7. 查所有绑定此证书的 `Website`：
   - 调 `createPemFile` 写证书到网站的 vhost 目录 — `:591-598`
   - 调 `opNginx(nginxInstall.ContainerName, NginxReload)` 重载 OpenResty（`docker exec <container> nginx -s reload`）— `:599-605`
8. 如果证书被设为"面板证书"（setting `SSL=enable` + `SSLID=this.id`），调 `reloadSystemSSL` 把 `data/secret/server.crt` + `server.key` 覆盖并通知 core 端 reload TLS；最后如果 `push_node=true` 走 `pushSSLToNode` 推给子节点 — `:607-612`

**被调用函数**：
- `websiteSSLRepo.GetFirst` / `TryMarkApplying` / `Save` — GORM 操作
- `websiteAcmeRepo.GetFirst` — 查 ACME 账户
- `websiteDnsRepo.GetFirst` — 查 DNS 账户
- `websiteRepo.GetBy` — 查所有绑定此证书的网站
- `getAppInstallByKey(constant.AppOpenresty)` — 查 OpenResty 安装记录
- `newWebsiteSSLLegoClient` (`service/website_ssl.go:70`) — 拿 lego 客户端
- `ssl.GetPrivateKeyByType` (`utils/ssl/acme.go:378`) — 加载/生成私钥
- `withLegoLoggerTimeout` (`service/website_ssl.go:62`) — 切 logger + 加超时
- `newWebsiteSSLLogger` (`service/website_ssl.go:324`) — 创建日志文件
- `client.UseDns` / `client.UseHTTP` / `client.ObtainSSL` / `client.ObtainIPSSL` (`utils/ssl/client.go`) — lego 调用
- `parseCertificatePEM` (`service/website_ssl.go:635`) — 解析证书
- `saveCertificateFile` — 写证书文件
- `createPemFile` — 写网站证书文件
- `opNginx` (`utils/nginx`) — 触发 OpenResty 重载
- `reloadSystemSSL` (`service/website_ssl.go:346`) — 重载面板自己的证书
- `pushSSLToNode` / `pushSSLToNodeWithNewLogger` — 推证书到子节点
- `handleError` (`service/website_ssl.go:643`) — 统一错误处理

---

### 6.3 `func (w WebsiteSSLService) Upload(req request.WebsiteSSLUpload) error`  —  `service/website_ssl.go:830`

**一句话作用**：**用户自己已有证书时**（不想用 ACME 申请），支持两种方式上传：（1）"paste" 模式粘贴 PEM 文本；（2）"local" 模式读服务器上的证书文件路径。解析证书后写库 + 推节点。

**参数说明**：
- `req.Type` (string) — `paste` 或 `local`
- `req.PrivateKey` / `req.Certificate` (string) — paste 模式下的私钥 + 证书内容
- `req.PrivateKeyPath` / `req.CertificatePath` (string) — local 模式下的文件路径
- `req.SSLID` (uint) — 如果 > 0 则更新已有证书；否则新建
- `req.Description` / `req.PushNode` / `req.Nodes` — 备注和多节点推送

**执行流程**（6 步 + 行号）：
1. 构造 `WebsiteSSL{Provider: Manual, Status: SSLReady}`；如果 `SSLID>0` 则加载已有记录 — `:831-845`
2. 根据 `Type` 走不同分支：`local` 用 `files.NewFileOp().GetContent` 读文件；`paste` 直接用入参；把内容写到 `websiteSSL.PrivateKey` 和 `websiteSSL.Pem` — `:846-871`
3. 校验私钥 PEM 格式（`pem.Decode` 不为空）— `:873-876`
4. 解析证书：从 fullchain 里找到 leaf cert（`cert.DNSNames` 或 `cert.IPAddresses` 至少有一个），拿 `ExpireDate`/`StartDate`/`Type`/`Organization` — `:878-907`
5. 提取主域名 + SAN：从 `cert.DNSNames[0]` 取主域名，剩余进 `domains`；如果没 DNSNames 就从 `cert.IPAddresses` 取 — `:909-926`
6. 根据 `ID` 判断新建还是更新：更新时先调 `UpdateSSLConfig`（写 vhost 配置 + reload），再 `Save` + `pushSSLToNodeWithNewLogger`；新建时直接 `Create` + push — `:928-940`

**被调用函数**：
- `files.NewFileOp().Stat` / `GetContent` — 文件操作
- `pem.Decode` / `x509.ParseCertificate` — 证书解析
- `buserr.New` — 错误国际化
- `setSSLPushConfig` — 推送配置
- `UpdateSSLConfig` — 同步更新网站 vhost 配置
- `websiteSSLRepo.GetFirst` / `Create` / `Save` — GORM 操作
- `pushSSLToNodeWithNewLogger` — 推子节点

---

### 6.4 `func (w WebsiteSSLService) Delete(ids []uint) error`  —  `service/website_ssl.go:699`

**一句话作用**：批量删除证书记录，**有网站正在用 / 证书是面板证书 / 正在申请中**这三种情况会被拦截；如果是 ACME 签发的证书还会在后台 goroutine 调 `client.RevokeSSL` 告诉 CA 吊销证书。

**参数说明**：
- `ids` ([]uint) — 要删除的证书 ID 列表

**执行流程**（5 步 + 行号）：
1. 遍历每个 id：先用 `websiteRepo.GetBy(WithWebsiteSSLID(id))` 查有没有网站在用，有就记进 `websiteSSLS` 跳过 — `:704-711`
2. 如果证书被设为"面板证书"（setting `SSL=enable` + `SSLID==id`），返回 `ErrDeleteWithPanelSSL` 拦截 — `:712-719`
3. 加载证书；如果状态是 `SSLApply`（正在申请）则记进 `applySSLS` 跳过 — `:720-727`
4. 如果是 ACME 签发的证书（`Provider != Manual && != SelfSigned`），启动 goroutine 调 `client.RevokeSSL(pem)` 告诉 CA 吊销 — `:728-747`
5. 调 `websiteSSLRepo.DeleteBy(WithByID(id))` 删记录；最后如果有 `websiteSSLS` 或 `applySSLS` 残留返回国际化错误 — `:748-756`

**被调用函数**：
- `websiteRepo.GetBy` — 反向查网站
- `websiteSSLRepo.GetFirst` / `DeleteBy` — GORM 操作
- `settingRepo.Get` — 查系统设置
- `websiteAcmeRepo.GetFirst` — 查 ACME 账户（吊销用）
- `newWebsiteSSLLegoClient` (`service/website_ssl.go:70`) — 拿吊销用的 lego 客户端
- `withLegoLogger` (`service/website_ssl.go:47`) — 切 lego logger
- `client.RevokeSSL` (`utils/ssl/client.go`) — 调 lego 吊销
- `buserr.New` / `buserr.WithName` — 错误国际化

---

### 6.5 `func (w WebsiteSSLService) Update(update request.WebsiteSSLUpdate) error`  —  `service/website_ssl.go:759`

**一句话作用**：修改证书记录（主域名、SAN、DNS 服务器、是否自动续期、多节点推送等），**不触发重新申请**（改 Provider/AcmeAccount 也不会重新走 ACME 流程）。

**参数说明**：
- `update.ID` (uint) — 证书 ID（必填）
- `update.PrimaryDomain` / `update.OtherDomains` / `update.Description` — 基础信息
- `update.Provider` / `update.AcmeAccountID` / `update.DnsAccountID` — 申请配置
- `update.AutoRenew` / `update.DisableCNAME` / `update.SkipDNS` / `update.Nameserver1/2` — 申请行为配置
- `update.PushDir` / `update.Dir` / `update.ExecShell` / `update.Shell` — 部署行为
- `update.PushNode` / `update.Nodes` — 多节点推送（仅 Master + XPack 生效）

**执行流程**（5 步 + 行号）：
1. 加载 `WebsiteSSL` 记录 — `:760-763`
2. 构造 `updateParams` map：基础字段 + PushNode（Master + XPack 校验）+ ExecShell 的 Shell 内容 — `:764-786`
3. 如果原来不是 SelfSigned/Manual，加载 ACME 账户并更新 `acme_account_id` — `:787-793`
4. 如果启用 PushDir 校验目录（不存在就 `CreateDir` 自动创建）— `:795-801`
5. 解析 `OtherDomains` 校验每个域名格式；按 Provider 类型决定 `auto_renew` / `dns_account_id`；最后调 `websiteSSLRepo.SaveByMap` 一次性 UPDATE — `:802-827`

**被调用函数**：
- `websiteSSLRepo.GetFirst` / `SaveByMap` — GORM 操作
- `websiteAcmeRepo.GetFirst` — 查 ACME 账户
- `websiteDnsRepo.GetFirst` — 查 DNS 账户
- `files.NewFileOp().Stat` / `CreateDir` — 文件操作
- `common.IsValidDomain` — 域名格式校验
- `normalizeSSLPushConfig` (`service/website_ssl.go:283`) — 标准化推送配置
- `buserr.WithName` — 错误国际化

---

### 6.6 `func (w WebsiteSSLService) PushToNode(req request.WebsiteSSLPush) error`  —  `service/website_ssl.go:943`

**一句话作用**：**XPack 多节点功能**——把当前节点的证书推到指定子节点；可以选择同步推（`Sync=true` 阻塞）或异步推（通过 `task` 任务系统后台跑）。

**参数说明**：
- `req.ID` (uint) — 证书 ID
- `req.PushNode` (bool) — 是否启用推送
- `req.Nodes` (string) — 子节点 ID 列表（逗号分隔）
- `req.Sync` (bool) — true 同步推，false 异步推
- `req.TaskID` (string) — 异步任务 ID（用于进度跟踪）

**执行流程**（6 步 + 行号）：
1. 校验当前是 Master 节点（`global.IsMaster`） + XPack 已启用 — `:944-949`
2. 标准化推送配置 + 校验 `pushNode=true` 且 `Nodes` 非空 — `:950-953`
3. 加载 `WebsiteSSL` 记录；如果是 `FromMaster` 类型不允许被推回 Master；证书必须已 `SSLReady` — `:954-963`
4. 校验没有同 ID 的推送任务在跑（`task.CheckResourceTaskIsExecuting`）— `:964-966`
5. `SaveByMap` 写 push_node/nodes 字段；如果是 `Sync=true` 直接调 `xpack.MultiNodeProvider.PushSSLToNode` 同步推 — `:967-978`
6. 否则构造 `task.NewTask` 异步任务，加子任务 "开始推送证书到节点" → 调 `PushSSLToNode` → goroutine 里 `task.Execute()` — `:980-998`

**被调用函数**：
- `global.IsMaster` — 节点类型判断
- `xpack.MultiNodeProvider.IsXpack()` — XPack 启用检查
- `normalizeSSLPushConfig` — 配置标准化
- `websiteSSLRepo.GetFirst` / `SaveByMap` — GORM 操作
- `task.CheckResourceTaskIsExecuting` (`task/task.go`) — 任务去重
- `task.NewTask` / `task.GetTaskName` / `task.AddSubTask` / `pushTask.Execute` — 异步任务
- `xpack.MultiNodeProvider.PushSSLToNode` — XPack 实际推送逻辑

---

### 6.7 `func (w WebsiteSSLService) GetDNSResolve(req request.WebsiteDNSReq) ([]response.WebsiteDNSRes, error)`  —  `service/website_ssl.go:656`

**一句话作用**：**申请证书前的"前置体检"**——查每个待申请域名的 DNS 解析是否正确指向本机 IP（避免 CA 验证失败），结果返回每条记录的 `Domain/Key/Value/Err`。

**参数说明**：
- `req.AcmeAccountID` (uint) — ACME 账户 ID
- `req.WebsiteSSLID` (uint) — 证书记录 ID（用来取域名列表）

**执行流程**（3 步 + 行号）：
1. 加载 `AcmeAccount` 和 `WebsiteSSL`，构造 `ssl.NewCustomAcmeClient` 拿一个轻量客户端（不发起注册）— `:657-667`
2. 调 `client.GetDNSResolve(ctx, websiteSSL)` 内部会遍历每个域名，DNS 查询 A/AAAA 记录对比本机 IP — `:668-672`
3. 把 `map[domain]resolve` 转成 `[]WebsiteDNSRes{Domain/Key/Value/Err}` 返回 — `:673-682`

**被调用函数**：
- `websiteAcmeRepo.GetFirst` / `websiteSSLRepo.GetFirst` — GORM
- `ssl.NewCustomAcmeClient` (`utils/ssl/manual_client.go`) — 手动客户端
- `client.GetDNSResolve` (`utils/ssl/client.go`) — 实际 DNS 查询

---

### 6.8 `func NewAcmeClientWithContext(ctx context.Context, acmeAccount *model.WebsiteAcmeAccount, systemProxy *dto.SystemProxy) (*AcmeClient, error)`  —  `utils/ssl/acme.go:142`

**一句话作用**：**ACME 账户注册器**——如果账户里没存 `URL`（CA 注册后的 endpoint），就先调 CA 注册接口拿一个新账户，存下私钥和 URL；否则直接用已有账户构造 lego `AcmeClient`。

**参数说明**：
- `ctx` (context.Context) — 上下文（用于 EAB 获取的 HTTP 请求）
- `acmeAccount` (*model.WebsiteAcmeAccount) — ACME 账户模型（要填好 Email/Type/KeyType/PrivateKey/URL）
- `systemProxy` (*dto.SystemProxy) — 系统代理配置（HTTP_PROXY/HTTPS_PROXY）

**执行流程**（4 步 + 行号）：
1. 调 `newAcmeClient` 构造底层 lego 客户端（解析或生成私钥，构造 `AcmeUser` + `lego.NewClient`）— `:142-146` + `acme.go:194-238`
2. 如果 account.Type 是 `zerossl` / `google` / `freessl` / `custom` 用了 EAB：
   - `zerossl` 先调 ZeroSSL 的 EAB API 拿 `eab_kid` + `eab_hmac_key` — `:148-162`
   - 调 `client.Client.Registration.RegisterWithExternalAccountBinding` 用 EAB 注册 — `:163-172`
3. 否则普通 `client.Client.Registration.Register` 走标准注册 — `:173-178`
4. 把 `client.User.Registration = reg` 填上；把 CA 返回的 `reg.Location` 存到 `acmeAccount.URL`；如果私钥为空就 `GetPrivateKey` 导出 PEM；返回 — `:179-192`

**被调用函数**：
- `newAcmeClient` (`utils/ssl/acme.go:194`) — 构造 lego Client
- `getZeroSSLEabCredentials` (`utils/ssl/acme.go:338`) — ZeroSSL 专用 EAB 获取
- `parsePrivateKeyPEM` (`utils/ssl/acme.go:39`) — 解析私钥
- `certcrypto.GeneratePrivateKey` — 生成新私钥
- `lego.NewClient` + `lego.NewConfig` — lego 库
- `GetPrivateKey` (`utils/ssl/acme.go:108`) — 私钥 → PEM

---

### 6.9 `func newAcmeClient(acmeAccount *model.WebsiteAcmeAccount, proxy *dto.SystemProxy, registration *legoacme.ExtendedAccount) (*AcmeClient, error)`  —  `utils/ssl/acme.go:194`

**一句话作用**：**lego 客户端的真正构造器**——加载/生成 ACME 账户私钥 → 包装成 `AcmeUser` → 调 `lego.NewConfig` + `lego.NewClient` 拿一个能直接调用的 lego 客户端。

**参数说明**：
- `acmeAccount` — 账户模型（用 KeyType/PrivateKey/CaDirURL/Type）
- `proxy` — 系统代理
- `registration` — 已有注册信息（可为 nil）

**执行流程**（3 步 + 行号）：
1. 调 `normalizeKeyType` 标准化 KeyType（v4 旧值 `P256`/`2048` → v5 新值 `EC256`/`RSA2048`）；如果 `acmeAccount.PrivateKey` 非空就 `parsePrivateKeyPEM`，否则 `certcrypto.GeneratePrivateKey` 新生成 — `:200-218`
2. 构造 `AcmeUser{Email, Registration, Key}` + `NewConfigWithProxy(myUser, accountType, CaDirURL, proxy)` 设 CA Directory URL + UserAgent="1Panel" + 带代理的 HTTPClient + 60s 超时 — `:220-229` + `acme.go:259-280`
3. 包装成 `AcmeClient{User, Client, Config}` 返回 — `:230-237`

**被调用函数**：
- `normalizeKeyType` (`utils/ssl/acme.go:80`)
- `parsePrivateKeyPEM` (`utils/ssl/acme.go:39`)
- `certcrypto.GeneratePrivateKey` (lego 库)
- `NewConfigWithProxy` (`utils/ssl/acme.go:259`) — 构造 lego.Config
- `getCaDirURL` (`utils/ssl/acme.go:240`) — CA 目录映射
- `createHTTPClientWithProxy` (`utils/ssl/acme.go:300`) — 带代理的 HTTP 客户端
- `lego.NewClient` (lego 库)

---

## 7. 类/结构体讲解

### 7.1 `type WebsiteSSL struct`  —  `model/website_ssl.go:11`

**一句话作用**：一张 SSL 证书的"身份证 + 健康证 + 部署位置"——记录从 CA 申请到的私钥 + 证书链、过期时间、申请方式、绑定的网站等所有信息。

**字段说明**（每字段 1 句通俗解释）：
- `BaseModel` — 继承自 `BaseModel`，提供 `id` / `created_at` / `updated_at` 三个公共字段。
- `PrimaryDomain string` — 主域名（如 `example.com`），证书的 CN（Common Name）。
- `PrivateKey string` — PEM 格式的私钥（用 `-----BEGIN RSA PRIVATE KEY-----` 或 `-----BEGIN EC PRIVATE KEY-----` 包起来）。
- `Pem string` — 完整证书链 fullchain = 站点证书 + 中间证书（浏览器要靠中间证书才能验证根 CA）。
- `Domains string` — SAN（Subject Alternative Name）多域名，逗号分隔（如 `www.example.com,api.example.com`）。
- `CertURL string` — CA 返回的"证书吊销 URL"，吊销时调这个 URL 通知 CA。
- `Type string` — 颁发者 CN，如 `R3`（Let's Encrypt 的中间证书 CN）。
- `Provider string` — 申请方式枚举：`dnsAccount`（DNS 账户方式）/ `http`（HTTP-01）/ `dnsManual`（手动 DNS）/ `selfSigned` / `manual`（上传）/ `fromMaster`（从主节点导入）。
- `Organization string` — 颁发者组织，如 `Let's Encrypt` / `ZeroSSL`。
- `DnsAccountID uint` — DNS 账户外键到 `website_dns_accounts.id`（DNS-01 方式用）。
- `AcmeAccountID uint` — ACME 账户外键到 `website_acme_accounts.id`。
- `CaID uint` — 自签名 CA ID（仅 selfSigned 类型使用）。
- `AutoRenew bool` — 是否自动续期（cron 每天会扫这个字段）。
- `ExpireDate time.Time` — 证书过期时间（从证书 X.509 `NotAfter` 字段解析）。
- `StartDate time.Time` — 证书生效时间（从 X.509 `NotBefore` 字段解析）。
- `Status string` — 状态枚举：`init`（刚创建）/ `ready`（签发完成）/ `applying`（申请中）/ `applyError`（申请失败）/ `error`（其他错误）/ `systemRestart`（系统重启中断）。
- `Message string` — 申请失败时的详细错误信息（用户可看日志）。
- `KeyType string` — 私钥算法：`EC256` / `EC384` / `RSA2048` / `RSA3072` / `RSA4096`。
- `PushDir bool` + `Dir string` — 是否额外把证书推到一个目录（如 `/etc/nginx/ssl/`），便于集成其他服务。
- `Description string` — 用户备注。
- `SkipDNS bool` — DNS-01 验证时是否跳过 propagation check（加速但增加失败风险）。
- `Nameserver1 string` + `Nameserver2 string` — 自定义 DNS 服务器（用于查询，不影响权威 DNS）。
- `DisableCNAME bool` — 禁用 CNAME 跟踪（DNS-01 严格模式下用）。
- `ExecShell bool` + `Shell string` — 申请后是否执行用户脚本（用于触发外部重启）。
- `MasterSSLID uint` — 如果是 `fromMaster` 类型，记录主节点上的证书 ID。
- `Nodes string` + `PushNode bool` — 多节点推送的子节点 ID 列表。
- `PrivateKeyPath string` + `CertPath string` — local 上传模式下的文件路径。
- `IsIp bool` — 是否是 IP 证书（IP cert 14 天短周期，续期阈值 72 小时而不是 30 天）。
- `AcmeAccount WebsiteAcmeAccount` (gorm:`-:migration`) — 关联对象（API 响应里包含，方便前端展示邮箱）。
- `DnsAccount WebsiteDnsAccount` (gorm:`-:migration`) — 关联对象。
- `Websites []Website` (gorm:`-:migration`) — 反向关联（哪些网站在用这张证书）。

**方法**：
- `func (w WebsiteSSL) TableName() string` (`model/website_ssl.go:51`) — 返回 `"website_ssls"`，GORM 用这个名建表。
- `func (w WebsiteSSL) GetLogPath() string` (`model/website_ssl.go:55`) — 返回日志文件路径 `global.Dir.SSLLogDir/<primary_domain>-ssl-<id>.log`。

---

### 7.2 `type WebsiteAcmeAccount struct`  —  `model/website_acme_account.go:3`

**一句话作用**：CA 颁发的"ACME 协议身份"——邮箱 + 私钥 + URL 就是你在 CA（Let's Encrypt / ZeroSSL）注册的身份；申请证书时拿这个身份去和 CA 通信。

**字段说明**：
- `BaseModel` — 公共字段。
- `Email string` (not null) — 注册邮箱（CA 通知用，Let's Encrypt 必填）。
- `URL string` (not null) — CA 返回的账户 endpoint（`https://acme-v02.api.letsencrypt.org/acme/acct/12345`），后续所有 ACME 请求都带这个 URL 标识身份。
- `PrivateKey string` (not null) — 账户私钥 PEM（ACME 协议用 JWS 签名每个请求）。
- `Type string` (default: `letsencrypt`) — CA 类型：`letsencrypt` / `zerossl` / `buypass` / `google` / `freessl` / `custom`。
- `EabKid string` + `EabHmacKey string` — External Account Binding 凭据（ZeroSSL/Google/FreeSSL 必填，Let's Encrypt 不用）。
- `KeyType string` (default: `RSA2048`) — 账户私钥算法。
- `UseProxy bool` (default: false) — ACME 请求是否走系统代理。
- `CaDirURL string` — 自定义 CA 的 ACME Directory URL（仅 `custom` 类型使用）。
- `UseEAB bool` — 自定义 CA 是否使用 EAB。

**方法**：
- `func (w WebsiteAcmeAccount) TableName() string` (`model/website_acme_account.go:17`) — 返回 `"website_acme_accounts"`。

---

### 7.3 `type AcmeClient struct`  —  `utils/ssl/client.go:42`

**一句话作用**：**lego 库的 Go 包装**——把 lego 官方的 `lego.Client` + `AcmeUser` + `Config` 装在一起，1Panel 自己加 DNS provider / HTTP challenge 等扩展。

**字段说明**：
- `Config *lego.Config` — lego 客户端配置（CA URL、HTTP 客户端、UserAgent 等）。
- `Client *lego.Client` — lego 主客户端（提供 `Registration` / `Certificate` 等子模块）。
- `User *AcmeUser` — ACME 账户身份。
- `ProxyURL string` — 系统代理 URL（debug 用）。
- `dnsChallengeConfig *dnsChallengeConfig` — DNS-01 challenge 内部配置（`recursiveNameservers` / `disableCNAME`）。

**方法（部分）**：
- `func (c *AcmeClient) UseDns(dnsType DnsType, params string, websiteSSL model.WebsiteSSL) error` (`client.go:82`) — 绑定 DNS provider（阿里云/腾讯云/华为云等），设自定义 nameservers + SkipDNS/DisableCNAME 等 challenge option。
- `func (c *AcmeClient) UseHTTP(httpRoot string) error` (`client.go:~140`) — 绑定 HTTP-01 challenge（写 challenge 文件到 OpenResty 根目录的 `.well-known/acme-challenge/`）。
- `func (c *AcmeClient) ObtainSSL(ctx, domains, privateKey) (certificate.Resource, error)` (`client.go:~150`) — 域名证书申请（多域名走 SAN）。
- `func (c *AcmeClient) ObtainIPSSL(ctx, ip, privateKey) (certificate.Resource, error)` (`client.go:~160`) — IP 证书申请（短周期 14 天）。
- `func (c *AcmeClient) RevokeSSL(cert []byte) error` (`client.go:~170`) — 吊销证书（调 CA 的 revocation endpoint）。

---

### 7.4 `type AcmeUser struct`  —  `utils/ssl/acme.go:90`

**一句话作用**：实现 lego `registration.User` 接口——把 1Panel 的 `WebsiteAcmeAccount` 适配给 lego 用。

**字段说明**：
- `Email string` — 账户邮箱。
- `Registration *legoacme.ExtendedAccount` — CA 注册信息（含 URL/状态）。
- `Key crypto.Signer` — 账户私钥（lego v5 把字段类型从 `crypto.PrivateKey` 改成了 `crypto.Signer`）。

**方法**：
- `func (u *AcmeUser) GetEmail() string` (`acme.go:96`) — lego 接口方法。
- `func (u *AcmeUser) GetRegistration() *legoacme.ExtendedAccount` (`acme.go:100`) — lego 接口方法。
- `func (u *AcmeUser) GetPrivateKey() crypto.Signer` (`acme.go:104`) — lego 接口方法（v5 类型）。

---

### 7.5 `type WebsiteSSLDTO struct`  —  `dto/response/website_ssl.go:5`

**一句话作用**：API 响应体——在 `WebsiteSSL` 基础上加 `LogPath` 字段告诉前端证书的日志文件路径。

**字段说明**：
- `model.WebsiteSSL` — 嵌入基础模型。
- `LogPath string` — 日志文件路径（`/opt/1panel/1panel/log/ssl/<domain>-ssl-<id>.log`）。

---

### 7.6 `type WebsiteSSLCreate struct`  —  `dto/request/website_ssl.go:17`

**一句话作用**：创建证书的 API 请求体——前端表单提交的内容。

**关键字段**：
- `PrimaryDomain string` (validate: required) — 主域名（必填）。
- `OtherDomains string` — 多域名（每行一个）。
- `Provider string` (validate: required) — 申请方式。
- `AcmeAccountID uint` (validate: required) — ACME 账户 ID。
- `DnsAccountID uint` — DNS 账户 ID（仅 dnsAccount 方式必填）。
- `AutoRenew bool` — 自动续期。
- `KeyType string` — 私钥类型。
- `Apply bool` — 是否立即申请。
- `PushDir bool` + `Dir string` — 推送目录。
- `ID uint` — 仅供 Update 时使用（区分新建/更新）。
- `DisableCNAME bool` + `SkipDNS bool` — DNS-01 高级选项。
- `Nameserver1` + `Nameserver2` — 自定义 DNS。
- `ExecShell bool` + `Shell string` — 申请后脚本。
- `PushNode bool` + `Nodes string` — 多节点推送。
- `IsIp bool` — 是否是 IP 证书。

---

### 7.7 `type WebsiteSSLService struct`  —  `service/website_ssl.go:40`

**一句话作用**：SSL 业务的统一入口——所有 handler 调 `websiteSSLService.XXX()` 都走这个空 struct 的方法（Go 里"namespace"模式）。

**字段**：无（空 struct）。但有 4 个 package-level 变量：
- `websiteSSLRepo` — 注入的 repo 实例
- `websiteAcmeRepo` — ACME 账户 repo
- `websiteDnsRepo` — DNS 账户 repo
- `websiteRepo` — Website repo（关联用）

---

## 8. Mermaid 时序图

### 8.1 ACME 申请证书主流程（HTTP-01 方式）

```mermaid
sequenceDiagram
    autonumber
    participant U as User<br/>(浏览器/前端)
    participant H as Handler<br/>api/v2/website_ssl.go
    participant S as Service<br/>service/website_ssl.go
    participant AR as ACME Repo<br/>repo/website_acme_account.go
    participant SR as SSL Repo<br/>repo/website_ssl.go
    participant L as lego AcmeClient<br/>utils/ssl/client.go
    participant CA as Let's Encrypt<br/>acme-v02.api.letsencrypt.org
    participant OR as OpenResty<br/>(本机)
    participant FS as 本地文件

    U->>H: POST /websites/ssl<br/>{primaryDomain, provider=http, acmeAccountID}
    H->>S: Create(createReq)
    S->>AR: GetFirst(acmeAccountID)
    AR-->>S: WebsiteAcmeAccount
    S->>SR: Create(websiteSSL{Status:init})
    SR-->>S: created
    Note over S: 启动 goroutine 调 obtainSSL

    S->>SR: GetFirst(websiteSSL.ID)
    SR-->>S: WebsiteSSL
    S->>SR: TryMarkApplying(id) [CAS]
    SR-->>S: marked=true
    S->>S: newWebsiteSSLLogger() 创建日志文件
    S->>L: NewRegisterClient(acmeAccount, proxy)
    L->>CA: POST /acme/new-acct {termsOfServiceAgreed}
    CA-->>L: {account URL, status: valid}
    L-->>S: AcmeClient ready
    S->>L: UseHTTP("/opt/1panel/1panel/app/openresty/root")
    L->>OR: 准备 .well-known/acme-challenge/ 目录

    S->>L: ObtainSSL(ctx, [example.com], privateKey)
    L->>CA: POST /acme/new-order {identifiers: [example.com]}
    CA-->>L: {order URL, authorizations URL}
    L->>CA: GET /acme/authz/<auth-id>
    CA-->>L: {http-01 challenge token, keyAuthorization}
    L->>FS: 写 .well-known/acme-challenge/<token>
    FS-->>L: ok
    L->>CA: POST /acme/chall/<chal-id> {keyAuthorization}
    Note over L,CA: CA 发起 HTTP GET http://example.com/.well-known/acme-challenge/<token>
    OR-->>CA: 返回 keyAuthorization
    CA-->>L: {status: valid}
    L->>CA: POST /acme/order/<order-id>/finalize {csr}
    CA-->>L: {certificate URL}
    L->>CA: GET /acme/cert/<cert-id>
    CA-->>L: fullchain.pem (PEM)
    L-->>S: certificate.Resource{PrivateKey, Certificate}

    S->>S: parseCertificatePEM() 解析 ExpireDate/Type
    S->>SR: Save(websiteSSL{Status:ready})
    S->>OR: 写证书到 conf.d/ssl/example.com/
    S->>OR: nginx -s reload
    S->>U: 200 OK (Create 返回 ID)
```

### 8.2 DNS-01 申请流程（用 DNS 账户方式）

```mermaid
sequenceDiagram
    autonumber
    participant U as User
    participant S as Service.obtainSSL
    participant L as lego AcmeClient
    participant DP as DNS Provider<br/>(阿里云/腾讯云/...)
    participant CA as Let's Encrypt
    participant DNS as 公共 DNS

    U->>S: Create(provider=dnsAccount)
    S->>L: UseDns(dnsType="Aliyun", auth, websiteSSL)
    L->>DP: getDNSProviderConfig() 构造 Provider
    Note over L,DP: 设 custom nameservers + challenge options
    L->>CA: new-order
    CA-->>L: dns-01 challenge token
    L->>DP: Add TXT record<br/>_acme-challenge.example.com
    DP-->>DNS: 同步 TXT 记录
    Note over L,DNS: 等待 DNS propagation (默认轮询)
    L->>DNS: 查询 _acme-challenge.example.com TXT
    DNS-->>L: 返回 challenge 值
    L->>CA: POST challenge {keyAuthorization}
    CA-->>L: status: valid
    L->>CA: finalize(csr)
    CA-->>L: certificate
    L->>DP: CleanUp 删 TXT 记录
```

### 8.3 证书自动续期时序

```mermaid
sequenceDiagram
    autonumber
    participant CT as Cron Trigger<br/>(每天 00:00)
    participant J as cron/job/ssl.go: Run
    participant S as service.SyncSystemSSL
    participant SR as SSL Repo
    participant SS as WebsiteSSLService

    CT->>J: Run()
    J->>SR: List() 拿所有证书
    J->>S: SyncSystemSSL() 同步面板证书
    Note over S: 比对 DB 和 disk 证书<br/>不一致则重写 server.crt + reload
    loop 每个 SSL
        J->>J: 判断是否续期
        Note over J: AutoRenew=true<br/>Provider != manual/dnsManual<br/>Status != applying
        alt 是 IP 证书
            J->>J: expireDate - now < 72h ?
        else 域名证书
            J->>J: expireDate - now < 720h (30d) ?
        end
        alt 需要续期
            J->>SS: AutoRenewSSL(id)
            SS->>SS: obtainSSL(id, autoRenew=true)
            SS->>SS: 用 O_APPEND 打开日志 + 加 banner
            SS->>SS: 走完整 ACME 流程（同 8.1）
            SS->>SS: 成功后 reloadSystemSSL + pushSSLToNode
        end
    end
    J->>CT: 完成
```

---

## 9. Let's Encrypt / ACME 协议说明

### 9.1 什么是 ACME

**ACME**（Automatic Certificate Management Environment）是 IETF 标准协议（RFC 8555），由 Let's Encrypt 主导设计，**让自动化申请 SSL 证书成为可能**。以前你得手动登录 CA 网站填表、验证域名归属、等几天，ACME 把这套流程自动化成 6 个 HTTP API 调用，2 分钟搞定。

1Panel 用了 Go 语言最流行的 ACME 客户端库 **lego v5**（`github.com/go-acme/lego/v5`）。

### 9.2 两种核心验证方式（Challenge）

| 维度 | HTTP-01 | DNS-01 |
|---|---|---|
| **原理** | CA 发起 `GET http://你的域名/.well-known/acme-challenge/<token>`，你的服务器返回 `keyAuthorization` | CA 查询 `TXT _acme-challenge.example.com` 记录，你预先在 DNS 服务商 API 加了这条记录 |
| **优点** | 简单、lego 自动写到 OpenResty root | 支持通配符 `*.example.com` |
| **缺点** | 不支持通配符；要求 80 端口对外可达 | 需要 DNS 厂商支持 API（阿里云/腾讯云/华为云/Cloudflare/GoDaddy 等 10+ 家） |
| **适用场景** | 单域名 / 多域名（不用通配） | 多子域名批量 / 通配证书 |
| **1Panel 入口** | `provider=http` | `provider=dnsAccount` |
| **第三方入口** | — | `provider=dnsManual`（手动到 DNS 后台加记录） |

### 9.3 1Panel 支持的 CA

| Type | CA Directory URL | EAB 必填 | 备注 |
|---|---|---|---|
| `letsencrypt` | `https://acme-v02.api.letsencrypt.org/directory` | 否 | 全球最流行，证书 90 天 |
| `zerossl` | `https://acme.zerossl.com/v2/DV90` | 是 | 90 天免费，1Panel 自动调 API 拿 EAB |
| `buypass` | `https://api.buypass.com/acme/directory` | 否 | 180 天免费 |
| `google` | `https://dv.acme-v02.api.pki.goog/directory` | 是 | Google Trust Services，90 天 |
| `freessl` | `https://acmepro.freessl.cn/v2/DV` | 是 | 国内免费，需注册 |
| `custom` | 自定义 `CaDirURL` | 可选 | 接自己的 ACME CA |

EAB（External Account Binding）= 预先在 CA 申请"应用身份"，让 ACME 知道你代表哪个用户。ZeroSSL/Google 必须用 EAB。

### 9.4 续期机制

Let's Encrypt 颁发的证书有效期 **90 天**。1Panel 通过 `cron/job/ssl.go` 每天扫一次，**域名证书 < 30 天**（720h）或 **IP 证书 < 72 小时**（IP cert 14 天有效期，续期更频繁）就触发 `AutoRenewSSL(id)`。

```mermaid
flowchart LR
    A[证书签发<br/>ExpireDate = now+90d] --> B{每天 cron 扫}
    B -->|ExpireDate - now > 30d| C[跳过]
    B -->|ExpireDate - now <= 30d| D[AutoRenewSSL]
    D --> E[走 ACME 重新签发]
    E --> F[ExpireDate = now+90d]
    F --> B
```

### 9.5 ACME 协议流程（高层）

```
1. Register   客户端 → CA:  "我邮箱是 X，这是我公钥"  → CA 返回 account URL
2. New-Order  客户端 → CA:  "我想给 example.com 签证书"  → CA 返回 order URL
3. Get-Authz  客户端 → CA:  "看 example.com 的授权"     → CA 返回 challenge token
4. Satisfy    客户端本地操作：HTTP-01 写文件 / DNS-01 加 TXT 记录
5. Validate   CA 主动验证（HTTP GET 或 DNS 查询）
6. Finalize   客户端 → CA:  "传 CSR（Certificate Signing Request）" → CA 返回证书 URL
7. Download   客户端 → CA:  GET 证书 URL → 拿 fullchain.pem + privkey.pem
```

---

## 10. 自动续期 Cron 任务详解

### 10.1 文件位置

`agent/cron/job/ssl.go`（62 行，简洁）

### 10.2 触发时机

由 `agent/init/runtime/loader.go` 在节点启动时注册到定时调度器，**每天 00:00** 触发一次。

### 10.3 执行流程（Run 函数详解）

```go
// cron/job/ssl.go:21
func (ssl *ssl) Run() {
    sslRepo := repo.NewISSLRepo()        // 拿 repo
    sslService := service.NewIWebsiteSSLService()  // 拿 service
    sslList, _ := sslRepo.List()         // 拿所有证书
    nyc, _ := time.LoadLocation(common.LoadTimeZoneByCmd())  // 拿系统时区

    // 步骤 1: 同步面板证书（recovery 机制）
    service.SyncSystemSSL()  // service/website_ssl.go:379

    // 步骤 2: 遍历判断续期
    now := time.Now().Add(10 * time.Second)  // 给 10s 缓冲
    for _, s := range sslList {
        // 过滤掉不需要续期的
        if !s.AutoRenew || s.Provider == "manual" || s.Provider == "dnsManual" || s.Status == "applying" {
            continue
        }
        expireDate := s.ExpireDate.In(nyc)  // 转用户时区
        sub := expireDate.Sub(now)          // 剩余时间

        // 步骤 3: 触发续期
        if s.IsIp && sub.Hours() < 72 || !s.IsIp && sub.Hours() < 720 {
            if s.Provider == constant.SelfSigned {
                // 自签名证书走 CA 模块
                caService := service.NewIWebsiteCAService()
                caService.ObtainSSL(request.WebsiteCAObtain{
                    ID:    s.CaID,
                    SSLID: s.ID,
                    Renew: true, Unit: "year", Time: 10,
                })
            } else {
                // ACME 证书
                sslService.AutoRenewSSL(s.ID)
            }
        }
    }
}
```

### 10.4 续期 vs 首次申请的区别

| 维度 | 首次申请（Create → obtainSSL） | 自动续期（cron → AutoRenewSSL） |
|---|---|---|
| 入口 | `CreateWebsiteSSL` handler | `cron/job/ssl.go: Run` |
| 触发方式 | 用户手动 | 每天定时 |
| 日志模式 | `O_TRUNC`（清空旧日志） | `O_APPEND`（追加）+ 加 banner `==== [time] auto renew attempt ====` |
| 并发控制 | `TryMarkApplying` (CAS) | `TryMarkApplying` (CAS) + cron 本身不会并发 |
| 错误处理 | 通过 handler 返回 500 | 仅 `global.LOG.Errorf` 写日志 |
| 续期触发后还做 | `reloadSystemSSL` + `pushSSLToNode` | 同（共用 `obtainSSL(id, true)`） |

### 10.5 SyncSystemSSL 机制（防丢证书）

`service/website_ssl.go:379` 的 `SyncSystemSSL` 是个**保险机制**：

- 问题：续期时如果 `nginx -s reload` 失败，DB 已经有新证书了但磁盘还是旧证书。
- 解决：cron 启动时调一次，比对 `data/secret/server.crt` 和 DB 里的 Pem：
  - 一致 → no-op
  - 不一致 → 重写磁盘 + 通知 core reload

这是个"对账"机制，确保任何时候面板自己的证书都是最新的。

---

## 11. 跟其他模块的关系

### 11.1 必须模块：Website（网站）

- `website_ssls.WebsiteSSLID` 字段（其实是 `websites` 表的 `website_ssl_id`）反向关联。
- 申请证书成功后，`createPemFile(website, ssl)` (`service/website_ssl.go:595`) 把证书写到 `website.WebsiteSSLDir`。
- 删除证书时 `websiteRepo.GetBy(WithWebsiteSSLID(id))` 校验有没有网站在用（service/website_ssl.go:705）。

### 11.2 集成：Nginx / OpenResty

- HTTP-01 challenge 时写文件到 `appInstall.GetPath()/root/.well-known/acme-challenge/`。
- 证书签发后写 `appInstall.GetPath()/conf.d/ssl/<domain>/fullchain.pem + privkey.pem`。
- `opNginx(containerName, NginxReload)` 触发 `docker exec <openresty-container> nginx -s reload`。

### 11.3 集成：App 应用（AppOpenresty）

- `getAppInstallByKey(constant.AppOpenresty)` 拿 OpenResty 安装记录。
- OpenResty 必须是已安装的应用，否则 HTTP-01 challenge 无法工作。

### 11.4 可选模块：DNS Provider（10+ 家）

`utils/ssl/dns_provider.go` 实现 `getDNSProviderConfig(dnsType, params, httpClient)`，根据 dnsType 选择对应 DNS 厂商：
- 阿里云 / 腾讯云 / 华为云 / 百度云 / 京东云 / GoDaddy / NameSilo / Namecheap / Cloudflare / Dnspod
- 完整列表见 `agent/utils/ssl/dns_provider.go`

### 11.5 可选模块：Xpack 多节点

- `xpack.MultiNodeProvider.IsXpack()` 判断是否启用多节点。
- `xpack.MultiNodeProvider.PushSSLToNode(ssl)` 实际推送逻辑（不开源）。
- `Middleware: certificate.go` 在 Worker 节点校验证书 + Proxy-Id。

### 11.6 可选模块：CA（自签名）

- `provider=selfSigned` 走 `agent/app/service/website_ca.go`（不在本模块）。
- cron 续期时如果 `Provider==selfSigned` 调 `caService.ObtainSSL(WebsiteCAObtain{Renew:true})`。

### 11.7 共用模块：Task 任务系统

- `PushToNode` 异步模式用 `task.NewTask` + `AddSubTask` 包装。
- `task.CheckResourceTaskIsExecuting` 防止同一资源并发推。

### 11.8 共用模块：Setting 系统设置

- `settingRepo.Get(WithByKey("SSL"))` 读 `SSL=enable/disable`。
- `settingRepo.Get(WithByKey("SSLID"))` 读面板证书 ID。
- 删除证书时校验是否是面板证书（service/website_ssl.go:712-718）。

---

## 12. 行号索引（func → file:line）

### 12.1 Service 层

| 函数 | 文件:行号 | 一句话作用 |
|---|---|---|
| `func (w WebsiteSSLService) Page` | `service/website_ssl.go:119` | 分页查询证书列表 |
| `func (w WebsiteSSLService) GetSSL` | `service/website_ssl.go:145` | 按 ID 查证书 |
| `func (w WebsiteSSLService) Search` | `service/website_ssl.go:155` | 按 ACME 账户 ID 查证书列表 |
| `func (w WebsiteSSLService) Create` | `service/website_ssl.go:180` | 创建证书 + 启动 ACME 申请 |
| `func (w WebsiteSSLService) ObtainSSL` | `service/website_ssl.go:419` | 手动触发 ACME 申请（API 入口） |
| `func (w WebsiteSSLService) AutoRenewSSL` | `service/website_ssl.go:423` | 触发自动续期（cron 入口） |
| `func (w WebsiteSSLService) obtainSSL` | `service/website_ssl.go:427` | 申请核心引擎（私有） |
| `func (w WebsiteSSLService) GetDNSResolve` | `service/website_ssl.go:656` | DNS 解析预检 |
| `func (w WebsiteSSLService) GetWebsiteSSL` | `service/website_ssl.go:685` | 按 websiteId 查关联证书 |
| `func (w WebsiteSSLService) Delete` | `service/website_ssl.go:699` | 批量删除证书 + 吊销 |
| `func (w WebsiteSSLService) Update` | `service/website_ssl.go:759` | 修改证书配置 |
| `func (w WebsiteSSLService) Upload` | `service/website_ssl.go:830` | 上传/导入证书 |
| `func (w WebsiteSSLService) PushToNode` | `service/website_ssl.go:943` | 推证书到子节点（XPack） |
| `func (w WebsiteSSLService) DownloadFile` | `service/website_ssl.go:1001` | 打包下载证书 zip |
| `func (w WebsiteSSLService) SyncForRestart` | `service/website_ssl.go:1029` | 系统启动时把 applying 状态改成 systemRestart |
| `func (w WebsiteSSLService) ImportMasterSSL` | `service/website_ssl.go:1044` | 从主节点导入证书 |
| `func withLegoLogger` | `service/website_ssl.go:47` | 临时切 lego logger |
| `func withLegoLoggerTimeout` | `service/website_ssl.go:62` | 切 logger + 加 ctx 超时 |
| `func newWebsiteSSLLegoClient` | `service/website_ssl.go:70` | 拿/注册 lego client |
| `func printSSLLog` | `service/website_ssl.go:276` | 写国际化日志 |
| `func normalizeSSLPushConfig` | `service/website_ssl.go:283` | 标准化推送配置 |
| `func setSSLPushConfig` | `service/website_ssl.go:291` | 设 SSL 推送字段（Master + XPack 校验） |
| `func pushSSLToNode` | `service/website_ssl.go:301` | 推证书到节点 |
| `func pushSSLToNodeWithNewLogger` | `service/website_ssl.go:311` | 同上 + 新 logger |
| `func newWebsiteSSLLogger` | `service/website_ssl.go:324` | 创建日志文件 + logger |
| `func reloadSystemSSL` | `service/website_ssl.go:346` | 同步面板证书到磁盘 |
| `func SyncSystemSSL` | `service/website_ssl.go:379` | 启动时面板证书对账 |
| `func runShellScriptFile` | `service/website_ssl.go:618` | 执行用户脚本 |
| `func parseCertificatePEM` | `service/website_ssl.go:635` | 解析证书 |
| `func handleError` | `service/website_ssl.go:643` | 统一错误处理 |

### 12.2 ACME 账户 Service

| 函数 | 文件:行号 | 一句话作用 |
|---|---|---|
| `func (w WebsiteAcmeAccountService) Page` | `service/website_acme_account.go:27` | 分页查询 ACME 账户 |
| `func (w WebsiteAcmeAccountService) Create` | `service/website_acme_account.go:38` | 创建 ACME 账户（注册到 CA） |
| `func (w WebsiteAcmeAccountService) Delete` | `service/website_acme_account.go:83` | 删除 ACME 账户 |
| `func (w WebsiteAcmeAccountService) Update` | `service/website_acme_account.go:90` | 修改 ACME 账户（仅 UseProxy） |

### 12.3 Repo 层

| 函数 | 文件:行号 | 一句话作用 |
|---|---|---|
| `func (w WebsiteSSLRepo) WithByAlias` | `repo/website_ssl.go:36` | 按别名查 |
| `func (w WebsiteSSLRepo) WithByAcmeAccountId` | `repo/website_ssl.go:42` | 按 ACME 账户 ID 查 |
| `func (w WebsiteSSLRepo) WithByDnsAccountId` | `repo/website_ssl.go:48` | 按 DNS 账户 ID 查 |
| `func (w WebsiteSSLRepo) WithByCAID` | `repo/website_ssl.go:54` | 按 CA ID 查 |
| `func (w WebsiteSSLRepo) WithByDomain` | `repo/website_ssl.go:60` | 按主域名/SAN 模糊查 |
| `func (w WebsiteSSLRepo) WithByMasterSSLID` | `repo/website_ssl.go:66` | 按主节点证书 ID 查 |
| `func (w WebsiteSSLRepo) Page` | `repo/website_ssl.go:72` | 分页 + Preload AcmeAccount/DnsAccount/Websites |
| `func (w WebsiteSSLRepo) GetFirst` | `repo/website_ssl.go:81` | 查单条 + Preload |
| `func (w WebsiteSSLRepo) List` | `repo/website_ssl.go:90` | 列表 + Preload |
| `func (w WebsiteSSLRepo) TryMarkApplying` | `repo/website_ssl.go:99` | CAS 抢占 applying 状态 |
| `func (w WebsiteSSLRepo) Create` | `repo/website_ssl.go:112` | 创建 |
| `func (w WebsiteSSLRepo) Save` | `repo/website_ssl.go:116` | Save（更新所有字段） |
| `func (w WebsiteSSLRepo) SaveByMap` | `repo/website_ssl.go:120` | 按 map 更新 |
| `func (w WebsiteSSLRepo) DeleteBy` | `repo/website_ssl.go:126` | 按条件删 |
| `func (*WebsiteAcmeAccountRepo) WithEmail/WithType` | `repo/website_acme_account.go:25/30` | DBOption |
| `func (*WebsiteAcmeAccountRepo) Page/GetFirst/Create/Save/DeleteBy` | `repo/website_acme_account.go:36-63` | GORM 增删改查 |

### 12.4 Router 层

| 路由 | 文件:行号 | 一句话作用 |
|---|---|---|
| `WebsiteSSLRouter.InitRouter` | `router/ro_website_ssl.go:11` | 注册 13 个 SSL 路由 |
| `WebsiteAcmeAccountRouter.InitRouter` | `router/ro_website_acme_account.go:11` | 注册 4 个 ACME 账户路由 |

### 12.5 Handler 层（api/v2/website_ssl.go）

| Handler | 文件:行号 | 一句话作用 |
|---|---|---|
| `PageWebsiteSSL` | `api/v2/website_ssl.go:26` | 分页查询 |
| `ListWebsiteSSL` | `api/v2/website_ssl.go:50` | 列表 |
| `CreateWebsiteSSL` | `api/v2/website_ssl.go:72` | 创建证书 |
| `ApplyWebsiteSSL` | `api/v2/website_ssl.go:94` | 申请证书 |
| `GetDNSResolve` | `api/v2/website_ssl.go:114` | DNS 解析预检 |
| `DeleteWebsiteSSL` | `api/v2/website_ssl.go:136` | 删除证书 |
| `GetWebsiteSSLByWebsiteId` | `api/v2/website_ssl.go:156` | 按 website 查证书 |
| `GetWebsiteSSLById` | `api/v2/website_ssl.go:178` | 按 ID 查证书 |
| `UpdateWebsiteSSL` | `api/v2/website_ssl.go:201` | 更新证书 |
| `PushWebsiteSSLToNode` | `api/v2/website_ssl.go:222` | 推节点 |
| `UploadWebsiteSSL` | `api/v2/website_ssl.go:243` | 上传 PEM |
| `UploadSSLFile` | `api/v2/website_ssl.go:268` | 上传文件 |
| `DownloadWebsiteSSL` | `api/v2/website_ssl.go:340` | 下载证书 zip |
| `ImportMasterSSL` | `api/v2/website_ssl.go:370` | 从主节点导入 |
| `readUploadedFile` | `api/v2/website_ssl.go:321` | 读 multipart 文件 |
| `PageWebsiteAcmeAccount` | `api/v2/website_acme_account.go:18` | ACME 账户分页 |
| `CreateWebsiteAcmeAccount` | `api/v2/website_acme_account.go:43` | 创建 ACME 账户 |
| `DeleteWebsiteAcmeAccount` | `api/v2/website_acme_account.go:65` | 删除 |
| `UpdateWebsiteAcmeAccount` | `api/v2/website_acme_account.go:86` | 更新 |

### 12.6 工具层 (utils/ssl/)

| 函数 | 文件:行号 | 一句话作用 |
|---|---|---|
| `func parsePrivateKeyPEM` | `utils/ssl/acme.go:39` | 解析 PKCS#8/EC/RSA 私钥 |
| `func normalizeKeyType` | `utils/ssl/acme.go:80` | v4 KeyType 字符串转 v5 |
| `func (u *AcmeUser) GetEmail/GetRegistration/GetPrivateKey` | `utils/ssl/acme.go:96/100/104` | lego User 接口 |
| `func GetPrivateKey` | `utils/ssl/acme.go:108` | crypto.PrivateKey → PEM bytes |
| `func NewRegisterClient` | `utils/ssl/acme.go:138` | 注册 ACME 账户（无 context） |
| `func NewRegisterClientWithContext` | `utils/ssl/acme.go:142` | 注册 ACME 账户（带 ctx） |
| `func newAcmeClient` | `utils/ssl/acme.go:194` | 构造 lego Client |
| `func getCaDirURL` | `utils/ssl/acme.go:240` | CA 类型 → Directory URL |
| `func NewConfigWithProxy` | `utils/ssl/acme.go:259` | 构造 lego.Config |
| `func initCertPool` | `utils/ssl/acme.go:282` | 读 LEGO_CA_CERTIFICATES |
| `func createHTTPClientWithProxy` | `utils/ssl/acme.go:300` | 带代理的 HTTP 客户端 |
| `func getZeroSSLEabCredentials` | `utils/ssl/acme.go:338` | 调 ZeroSSL API 拿 EAB |
| `func GetPrivateKeyByType` | `utils/ssl/acme.go:378` | 拿/生成 SSL 私钥 |
| `func getWebsiteSSLDomains` | `utils/ssl/acme.go:394` | 合并主域名+SAN |
| `func isHTTP503Error` | `utils/ssl/acme.go:408` | 判断 lego 错误是否是 503 |
| `func NewAcmeClient` | `utils/ssl/client.go:56` | 已有账户 → AcmeClient |
| `func NewAcmeClientWithContext` | `utils/ssl/client.go:60` | 同上带 ctx |
| `func (c *AcmeClient) UseDns` | `utils/ssl/client.go:82` | 绑定 DNS provider |
| `func (c *AcmeClient) UseHTTP` | `utils/ssl/client.go:~140` | 绑定 HTTP challenge |
| `func (c *AcmeClient) ObtainSSL` | `utils/ssl/client.go:~150` | 域名证书申请 |
| `func (c *AcmeClient) ObtainIPSSL` | `utils/ssl/client.go:~160` | IP 证书申请 |
| `func (c *AcmeClient) RevokeSSL` | `utils/ssl/client.go:~170` | 吊销证书 |
| `func (c *AcmeClient) GetDNSResolve` | `utils/ssl/client.go:~180` | DNS 解析预检 |
| `func waitForRetry` | `utils/ssl/client.go:255` | 等 30s 重试 |
| `func getDNSProviderConfig` | `utils/ssl/dns_provider.go:103` | DNS 厂商 → challenge.Provider |

### 12.7 Cron / Middleware

| 函数 | 文件:行号 | 一句话作用 |
|---|---|---|
| `func NewSSLJob` | `cron/job/ssl.go:17` | 构造 cron job 实例 |
| `func (ssl *ssl) Run` | `cron/job/ssl.go:21` | 每天 00:00 跑 |
| `func Certificate` | `middleware/certificate.go:17` | 节点证书 + Proxy-Id 校验 |
| `func CloseDirectly` | `middleware/certificate.go:42` | 校验失败关闭 TCP 连接 |

### 12.8 Model / DTO

| 类型 | 文件:行号 | 一句话作用 |
|---|---|---|
| `type WebsiteSSL` | `model/website_ssl.go:11` | 证书 GORM 模型 |
| `func (w WebsiteSSL) TableName` | `model/website_ssl.go:51` | 表名 website_ssls |
| `func (w WebsiteSSL) GetLogPath` | `model/website_ssl.go:55` | 日志文件路径 |
| `type WebsiteAcmeAccount` | `model/website_acme_account.go:3` | ACME 账户 GORM 模型 |
| `type WebsiteSSLDTO` | `dto/response/website_ssl.go:5` | 响应体 + LogPath |
| `type WebsiteDNSRes` | `dto/response/website_ssl.go:10` | DNS 解析预检响应 |
| `type WebsiteAcmeAccountDTO` | `dto/response/website_ssl.go:17` | ACME 账户 DTO |
| `type WebsiteDnsAccountDTO` | `dto/response/website_ssl.go:21` | DNS 账户 DTO |
| `type WebsiteSSLSearch` | `dto/request/website_ssl.go:5` | 搜索条件 |
| `type WebsiteSSLListReq` | `dto/request/website_ssl.go:13` | 列表请求 |
| `type WebsiteSSLCreate` | `dto/request/website_ssl.go:17` | 创建请求 |
| `type WebsiteDNSReq` | `dto/request/website_ssl.go:41` | DNS 解析预检请求 |
| `type WebsiteSSLRenew` | `dto/request/website_ssl.go:46` | 续期请求 |
| `type WebsiteSSLApply` | `dto/request/website_ssl.go:50` | 申请请求 |
| `type WebsiteSSLUpdate` | `dto/request/website_ssl.go:98` | 更新请求 |
| `type WebsiteSSLUpload` | `dto/request/website_ssl.go:121` | 上传请求 |
| `type WebsiteSSLPush` | `dto/request/website_ssl.go:133` | 推送请求 |
| `type WebsiteSSLFileUpload` | `dto/request/website_ssl.go:178` | 文件上传请求 |
| `type WebsiteAcmeAccountCreate` | `dto/request/website_ssl.go:61` | ACME 账户创建 |
| `type WebsiteAcmeAccountUpdate` | `dto/request/website_ssl.go:72` | ACME 账户更新 |
| `type WebsiteDnsAccountCreate` | `dto/request/website_ssl.go:77` | DNS 账户创建 |
| `type WebsiteDnsAccountUpdate` | `dto/request/website_ssl.go:83` | DNS 账户更新 |
| `type WebsiteResourceReq` | `dto/request/website_ssl.go:90` | ID 资源请求 |
| `type WebsiteBatchDelReq` | `dto/request/website_ssl.go:94` | 批量删除请求 |

---

## 13. 关键文件清单（按重要性）

| 文件 | 重要性 | 一句话说明 |
|---|---|---|
| `agent/app/service/website_ssl.go` | ⭐⭐⭐⭐⭐ | SSL 业务主线（1090 行），含 obtainSSL 主流程 |
| `agent/utils/ssl/acme.go` | ⭐⭐⭐⭐⭐ | ACME 协议客户端工厂（423 行） |
| `agent/app/api/v2/website_ssl.go` | ⭐⭐⭐⭐ | 14 个 HTTP handler |
| `agent/utils/ssl/client.go` | ⭐⭐⭐⭐ | AcmeClient (UseDns/UseHTTP/ObtainSSL/RevokeSSL) |
| `agent/cron/job/ssl.go` | ⭐⭐⭐ | 每天 00:00 自动续期 |
| `agent/app/repo/website_ssl.go` | ⭐⭐⭐ | 证书 GORM 仓储 |
| `agent/app/model/website_ssl.go` | ⭐⭐⭐ | WebsiteSSL 表结构 |
| `agent/app/dto/request/website_ssl.go` | ⭐⭐ | 7 个请求体定义 |
| `agent/app/dto/response/website_ssl.go` | ⭐ | 3 个响应体 |
| `agent/app/service/website_acme_account.go` | ⭐ | ACME 账户业务 |
| `agent/app/repo/website_acme_account.go` | ⭐ | ACME 账户仓储 |
| `agent/app/model/website_acme_account.go` | ⭐ | ACME 账户表 |
| `agent/app/api/v2/website_acme_account.go` | ⭐ | 4 个 ACME 账户 handler |
| `agent/router/ro_website_ssl.go` | ⭐ | 13 个 SSL 路由 |
| `agent/router/ro_website_acme_account.go` | ⭐ | 4 个 ACME 账户路由 |
| `agent/middleware/certificate.go` | ⭐ | 节点证书校验（XPack 多节点） |
| `agent/utils/ssl/dns_provider.go` | ⭐ | 10+ DNS 厂商 provider |
| `agent/utils/ssl/manual_client.go` | ⭐ | 手动 DNS 申请客户端 |
| `agent/constant/website.go` | ref | 常量: SSLInit/SSLError/SSLReady/SSLApply 等 |

---

## 14. 总结：12-ssl 模块的生命周期

```mermaid
flowchart TD
    A[用户访问 1Panel 网站管理] --> B[创建证书]
    B --> C{Provider 类型}
    C -->|http| D[HTTP-01: OpenResty 写 .well-known]
    C -->|dnsAccount| E[DNS-01: DNS 厂商 API 加 TXT]
    C -->|dnsManual| F[手动: 用户去 DNS 后台加]
    C -->|selfSigned| G[走 CA 模块自签名]
    C -->|manual| H[直接上传 PEM]
    D --> I[CA 验证 → 签发]
    E --> I
    F --> J[用户点 '继续' 触发实际申请]
    J --> I
    G --> K[签发自签名证书]
    H --> L[解析证书入库]
    I --> M[写库 Status=ready]
    K --> M
    L --> M
    M --> N[写证书到 OpenResty]
    N --> O[nginx -s reload]
    O --> P{面板证书?}
    P -->|是| Q[替换 server.crt + 通知 core]
    P -->|否| R[完成]
    Q --> S{XPack 多节点?}
    S -->|是| T[pushSSLToNode]
    S -->|否| R
    T --> R
    R --> U[等待过期]
    U --> V{每天 00:00 cron}
    V --> W{过期 < 30d?}
    W -->|是| X[AutoRenewSSL]
    W -->|否| U
    X --> D
```

一个 SSL 证书在 1Panel 里的一生：用户创建 → 选 provider → ACME 验证 → 写库 → 重载 Nginx → 等过期 → cron 自动续期 → 循环往复。

---

> **文档版本**: v3 (2026-08-26)
> **总行数**: ~1,000 行
> **覆盖函数**: 80+（service 28 + repo 16 + utils 16 + handler 18 + cron/middleware 4 + model 5 + dto 14）
> **覆盖 Mermaid 图**: 6（HR.md 章节 8 + 9 + 10 + 14）
> **配套可视化**: `visual-atlas.html`（8+ 独立 Mermaid 图）

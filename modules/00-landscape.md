# 1Panel v2 — 模块全景图

> 1Panel (`dev-v2` @ `7915230`) 全部模块地图。**目的**：让你快速判断"哪些模块值得精读、哪些只需要知道存在"。
>
> 详细注解见同目录下 `01-*.md`、`02-*.md`... 等（按你挑的模块出）。

---

## 0. 总体架构

```
┌─────────────────────────────────────────────────────────────┐
│  Frontend  (Vue 3 SPA, 488 view files)                       │
│  /src/views/{ai, app-store, container, database, host, ...}│
└──────────────────┬───────────────────────────────────────────┘
                   │ HTTPS / JSON
┌──────────────────┴───────────────────────────────────────────┐
│  Core  (1panel-core, 1panel-core 主控端, Go)                  │
│  /core/cmd/server/main.go + /core/app + /core/router         │
│  作用：认证 / 路由转发 / 中心化设置 / 多 Agent 调度           │
└──────────────────┬───────────────────────────────────────────┘
                   │ 内部 RPC
       ┌───────────┴────────────┐
       │                        │
┌──────┴────────┐      ┌───────┴────────┐
│  Agent 1      │      │  Agent 2       │  ...
│  (被控端)     │      │  (被控端)     │
│  /agent/...   │      │  /agent/...   │
│  真实干活：   │      │               │
│  Docker/iptables/Shell/...     │
└───────────────┘      └───────────────┘
```

**Core 是薄代理，Agent 干实事。** 每个被管机器装一个 agent。

---

## 1. 业务模块清单（按 agent/service 的 .go 文件归类）

### 1.1 应用商店 App Store
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/app.go` | 976 | 已装应用 CRUD（list / install / uninstall / update） |
| `service/app_install.go` | 949 | 安装流程编排（参数化 + 模板化） |
| `service/app_upgrade.go` | 872 | 升级流程（含 ignore_upgrade 跳过逻辑） |
| `service/app_sync_task.go` | 530 | 定时同步任务（从 1Panel 官方仓库拉新版本） |
| `service/app_utils.go` | **2119** | 应用元数据 / 依赖关系 / helper |
| `api/v2/app.go` + `app_install.go` | 595 | HTTP 入口 |

> **看点**：1Panel 的"一键装 WordPress"全在这；`app_utils.go` 是 2119 行的工具大杂烩，估计是 hot path。

### 1.2 容器 Container
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/container.go` | **1961** | 容器 CRUD（list / start / stop / restart / kill） |
| `service/container_compose.go` | 720 | docker compose up/down 管理 |
| `service/container_update.go` | 657 | 容器重建 / image 拉取 |
| `service/container_network.go` | 165 | 自定义网络 |
| `service/container_volume.go` | 140 | volume 管理 |
| `service/docker.go` | 409 | Docker daemon 自身管控 |
| `api/v2/container.go` | 891 | HTTP 入口 |

> **看点**：`container.go` 接近 2000 行，跟 Docker Engine API 1:1 映射；1Panel 把它当 Docker UI 来卖。

### 1.3 网站 Website (Nginx)
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/website.go` | **2370** | 网站 CRUD（default / reverse proxy / static） |
| `service/website_utils.go` | 1783 | Nginx 配置生成 / 解析 |
| `service/nginx_module.go` | 1307 | Nginx 二进制操作（reload / test config） |
| `service/website_ssl.go` | 1024 | SSL 证书管理（自签 / Let's Encrypt / 上传） |
| `service/website_*.go` (×6) | 1400+ | 反向代理 / LB / 重写 / 操作日志 |
| `api/v2/website*.go` (×5) | 1900+ | HTTP 入口 |

> **看点**：Nginx 配置生成是核心，**不是直接调 nginx -s reload**，而是把整个 conf 文件渲染后写盘再 reload；这部分对 Sirius Cloud 网站管理有参考价值。

### 1.4 数据库 Database
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/database.go` | 364 | 统一数据库 CRUD 接口 |
| `service/database_mysql.go` | **1488** | MySQL 专属（增删改查 + 备份恢复 + 主从） |
| `service/database_postgresql.go` | 413 | PostgreSQL |
| `service/database_redis.go` | 272 | Redis |
| `service/database_mongodb.go` | 878 | MongoDB |
| `service/database_common.go` | 105 | 公共逻辑（连接池 / 凭据加密） |
| `api/v2/database_*.go` (×4) | 1140+ | HTTP 入口 |

> **看点**：MySQL 模块 1488 行，里面估计有 "**通过 SSH 隧道连远端 MySQL**" 这种典型运维场景的实现（你的真实测试环境 10.10.10.100 用得上）。

### 1.5 备份 Backup + 快照 Snapshot
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/backup.go` | 662 | 备份任务编排 |
| `service/backup_app.go` | 573 | 应用级备份 |
| `service/backup_container.go` | 955 | 容器级备份（commit + export） |
| `service/backup_compose.go` | 673 | compose 项目备份 |
| `service/backup_database*.go` (×5) | 1500+ | 各数据库备份 |
| `service/backup_website.go` | 350 | 网站文件备份 |
| `service/snapshot.go` + `snapshot_*.go` (×4) | 1700+ | 快照 / 回滚 |

> **看点**：每个对象类型（app / container / compose / db / website）都有独立 backup 文件——典型的"多态备份"模式；跟你 Sirius Cloud L2 备份需求几乎 1:1。

### 1.6 定时任务 Cronjob
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/cronjob.go` | 783 | cron 任务 CRUD |
| `service/cronjob_backup.go` | 628 | "自动备份"型任务 |
| `service/cronjob_helper.go` | 472 | cron 表达式解析 / 下次执行时间计算 |

> **看点**：1Panel 的 cron 是面向用户的（"每周日凌晨 3 点自动备份数据库"），不是面向系统的。可以学到"用户友好 cron"设计。

### 1.7 告警 Alert
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/alert.go` | 683 | 告警规则管理 |
| `service/alert_helper.go` | 1019 | 触发评估逻辑（CPU > 80% 持续 5 分钟） |
| `service/alert_sender.go` | 379 | 通知渠道（钉钉 / 飞书 / 企微 / Webhook / 邮件） |

> **看点**：告警 + 通知是多渠道的，**不是单一发送器**。每个渠道一个 sender。

### 1.8 文件管理 File
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/file.go` | **1587** | 文件 CRUD（list / upload / download / edit） |
| `service/file_history.go` | 688 | 文件操作历史（恢复误删） |
| `service/file_share.go` | 350 | 分享链接 |
| `service/file_transfer.go` | 66 | 大文件分片传输 |

> **看点**：1587 行，**对位你 L1/L2 资产管理需求**。

### 1.9 AI Agent (v2 新增)
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/agents.go` | **1635** | AI Agent 编排（核心入口） |
| `service/agents_utils.go` | **1518** | Agent runtime / context 管理 |
| `service/agents_channels.go` | 1580 | 通信渠道（IM / API / CLI） |
| `service/agents_skills.go` | 863 | Skills 插件系统 |
| `service/agents_website.go` | 305 | 通过 Web UI 操作 Agent |
| `service/agents_hermes*.go` (×5) | 1700+ | 集成的 Hermes Agent 后端 |
| `service/mcp_server.go` | 1006 | MCP 协议 server（给 LLM 调用工具） |
| `service/ai.go` | 496 | AI 工具调用 / 上下文拼装 |
| `api/v2/agents.go` | **1427** | HTTP 入口 |
| `api/v2/mcp_server.go` | 217 | MCP HTTP 入口 |

> **看点**：**v2 的核心新特性** —— "lightweight AI management platform"。`agents.go` + `agents_utils.go` 加起来 3153 行，估计算 1Panel v2 50% 的代码量。可以学到"AI Agent as a service" 完整设计。
>
> ⚠️ `agents_hermes*.go` 是集成的 [Hermes](https://github.com/) —— 1Panel 跟 Hermes 项目的合作。

### 1.10 主机管理 Host
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/host.go` | 320 | 主机清单 |
| `service/host_tool.go` | 625 | 工具：重启 / 关机 / 改 hostname |
| `service/monitor.go` | 642 | CPU/内存/磁盘/网络指标采集 |
| `service/device.go` | 433 | 设备/磁盘 |
| `service/device_clean.go` | **1118** | 磁盘清理（找大文件 / 临时文件） |
| `service/disk.go` + `disk_utils.go` | 679 | 磁盘管理 |
| `service/gpu.go` | — | GPU 信息（NVIDIA 容器场景） |
| `service/process.go` | 80 | 进程列表 |
| `api/v2/host.go` + `host_tool.go` | 412 | HTTP 入口 |

> **看点**：`monitor.go` 642 行的指标采集，**对位你 L1 readyz / L3 Obs 需求**。

### 1.11 运行时 Runtime (AI 模型运行)
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/runtime.go` | **1382** | 模型运行时管理（启动 / 停止 / 调用） |
| `service/runtime_utils.go` | 1093 | runtime 公共 |
| `service/runtime_diagnostics.go` | 386 | 故障诊断 |
| `service/tensorrt_llm.go` | 377 | TensorRT-LLM 集成 |
| `service/vllm_upgrade.go` | 127 | vLLM 升级 |
| `api/v2/runtime*.go` | 610+ | HTTP 入口 |

> **看点**：v2 新增的 AI 推理运行时管理（vLLM / TensorRT-LLM），**跟 AI Agent 模块配合**。

### 1.12 防火墙 / 安全（已研究）
| 文件 | 行 | 状态 |
|---|---:|---|
| `service/firewall_service.go` | 2529 | ✅ 已分析（见 `firewall-architecture.md`） |
| `service/firewall_docker.go` | 556 | ✅ 已分析 |
| `service/firewall_setting.go` | 269 | ✅ 已分析 |
| `service/ssh.go` | **1603** | 🔲 未读（SSH 终端 + 凭据） |
| `service/fail2ban.go` | 230 | 🔲 未读（fail2ban 集成） |
| `service/clam.go` | 477 | 🔲 未读（ClamAV 防病毒） |

### 1.13 工具 / 系统
| 文件 | 行 | 职责 |
|---|---:|---|
| `service/dashboard.go` | 626 | 首页 dashboard 数据 |
| `service/setting.go` | 325 | 系统设置 |
| `service/group.go` | 88 | 用户组 |
| `service/favorite.go` | 76 | 收藏夹 |
| `service/logs.go` | 527 | 日志查看 |
| `service/ftp.go` | 271 | FTP 服务 |
| `service/snapshot.go` | 420 | 系统快照（跟备份快照不同） |
| `service/php_extensions.go` | 77 | PHP 扩展管理 |
| `service/recycle_bin.go` | 479 | 回收站 |
| `service/compose_template.go` | 88 | compose 模板 |

### 1.14 跨切关注点（基础库）
| 路径 | 职责 |
|---|---|
| `agent/init/` | 启动初始化（migration / secret 生成） |
| `agent/router/` | Gin 路由注册 |
| `agent/middleware/` | 鉴权 / 限流 / 日志 / CORS |
| `agent/global/` | 全局变量（DB / LOG / CONFIG） |
| `agent/constant/` | 枚举 / 字符串常量 |
| `agent/buserr/` | 业务错误码 |
| `agent/log/` | 日志封装 |
| `agent/i18n/` | 国际化 |
| `agent/utils/firewall/` | 防火墙工具（已分析） |
| `agent/utils/cmd/` | shell 调用封装 |
| `agent/utils/ctl/` | systemctl / process 管理 |
| `agent/utils/nginx/` | Nginx 工具 |
| `agent/utils/mysql_utils/` | MySQL 客户端 |
| ... 其他 utils | 各种零碎工具 |

---

## 2. core/ 主控端（172 .go 文件）

`core/` 跟 `agent/` 目录结构类似，但**精简得多**：
- `core/cmd/server/main.go` — 入口（仅 38 行）
- `core/router/` — 路由（10 文件，**包含对所有 agent 的 RPC 转发**）
- `core/app/` — 业务（52 文件）
- `core/utils/` — 工具（42 文件）

**重点看**：
- `core/router/` 怎么把 HTTP 请求转给 agent
- `core/app/` 的 service 是不是只是 agent service 的"薄包装"
- `core/init/migration/` 里**1Panel v1 → v2 的数据迁移**（这条 v1 升 v2 的 SQL 转换可能跟你 Sirius Cloud 跨版本兼容有关）

---

## 3. frontend/ Vue 3 SPA

488 个 view 文件，按业务域分目录：
| 目录 | 文件 | 含义 |
|---|---:|---|
| `ai/` | 64 | **AI Agent UI** (v2 新增) |
| `website/` | 120 | 网站管理 |
| `host/` | 76 | 主机 / 监控 / 防火墙 |
| `database/` | 47 | 数据库管理 |
| `container/` | 44 | 容器 |
| `setting/` | 41 | 系统设置 |
| `toolbox/` | 35 | 工具箱 |
| `app-store/` | 23 | 应用商店 |
| `terminal/` | 12 | 终端 |
| `cronjob/` | 11 | 定时任务 |
| `log/` | 8 | 日志查看 |
| `home/` | 4 | 首页 |
| `login/` | 2 | 登录 |

技术栈：Vue 3.5 + Element Plus 2.14 + Pinia 3 + vue-router 5 + Vite 8 + TypeScript 6 + ECharts 6 + xterm 5 + monaco-editor + CodeMirror 6 + noVNC + @vue-office (docx/excel preview)

> **对你的价值**：UI 模式 / 状态管理 / 路由权限设计可以直接借鉴，但**视觉风格不要抄**（GPL v3 + 你有自己设计系统）。

---

## 4. 阅读路线建议

### 第一次接触（30 分钟内搞懂整体）
1. `agent/cmd/server/main.go` — agent 启动流程
2. `agent/router/` — HTTP 路由结构
3. `agent/app/api/v2/agents.go`（虽然叫 agents.go 但其实是路由总入口）— 看 200+ 个 endpoint 怎么组织
4. `agent/app/service/entry.go` — 业务入口

### 第二次（按你 Sirius Cloud 需求挑）

| Sirius Cloud 需求 | 对应 1Panel 模块 | 推荐度 |
|---|---|---|
| **L2 主机纳管 + 防火墙** | firewall (已读) / host | ⭐⭐⭐⭐⭐ |
| **L2 应用部署** | app-store / container | ⭐⭐⭐⭐ |
| **L2 数据库** | database/mysql | ⭐⭐⭐⭐ |
| **L3 监控告警** | monitor / alert | ⭐⭐⭐⭐ |
| **L3 备份恢复** | backup / snapshot | ⭐⭐⭐⭐ |
| **L3 定时任务** | cronjob | ⭐⭐⭐ |
| **L1 文件管理 / 资产管理** | file | ⭐⭐⭐ |
| **新功能：AI Agent 平台** | agents_*/mcp_server | ⭐⭐⭐⭐⭐（前沿方向）|

### 第三次（v2 的杀手锏）
- **AI Agent 模块**（`agents.go` + `agents_utils.go` + `mcp_server.go`）—— 1Panel v2 押注的方向，值得细读

---

## 5. 模块注解文档计划

每个详细注解文档固定结构：

```markdown
# {模块名} — 详细注解

## 0. 模块定位
## 1. 核心文件清单
## 2. 关键数据模型（DB / DTO）
## 3. 关键 API 列表
## 4. 实现逻辑（贴代码 + 解释）
   ### 4.1 入口流程
   ### 4.2 关键业务函数（每函数 1 段：做什么 / 怎么做的 / 注意点）
   ### 4.3 错误处理 / 边界
## 5. 跟 Sirius Cloud 的对位 / 借鉴价值
## 6. 避坑 / 不建议照搬
```

每个文档预计 **400-800 行 markdown**（含 50-100 段代码引用）。

---

## 6. 文档产出进度

| 状态 | 模块 | 文档 |
|---|---|---|
| ✅ 已完成 | 防火墙 v2 | `../firewall-architecture.md` |
| 🟡 框架已建（README 占位） | App Store | `01-app-store/README.md` |
| 🟡 框架已建 | Container | `02-container/README.md` |
| 🟡 框架已建 | Website | `03-website/README.md` |
| 🟡 框架已建 | Database | `04-database/README.md` |
| 🟡 框架已建 | Backup & Snapshot | `05-backup-snapshot/README.md` |
| 🟡 框架已建 | Cronjob | `06-cronjob/README.md` |
| 🟡 框架已建 | Alert | `07-alert/README.md` |
| 🟡 框架已建 | File | `08-file/README.md` |
| 🟡 框架已建 | AI Agent | `09-ai-agent/README.md` |
| 🟡 框架已建 | Host & Monitor | `10-host-monitor/README.md` |
| 🟡 框架已建 | Runtime (AI) | `11-runtime-ai/README.md` |
| 🟡 框架已建（部分已读） | Security | `12-security/README.md` |
| 🟡 框架已建 | Frontend | `13-frontend/README.md` |
| ⏳ 待选 | （选中后填充 §4 实现逻辑） | |
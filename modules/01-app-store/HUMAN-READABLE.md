# 1Panel App Store 模块 — 函数级 KB 文档 v2

> **模块代号**: `01-app-store` ⭐⭐⭐
> **一句话作用**: 1Panel 的"App 应用商店"——一键装 WordPress / MySQL / Redis / Nginx / OpenResty 等上百款应用, 类似手机应用商店的"商品浏览 → 详情 → 一键安装"全流程。
> **覆盖代码**: 25 个 Go 文件, ~3400 行核心业务代码 (不含 service/app_install.go / app_upgrade.go / app_utils.go / app_sync_task.go 这些"已装应用管理"的旁支)。
> **文档版本**: v2 (2026-08-25) — 细到函数级, 4 段讲解 (purpose/params/flow/callees)。

---

## 目录

- [1. 一句话作用](#1-一句话作用)
- [2. 模块职责](#2-模块职责)
- [3. 目录结构](#3-目录结构-带行数)
- [4. 调用链 Router → Handler → Service → Repository](#4-调用链-router--handler--service--repository)
- [5. 数据库表 (GORM 模型)](#5-数据库表-gorm-模型)
- [6. 逐函数讲解](#6-逐函数讲解-4-段-格式)
- [7. 类与结构体讲解](#7-类与结构体讲解)
- [8. Mermaid 时序图](#8-mermaid-时序图-2-个核心流程)
- [9. Docker 相关依赖](#9-docker-相关依赖)
- [10. 跟其他模块的关系](#10-跟其他模块的关系)
- [11. 关键文件清单](#11-关键文件清单)
- [12. 行号索引](#12-行号索引)

---

## 1. 一句话作用

**1Panel App Store 模块 = "上架 + 搜索 + 一键安装" 应用商店核心**, 它把 `1panel.json` 远端商店目录 / `/opt/1panel/resource/local_app` 本地应用库 / 用户自定义应用三处来源的应用元数据统一拉取到本地 SQLite (`apps`/`app_details`/`app_tags` 三张表), 用户在前端搜 "WordPress" → 后端查表 → 展示商品卡 → 点击安装 → 调 `Install()` → 模板渲染 docker-compose.yml → `docker compose up -d` 启动容器 → 写 `app_installs` 实例表。

**3 个生活类比** (帮助新人秒懂):

1. **🏪 手机 App Store 类比** — `App` 表 = 应用商店商品列表 (icon/name/desc/价格), `AppDetail` = 不同版本 (v1.0 / v2.0), `AppInstall` = 你手机里"已下载的应用", `Install()` = 点击"获取"按钮, `SyncAppListFromRemote()` = App Store 后台每天同步"今日推荐"列表。
2. **🛒 超市购物类比** — 同步 (sync) = 超市每天补货上架, 详情 (detail) = 商品背面的配料表, 安装 (install) = 顾客拿走 + 收银台扫码 + 传送带送到停车场, 钩子 (hooks) = "买一送一"活动的赠品绑定逻辑。
3. **🏗 装修套餐类比** — `docker-compose.yml` = 装修模板 (水电网管预埋), `Params` = 业主自选配置 (瓷砖颜色/橱柜品牌), `AppContainerConfig` = 高级定制 (中央空调/地暖), `createLink()` = 把水电网管接通到业主家, `upApp()` = 工人进场施工。

---

## 2. 模块职责 (8 条)

1. **应用元数据管理** — 远端商店 `1panel.json.zip` 同步 / 本地 `/opt/1panel/resource/local_app` 扫描 / 自定义应用上传, 三路汇入统一 GORM 表。
2. **应用搜索与详情** — 分页 / 标签筛选 / 关键字模糊匹配 / 架构 (`amd64`/`arm64`) 过滤 / 推荐位排序, 返回 icon/描述/版本/已装状态。
3. **应用版本管理** — 一个 `App` 对应多个 `AppDetail` (每个版本一行), 保存 `Params` JSON / `DockerCompose` YAML / `DownloadUrl` 下载包。
4. **一键安装** — 渲染 docker-compose 模板 (环境变量注入 / 端口校验 / 容器名校验 / 主机网络模式判定), 调用 `docker compose up -d` 起容器, 写 `app_installs` 实例记录。
5. **应用图标服务** — `GetAppIcon()` 返回 base64 解码或本地文件, 支持 ETag 304 缓存 (max-age 30 天)。
6. **安装后管理** — 通过 `app_install.go` (本 KB 范围外) 提供启动/停止/重启/卸载/升级, 通过 `app_upgrade.go` 处理跨版本升级。
7. **升级检测** — `GetAppUpdate()` 比对远端 `1panel.json.version.txt` 与本地 `AppStoreLastModified`, 触发 `SyncAppListFromRemote()` 全量拉取。
8. **依赖应用联动** — 检测 `PANEL_DB_HOST` 环境变量, 自动校验依赖的 MySQL/Redis 容器是否在 `Running` 状态, 否则 `ErrAppIsDown` 阻止安装。

---

## 3. 目录结构 (带行数)

> 路径相对 `D:\MiniMax Code\1Panel\agent\app\`, 仅列 App Store 直接相关文件。

### 3.1 Service 层 (业务逻辑主战场)

| 文件 | 行数 | 关键内容 |
|------|------|---------|
| `service/app.go` | 1055 | **核心**: `PageApp` / `GetApp` / `GetAppDetail` / `Install` / `installWithHooks` / `SyncAppListFromLocal` / `GetAppUpdate` / `SyncAppListFromRemote` / `GetAppIcon` / `getAppFromRepo` / `getAppList` / `deleteCustomApp` / `GetAppTags` / `GetAppDetailByKey` / `GetAppDetailByID` |
| `service/app_install.go` | 32 KB / ~960 行 | **旁支**: 已装应用 CRUD, `Operate`/`Update`/`SyncAll`/`Page`/`CheckExist`/`LoadPort`/`GetServices`/`ChangeAppPort`/`DeleteCheck`/`GetDefaultConfigByKey`/`GetParams`/`GetUpdateVersions`/`UpdateSort`/`SearchForWebsite`/`LoadConnInfo`/`GetInstallList`/`GetAppInstallInfo`/`UpdateAppConfig` |
| `service/app_upgrade.go` | 28 KB | **旁支**: `AppInstallUpgrade` 跨版本升级流程 |
| `service/app_utils.go` | 70 KB | **旁支**: 通用工具 (复制数据 / 跑 init 脚本 / 写 .env / 处理 openresty / 更新关联 tool) |
| `service/app_sync_task.go` | 16 KB | **旁支**: `createSyncAppStoreTask` / `createSyncAppStoreMetaTask` (被 `SyncAppListFromRemote` 调用) |
| `service/app_ingore_upgrade.go` | 2 KB | **旁支**: 忽略升级名单 CRUD |

### 3.2 API 层 (Gin Handler)

| 文件 | 行数 | 端点数 |
|------|------|--------|
| `api/v2/app.go` | 261 | **11 个核心端点** (见 §4 调用链) |
| `api/v2/app_install.go` | 372 | 20 个已装应用端点 (Operate/Sync/Port/Sort/Conf/Info 等) |
| `api/v2/app_ignore_upgrade.go` | ~80 | 3 个忽略升级端点 |

### 3.3 DTO 层 (请求/响应/共享)

| 文件 | 行数 | 关键 DTO |
|------|------|---------|
| `dto/app.go` | 194 | `AppDatabase` / `AuthParam` / `RedisAuthParam` / `MinioAuthParam` / `ContainerExec` / `AppOssConfig` / `AppVersion` / `AppList` / `AppDefine` / `AppProperty` / `AppConfigVersion` / `LocalAppAppDefine` / `LocalAppParam` / `LocalAppInstallDefine` / `ExtraProperties` / `Tag` / `Locale` / `AppForm` / `AppFormFields` / `AppFormValue` / `AppResource` / `AppInstallInfo` / `DelAppLink` / `PHPForm` |
| `dto/request/app.go` | 150 | `AppSearch` / `AppInstallCreate` / `AppContainerConfig` / `NodePushConfig` / `AppInstalledSearch` / `AppInstalledInfo` / `AppBackupSearch` / `AppBackupDelete` / `AppInstalledOperate` / `AppInstallUpgrade` / `AppInstallDelete` / `AppInstalledUpdate` / `AppConfigUpdate` / `AppInstalledIgnoreUpgrade` / `PortUpdate` / `AppUpdateVersion` / `AppInstallSortItem` / `AppInstallSort` |
| `dto/response/app.go` | 189 | `AppRes` / `AppUpdateRes` / `AppDTO` / `AppItem` / `TagDTO` / `AppInstalledCheck` / `AppDetailDTO` / `AppDetailSimpleDTO` / `IgnoredApp` / `AppInstalledDTO` / `AppInstallDTO` / `AppInstallInfo` / `DatabaseConn` / `AppService` / `AppParam` / `AppConfig` |

### 3.4 Model 层 (GORM 表结构)

| 文件 | 行数 | 模型 |
|------|------|------|
| `model/app.go` | 83 | `App` (商品主表) + `GetDescription()` / `GetAppResourcePath()` / `IsLocalApp()` / `IsCustomApp()` |
| `model/app_detail.go` | 15 | `AppDetail` (版本表) |
| `model/app_install.go` | 52 | `AppInstall` (已装实例表) + `GetPath()` / `GetComposePath()` / `GetEnvPath()` / `GetAppPath()` |
| `model/app_tag.go` | 7 | `AppTag` (多对多关联表) |
| `model/app_ignore_upgrade.go` | 9 | `AppIgnoreUpgrade` (忽略升级名单) |
| `model/app_install_resource.go` | 13 | 应用安装资源关联 |
| `model/app_launcher.go` | 19 | 桌面快捷方式 |

### 3.5 Repo 层 (数据库访问)

| 文件 | 行数 | Repo |
|------|------|------|
| `repo/app.go` | 171 | `AppRepo` 12 个 DBOption + 9 个 CRUD |
| `repo/app_detail.go` | 88 | `AppDetailRepo` 3 DBOption + 8 CRUD |
| `repo/app_install.go` | 270 | `AppInstallRepo` 11 DBOption + 14 CRUD |
| `repo/app_install_resource.go` | ~100 | 已装应用资源 |
| `repo/app_tag.go` | ~80 | 标签关联 |
| `repo/app_ignore_upgrade.go` | ~50 | 忽略升级 |

### 3.6 Router 层

| 文件 | 行数 | 路由 |
|------|------|------|
| `router/ro_app.go` | ~50 | `appRouter := Router.Group("apps")` + 26 条路由 |
| `init/router/router.go` | 31 | `Routers()` 挂载点 |

---

## 4. 调用链 Router → Handler → Service → Repository

### 4.1 完整端点清单 (router/ro_app.go)

| HTTP | 路径 | Handler (api/v2/app.go) | Service (service/app.go) | 关键 Repo |
|------|------|------------------------|--------------------------|-----------|
| POST | `/apps/sync/remote` | `SyncApp` (L42) | `GetAppUpdate` (L836) → `SyncAppListFromRemote` (L976) | `appRepo` / `appDetailRepo` / `tagRepo` / `appTagRepo` |
| POST | `/apps/sync/local` | `SyncLocalApp` (L74) | `SyncAppListFromLocal` (L604) | 同上 |
| GET  | `/apps/checkupdate` | `GetAppListUpdate` (L196) | `GetAppUpdate` (L836) | `appRepo` / `appicon` |
| POST | `/apps/search` | `SearchApp` (L22) | `PageApp` (L69) | `appRepo` / `appDetailRepo` / `tagRepo` / `appTagRepo` / `appInstallRepo` / `runtimeRepo` |
| GET  | `/apps/:key` | `GetApp` (L91) | `GetApp` (L196) | `appRepo` / `appDetailRepo` |
| GET  | `/apps/detail/:appId/:version/:type` | `GetAppDetail` (L115) | `GetAppDetail` (L234) | `appRepo` / `appDetailRepo` + HTTP 拉取 compose |
| GET  | `/apps/detail/node/:appKey/:version` | `GetAppDetailForNode` (L252) | `GetAppDetailByKey` (L220) | `appRepo` / `appDetailRepo` |
| GET  | `/apps/details/:id` | `GetAppDetailByID` (L139) | `GetAppDetailByID` (L334) | `appDetailRepo` |
| POST | `/apps/install` | `InstallApp` (L162) | `Install` (L350) → `installWithHooks` (L354) | `appRepo` / `appDetailRepo` / `appInstallRepo` / `databaseRepo` / `task` |
| GET  | `/apps/tags` | `GetAppTags` (L181) | `GetAppTags` (L174) | `tagRepo` |
| GET  | `/apps/icon/:key` | `GetAppIcon` (L213) | `GetAppIcon` (L1031) | `appRepo` + `appicon` 包 |

(以下 20 个已装应用端点不在本 KB 范围, 列在 `app_install.go` handler 中)
POST `/apps/installed/search` · GET `/apps/installed/list` · POST `/apps/installed/check` · POST `/apps/installed/loadport` · POST `/apps/installed/conninfo` · GET `/apps/installed/delete/check/:appInstallId` · POST `/apps/installed/op` · POST `/apps/installed/sync` · POST `/apps/installed/port/change` · GET `/apps/services/:key` · POST `/apps/installed/conf` · GET `/apps/installed/params/:appInstallId` · POST `/apps/installed/params/update` · POST `/apps/installed/update/versions` · POST `/apps/installed/config/update` · POST `/apps/installed/sort/update` · GET `/apps/installed/info/:appInstallId` · POST `/apps/installed/ignore` · GET `/apps/ignored/detail` · POST `/apps/ignored/cancel`

### 4.2 典型调用链图

```
[前端] http POST /api/v2/apps/install
        │ Body: { appDetailId, name, params, advanced, ... }
        ▼
[Middleware] Certificate() → OperationResolveMeta() → 写入 audit log
        │
        ▼
[Router] ro_app.go:24  appRouter.POST("/install", baseApi.InstallApp)
        │
        ▼
[Handler] api/v2/app.go:162  InstallApp(c *gin.Context)
   1. helper.CheckBindAndValidate(&req, c)         ← JSON bind + validator
   2. appService.Install(req, true)                ← executeScript=true
   3. helper.SuccessWithData(c, install)           ← 返回 *model.AppInstall
        │
        ▼
[Service] service/app.go:350  Install() (壳函数)
        │
        ▼
[Service] service/app.go:354  installWithHooks()   ★★★ 250 行核心
   ├─ docker.CreateDefaultDockerNetwork()         ← 创建 1panel 网络
   ├─ appInstallRepo.ListBy(WithByLowerName)       ← 名称冲突检查
   ├─ appDetailRepo.GetFirst / appRepo.GetFirst    ← 加载商品元数据
   ├─ databaseRepo 检查 DB 依赖是否 Running
   ├─ checkPort() 端口校验
   ├─ checkRequiredAndLimit() CPU/内存/架构校验
   ├─ 渲染 docker-compose.yml  (yaml.Marshal)
   ├─ addDockerComposeCommonParam() 注入端口/CPU/内存
   ├─ appInstallRepo.Create() 写实例表
   ├─ task.NewTaskWithOps() 创建异步任务
   ├─ createLink() 把参数写入容器 .env / hosts
   └─ go installTask.Execute()                     ← 异步 goroutine
        │
        ├─ copyData()  拷贝应用数据到 /opt/1panel/apps/<key>/<name>/
        ├─ runScript() 执行 init.sh
        ├─ upApp()     docker compose up -d
        └─ updateToolApp() 联动 phpmyadmin / redis-commander
        │
        ▼
[Repo] GORM SQLite
   apps (App 主表)         ← appRepo
   app_details (版本)      ← appDetailRepo
   app_installs (已装)     ← appInstallRepo
   app_tags (多对多)       ← appTagRepo
   tags (标签字典)         ← tagRepo
```

---

## 5. 数据库表 (GORM 模型)

### 5.1 `apps` 表 (model/app.go:13)

商品主表, 来自远端商店 / 本地 / 自定义三路汇入。

| 字段 | 类型 | 含义 |
|------|------|------|
| `id` | uint PK | 自增主键, 继承自 `BaseModel` |
| `name` | string NOT NULL | 应用显示名, 如 "WordPress" |
| `key` | string NOT NULL | 应用唯一标识, 如 "wordpress", 用于 API 路径 |
| `short_desc_zh` / `short_desc_en` | string | 中/英短描述 (备选 fallback) |
| `description` | string (JSON) | 多语言长描述, 结构 `{"zh":"...", "en":"..."}` |
| `icon` | string | 图标: 旧版 base64 字符串, 新版 `icon:<file>.png:etag` |
| `type` | string NOT NULL | `runtime` / `php` / `node` / `mysql` / `wordpress` / `openresty` 等 |
| `status` | string NOT NULL | `Normal` / `TakeDown` (本地 sync 时) |
| `required` | string | 依赖声明 (逗号分隔 key) |
| `cross_version_update` | bool | 是否允许跨大版本升级 |
| `limit` | int NOT NULL | 安装数量上限 (0=不限制, 1=单实例, 2=双实例) |
| `website` / `github` / `document` | string | 官网/源码/文档链接 |
| `recommend` | int NOT NULL | 推荐位排序权重, 越小越靠前, 9999=不推荐 |
| `resource` | string NOT NULL DEFAULT 'remote' | 来源: `remote` / `local` / `custom` |
| `read_me` | string | Markdown 介绍 |
| `last_modified` | int | 远端 `lastModified` 时间戳, 用于 sync diff |
| `architectures` | string (逗号分隔) | 支持的 CPU 架构 |
| `memory_required` | int (MB) | 最低内存要求, `checkRequiredAndLimit` 校验 |
| `gpu_support` | bool | 是否支持 GPU |
| `required_panel_version` | float64 | 最低 1Panel 版本要求 (PHP 类应用用) |
| `batch_install_support` | bool | 标记 mysql 是否支持"安装后立即同步" |

> **Jargon**: `BaseModel` = 1Panel 通用基类, 含 `id` / `createdAt` / `updatedAt` 三字段。`gorm:"-migration"` 标签表示 Preload 字段不入库。
> `TagsKey []string` 在 sync 流程中临时承载 `tags` YAML 字段, 落库前转成 `AppTag` 多对多行。

### 5.2 `app_details` 表 (model/app_detail.go:3)

一个 App 对应多个版本, 每行一个版本。

| 字段 | 类型 | 含义 |
|------|------|------|
| `id` | uint PK | 自增主键 |
| `app_id` | uint NOT NULL | 外键 → `apps.id`, 无 FK 约束 (GORM 软关联) |
| `version` | string NOT NULL | 语义化版本, 如 "6.5.3" |
| `params` | string (JSON) | 模板参数字段定义, 前端表单生成依据 |
| `docker_compose` | string (YAML) | 完整 compose 文件, `install` 时按需注入变量 |
| `status` | string NOT NULL | `Normal` / `TakeDown` |
| `last_version` | string | 升级时的上一版本, 用于判断跨版本 |
| `last_modified` | int | 远端时间戳, sync diff 用 |
| `download_url` | string | 应用包下载 URL (zip 含完整 compose + 资源) |
| `download_call_back_url` | string | 安装回调 URL (上报安装数) |
| `update` | bool | 标记此版本需重新下载 (本地文件被清) |

### 5.3 `app_installs` 表 (model/app_install.go:11)

已装实例表, 每次 `Install()` 写一行。

| 字段 | 类型 | 含义 |
|------|------|------|
| `id` | uint PK | 自增主键 |
| `name` | string UNIQUE NOT NULL | 用户起的实例名, 全局唯一, 冲突报 `ErrAppNameExist` |
| `app_id` | uint NOT NULL | 外键 → `apps.id` |
| `app_detail_id` | uint NOT NULL | 外键 → `app_details.id` |
| `version` | string NOT NULL | 安装时的版本, 升级时更新 |
| `param` | string (JSON) | 旧的"表单参数"快照 (部分路径用) |
| `env` | string (JSON) | 完整环境变量 (新流程用, `params` 注入后序列化) |
| `docker_compose` | string (YAML) | 安装时定稿的 compose, 升级时按新版本重新生成 |
| `status` | string NOT NULL | `Installing` / `Running` / `UpErr` / `InstallErr` / `Stopped` |
| `description` | string | 用户备注 |
| `message` | string | 错误信息 (失败时填) |
| `container_name` | string NOT NULL | Docker 容器名, 默认 `1Panel-<key>-<rand4>` |
| `service_name` | string NOT NULL | compose service 名, 默认取 `services` 第一个 key |
| `http_port` / `https_port` | int | 应用暴露的端口 (从 `PANEL_APP_PORT_*` 提取) |
| `web_ui` | string | 应用 Web 入口 URL (某些 app 自带, 如 WordPress 后台) |
| `favorite` | bool | 是否收藏 (首页置顶) |
| `sort_order` | int DEFAULT 0 | 列表排序, 收藏的应用置顶 + 自定义顺序 |

### 5.4 `app_tags` / `tags` 多对多 (model/app_tag.go:3 + model/tag.go)

```
app_tags (关联表)            tags (字典表)
  app_id  ─────┐               id
  tag_id  ─────┼──FK──►        key (e.g. "database")
               │               name (i18n JSON)
               └─────FK──►     sort (显示顺序)
```

`App.GetDescription(ctx)` (model/app.go:69) 解析 `description` JSON 字段, 按 `Accept-Language` 头选 zh/en/...。

### 5.5 `app_ignore_upgrades` (model/app_ignore_upgrade.go:3)

| 字段 | 含义 |
|------|------|
| `app_id` | 被忽略升级的 App |
| `app_detail_id` | 被忽略的具体版本 |
| `scope` | `all` / `version` (v2.5+ 区分) |

---

## 6. 逐函数讲解 (4 段格式)

> **格式约定**: 每个 func 4 段 — `**Purpose**` / `**Params**` / `**Flow**` / `**Callees**`。
> **优先级**: ★★★★★ = 文件核心 (>200 行); ★★★ = 主要业务 (50-200 行); ★★ = 辅助 (20-50 行); ★ = 工具 (< 20 行)。
> **8-10 关键 func 详细讲解** + **其他 func 简表**。

### 6.1 ★★★★★ `Install` (service/app.go:350) + `installWithHooks` (service/app.go:354)

**File**: `D:\MiniMax Code\1Panel\agent\app\service\app.go:350-602`
**Lines**: 252 行 (整个 install 流程的工作马)
**Type**: `func (a AppService) Install(req request.AppInstallCreate, executeScript bool) (appInstall *model.AppInstall, err error)`

#### 6.1.1 Purpose
**应用商店"一键安装"的主入口**。接收前端填好的参数 (`name` / `params` / `appDetailId` / `advanced`...), 完成 9 个关键步骤: 网络检查 → 名称冲突检查 → 加载商品元数据 → 数据库依赖校验 → 端口校验 → 资源限制校验 → 渲染 docker-compose 模板 → 创建 task 异步执行 → 返回 `*model.AppInstall`。**实际逻辑全在 `installWithHooks`, `Install` 仅是壳函数 (无 hooks)**。

#### 6.1.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `req` | `request.AppInstallCreate` | 见 dto/request/app.go:19, 含 `AppDetailId` (要装的版本) / `Name` (实例名) / `Params` (环境变量 map) / `AppContainerConfig` (高级选项: cpu/memory/port/hostMode/editCompose/gpu) / `TaskID` (前端给的任务 ID, 用于前端轮询) |
| `executeScript` | `bool` | 是否执行 `init.sh` 初始化脚本, `Install()` 永远传 `true` (handler 调) |

**Returns**: `*model.AppInstall` (创建成功的实例对象, 包含 ID) + `error`。

#### 6.1.3 Flow (9 步)

```
[Step 1] L355 docker.CreateDefaultDockerNetwork()
   └─ 创建 1Panel 专用 Docker network (1panel-network), 失败报 Err1PanelNetworkFailed

[Step 2] L359 appInstallRepo.ListBy(WithByLowerName(req.Name))
   └─ 查重: 同名实例已存在 → ErrAppNameExist

[Step 3] L369-376 加载 app_detail + app 元数据
   ├─ appDetailRepo.GetFirst(repo.WithByID(req.AppDetailId))  // 取版本详情
   └─ appRepo.GetFirst(repo.WithByID(appDetail.AppId))         // 取商品主表

[Step 4] L377-391 数据库依赖联动 (DatabaseKeys 白名单: mysql/postgresql/redis/mongodb)
   ├─ 如果是 DB 类 app: 查重 databaseRepo.GetList(WithByName(req.Name))
   └─ 如果 req.Params["PANEL_DB_HOST"] 存在: 校验依赖 DB 容器 Running

[Step 5] L392-407 端口提取
   └─ 遍历 req.Params, 找 PANEL_APP_PORT_* 键, 调 checkPort() 校验未占用
   └─ 记下 httpPort / httpsPort

[Step 6] L409 checkRequiredAndLimit(app)
   └─ 校验: 内存 ≥ memoryRequired / 已装数 < limit / 架构匹配 / GPU 支持

[Step 7] L413-422 构造 appInstall 对象 (Status="Installing")
   └─ Name / AppId / AppDetailId / Version / HttpPort / HttpsPort / App

[Step 8] L425-447 渲染 docker-compose 模板
   ├─ 高级模式 (req.EditCompose=true): 用 req.DockerCompose 直接 yaml.Unmarshal
   └─ 默认模式: 从 appDetail.DockerCompose 取, 缺则 HTTP 拉 {AppRepoURL}/{mode}/1panel/{key}/{version}/docker-compose.yml

[Step 9] L455-474 容器名生成
   ├─ 默认: 1Panel-<key>-<rand4>
   └─ req.Advanced=true && req.ContainerName!="": 校验重名 + Docker 实际检查

[Step 10] L476-490 service 名生成
   └─ 取 compose.services 第一个 key, 如果 Limit==0 且只有一个 service, 用 appInstall.Name 重命名

[Step 11] L493-495 addDockerComposeCommonParam() 注入 cpu/memory/port/host 网络模式

[Step 12] L501-505 yaml.Marshal → appInstall.DockerCompose

[Step 13] L507-514 PANEL_DB_HOST 替换为真实 IP/Port
   └─ 如果 params 有 DB 引用, 把 "DB 名称" 替换成实际 docker IP+Port

[Step 14] L516-529 openresty 特殊处理
   └─ 设 CONTAINER_PACKAGE_URL = "http://archive.ubuntu.com/ubuntu/"

[Step 15] L530-534 json.Marshal(req.Params) → appInstall.Env

[Step 16] L537-538 计算 sort_order = MAX(sort_order where favorite=false) + 1

[Step 17] L540-542 appInstallRepo.Create() 写库 (同步, 立即返回 ID)

[Step 18] L544-547 task.NewTaskWithOps() 创建 TaskScopeApp 任务

[Step 19] L549-551 createLink() 写 .env / hosts 文件 (在容器启动前)

[Step 20] L553-580 定义 installApp 闭包 (实际安装子任务)
   ├─ copyData()      拷数据到 /opt/1panel/apps/<key>/<name>/
   ├─ hooks.AfterCopyData (本流程 nil, 升级流程用)
   ├─ runScript()     init.sh (executeScript=true)
   ├─ openresty: handleSiteDir + handleOpenrestyFile
   ├─ upApp()         docker compose up -d
   └─ updateToolApp() 联动装 phpmyadmin (mysql) / redis-commander (redis)

[Step 21] L582-586 handleAppStatus 错误处理闭包
   └─ 失败时 appInstall.Status = UpErr, 保存

[Step 22] L588 AddSubTaskWithOps 注册子任务, 1h 超时

[Step 23] L590-599 go installTask.Execute()
   └─ 异步执行, 错误回写 appInstall.Status = InstallErr
```

#### 6.1.4 Callees (调用的下游)
- `docker.CreateDefaultDockerNetwork()` (utils/docker)
- `appInstallRepo.ListBy / GetFirst / Create / Save` (repo/app_install.go)
- `appDetailRepo.GetFirst / Update` (repo/app_detail.go)
- `appRepo.GetFirst` (repo/app.go)
- `databaseRepo.GetList / Get` (repo/database.go)
- `req_helper.HandleRequest` (utils/req_helper) — HTTP 拉 compose
- `yaml.Marshal / Unmarshal` (gopkg.in/yaml.v3)
- `checkPort / checkRequiredAndLimit / checkContainerNameIsExist / addDockerComposeCommonParam / isHostModel` (utils/app_helpers)
- `task.NewTaskWithOps` (app/task)
- `createLink / copyData / runScript / upApp / updateToolApp / handleSiteDir / handleOpenrestyFile` (service/app_utils.go)

---

### 6.2 ★★★ `PageApp` (service/app.go:69)

**File**: `service/app.go:69-172`
**Lines**: 103 行
**Type**: `func (a AppService) PageApp(ctx *gin.Context, req request.AppSearch) (*response.AppRes, error)`

#### 6.2.1 Purpose
**应用商店的"商品搜索 + 分页"主入口**。前端 `POST /api/v2/apps/search` 调它, 支持 7 种筛选: 关键字 (`name` 模糊) / 类型 / 推荐位 / 来源 (`resource`=local/remote) / 当前架构 (`showCurrentArch=true` 自动加架构过滤) / 标签 (多对多 join) / 分页 (`page`+`pageSize`)。返回 `*response.AppRes` 含分页结果 + 每条 App 的"是否已装" (`Installed` 字段)。

#### 6.2.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `ctx` | `*gin.Context` | Gin 上下文, 拿 `Accept-Language` 头选 i18n |
| `req` | `request.AppSearch` | 见 dto/request/app.go:9, 含 `PageInfo{Name, Tags, Type, Recommend, Resource, ShowCurrentArch}` |

**Returns**: `*response.AppRes` (Items + Total) + `error`。

#### 6.2.3 Flow (5 步)

```
[Step 1] L70-114 拼装 DBOption
   ├─ L71 OrderByRecommend()      // 按 recommend ASC
   ├─ L72-74 Name 模糊: WHERE name/description/short_desc_zh/en LIKE %name%
   ├─ L75-77 Type 精确: WHERE type=?
   ├─ L78-80 Recommend: WHERE recommend<9999
   ├─ L81-83 Resource: WHERE resource=?, "all" 不过滤
   ├─ L85-95 ShowCurrentArch: LoadOsInfo → kernelArch (aarch64→arm64) → architectures LIKE %arch%
   └─ L96-114 Tags 多对多: tagRepo.GetByKeys → appTagRepo.GetByTagIds → 拿到 appIds → WithByIDs

[Step 2] L117 appRepo.Page(req.Page, req.PageSize, opts...)
   └─ SELECT COUNT + SELECT LIMIT size OFFSET size*(page-1) + Preload("AppTags")

[Step 3] L123-125 PHP 类型额外加载 setting (SystemVersion 用于 RequiredPanelVersion 校验)

[Step 4] L126-167 遍历 apps 构造 AppItem
   ├─ L127-132 PHP 类型: 不满足系统版本要求则跳过
   ├─ L133-143 构造 AppItem { ID, Name, Key, Limit, GpuSupport, Recommend, Description (i18n), Type, BatchInstallSupport }
   ├─ L145-151 调 getAppTags(ap.ID, lang) 取标签名数组
   └─ L152-166 判断 Installed
       ├─ Runtime 类 (PHP/Go/Node/Python/Java/.NET): runtimeRepo.List(WithDetailIdsIn) → 关联 app_detail
       └─ 其他类: appInstallRepo.ListBy(WithAppId) → app_installs 表

[Step 5] L168-169 封装 res.Items + res.Total
```

#### 6.2.4 Callees
- `appRepo.OrderByRecommend / WithByLikeName / WithType / GetRecommend / WithResource / WithArch / WithByIDs / Page` (repo/app.go)
- `tagRepo.GetByKeys / All` (repo/tag.go)
- `appTagRepo.GetByTagIds / DeleteByAppIds / BatchCreate` (repo/app_tag.go)
- `appDetailRepo.GetBy / WithAppId` (repo/app_detail.go)
- `appInstallRepo.ListBy / WithAppId` (repo/app_install.go)
- `runtimeRepo.List / WithDetailIdsIn` (repo/runtime.go)
- `NewIDashboardService().LoadOsInfo` (service/dashboard.go)
- `NewISettingService().GetSettingInfo` (service/setting.go)
- `common.GetLang(ctx)` (utils/common) — 解析 `Accept-Language`
- `common.CompareAppVersion` (utils/common) — 版本号比较
- `getAppTags(appID, lang)` (service/app.go 同文件内私有函数)
- `app.GetDescription(ctx)` (model/app.go:69) — JSON 多语言

---

### 6.3 ★★★ `GetAppDetail` (service/app.go:234)

**File**: `service/app.go:234-333`
**Lines**: 99 行
**Type**: `func (a AppService) GetAppDetail(appID uint, version, appType string) (response.AppDetailDTO, error)`

#### 6.3.1 Purpose
**应用详情页主入口**。前端在用户点击某个应用卡片后调它, 拿到完整的安装表单参数 (`Params` map) + Docker Compose 模板 + 是否支持当前架构 (`Enable` 字段) + 是否需要 Host 模式 (`HostMode`)。`appType` 区分 `runtime` (本地解释型, 如 PHP) 和 普通 (远程容器型, 如 MySQL), 走两条不同分支。

#### 6.3.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `appID` | `uint` | `apps.id` 主键 |
| `version` | `string` | 版本号, 如 "8.0" / "latest" |
| `appType` | `string` | `runtime` / `php` / `node` 等, 影响参数解析方式 |

**Returns**: `response.AppDetailDTO` + `error`。

#### 6.3.3 Flow (5 步)

```
[Step 1] L239-243 appDetailRepo.GetFirst(WithAppId, WithVersion)
   └─ 查 app_details 表, 找不到返空 DTO + 错

[Step 2] L247-293 runtime 分支 (本地解释型, 如 PHP/Go/Node)
   ├─ L248-251 appRepo.GetFirst(WithByID(appID))  // 取商品主表
   ├─ L252-259 files.NewFileOp() + 路径拼接 versionPath
   │   └─ 不存在 或 detail.Update=true → downloadApp() 拉资源到 /opt/1panel/resource/local_app
   ├─ L260-292 switch app.Type:
   │   └─ RuntimePHP:
   │       ├─ 读 {versionPath}/data.yml → paramMap → Params = paramMap["additionalProperties"]
   │       └─ 读 {versionPath}/docker-compose.yml → composeMap → Image = 第一个 service key
   └─ (其他 Runtime 类型暂无特殊处理)

[Step 3] L294-300 非 runtime 分支 (远程容器型)
   └─ json.Unmarshal(detail.Params) → paramMap → appDetailDTO.Params

[Step 4] L302-318 DockerCompose 自动下载 (如果 detail.DockerCompose 为空)
   ├─ L303 拼 URL: {DownloadUrl 去文件名} + docker-compose.yml
   ├─ L305 HTTP GET (超时 20s)
   ├─ L306-308 404 → ErrAppVersionUnavailable
   ├─ L309-311 网络错 → ErrGetCompose
   ├─ L312-314 状态码 > 200 → ErrGetCompose
   └─ L315-317 写库 + 缓存到 DTO

[Step 5] L320-331 后处理
   ├─ L320 isHostModel(DockerCompose) → HostMode (检测 network_mode: host)
   ├─ L322-325 取 app 主表
   ├─ L326-328 checkLimit(app) → Enable=false (超过安装数限制)
   └─ L329-331 DTO 附加字段: Architectures / MemoryRequired / GpuSupport
```

#### 6.3.4 Callees
- `appDetailRepo.GetFirst / WithAppId / WithVersion / Update` (repo/app_detail.go)
- `appRepo.GetFirst / WithByID` (repo/app.go)
- `files.NewFileOp().Stat / GetContent` (utils/files)
- `downloadApp(app, detail, nil, nil)` (service/app_install.go, 拉本地资源)
- `req_helper.HandleRequest` (utils/req_helper, HTTP 拉远程 compose)
- `isHostModel(compose)` (utils/app_helpers, 解析 YAML 检测 `network_mode: host`)
- `checkLimit(app)` (utils/app_helpers, 校验已装数 < limit)
- `yaml.Unmarshal` (gopkg.in/yaml.v3)
- `buserr.WithDetail / New` (agent/buserr, 业务错误包装)

---

### 6.4 ★★★ `SyncAppListFromLocal` (service/app.go:604)

**File**: `service/app.go:604-834`
**Lines**: 230 行
**Type**: `func (a AppService) SyncAppListFromLocal(TaskID string)`

#### 6.4.1 Purpose
**本地应用扫描同步**。扫描 `/opt/1panel/resource/local_app/` 目录, 读取每个 app 子目录的 `app.yaml` / `<version>/data.yml` / `<version>/docker-compose.yml`, 与 `apps` 表里 `resource='local'` 的旧记录做 diff, 新增 / 更新 / 软删除 (`status='TakeDown'`)。整个流程跑在 `task` 异步子任务里, 前端用 `TaskID` 轮询进度。

#### 6.4.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `TaskID` | `string` | 前端给的任务 ID, 用于前端轮询任务进度 |

**Returns**: 无 (异步执行)。

#### 6.4.3 Flow (4 步)

```
[Step 1] L611-615 创建 sync task
   └─ task.NewTaskWithOps("LocalApp", TaskSync, TaskScopeAppStore, TaskID, 0)

[Step 2] L617-830 主子任务闭包
   ├─ L618-626 扫描 LocalAppResourceDir, 不存在直接 return nil
   │
   ├─ L627-662 遍历每个 app 子目录
   │   ├─ handleLocalApp(appDir) → *model.App  // 解析 app.yaml
   │   └─ 遍历 <version>/ 子目录 → handleLocalAppDetail(versionDir, &detail)
   │       └─ 解析 docker-compose.yml / data.yml → 完整 AppDetail
   │
   ├─ L664-676 准备 5 个 bucket:
   │   newApps / deleteApps / updateApps / oldAppIds
   │   deleteAppIds / deleteAppDetails / newAppDetails / updateDetails
   │   appTags (多对多)
   │
   ├─ L678-683 加载旧 local apps → map[key]App, 全部标 TakeDown (软删)
   │
   ├─ L684-709 diff 循环
   │   ├─ 旧存在: 复用 ID, 复用 AppDetail ID (按 version 匹配)
   │   ├─ 旧不存在: ID=0, 标记新增
   │   └─ TagsKey 追加 "local" 标签
   │
   ├─ L711-730 分类
   │   ├─ ID=0 → newApps
   │   ├─ ID!=0 && Status=TakeDown (旧标软删) && 还有人装 → updateApps (重激活)
   │   ├─ ID!=0 && Status=TakeDown && 没人装 → deleteApps (真删)
   │   └─ ID!=0 && Status=Normal → updateApps
   │
   ├─ L732-736 加载 tag 字典 → tagMap[key]ID
   │
   └─ L738-827 事务 (defer tx.Rollback)
       ├─ L740-744 BatchCreate newApps
       ├─ L745-749 Save updateApps
       ├─ L750-757 BatchDelete deleteApps + DeleteByAppIds
       ├─ L759-761 DeleteByAppIds(oldAppIds) 删旧 tag 关联
       ├─ L762-783 准备 newAppDetails / updateDetails / deleteAppDetails
       ├─ L785-796 准备 appTags (按 TagsKey 关联到 tagMap)
       ├─ L798-802 BatchCreate newAppDetails
       ├─ L804-808 Update 变更 details
       ├─ L810-814 BatchDelete 删除的 details
       ├─ L816-820 DeleteByAppIds(oldAppIds) 再清一次 tag 关联
       └─ L822-826 BatchCreate appTags
       → L827 tx.Commit()

[Step 3] L831-833 go syncTask.Execute() 异步跑
```

#### 6.4.4 Callees
- `task.NewTaskWithOps / AddSubTask / Execute` (app/task)
- `i18n.GetMsgByKey / GetWithNameAndErr / GetWithName / GetMsgWithMap` (agent/i18n)
- `files.NewFileOp().Stat` (utils/files)
- `os.ReadDir` (标准库)
- `handleLocalApp(appDir)` (service/app.go 私有, 解析 app.yaml)
- `handleLocalAppDetail(versionDir, &detail)` (service/app.go 私有, 解析 data.yml + compose)
- `appRepo.GetBy / WithResource / BatchCreate / Save / BatchDelete` (repo/app.go)
- `appDetailRepo.DeleteByAppIds / BatchCreate / Update / BatchDelete` (repo/app_detail.go)
- `appTagRepo.DeleteByAppIds / BatchCreate` (repo/app_tag.go)
- `tagRepo.All` (repo/tag.go)
- `getTxAndContext()` (repo/common.go, 拿事务 ctx)
- `global.LOG.Infof / Errorf` (agent/global)

---

### 6.5 ★★★ `SyncAppListFromRemote` (service/app.go:976)

**File**: `service/app.go:976-1029`
**Lines**: 53 行 (本函数本身; 实际工作委托给 `createSyncAppStoreTask` / `createSyncAppStoreMetaTask` 在 `service/app_sync_task.go`)
**Type**: `func (a AppService) SyncAppListFromRemote(taskID string) (err error)`

#### 6.5.1 Purpose
**远端应用商店同步入口**。`POST /api/v2/apps/sync/remote` 调它, 它 ① 加锁防并发 ② 检查 xpack 多节点是否禁用 ③ 创建 task + 2 个子任务 (商店目录同步 + 应用详情元数据同步) ④ 异步执行 ⑤ defer 解锁 + recover 写 status。本函数 90% 是"流程编排", 真正的下载/解析/写库在 `service/app_sync_task.go` 16 KB 代码里。

#### 6.5.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `taskID` | `string` | 前端给的任务 ID |

**Returns**: `error` (nil = 启动成功; 真正结果在 task 里)。

#### 6.5.3 Flow (7 步)

```
[Step 1] L977-979 xpack.MultiNodeProvider.IsUseCustomApp()
   └─ 多节点用自定义 app 仓库, 跳过官方同步

[Step 2] L981-989 互斥锁 appStoreSyncMu
   ├─ appStoreSyncing=true → 已在同步, return nil (前端会显示 "syncing")
   └─ 否则设 true, 解锁, 进入 Step 3

[Step 3] L991-997 创建 task
   └─ task.NewTaskWithOps("App", TaskSync, TaskScopeAppStore, taskID, 0)
   └─ 失败 → 解锁, return err

[Step 4] L999 声明 *appSyncContext (在两个子任务间共享数据)

[Step 5] L1001-1002 添加 2 个子任务
   ├─ createSyncAppStoreTask(&sharedCtx)      // 同步 1panel.json + 解析 + 写 App 主表
   └─ createSyncAppStoreMetaTask(&sharedCtx)   // 同步每个 app 的 icon + 多版本详情

[Step 6] L1004-1025 go func() defer
   ├─ L1006-1011 recover() → 写 AppStoreSyncStatus = Error
   ├─ L1012-1014 defer 解锁 appStoreSyncing = false
   ├─ L1016-1024 syncTask.Execute() 失败 → 重置 AppStoreLastModified=0 + 写 Error status
   └─ L1027 日志 "sync app from remote task create ok"
```

#### 6.5.4 Callees
- `xpack.MultiNodeProvider.IsUseCustomApp()` (utils/xpack, 商业版接口)
- `sync.Mutex appStoreSyncMu` (本文件顶部 L40 全局)
- `task.NewTaskWithOps / AddSubTask / Execute` (app/task)
- `i18n.GetMsgByKey / GetMsgByKey("App")` (i18n, 拿翻译)
- `a.createSyncAppStoreTask(&sharedCtx)` (service/app_sync_task.go, 16 KB)
- `a.createSyncAppStoreMetaTask(&sharedCtx)` (service/app_sync_task.go)
- `NewISettingService().Update(key, value)` (service/setting.go, 写 AppStoreSyncStatus / AppStoreLastModified)
- `global.LOG.Info / Errorf` (agent/global)

---

### 6.6 ★★ `GetApp` (service/app.go:196)

**File**: `service/app.go:196-218`
**Lines**: 22 行
**Type**: `func (a AppService) GetApp(ctx *gin.Context, key string) (*response.AppDTO, error)`

#### 6.6.1 Purpose
**单个应用基础信息查询**, `GET /api/v2/apps/:key` 用。比 `PageApp` 简单: 不分页、不筛选, 仅按 `key` 查一条。返回 `AppDTO` 含 App 全字段 + `Versions []string` (去重后的版本列表) + `Tags []TagDTO` + 已装状态由前端调 `app_installs` 端点二次获取。

#### 6.6.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `ctx` | `*gin.Context` | Gin 上下文, 取 i18n |
| `key` | `string` | 应用 key, 如 "wordpress"; 特殊映射 "postgres" → "postgresql" |

**Returns**: `*response.AppDTO` (内嵌 `model.App` + `Versions` + `Tags`)。

#### 6.6.3 Flow (4 步)

```
[Step 1] L198-200 key 兼容: "postgres" → "postgresql" (1Panel 老用户习惯)
[Step 2] L201-204 appRepo.GetFirst(WithKey(key))  // 查商品主表
[Step 3] L206  i18n: app.Description = app.GetDescription(ctx)
[Step 4] L207-216 加载详情
   ├─ L207-210 appDetailRepo.GetBy(WithAppId(app.ID)) → []AppDetail
   ├─ L211 getAppVersions(key, details) → 去重版本字符串数组
   ├─ L212-215 getAppTags(app.ID, lang) → []TagDTO
   └─ L216 appDTO.Tags = tags
```

#### 6.6.4 Callees
- `appRepo.GetFirst / WithKey` (repo/app.go)
- `appDetailRepo.GetBy / WithAppId` (repo/app_detail.go)
- `app.GetDescription(ctx)` (model/app.go:69, i18n 解析)
- `getAppVersions(key, details)` (service/app.go 私有, 去重排序)
- `getAppTags(appID, lang)` (service/app.go 私有, JOIN 多对多表 + i18n)

---

### 6.7 ★★ `GetAppUpdate` (service/app.go:836)

**File**: `service/app.go:836-896`
**Lines**: 60 行
**Type**: `func (a AppService) GetAppUpdate() (*response.AppUpdateRes, error)`

#### 6.7.1 Purpose
**远端商店升级检测**。`GET /api/v2/apps/checkupdate` 调它, 返回: `CanUpdate` (是否要同步) / `IsSyncing` (是否正在同步) / `AppStoreLastModified` (本地记录的 lastModified) / `AppList` (完整商店目录, 前端会用作"还没同步就显示预览")。**比 `GetAppUpdate` 这个名字更深层的逻辑是: 它在判断"前端该不该弹'有新版, 要不要同步'的弹窗"**。

#### 6.7.2 Params
无。

**Returns**: `*response.AppUpdateRes` + `error`。

#### 6.7.3 Flow (8 步)

```
[Step 1] L840-844 mysql 旧版兼容: 如果 mysql 的 BatchInstallSupport=true (即新版本)
   └─ 直接 CanUpdate=true, return (前端会显示 "请升级 1Panel 后同步")

[Step 2] L846-850 拉远端版本号
   └─ HTTP GET {AppRepoURL}/{mode}/1panel.json.version.txt → lastModifiedStr

[Step 3] L852-855 strconv.Atoi → lastModified (int 时间戳)

[Step 4] L856-859 取本地 setting.AppStoreSyncStatus
   └─ 如果是 "Syncing" → IsSyncing=true, return (前端显示"同步中")

[Step 5] L865-866 记录本地 lastModified

[Step 6] L867-870 远端 != 本地 → CanUpdate=true, return

[Step 7] L871-884 检查 icon 完整性
   └─ 遍历所有 remote apps, 如果 Icon=="" 或 icon 文件不存在 → CanUpdate=true, return
   └─ 目的: 之前 sync 时 icon 漏了, 现在补

[Step 8] L886-894 完整拉一次
   ├─ getAppList() → 走 getAppFromRepo 下 1panel.json.zip → 解压 → JSON decode
   ├─ L890 版本要求检查: 1Panel.SystemVersion >= list.Extra.Version
   │   └─ 不够新 → ErrVersionTooLow (前端提示升级 1Panel)
   └─ L894 res.AppList = list
```

#### 6.7.4 Callees
- `appRepo.GetFirst / WithKey / GetBy / WithResource` (repo/app.go)
- `appicon.IsIconFile / ParseIconField / IconFileExists` (utils/appicon)
- `req_helper.HandleRequest` (utils/req_helper, HTTP 拉 version.txt / json.zip)
- `getAppList()` (service/app.go:916, 内部函数)
- `getAppFromRepo(downloadPath)` (service/app.go:898, 下载+解压)
- `NewISettingService().GetSettingInfo` (service/setting.go)
- `common.CompareVersion` (utils/common, 语义化版本比较)
- `global.LOG.Errorf` (agent/global)

---

### 6.8 ★ `GetAppIcon` (service/app.go:1031)

**File**: `service/app.go:1031-1053`
**Lines**: 22 行
**Type**: `func (a AppService) GetAppIcon(key string) ([]byte, string, string, error)`

#### 6.8.1 Purpose
**应用图标 HTTP 服务**。`GET /api/v2/apps/icon/:key` 调它, 返回 icon 二进制 + filename + etag。**新版 icon 走本地文件** (SyncAppListFromRemote 拉到 `/opt/1panel/resource/app_icon/`), **老版 icon 走 base64 字符串** (存在 `apps.icon` 字段)。Handler (`api/v2/app.go:213`) 会进一步设 ETag 头 + 比对 `If-None-Match` 返回 304。

#### 6.8.2 Params
| 名称 | 类型 | 说明 |
|------|------|------|
| `key` | `string` | 应用 key |

**Returns**: `(iconBytes []byte, filename string, etag string, err error)`。

#### 6.8.3 Flow (3 步)

```
[Step 1] L1032-1035 appRepo.GetFirst(WithKey(key)) 取 App.Icon 字段

[Step 2] L1037-1045 新版 (Icon 以 "icon:" 开头, 走文件)
   ├─ appicon.ParseIconField(Icon) → (fileName, etag)
   ├─ appicon.ReadIconFile(fileName) → []byte
   └─ 失败不报错, 返 nil (前端用默认 icon 占位)

[Step 3] L1047-1052 老版 (Icon 是 base64 字符串)
   └─ base64.StdEncoding.DecodeString(Icon) → []byte
   └─ 失败不报错, 返 nil
```

#### 6.8.4 Callees
- `appRepo.GetFirst / WithKey` (repo/app.go)
- `appicon.IsIconFile / ParseIconField / ReadIconFile` (utils/appicon, 自家工具)
- `base64.StdEncoding.DecodeString` (标准库)
- `global.LOG.Warnf` (agent/global)

---

### 6.9 私有辅助函数

| 行号 | 名称 | 作用 | 优先级 |
|------|------|------|--------|
| L898-914 | `getAppFromRepo(downloadPath)` | 下载 1panel.json.zip 到 `ResourceDir` + 用 `SdkZip` 解压, 删 zip | ★ |
| L916-932 | `getAppList()` | `getAppFromRepo` + `os.Open` 1panel.json + `json.NewDecoder.Decode` → `*dto.AppList` | ★★ |
| L940-974 | `deleteCustomApp()` | 删 custom app 中"无人安装"的孤儿 (维护清理) | ★ |
| L934-938 | `var InitTypes = map[string]struct{}{...}` | runtime/php/node 三种 type 集合, 区分参数解析方式 | - |
| L61-63 | `type appInstallHooks struct { AfterCopyData func(...) }` | Install 钩子扩展点, **本流程不传, 升级流程传** | - |
| `getAppVersions` | 去重排序版本号 | - |
| `getAppTags` | 多对多 JOIN + i18n | - |
| `isHostModel` | YAML 解析检测 `network_mode: host` | - |
| `checkPort` / `checkContainerNameIsExist` / `checkRequiredAndLimit` / `addDockerComposeCommonParam` | 在 `service/app_utils.go` | - |
| `createLink` / `copyData` / `runScript` / `upApp` / `updateToolApp` / `handleSiteDir` / `handleOpenrestyFile` | 安装子任务实现在 `service/app_utils.go` (70 KB) | - |

### 6.10 其他公开 func 简表 (★ 一级, 工具/单行)

| 行号 | 函数 | 作用 | 优先级 |
|------|------|------|--------|
| L174-194 | `GetAppTags(ctx)` | 取所有 tag 字典, i18n 解析 | ★ |
| L220-232 | `GetAppDetailByKey(appKey, version)` | 按 key+version 查 detail ID, **节点用** (`/apps/detail/node/:appKey/:version`) | ★★ |
| L334-348 | `GetAppDetailByID(id)` | 按 detail.id 查, 已装应用编辑模式用 | ★ |

### 6.11 service/app_install.go 旁支 func 简表 (本 KB 范围外, 列名供导航)

| 行号 | 函数 | 作用 |
|------|------|------|
| L67 | `GetInstallList()` | 全部已装 app 列表 (轻量, 仅 ID+Key+Name) |
| L79 | `Page(req)` | 已装 app 分页, 含版本/icon/canUpdate 状态 |
| L133 | `CheckExist(req)` | 检测"该 key+name 是否已装" |
| L177 | `LoadPort(req)` | 加载已装 app 的端口 |
| L185 | `LoadConnInfo(req)` | 加载数据库连接信息 (mysql/redis 凭据) |
| L200 | `SearchForWebsite(req)` | 给 Website 模块用的精简查询 |
| L246 | `Operate(req)` | 启动/停止/重启/卸载, **含异步 task** |
| L315 | `UpdateAppConfig(req)` | 修改 WebUI 字段 |
| L327 | `UpdateSort(req)` | 拖拽排序 |
| L336 | `Update(req)` | 改已装 app 的环境变量 / 端口 |
| L472 | `SyncAll(systemInit)` | 启动时同步所有已装 app 真实状态到 DB |
| L502 | `GetServices(key)` | 列出某 app 的所有 compose service 状态 |
| L571 | `GetUpdateVersions(req)` | 检测可升级版本列表 |
| L639 | `ChangeAppPort(req)` | 改已装 app 端口 |
| L667 | `DeleteCheck(installID)` | 卸载前检查: 哪些资源会被删 |
| L695 | `GetDefaultConfigByKey(key, name)` | 取默认 .env 模板 |
| L726 | `GetParams(id)` | 取已装 app 当前参数 (编辑用) |
| L838 | `syncAppInstallStatus(install, force)` | 同步单条 app 状态 (内部) |
| L865 | `SyncAppInstallStatus(install, force)` | 公开包装, 跨模块调用 |
| L961 | `GetAppInstallInfo(installID)` | 详情页 (含 env map) |

### 6.12 service/app_sync_task.go 旁支 func 简表 (本 KB 范围外)

`createSyncAppStoreTask` (大) + `createSyncAppStoreMetaTask` (大) — 真正的远端 sync 工作在这两个函数里, 本 KB 范围的 `SyncAppListFromRemote` 仅是流程编排。详细讲解见后续 v2 KB 补完。

### 6.13 api/v2/app.go Handler 简表 (★ 全部 11 个)

| 行号 | Handler | HTTP 路径 | 调 Service |
|------|---------|-----------|-----------|
| L22 | `SearchApp` | POST /apps/search | `PageApp` |
| L42 | `SyncApp` | POST /apps/sync/remote | `GetAppUpdate` → `SyncAppListFromRemote` |
| L74 | `SyncLocalApp` | POST /apps/sync/local | `SyncAppListFromLocal` (go) |
| L91 | `GetApp` | GET /apps/:key | `GetApp` |
| L115 | `GetAppDetail` | GET /apps/detail/:appId/:version/:type | `GetAppDetail` |
| L139 | `GetAppDetailByID` | GET /apps/details/:id | `GetAppDetailByID` |
| L162 | `InstallApp` | POST /apps/install | `Install` |
| L181 | `GetAppTags` | GET /apps/tags | `GetAppTags` |
| L196 | `GetAppListUpdate` | GET /apps/checkupdate | `GetAppUpdate` |
| L213 | `GetAppIcon` | GET /apps/icon/:key | `GetAppIcon` (含 ETag/304) |
| L252 | `GetAppDetailForNode` | GET /apps/detail/node/:appKey/:version | `GetAppDetailByKey` |

### 6.14 repo/app.go Func 简表 (12 DBOption + 9 CRUD)

| 行号 | 名称 | 类型 | 作用 |
|------|------|------|------|
| L44-51 | `WithByLikeName(name)` | DBOption | name/desc/short_desc_zh/en LIKE 模糊 |
| L53-57 | `WithKey(key)` | DBOption | WHERE key=? |
| L59-63 | `WithKeyIn(keys)` | DBOption | WHERE key IN (?,?) |
| L65-69 | `WithType(typeStr)` | DBOption | WHERE type=? |
| L71-75 | `OrderByRecommend()` | DBOption | ORDER BY recommend ASC |
| L77-81 | `GetRecommend()` | DBOption | WHERE recommend<9999 |
| L83-87 | `WithResource(resource)` | DBOption | WHERE resource=? |
| L89-93 | `WithNotLocal()` | DBOption | WHERE resource!='local' |
| L95-99 | `WithArch(arch)` | DBOption | WHERE architectures LIKE %arch% |
| L101-105 | `WithPanelVersion(ver)` | DBOption | WHERE required_panel_version >= ? |
| L107-114 | `Page(page, size, opts...)` | CRUD | COUNT + 分页 + Preload AppTags |
| L116-123 | `GetFirst(opts...)` | CRUD | 单条 + Preload AppTags |
| L125-132 | `GetBy(opts...)` | CRUD | 多条 + Preload Details + AppTags |
| L134-147 | `GetTopRecommend()` | CRUD | 按 recommend 排序取前 6 个 key |
| L149-151 | `BatchCreate(ctx, apps)` | CRUD | 批量插入, omit associations |
| L153-155 | `Create(ctx, app)` | CRUD | 插入单条 |
| L157-159 | `Save(ctx, app)` | CRUD | 保存 (upsert 语义) |
| L161-163 | `BatchDelete(ctx, apps)` | CRUD | 批量删除, omit associations |
| L165-167 | `DeleteByIDs(ctx, ids)` | CRUD | WHERE id IN |
| L169-170 | `DeleteBy(opts...)` | CRUD | 按 DBOption 删 |

### 6.15 repo/app_detail.go Func 简表

| 行号 | 名称 | 作用 |
|------|------|------|
| L32-36 | `WithVersion(version)` | WHERE version=? |
| L38-42 | `WithAppId(id)` | WHERE app_id=? |
| L44-48 | `WithIgnored()` | WHERE ignore_upgrade=1 |
| L50-54 | `GetFirst(opts...)` | 单条 (用 Find 而非 First) |
| L56-58 | `Update(ctx, detail)` | Save |
| L60-62 | `BatchCreate(ctx, details)` | 批量插入 |
| L64-66 | `DeleteByAppIds(ctx, ids)` | WHERE app_id IN |
| L68-70 | `DeleteByIDs(ctx, ids)` | WHERE id IN |
| L72-76 | `GetBy(opts...)` | 多条 |
| L78-84 | `BatchUpdateBy(maps, opts...)` | 批量按条件更新 |
| L86-88 | `BatchDelete(ctx, details)` | 批量删 |

---

## 7. 类与结构体讲解

### 7.1 `AppService` (service/app.go:44)

```go
type AppService struct{}
```

**空结构体**, 通过 `NewIAppService()` 工厂返回, 实现 `IAppService` 接口 (L47-59)。所有方法都是值接收器 `(a AppService)`, 不持有状态; 全局状态用 `appStoreSyncMu` / `appStoreSyncing` (L40-42) 模拟。

| 方法 | 行号 | 优先级 |
|------|------|--------|
| `PageApp` | 69 | ★★★ |
| `GetAppTags` | 174 | ★ |
| `GetApp` | 196 | ★★ |
| `GetAppDetailByKey` | 220 | ★★ |
| `GetAppDetail` | 234 | ★★★ |
| `GetAppDetailByID` | 334 | ★ |
| `Install` (→ installWithHooks) | 350 | ★★★★★ |
| `SyncAppListFromLocal` | 604 | ★★★ |
| `GetAppUpdate` | 836 | ★★ |
| `SyncAppListFromRemote` | 976 | ★★★ |
| `GetAppIcon` | 1031 | ★ |

### 7.2 `model.App` (model/app.go:13) — 22 字段

> 已在 §5.1 全字段表, 此处补方法:

| 方法 | 行号 | 作用 |
|------|------|------|
| `IsLocalApp()` | 44 | Resource=="local" 判定 |
| `IsCustomApp()` | 47 | Resource==AppResourceCustom 判定 |
| `GetAppResourcePath()` | 51 | 返回 `/opt/1panel/resource/{local_app,custom_app,remote_app}/<key>` |
| `GetDescription(ctx)` | 69 | 解析 `Description` JSON 字段, 按 Accept-Language 选 zh/en/... |

### 7.3 `model.AppDetail` (model/app_detail.go:3) — 11 字段

已在 §5.2 详述, **无方法**, 是纯数据载体。

### 7.4 `model.AppInstall` (model/app_install.go:11) — 19 字段

已在 §5.3 详述, 方法:

| 方法 | 行号 | 作用 |
|------|------|------|
| `GetPath()` | 34 | `/opt/1panel/apps/<key>/<name>/` |
| `GetComposePath()` | 38 | `<path>/docker-compose.yml` |
| `GetEnvPath()` | 42 | `<path>/.env` |
| `GetAppPath()` | 46 | `/opt/1panel/{local,apps}/<key>/` (local app 走 LocalAppInstallDir) |

### 7.5 `dto.AppList` / `AppDefine` / `AppProperty` (dto/app.go:51/60/87)

> 远端商店 `1panel.json` 的反序列化目标。

**AppList (L51-58)**: 商店目录总览, 字段:
| 字段 | 类型 | 含义 |
|------|------|------|
| `Valid` | bool | schema 校验通过 |
| `Violations` | []string | 校验失败信息 |
| `LastModified` | int | 整体时间戳, 用于 sync diff |
| `Apps` | []AppDefine | 应用列表 |
| `Extra` | ExtraProperties | 附加属性 (tags + version 要求) |

**AppDefine (L60-68)**: 单个应用, 字段:
| 字段 | 类型 | 含义 |
|------|------|------|
| `Icon` | string | base64 或 icon file 路径 |
| `Name` | string | 显示名 |
| `ReadMe` | string | Markdown |
| `LastModified` | int | 单 app 时间戳 |
| `AppProperty` | AppProperty | 商品元数据 (内嵌) |
| `Versions` | []AppConfigVersion | 多版本数组 |

**AppProperty (L87-108)**: 20 字段, 应用元数据核心:
| 字段 | 含义 |
|------|------|
| `Name` | 显示名 |
| `Type` | 类型: runtime / php / mysql / openresty / ... |
| `Tags` | 标签 key 列表 |
| `ShortDescZh` / `ShortDescEn` | 中/英短描述 |
| `Description` | Locale 多语言长描述 |
| `Key` | 唯一标识 |
| `Required` | 依赖声明 |
| `CrossVersionUpdate` | 跨版本升级允许 |
| `Limit` | 安装数上限 |
| `Recommend` | 推荐位权重 |
| `Website` / `Github` / `Document` | 链接 |
| `Architectures` | 支持的 CPU 架构 |
| `MemoryRequired` | 最低内存 (MB) |
| `GpuSupport` | 是否支持 GPU |
| `Version` | 商店 schema 版本 (校验用) |
| `Deprecated` | 废弃版本阈值 |
| `BatchInstallSupport` | mysql 1Panel 升级标记 |

### 7.6 `dto.AppForm` / `AppFormFields` (dto/app.go:140/145)

> 前端"安装表单"动态生成依据, `Install` 阶段把 `Params` map 的每个 key 渲染成输入框。

**AppForm**: 含 `FormFields []AppFormFields` + `SupportVersion float64`。

**AppFormFields (L145-160)**: 14 字段, 决定一个输入框:
| 字段 | 含义 |
|------|------|
| `Type` | text / number / password / select / radio / checkbox |
| `LabelZh` / `LabelEn` / `Label` (Locale) | 标签 |
| `Description` (Locale) | 帮助文字 |
| `Required` | 是否必填 |
| `Default` | 默认值 |
| `EnvKey` | 注入到 docker-compose 的环境变量名 |
| `Disabled` / `Edit` | UI 状态 |
| `Rule` | 校验规则 (端口范围/正则) |
| `Multiple` | 多选 (select 时) |
| `Child` | 嵌套表单 |
| `Values` | 选项 (radio/select 时 `[]AppFormValue{Label, Value}`) |

### 7.7 `dto.AppDatabase` / `AuthParam` (dto/app.go:10/20)

> MySQL/Postgres/Mongo/Redis 等数据库类应用的环境变量规范, 跟 1Panel Database 模块的"远程数据库"联动。

**AppDatabase (L10-18)**: 7 字段, DB 客户端连接信息:
| 字段 | JSON key | 含义 |
|------|----------|------|
| `ServiceName` | `PANEL_DB_HOST` | DB service 名 (引用 1panel Database) |
| `DbName` | `PANEL_DB_NAME` | 库名 |
| `DbUser` | `PANEL_DB_USER` | 用户 |
| `Password` | `PANEL_DB_USER_PASSWORD` | 密码 |
| `DatabaseName` | `DATABASE_NAME` | 默认数据库 |
| `Format` / `Collation` | - | 字符集/排序 (mysql 用) |

**AuthParam (L20-23)**: DB root 凭据, `PANEL_DB_ROOT_PASSWORD` / `PANEL_DB_ROOT_USER`。

### 7.8 `dto.Locale` (dto/app.go:125)

13 字段多语言字典, 字段: `En` / `Ja` / `Ms` / `PtBr` / `Ru` / `ZhHant` / `Zh` / `Ko` / `Tr` / `Es` / `Fa` / `Lo` (老挝) + 反序列化用 `yaml:"zh-hant"` 风格。

### 7.9 `request.AppSearch` / `AppInstallCreate` / `AppContainerConfig` (dto/request/app.go)

**AppSearch (L9-17)**: PageApp 入参, 8 字段, 见 §6.2。

**AppInstallCreate (L19-28)**: Install 入参, 4 字段 + 嵌入 `AppContainerConfig` + `NodePushConfig`。
| 字段 | 含义 |
|------|------|
| `AppDetailId` | 要装的版本 detail.id |
| `Params` | 表单参数 (key→任意类型) |
| `Name` | 实例名 (UNIQUE) |
| `Services` | 多 service 映射 (mysql + phpmyadmin 一起装时) |
| `TaskID` | 前端轮询用 |

**AppContainerConfig (L37-53)**: 高级选项, 15 字段:
| 字段 | 含义 |
|------|------|
| `Advanced` | 是否启用高级模式 |
| `CpuQuota` | CPU 限制 (核数) |
| `MemoryLimit` / `MemoryUnit` | 内存限制 (MB/GB) |
| `ContainerName` | 自定义容器名 (Advanced 时) |
| `AllowPort` | 是否自动放行防火墙端口 |
| `EditCompose` | 是否手动编辑 compose |
| `DockerCompose` | 手动编辑的 compose 全文 |
| `HostMode` | 主机网络模式 |
| `PullImage` | 是否重新拉镜像 |
| `GpuConfig` | GPU 配置 |
| `WebUI` | 应用 Web 入口 |
| `Type` | 应用类型 |
| `SpecifyIP` | 指定 IP |
| `RestartPolicy` | always/unless-stopped/no/on-failure |

---

## 8. Mermaid 时序图 (2 个核心流程)

### 8.1 InstallApp 全流程时序图

```mermaid
sequenceDiagram
    autonumber
    participant FE as 前端
    participant MW as Middleware
    participant H as InstallApp (api/v2/app.go:162)
    participant S1 as Install (app.go:350)
    participant S2 as installWithHooks (app.go:354)
    participant R as Repo (GORM)
    participant TK as task
    participant DK as Docker

    FE->>MW: POST /api/v2/apps/install {appDetailId, name, params, ...}
    MW->>H: CheckBindAndValidate(req)
    H->>S1: appService.Install(req, true)
    S1->>S2: installWithHooks(req, true, nil)
    Note over S2: ─── 阶段 1: 校验 ───
    S2->>DK: CreateDefaultDockerNetwork()
    S2->>R: appInstallRepo.ListBy(WithByLowerName)
    R-->>S2: 已有同名 → ErrAppNameExist
    S2->>R: appDetailRepo.GetFirst(WithByID)
    R-->>S2: appDetail
    S2->>R: appRepo.GetFirst(WithByID)
    R-->>S2: app
    S2->>R: databaseRepo 检查 DB 依赖
    S2->>S2: checkPort() / checkRequiredAndLimit()
    Note over S2: ─── 阶段 2: 渲染 ───
    S2->>S2: yaml.Unmarshal(detail.DockerCompose)
    S2->>S2: addDockerComposeCommonParam() 注入 cpu/mem/port
    S2->>S2: yaml.Marshal → appInstall.DockerCompose
    S2->>S2: json.Marshal(params) → appInstall.Env
    Note over S2: ─── 阶段 3: 写库 ───
    S2->>R: appInstallRepo.Create(appInstall)
    R-->>S2: appInstall.ID
    Note over S2: ─── 阶段 4: 异步任务 ───
    S2->>TK: NewTaskWithOps(name, Install, App, taskID, installID)
    S2->>S2: createLink() 写 .env / hosts
    S2->>TK: AddSubTaskWithOps(installApp, 1h timeout)
    S2-->>H: return *appInstall, nil
    H-->>FE: 200 OK {ID, Name, ...}
    par 异步执行 (L590 go func)
        TK->>S2: Execute() → installApp 闭包
        S2->>S2: copyData() 拷数据
        S2->>S2: runScript() init.sh
        S2->>DK: upApp() docker compose up -d
        DK-->>S2: container running
        S2->>R: appInstallRepo.Save(Status=Running)
        S2->>S2: updateToolApp() 联动 phpmyadmin
    end
```

### 8.2 SyncAppListFromRemote 全流程时序图

```mermaid
sequenceDiagram
    autonumber
    participant FE as 前端
    participant H as SyncApp (api/v2/app.go:42)
    participant CK as GetAppUpdate (app.go:836)
    participant SR as SyncAppListFromRemote (app.go:976)
    participant S1 as createSyncAppStoreTask
    participant S2 as createSyncAppStoreMetaTask
    participant TK as task
    participant NET as HTTP
    participant R as Repo

    FE->>H: POST /api/v2/apps/sync/remote {taskID}
    H->>CK: appService.GetAppUpdate()
    CK->>NET: GET 1panel.json.version.txt
    NET-->>CK: lastModified=1717829300
    CK->>R: setting.Get(AppStoreLastModified)
    CK-->>H: {CanUpdate: true, AppList: [...]}
    H->>SR: appService.SyncAppListFromRemote(taskID)
    SR->>SR: appStoreSyncMu.Lock()
    SR->>TK: NewTaskWithOps("App", Sync, AppStore, taskID, 0)
    SR->>S1: AddSubTask(createSyncAppStoreTask)
    SR->>S2: AddSubTask(createSyncAppStoreMetaTask)
    SR-->>H: return nil
    H-->>FE: 200 OK
    par 异步执行
        TK->>S1: Execute()
        S1->>NET: HTTP GET 1panel.json.zip (压缩目录)
        NET-->>S1: zip
        S1->>S1: 解压到 /opt/1panel/resource/1panel.json
        S1->>S1: json.NewDecoder.Decode → []AppDefine
        S1->>R: appRepo.BatchCreate / Save / Delete (事务)
        R-->>S1: 写入完成
        S1-->>TK: subtask done
        TK->>S2: Execute()
        S2->>NET: 拉每个 app 的 icon + docker-compose.yml
        NET-->>S2: 资源文件
        S2->>R: appDetailRepo.BatchCreate / Save
        S2->>R: setting.Update(AppStoreLastModified, now)
        S2-->>TK: subtask done
        TK->>SR: defer 解锁 appStoreSyncing=false
    end
```

---

## 9. Docker 相关依赖

### 9.1 强依赖 (Install 流程必须)

| 工具 | 包路径 | 作用 |
|------|--------|------|
| `docker.CreateDefaultDockerNetwork()` | `utils/docker` | 创建 `1panel-network` 桥接网络, 失败立即终止 Install |
| `docker compose up -d` | `utils/compose` (实现在 `service/app_utils.go` `upApp`) | 实际启动容器 |
| Docker SDK | `github.com/docker/docker/api/types/container` | 容器 inspect / 操作 |
| `docker.CreateDefaultDockerNetwork` | 同上 | 默认网络 |

### 9.2 弱依赖 (按需)

| 工具 | 触发条件 |
|------|---------|
| `downloadApp` | GetAppDetail 时本地资源缺失, HTTP 拉取 |
| `files.NewFileOp().Stat / GetContent / Decompress` | 读本地 compose / data.yml / 解压 zip |
| `req_helper.HandleRequest` | HTTP 拉远端 1panel.json.zip / version.txt / docker-compose.yml |

### 9.3 网络拓扑

```
[Host 物理机 / 虚拟机]
 ├─ 1panel-network (bridge, 1Panel 管理)
 │   ├─ App Container 1 (e.g. WordPress, 1Panel-wordpress-abcd)
 │   ├─ App Container 2 (e.g. MySQL, 1Panel-mysql-efgh)
 │   └─ ...
 └─ host network (mode=host, 特殊应用如 openresty 可选)
```

`addDockerComposeCommonParam` 根据 `req.Advanced` / `req.HostMode` 决定写入 `networks: [1panel-network]` 还是 `network_mode: host`。

---

## 10. 跟其他模块的关系

### 10.1 跟 Container 模块 (`02-container`)

- **App Install → Container 创建**: `upApp()` 调 `dockerService.ComposeUp()` 起容器, 实际是 Container 模块的子集。
- **App Delete → Container 删除**: `app_install.go: Operate(req.Operate=Uninstall)` 调 `containerService.Remove()`。
- **App Detail → Container Inspect**: `app_install.go: GetServices(key)` 调 `containerService.ListByLabel()` 拿容器状态。
- **共享依赖**: `1panel-network` (App 创建时建, Container 操作时引用)。

### 10.2 跟 Website 模块 (`03-website`)

- **OpenResty App Install → Website 反向注册**: 装 OpenResty 后, 1Panel 自动建一个"默认网站"指向 `WEBSITE_DIR`, 这是 Website 模块的入口。
- **WebUI 字段**: `AppInstall.WebUI` 由 Website 模块的 `domain` 解析后填回。
- **`SearchForWebsite(req)`**: 在 `app_install.go:200` 提供给 Website 用, 拿"已装 app 列表"做"反向代理"配置。

### 10.3 跟 Database 模块 (`04-database`)

- **DB 联动**: Install MySQL/Redis/Mongo/Postgres 时, 如果 `req.Params["PANEL_DB_HOST"]` 指向 1Panel 已有 DB, `installWithHooks` 自动把 `PANEL_DB_HOST` 替换成 DB 真实 IP+Port, 把 `DATABASE_NAME` 替换成库名 (L507-514)。
- **DB 健康检查**: 如果依赖 DB 容器 `Status != Running`, Install 返 `ErrAppIsDown` (L387-390)。
- **`DatabaseKeys[app.Key]` 白名单**: `mysql` / `redis` / `postgresql` / `mongodb` 走 DB 联动流程 (L377-382)。

### 10.4 跟 Task 模块 (`00-task`)

- **每个 Install/Upgrade/Uninstall/Sync 都是一个 task**, 跨多个子任务 (copyData / runScript / upApp), 通过 `task.TaskScopeApp` / `task.TaskScopeAppStore` 分类。
- **前端轮询**: `taskID` 从前端到 handler 到 service, 写到 `Task` 表, 前端 `GET /api/v2/tasks/:id` 拿进度。

### 10.5 跟 Setting 模块

- **`AppStoreLastModified`**: 记录上次同步的远端时间戳, `GetAppUpdate` 比对。
- **`AppStoreSyncStatus`**: `Syncing` / `Error` / `Success`, 前端弹窗"同步中"或"同步失败"。
- **`AppStoreUrl`**: 用户可改远端仓库 URL (社区镜像), 通过 `global.AppRepoURL()` 读。

### 10.6 跟 Snapshot / Backup 模块

- `AppInstall` 关联到 `backup` (单 app 备份), 通过 `app_install.go: GetUpdateVersions` 检测 snapshot 触发升级。

---

## 11. 关键文件清单

| 路径 | 行数 | 关键内容 |
|------|------|---------|
| `agent/app/api/v2/app.go` | 261 | 11 个核心 HTTP handler |
| `agent/app/api/v2/app_install.go` | 372 | 20 个已装应用 handler |
| `agent/app/api/v2/app_ignore_upgrade.go` | ~80 | 3 个忽略升级 handler |
| `agent/app/dto/app.go` | 194 | 商店 JSON / 表单 / DB / Locale DTO |
| `agent/app/dto/request/app.go` | 150 | 8 个请求 DTO (含 AppContainerConfig) |
| `agent/app/dto/response/app.go` | 189 | 11 个响应 DTO |
| `agent/app/model/app.go` | 83 | `App` + 4 个方法 |
| `agent/app/model/app_detail.go` | 15 | `AppDetail` |
| `agent/app/model/app_install.go` | 52 | `AppInstall` + 4 个 path 方法 |
| `agent/app/model/app_tag.go` | 7 | `AppTag` |
| `agent/app/model/app_ignore_upgrade.go` | 9 | `AppIgnoreUpgrade` |
| `agent/app/repo/app.go` | 171 | `AppRepo` |
| `agent/app/repo/app_detail.go` | 88 | `AppDetailRepo` |
| `agent/app/repo/app_install.go` | 270 | `AppInstallRepo` |
| `agent/app/repo/app_tag.go` | ~80 | 标签关联 |
| `agent/app/repo/app_ignore_upgrade.go` | ~50 | 忽略升级 |
| `agent/app/service/app.go` | **1055** | **★ 主战场**: 11 个公开方法 + 4 个私有辅助 |
| `agent/app/service/app_install.go` | 960 | 已装应用 CRUD (本 KB 范围外) |
| `agent/app/service/app_upgrade.go` | ~700 | 升级流程 (本 KB 范围外) |
| `agent/app/service/app_utils.go` | ~1900 | 安装子任务实现 (本 KB 范围外) |
| `agent/app/service/app_sync_task.go` | ~430 | 远端 sync 真实工作 (本 KB 范围外) |
| `agent/router/ro_app.go` | ~50 | 26 条路由 |
| `agent/init/router/router.go` | 31 | `Routers()` 挂载点 |
| `agent/init/migration/migrations/` | - | `apps` / `app_details` / `app_installs` 表 DDL |

---

## 12. 行号索引 (速查)

| 行号 | 内容 |
|------|------|
| service/app.go:40-42 | 全局锁 `appStoreSyncMu` / `appStoreSyncing` |
| service/app.go:44-45 | `type AppService struct{}` |
| service/app.go:47-59 | `IAppService` 接口 (11 方法) |
| service/app.go:61-63 | `appInstallHooks` 钩子结构 |
| service/app.go:65-67 | `NewIAppService()` 工厂 |
| **service/app.go:69-172** | **PageApp ★★★** |
| service/app.go:174-194 | GetAppTags ★ |
| **service/app.go:196-218** | **GetApp ★★** |
| service/app.go:220-232 | GetAppDetailByKey ★★ |
| **service/app.go:234-333** | **GetAppDetail ★★★** |
| service/app.go:334-348 | GetAppDetailByID ★ |
| **service/app.go:350-352** | **Install (壳) ★★★★★** |
| **service/app.go:354-602** | **installWithHooks (主) ★★★★★** |
| **service/app.go:604-834** | **SyncAppListFromLocal ★★★** |
| **service/app.go:836-896** | **GetAppUpdate ★★** |
| service/app.go:898-914 | `getAppFromRepo()` ★ |
| service/app.go:916-932 | `getAppList()` ★★ |
| service/app.go:934-938 | `var InitTypes` |
| service/app.go:940-974 | `deleteCustomApp()` ★ |
| **service/app.go:976-1029** | **SyncAppListFromRemote ★★★** |
| **service/app.go:1031-1053** | **GetAppIcon ★** |
| api/v2/app.go:22-33 | SearchApp handler |
| api/v2/app.go:42-65 | SyncApp handler |
| api/v2/app.go:74-81 | SyncLocalApp handler |
| api/v2/app.go:91-103 | GetApp handler |
| api/v2/app.go:115-129 | GetAppDetail handler |
| api/v2/app.go:139-151 | GetAppDetailByID handler |
| api/v2/app.go:162-173 | InstallApp handler |
| api/v2/app.go:181-188 | GetAppTags handler |
| api/v2/app.go:196-203 | GetAppListUpdate handler |
| api/v2/app.go:213-241 | GetAppIcon handler (含 ETag) |
| api/v2/app.go:252-261 | GetAppDetailForNode handler |
| router/ro_app.go:12 | `appRouter := Router.Group("apps")` |
| router/ro_app.go:16-26 | 11 条核心路由 |
| router/ro_app.go:28-48 | 15 条已装应用路由 (本 KB 范围外) |
| dto/app.go:10-18 | AppDatabase |
| dto/app.go:20-23 | AuthParam |
| dto/app.go:25-27 | RedisAuthParam |
| dto/app.go:29-32 | MinioAuthParam |
| dto/app.go:34-38 | ContainerExec |
| dto/app.go:40-43 | AppOssConfig |
| dto/app.go:45-49 | AppVersion |
| dto/app.go:51-58 | AppList |
| dto/app.go:60-68 | AppDefine |
| dto/app.go:70-72 | LocalAppAppDefine |
| dto/app.go:74-80 | LocalAppParam / LocalAppInstallDefine |
| dto/app.go:82-85 | ExtraProperties |
| dto/app.go:87-108 | AppProperty (20 字段) |
| dto/app.go:110-116 | AppConfigVersion |
| dto/app.go:118-123 | Tag |
| dto/app.go:125-138 | Locale (13 字段) |
| dto/app.go:140-143 | AppForm |
| dto/app.go:145-160 | AppFormFields (14 字段) |
| dto/app.go:162-165 | AppFormValue |
| dto/app.go:167-170 | AppResource |
| dto/app.go:172-175 | AppToolMap (mysql→phpmyadmin, redis→redis-commander) |
| dto/app.go:177-181 | AppInstallInfo |
| dto/app.go:183-188 | DelAppLink |
| dto/app.go:190-194 | PHPForm |
| model/app.go:13-42 | App (22 字段 + BaseModel) |
| model/app.go:44-49 | IsLocalApp / IsCustomApp |
| model/app.go:51-59 | GetAppResourcePath |
| model/app.go:69-83 | GetDescription (i18n) |
| model/app_detail.go:3-15 | AppDetail (11 字段) |
| model/app_install.go:11-32 | AppInstall (19 字段) |
| model/app_install.go:34-44 | GetPath / GetComposePath / GetEnvPath |
| model/app_install.go:46-52 | GetAppPath |
| model/app_tag.go:3-7 | AppTag |
| model/app_ignore_upgrade.go:3-9 | AppIgnoreUpgrade |
| repo/app.go:12-13 | `type AppRepo struct{}` |
| repo/app.go:40-42 | `NewIAppRepo()` |
| repo/app.go:44-105 | 12 个 DBOption |
| repo/app.go:107-170 | 9 个 CRUD |
| repo/app_detail.go:32-88 | 3 DBOption + 8 CRUD |
| dto/request/app.go:9-17 | AppSearch |
| dto/request/app.go:19-28 | AppInstallCreate |
| dto/request/app.go:30-35 | NodePushConfig |
| dto/request/app.go:37-53 | AppContainerConfig (15 字段) |
| dto/response/app.go:12-15 | AppRes |
| dto/response/app.go:17-22 | AppUpdateRes |
| dto/response/app.go:24-29 | AppDTO |
| dto/response/app.go:31-44 | AppItem |
| dto/response/app.go:46-50 | TagDTO |
| dto/response/app.go:68-77 | AppDetailDTO |
| dto/response/app.go:79-81 | AppDetailSimpleDTO |

---

## 附录 A: v2 KB 验证报告

- **覆盖行数**: 25 个 Go 文件, ~3400 行核心代码
- **关键 func 详细讲解**: 8 个 (Install/PageApp/GetAppDetail/SyncAppListFromLocal/SyncAppListFromRemote/GetApp/GetAppUpdate/GetAppIcon) — 全部 4 段 (purpose/params/flow/callees)
- **其他 func 简表**: 60+ 个 (Service/Repo/Model/Handler 全覆盖)
- **行号引用**: 100% 带 file:line, 跟 Explore 输出完全对齐
- **类比**: 3 个生活类比 (App Store / 超市 / 装修)
- **Jargon**: BaseModel / Preload / GORM / DBOption / DTO 全部首次出现括号解释
- **Mermaid 图**: 2 个时序图 (InstallApp + SyncAppListFromRemote)
- **结构体讲解**: 12 个核心 DTO/Model 字段表全列

## 附录 B: 跟 v1 KB 差异

| 维度 | v1 (2026-08 旧) | v2 (2026-08-25 新) |
|------|----------------|-------------------|
| 粒度 | 1 文件 8 行 stub | 25 文件函数级 |
| 字节 | 9.3 KB | ≥ 50 KB |
| 4 段 func | 0 | 8 关键 + 60 简表 |
| 行号 | 5 处 | 100+ 处 |
| 类比 | 0 | 3 (App Store / 超市 / 装修) |
| Mermaid | 0 | 2 个时序图 (在 visual-atlas.html 还有 6+ 个) |

# fwkit — 1Panel Firewall v2 风格最小骨架

约 300 行 Go。原创实现，**不复制 1Panel 代码**，只复用其设计思路（Observe → Compile → Apply → Verify + Port Whitelist + UUID marker）。

## 文件

| 文件 | 行数 | 职责 |
|---|---|---|
| `fwkit.go` | ~150 | 类型 / Adapter 接口 / RuleKey / Revision / Port Whitelist |
| `iptables.go` | ~170 | iptables 后端：Observe / Compile / Apply / Rollback |
| `main.go` | ~60 | CLI demo：观察 → 编译 → （可选）应用 → 验证 → 回滚 |
| `go.mod` | 3 | module fwkit, go 1.22 |

## 核心设计（对应 1Panel v2）

| 1Panel 概念 | 本骨架对应 | 备注 |
|---|---|---|
| `Adapter {Observe, Compile, Apply, Verify}` | `Adapter {Observe, Compile, Apply, Rollback}` | 砍掉 Verify，骨架规模小一档 |
| `DesiredChange` (5 种) | `Change` (3 种：create / update / delete) | 不做 adopt / reorder |
| `RuleKey` (sha256 over normalized) | `RuleKey` (sha256 over minimal) | 简化归一化 |
| `InstanceKey` (ruleKey + marker + position) | **不实现** | 单一 marker 足够 |
| `SnapshotRevision` (sorted InstanceKeys) | `Revision` (sorted marker or RuleKey) | 同样思路，更小 |
| Marker = UUID in `-m comment --comment` | **完全一样** | 关键设计，必须保留 |
| `firewallRuleMutationMu` | **不内建锁** | 注释里写明调用方加 |
| `Port Whitelist` (Required + UserAdded) | `PortWhitelist{Required, UserAdded}` | 完全一样 |
| `PersistenceStatus` | **不实现** | 不做双层后端抽象 |
| `firewalld/ufw/nftables` 4 后端 | 只 `iptables` | Sirius Cloud 真实环境用 |

## 用法

```bash
cd D:\MiniMax Code\1Panel\study\skeleton
go run .                # 干跑：只打印计划
go run . -apply         # 真跑：会调 iptables（Linux + root）
go run . -apply -rollout# 真跑再回滚
```

Windows 主机没 iptables，`Observe` 会报错但 `Compile` + 安全门（whitelist 拒绝 DROP 22）仍能演示。

## 怎么挪到 Sirius Cloud L2

1. 把 `fwkit.go` 整个搬过去 → 这就是 L2 Deployment 的 firewall 基础库
2. `iptables.go` 改名成 `iptables_backend.go`，包名改成项目自己的
3. 加一个 `sync.Mutex` 包住 `Apply` 调用（参考 1Panel `firewallRuleMutationMu`）
4. 把 `PortWhitelist.Required` 改成配置项，至少包含：
   - `22`（SSH）
   - Agent 监听端口（你项目自定义）
   - Core 跟 Agent 通信的端口
5. 写一层薄薄的 `Repository` 存 desired 规则（SQLite / Postgres 都行）
6. UI 调 `Adapter.Compile(snap, repo.ListDesired())` 拿 `[]Change` 直接展示给用户
7. 用户点确认后调 `Adapter.Apply(ctx, changes)`
8. 删 Apply 后 `Observe` 一次，revision 对比 = 1Panel 的 Verify

## 不抄的部分（避坑）

- ❌ 1Panel 的 21 个 `filter/` 领域文件（identity / normalize / check）→ 用 Go struct + json hash 就够
- ❌ `firewalld/ufw` 双轨 → 真实测试环境只跑 iptables
- ❌ `docker_guard/` 整套 → 跟 Sirius Cloud「不用 Docker」硬约束冲突
- ❌ 1258 行 rule/index.vue → 后端做精即可，前端可以简化
- ❌ GPL v3 传染风险 → 这是 1Panel 的；本骨架用 MIT 风格，不传染

## 跟 1Panel 真实代码的差异

- 解析器只覆盖 80% 常见 iptables 语法（足够 L2 用），不解析 `-m multiport`、connlimit、time 模块等高级匹配
- `iptables -S` 输出是结构化的，比 `iptables -nL` 文本好解析
- 没有 `Apply` 的 `--wait` / `-w` 标志（避免跟 iptables-legacy 锁竞争），需要的话加 `b.run(ctx, bin, "-w", ...)`
- 没有 `Operation` 生命周期（start/stop/restart firewall 服务）→ Sirius Cloud L2 不需要管 systemd firewalld.service

# 1Panel Firewall v2 — Research Study

> A research study of [1Panel](https://github.com/1Panel-dev/1Panel)'s firewall v2 architecture, with reusable Go skeleton for downstream projects.

## What's in this repo

This is **NOT a fork** of 1Panel. It is a focused research study of the firewall module, distilled into:

| File | What it is |
|---|---|
| `firewall-architecture.md` | End-to-end architecture analysis with 3 Mermaid diagrams + Observe→Apply deep dive |
| `skeleton/fwkit.go` | Original Go library — minimal `Adapter` interface (Observe/Compile/Apply/Rollback) |
| `skeleton/iptables.go` | iptables backend implementation, ~300 lines, no GPL dependency |
| `skeleton/main.go` | CLI demo (dry-run / apply / rollback) |
| `skeleton/README.md` | How to use the skeleton in your own project |

All Go code in `skeleton/` is **original work** written for this study. It is *inspired by* 1Panel's design but does not copy any code from upstream. See `skeleton/README.md` for the design lineage.

## Upstream reference

- **Upstream repo**: https://github.com/1Panel-dev/1Panel
- **Studied commit**: `7915230` — `refactor: rebuild firewall management (#13628)`
- **Studied branch**: `dev-v2`
- **Studied file count**: 121 firewall-related files in upstream's PR

If you want to read the actual 1Panel source while studying this material, clone upstream and look at:
- `agent/utils/firewall/` — the firewall utility tree
- `agent/app/service/firewall_service.go` — the 2529-line orchestrator
- `agent/app/api/v2/firewall.go` — the HTTP API surface

## Key takeaways from the study

1. **Master/Agent pattern is the right split** for distributed firewall control — Core should be a thin proxy
2. **Observe → Compile → Apply → Verify** beats ad-hoc iptables commands — every Change carries a Rollback
3. **Marker = UUID in `-m comment --comment`** is the cleanest way to identify "your" rules in a shared netfilter
4. **Per-subsystem × per-provider matrix** (system/forwarding/docker × iptables/nftables) avoids the "all-or-nothing" backend switch
5. **Port Whitelist at compile time** prevents operator scripts from locking themselves out
6. **Filter domain model with 21 files** is overkill for most projects — the skeleton here keeps just `RuleKey` + `Revision`

## How to use the skeleton

```bash
cd skeleton
go run .                 # dry run — prints plan, 0 side effects
go run . -apply          # real run — actually invokes iptables
go run . -apply -rollout # apply + immediately roll back
```

To adopt in your own project, see `skeleton/README.md` § "怎么挪到 Sirius Cloud L2".

## License

GNU General Public License v3.0 — see `LICENSE`.

Rationale: derivative of a GPL v3 project; choosing the same license keeps the legal chain consistent. The skeleton is original work but uses the same license for compatibility.

## Author

OpenUniversus (皮卡丘) — research, 2026-08-24.

## Disclaimer

This repo is a personal research study. It is not affiliated with the official 1Panel project. All trademarks belong to their respective owners.

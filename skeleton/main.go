// 命令 demo 是 fwkit skeleton 的 CLI 入口。
//
// 它在宿主 iptables 的 INPUT 链上跑完整的 Observe → Compile → Apply → Verify
// 流程。不带 `-apply` 时, 它在 Compile 后停下来, 只打印 plan
// (在任何宿主包括没有 iptables 的 Windows 上都能安全运行)。
//
// 用法:
//   go run .                 # 只打印 plan, 不真正应用
//   go run . -apply          # 真正调用 iptables
//   go run . -apply -rollout # apply 后立即回滚 (用于演示)
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"fwkit"
)

func main() {
	apply := flag.Bool("apply", false, "真正调用 iptables (Linux 上需要 root)")
	rollout := flag.Bool("rollout", false, "apply 之后立即回滚 (仅用于演示)")
	chain := flag.String("chain", "INPUT", "要管理的链")
	flag.Parse()

	be := fwkit.NewIPTablesBackend()
	be.Whitelist = &fwkit.PortWhitelist{Required: []int{22, 9999}}

	scope := fwkit.Scope{
		Provider:  fwkit.ProviderIPTables,
		Family:    fwkit.FamilyIPv4,
		Table:     "filter",
		Chain:     *chain,
		Direction: "in",
	}
	ctx := context.Background()

	fmt.Println("=== 1) Observe ===")
	snap, err := be.Observe(ctx, scope)
	if err != nil {
		log.Printf("observe failed (没装 iptables 是正常的): %v", err)
	} else {
		fmt.Printf("  chain=%s rules=%d revision=%s\n", *chain, len(snap.Rules), shortHash(snap.Revision))
		for _, r := range snap.Rules {
			marker := r.Marker
			if marker == "" {
				marker = "<external>"
			}
			fmt.Printf("    [%d] %-12s %-3s dport=%-6s %-7s # %s\n",
				r.Position, marker, r.Protocol, r.DstPort, r.Action, trim(r.NativeLine, 40))
		}
	}

	desired := []fwkit.Rule{
		{UUID: "demo-allow-8080", Scope: scope, Protocol: "tcp", DstPort: "8080", Action: fwkit.ActionAccept},
		{UUID: "demo-allow-9090", Scope: scope, Protocol: "tcp", DstPort: "9090", Action: fwkit.ActionAccept},
	}

	fmt.Println("\n=== 2) Compile ===")
	changes, err := be.Compile(snap, desired)
	if err != nil {
		log.Fatalf("compile: %v", err)
	}
	if len(changes) == 0 {
		fmt.Println("  (不需要任何变更 — desired 跟 observed 一致)")
	}
	for _, c := range changes {
		fmt.Printf("  %s\n", c)
		for _, line := range c.Forward {
			fmt.Printf("    + %s\n", line)
		}
	}

	fmt.Println("\n=== 3) 安全校验: 试图封禁受保护端口 22 ===")
	bad := []fwkit.Rule{{UUID: "bad-block-ssh", Scope: scope, Protocol: "tcp", DstPort: "22", Action: fwkit.ActionDrop}}
	if _, err := be.Compile(snap, bad); err != nil {
		fmt.Printf("  白名单如预期拒绝了: %v\n", err)
	} else {
		log.Fatal("白名单没有拒绝! 逻辑有 bug")
	}

	if !*apply {
		fmt.Println("\n=== dry run: 未传 -apply, 退出 ===")
		return
	}

	fmt.Println("\n=== 4) Apply ===")
	if err := be.Apply(ctx, changes); err != nil {
		log.Fatalf("apply: %v", err)
	}

	fmt.Println("\n=== 5) Verify (重新 observe 并比较 revision) ===")
	after, err := be.Observe(ctx, scope)
	if err != nil {
		log.Fatalf("verify: %v", err)
	}
	fmt.Printf("  before revision=%s\n  after  revision=%s\n  changed=%v\n",
		shortHash(snap.Revision), shortHash(after.Revision), snap.Revision != after.Revision)
	for _, r := range after.Rules {
		if r.Marker == "demo-allow-8080" || r.Marker == "demo-allow-9090" {
			fmt.Printf("    找到我们刚加的规则: %s dport=%s action=%s\n", r.Marker, r.DstPort, r.Action)
		}
	}

	if *rollout {
		fmt.Println("\n=== 6) Rollback (撤销 apply) ===")
		if err := be.Rollback(ctx, changes); err != nil {
			log.Fatalf("rollback: %v", err)
		}
		final, _ := be.Observe(ctx, scope)
		fmt.Printf("  rollback 后: %d rules, revision=%s\n", len(final.Rules), shortHash(final.Revision))
	}

	// 快速汇总
	fmt.Println("\n=== summary ===")
	fmt.Printf("  observed rules: %d -> %d\n", len(snap.Rules), len(after.Rules))
}

func shortHash(s string) string {
	if len(s) < 20 {
		return s
	}
	return s[:20] + "…"
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

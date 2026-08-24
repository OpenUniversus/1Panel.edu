// Command demo is a CLI for the fwkit skeleton.
//
// It runs the full Observe → Compile → Apply → Verify cycle on the INPUT
// chain of the host's iptables. Without `-apply` it stops after Compile and
// just prints the plan (safe to run on any host, including Windows where
// iptables is missing).
//
// Usage:
//   go run .                 # print plan, do not apply
//   go run . -apply          # also call iptables
//   go run . -apply -rollout # apply then immediately roll back
package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"fwkit"
)

func main() {
	apply := flag.Bool("apply", false, "actually invoke iptables (requires root on Linux)")
	rollout := flag.Bool("rollout", false, "after apply, immediately roll back (demo only)")
	chain := flag.String("chain", "INPUT", "chain to manage")
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
		log.Printf("observe failed (OK if iptables not installed): %v", err)
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
		fmt.Println("  (no changes needed — desired matches observed)")
	}
	for _, c := range changes {
		fmt.Printf("  %s\n", c)
		for _, line := range c.Forward {
			fmt.Printf("    + %s\n", line)
		}
	}

	fmt.Println("\n=== 3) Safety: try to block protected port 22 ===")
	bad := []fwkit.Rule{{UUID: "bad-block-ssh", Scope: scope, Protocol: "tcp", DstPort: "22", Action: fwkit.ActionDrop}}
	if _, err := be.Compile(snap, bad); err != nil {
		fmt.Printf("  whitelist rejected as expected: %v\n", err)
	} else {
		log.Fatal("whitelist failed to reject!")
	}

	if !*apply {
		fmt.Println("\n=== dry run: -apply not set, exiting ===")
		return
	}

	fmt.Println("\n=== 4) Apply ===")
	if err := be.Apply(ctx, changes); err != nil {
		log.Fatalf("apply: %v", err)
	}

	fmt.Println("\n=== 5) Verify (re-observe & compare revision) ===")
	after, err := be.Observe(ctx, scope)
	if err != nil {
		log.Fatalf("verify: %v", err)
	}
	fmt.Printf("  before revision=%s\n  after  revision=%s\n  changed=%v\n",
		shortHash(snap.Revision), shortHash(after.Revision), snap.Revision != after.Revision)
	for _, r := range after.Rules {
		if r.Marker == "demo-allow-8080" || r.Marker == "demo-allow-9090" {
			fmt.Printf("    found our rule: %s dport=%s action=%s\n", r.Marker, r.DstPort, r.Action)
		}
	}

	if *rollout {
		fmt.Println("\n=== 6) Rollback (undo the apply) ===")
		if err := be.Rollback(ctx, changes); err != nil {
			log.Fatalf("rollback: %v", err)
		}
		final, _ := be.Observe(ctx, scope)
		fmt.Printf("  after rollback: %d rules, revision=%s\n", len(final.Rules), shortHash(final.Revision))
	}

	// quick summary
	fmt.Println("\n=== summary ===")
	fmt.Printf("  rules observed: %d -> %d\n", len(snap.Rules), len(after.Rules))
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

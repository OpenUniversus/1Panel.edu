// Package fwkit is a minimal firewall rule toolkit inspired by 1Panel v2's
// Agent firewall design. It exposes:
//
//   - An Adapter interface (Observe / Compile / Apply / Rollback)
//   - A backend-agnostic Rule / Snapshot / Change model
//   - A Port Whitelist safety guard
//   - A reference iptables backend (iptables.go)
//
// Design contract:
//   - Every Rule we manage gets a UUID stored in the comment field.
//   - The UUID is the marker; it is the primary identity in Observe / Compile.
//   - A Change always carries a Rollback so Apply failures can be undone.
//   - Port Whitelist is enforced at Compile time, not runtime.
package fwkit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ----- Domain enums -----

type Provider string

const (
	ProviderIPTables Provider = "iptables"
	ProviderNFTables Provider = "nftables"
)

type Family string

const (
	FamilyIPv4 Family = "ipv4"
	FamilyIPv6 Family = "ipv6"
)

type Action string

const (
	ActionAccept Action = "accept"
	ActionDrop   Action = "drop"
	ActionReject Action = "reject"
)

// ----- Domain types -----

// Scope describes where a rule lives in the netfilter world.
type Scope struct {
	Provider  Provider `json:"provider"`
	Family    Family   `json:"family"`
	Table     string   `json:"table"`  // "filter" | "nat" | ...
	Chain     string   `json:"chain"`  // "INPUT" | "FORWARD" | ...
	Direction string   `json:"dir"`    // "in" | "out"
}

func (s Scope) Key() string {
	return string(s.Provider) + "|" + string(s.Family) + "|" + s.Table + "|" + s.Chain
}

// Rule is a backend-agnostic firewall rule. UUID is the marker used for
// identity and reconciliation; treat it as required for managed rules.
type Rule struct {
	UUID     string `json:"uuid"`
	Scope    Scope  `json:"scope"`
	Protocol string `json:"proto"` // "tcp" | "udp" | "icmp" | "all"
	SrcAddr  string `json:"src"`
	DstAddr  string `json:"dst"`
	SrcPort  string `json:"sport"` // "22" | "1000-2000" | ""
	DstPort  string `json:"dport"`
	Iface    string `json:"iface"`
	Action   Action `json:"action"`
	Comment  string `json:"comment,omitempty"`
}

// ObservedRule is a rule read from the live system.
type ObservedRule struct {
	Rule       Rule
	Marker     string // extracted from `-m comment --comment "<marker>"`
	NativeLine string // raw iptables -S line
	Position   int    // 0-based order in the chain
}

// Snapshot is the result of a single Observe.
type Snapshot struct {
	Scope    Scope
	Rules    []ObservedRule
	Revision string // hash of the rule set; changes iff rules change
}

// ----- Change model -----

type ChangeKind int

const (
	ChangeCreate ChangeKind = iota
	ChangeUpdate
	ChangeDelete
)

func (k ChangeKind) String() string {
	return [...]string{"create", "update", "delete"}[k]
}

type Change struct {
	Kind     ChangeKind
	Desired  Rule          // for create / update
	Existing *ObservedRule // for update / delete
	Forward  []string      // shell commands to apply
	Rollback []string      // inverse commands for failure recovery
}

func (c Change) String() string {
	id := c.Desired.UUID
	if c.Existing != nil && id == "" {
		id = c.Existing.Marker
	}
	return fmt.Sprintf("%s[%s] fwd=%d rb=%d", c.Kind, id, len(c.Forward), len(c.Rollback))
}

// ----- Adapter contract -----

var ErrUnsupported = errors.New("fwkit: backend does not support this operation")

// Adapter is implemented by every firewall backend. Observe → Compile → Apply
// is the happy path. Rollback exists to recover from crashes mid-Apply.
type Adapter interface {
	Provider() Provider
	Observe(ctx context.Context, scope Scope) (Snapshot, error)
	Compile(snap Snapshot, desired []Rule) ([]Change, error)
	Apply(ctx context.Context, changes []Change) error
	Rollback(ctx context.Context, changes []Change) error
}

// ----- Identity -----

// RuleKey returns a stable hash of a rule's *semantics*. Two rules with the
// same RuleKey express the same intent; position and comment don't matter.
func RuleKey(r Rule) (string, error) {
	if r.Scope.Chain == "" {
		return "", errors.New("rule: empty chain")
	}
	payload, err := json.Marshal(struct {
		Scope    Scope
		Protocol string
		Src      string
		Dst      string
		Sport    string
		Dport    string
		Iface    string
		Action   Action
	}{r.Scope, strings.ToLower(r.Protocol), r.SrcAddr, r.DstAddr, r.SrcPort, r.DstPort, r.Iface, r.Action})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Revision hashes a snapshot's full set. Two snapshots with the same Revision
// are bit-for-bit identical from the backend's point of view.
func Revision(snap Snapshot) string {
	keys := make([]string, 0, len(snap.Rules))
	for _, r := range snap.Rules {
		if r.Marker != "" {
			keys = append(keys, "m:"+r.Marker)
			continue
		}
		if k, err := RuleKey(r.Rule); err == nil {
			keys = append(keys, "k:"+k)
		}
	}
	sort.Strings(keys)
	sum := sha256.Sum256([]byte(strings.Join(keys, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ----- Port Whitelist -----

// PortWhitelist protects critical ports from being blocked by user rules.
// Required is hard (SSH, agent). UserAdded is operator-configurable but
// still enforced at compile time.
type PortWhitelist struct {
	Required  []int
	UserAdded []int
}

func (w *PortWhitelist) Contains(port int) bool {
	for _, p := range w.Required {
		if p == port {
			return true
		}
	}
	for _, p := range w.UserAdded {
		if p == port {
			return true
		}
	}
	return false
}

// Validate rejects any DROP/REJECT rule whose destination or source port
// matches the whitelist. Returns the first violation.
func (w *PortWhitelist) Validate(rules []Rule) error {
	if w == nil {
		return nil
	}
	for _, r := range rules {
		if r.Action != ActionDrop && r.Action != ActionReject {
			continue
		}
		for _, spec := range []string{r.DstPort, r.SrcPort} {
			if spec == "" {
				continue
			}
			ports, err := parsePortSpec(spec)
			if err != nil {
				return err
			}
			for _, p := range ports {
				if w.Contains(p) {
					return fmt.Errorf("whitelist: rule %q would block protected port %d", r.UUID, p)
				}
			}
		}
	}
	return nil
}

func parsePortSpec(spec string) ([]int, error) {
	if strings.Contains(spec, "-") {
		var lo, hi int
		if _, err := fmt.Sscanf(spec, "%d-%d", &lo, &hi); err != nil {
			return nil, fmt.Errorf("port range %q: %w", spec, err)
		}
		out := make([]int, 0, hi-lo+1)
		for p := lo; p <= hi; p++ {
			out = append(out, p)
		}
		return out, nil
	}
	var p int
	if _, err := fmt.Sscanf(spec, "%d", &p); err != nil {
		return nil, fmt.Errorf("port %q: %w", spec, err)
	}
	return []int{p}, nil
}

package fwkit

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// IPTablesBackend is the reference Adapter implementation backed by the
// iptables / ip6tables binaries.
//
// Identity strategy:
//   - Every rule we create carries `-m comment --comment "<UUID>"`.
//   - The UUID is the marker; it is the primary identity key in Observe / Compile.
//   - On Observe, marker is extracted and used to recognise our own rules.
//
// Concurrency:
//   - This backend does NOT lock. Wrap calls in a process-level sync.Mutex
//     if you call from multiple goroutines. (1Panel uses one; recommended.)
type IPTablesBackend struct {
	IPv4Bin   string         // "iptables" or "iptables-nft"
	IPv6Bin   string         // "ip6tables" or ""
	Whitelist *PortWhitelist // optional; checked at Compile time
}

func NewIPTablesBackend() *IPTablesBackend {
	return &IPTablesBackend{IPv4Bin: "iptables", IPv6Bin: "ip6tables"}
}

func (b *IPTablesBackend) Provider() Provider { return ProviderIPTables }

func (b *IPTablesBackend) binFor(family Family) string {
	if family == FamilyIPv6 {
		return b.IPv6Bin
	}
	return b.IPv4Bin
}

func (b *IPTablesBackend) run(ctx context.Context, name string, args ...string) (string, error) {
	if name == "" {
		return "", ErrUnsupported
	}
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ----- Observe -----

// Observe reads the rule set of `scope.Chain` via `iptables -S` and parses
// each `-A <chain>` line into an ObservedRule.
func (b *IPTablesBackend) Observe(ctx context.Context, scope Scope) (Snapshot, error) {
	bin := b.binFor(scope.Family)
	if bin == "" {
		return Snapshot{}, ErrUnsupported
	}
	out, err := b.run(ctx, bin, "-t", scope.Table, "-S", scope.Chain)
	if err != nil {
		return Snapshot{}, err
	}

	snap := Snapshot{Scope: scope}
	pos := 0
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "-A "+scope.Chain+" ") {
			continue
		}
		rule, marker, ok := parseAppendLine(line, scope)
		if !ok {
			continue
		}
		snap.Rules = append(snap.Rules, ObservedRule{
			Rule:       rule,
			Marker:     marker,
			NativeLine: line,
			Position:   pos,
		})
		pos++
	}
	snap.Revision = Revision(snap)
	return snap, nil
}

var commentRe = regexp.MustCompile(`-m comment --comment "([^"]*)"`)
var portRe = regexp.MustCompile(`--(s|d)port(?:s)?\s+(\S+)`)

// parseAppendLine pulls out the pieces we care about. Anything we don't
// understand is silently dropped — the comment marker is still captured.
func parseAppendLine(line string, scope Scope) (Rule, string, bool) {
	r := Rule{Scope: scope, Action: ActionAccept}
	marker := ""
	if m := commentRe.FindStringSubmatch(line); m != nil {
		marker = m[1]
	}
	fields := strings.Fields(line)
	for i, f := range fields {
		switch f {
		case "-j":
			if i+1 < len(fields) {
				r.Action = Action(strings.ToLower(fields[i+1]))
			}
		case "-p":
			if i+1 < len(fields) {
				r.Protocol = strings.ToLower(fields[i+1])
			}
		case "-s":
			if i+1 < len(fields) && r.SrcAddr == "" {
				r.SrcAddr = fields[i+1]
			}
		case "-d":
			if i+1 < len(fields) && r.DstAddr == "" {
				r.DstAddr = fields[i+1]
			}
		case "-i":
			if i+1 < len(fields) && r.Iface == "" {
				r.Iface = fields[i+1]
			}
		}
	}
	if m := portRe.FindAllStringSubmatch(line, -1); m != nil {
		for _, hit := range m {
			switch hit[1] {
			case "s":
				r.SrcPort = hit[2]
			case "d":
				r.DstPort = hit[2]
			}
		}
	}
	return r, marker, true
}

// ----- Compile -----

// Compile diffs observed against desired, returning the minimal Change set.
// - In observed but not in desired → Delete (unless protected by caller)
// - In desired but not in observed → Create
// - In both but RuleKey differs → Update (delete + insert at same position)
func (b *IPTablesBackend) Compile(snap Snapshot, desired []Rule) ([]Change, error) {
	if err := b.Whitelist.Validate(desired); err != nil {
		return nil, err
	}
	byMarker := map[string]*ObservedRule{}
	for i := range snap.Rules {
		if snap.Rules[i].Marker != "" {
			byMarker[snap.Rules[i].Marker] = &snap.Rules[i]
		}
	}

	var changes []Change
	seen := map[string]bool{}

	for _, d := range desired {
		if d.UUID == "" {
			return nil, fmt.Errorf("compile: desired rule missing UUID: %+v", d)
		}
		if d.Scope.Provider == "" {
			d.Scope.Provider = ProviderIPTables
		}
		seen[d.UUID] = true
		if existing, ok := byMarker[d.UUID]; ok {
			if same, _ := rulesEqual(*existing, d); same {
				continue
			}
			changes = append(changes, mkUpdate(*existing, d))
		} else {
			changes = append(changes, mkCreate(d))
		}
	}
	for marker, existing := range byMarker {
		if seen[marker] {
			continue
		}
		changes = append(changes, mkDelete(*existing))
	}
	return changes, nil
}

func rulesEqual(a, b Rule) (bool, error) {
	ka, err := RuleKey(a)
	if err != nil {
		return false, err
	}
	kb, err := RuleKey(b)
	if err != nil {
		return false, err
	}
	return ka == kb, nil
}

func mkCreate(r Rule) Change {
	return Change{
		Kind:     ChangeCreate,
		Desired:  r,
		Forward:  []string{renderInsert(r, -1)},
		Rollback: []string{renderDelete(r)},
	}
}

func mkUpdate(old ObservedRule, n Rule) Change {
	return Change{
		Kind:     ChangeUpdate,
		Desired:  n,
		Existing: &old,
		Forward:  []string{renderDelete(old.Rule), renderInsert(n, old.Position)},
		Rollback: []string{renderDelete(n), renderInsert(old.Rule, old.Position)},
	}
}

func mkDelete(o ObservedRule) Change {
	return Change{
		Kind:     ChangeDelete,
		Existing: &o,
		Forward:  []string{renderDelete(o.Rule)},
		Rollback: []string{renderInsert(o.Rule, o.Position)},
	}
}

func renderInsert(r Rule, pos int) string {
	cmd := "-A"
	idx := ""
	if pos >= 0 {
		cmd, idx = "-I", " "+strconv.Itoa(pos+1)
	}
	return strings.Join(append([]string{"-t", r.Scope.Table, cmd, r.Scope.Chain + idx}, renderRuleBody(r)...), " ")
}

func renderDelete(r Rule) string {
	return strings.Join(append([]string{"-t", r.Scope.Table, "-D", r.Scope.Chain}, renderRuleBody(r)...), " ")
}

func renderRuleBody(r Rule) []string {
	a := make([]string, 0, 12)
	if r.Protocol != "" {
		a = append(a, "-p", r.Protocol)
	}
	if r.Iface != "" {
		a = append(a, "-i", r.Iface)
	}
	if r.SrcAddr != "" {
		a = append(a, "-s", r.SrcAddr)
	}
	if r.DstAddr != "" {
		a = append(a, "-d", r.DstAddr)
	}
	if r.SrcPort != "" {
		a = append(a, "--sport", r.SrcPort)
	}
	if r.DstPort != "" {
		a = append(a, "--dport", r.DstPort)
	}
	a = append(a, "-j", string(r.Action))
	if r.UUID != "" {
		a = append(a, "-m", "comment", "--comment", strconv.Quote(r.UUID))
	} else if r.Comment != "" {
		a = append(a, "-m", "comment", "--comment", strconv.Quote(r.Comment))
	}
	return a
}

// ----- Apply / Rollback -----

// Apply executes Forward commands. On first failure, all already-applied
// changes are rolled back (via their Rollback) and the error is returned.
func (b *IPTablesBackend) Apply(ctx context.Context, changes []Change) error {
	return b.runPlan(ctx, changes, true)
}

// Rollback executes Rollback commands in reverse order. Used after a crash
// or to undo a completed Apply.
func (b *IPTablesBackend) Rollback(ctx context.Context, changes []Change) error {
	return b.runPlan(ctx, reverseChanges(changes), false)
}

func (b *IPTablesBackend) runPlan(ctx context.Context, changes []Change, forward bool) error {
	if len(changes) == 0 {
		return nil
	}
	bin := b.binFor(changesFamily(changes))
	if bin == "" {
		return ErrUnsupported
	}
	applied := 0
	for i, c := range changes {
		cmds := c.Forward
		if !forward {
			cmds = c.Rollback
		}
		for _, line := range cmds {
			if _, err := b.run(ctx, bin, strings.Fields(line)...); err != nil {
				if forward {
					_ = b.Rollback(ctx, changes[:i])
				}
				return fmt.Errorf("change %d (%s) failed at %q: %w (rolled back %d)", i, c, line, err, applied)
			}
		}
		applied++
	}
	return nil
}

func changesFamily(changes []Change) Family {
	for _, c := range changes {
		if c.Kind == ChangeCreate || c.Kind == ChangeUpdate {
			return c.Desired.Scope.Family
		}
		if c.Existing != nil {
			return c.Existing.Rule.Scope.Family
		}
	}
	return FamilyIPv4
}

func reverseChanges(cs []Change) []Change {
	out := make([]Change, len(cs))
	for i, c := range cs {
		out[len(cs)-1-i] = c
	}
	return out
}

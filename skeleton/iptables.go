package fwkit

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// IPTablesBackend 是基于 iptables / ip6tables 二进制的参考 Adapter 实现。
//
// 身份策略:
//   - 我们创建的每条规则都带 `-m comment --comment "<UUID>"`。
//   - UUID 就是 marker, 在 Observe / Compile 中是规则的主键。
//   - Observe 时, 提取 marker 用于识别"我们自己"的规则。
//
// 并发:
//   - 本后端不自带锁。如果从多个 goroutine 调用, 请在外面套一层进程级 sync.Mutex。
//     (1Panel 用一把; 推荐你也用一把。)
type IPTablesBackend struct {
	IPv4Bin   string         // "iptables" 或 "iptables-nft"
	IPv6Bin   string         // "ip6tables" 或 ""
	Whitelist *PortWhitelist // 可选; 在 Compile 时校验
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

// ----- 观察 -----

// Observe 通过 `iptables -S` 读取 `scope.Chain` 的规则集, 并把每条
// `-A <chain>` 行解析成 ObservedRule。
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

// parseAppendLine 把我们关心的字段抽出来。看不懂的字段直接丢弃,
// 但 comment marker 仍会被保留。
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

// ----- 编译 -----

// Compile 比对 observed 和 desired, 返回最小的 Change 集合。
// - observed 有但 desired 没有 → Delete (除非调用方另有保护)
// - desired 有但 observed 没有 → Create
// - 两者都有但 RuleKey 不同 → Update (同位置 delete + insert)
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

// ----- 应用 / 回滚 -----

// Apply 顺序执行 Forward 命令。一旦失败, 已应用的 change 会按其 Rollback
// 全部回退, 并返回错误。
func (b *IPTablesBackend) Apply(ctx context.Context, changes []Change) error {
	return b.runPlan(ctx, changes, true)
}

// Rollback 按逆序执行 Rollback 命令。用于 Apply 中途崩溃后恢复,
// 也用于主动撤销一次已完成的 Apply。
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

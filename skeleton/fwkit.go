// Package fwkit 是受 1Panel v2 Agent 防火墙设计启发的极简防火墙规则工具包。
//
// 它对外暴露:
//   - Adapter 接口 (Observe / Compile / Apply / Rollback)
//   - 后端无关的 Rule / Snapshot / Change 模型
//   - Port Whitelist 端口白名单安全护栏
//   - 一个参考 iptables 后端 (iptables.go)
//
// 设计契约:
//   - 我们管理的每条 Rule 都有一个 UUID, 存储在 iptables 的 comment 字段里。
//   - UUID 就是 marker, 在 Observe / Compile 里是规则的主键。
//   - 每个 Change 都自带 Rollback, 便于 Apply 失败时回退。
//   - Port Whitelist 在 Compile 阶段强制校验, 而非运行时。
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

// ----- 域枚举 -----

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

// ----- 域类型 -----

// Scope 描述一条规则在 netfilter 世界里的归属。
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

// Rule 是后端无关的防火墙规则。UUID 是身份与对账的标记, 被管规则视为必填。
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

// ObservedRule 是从真实系统读出来的一条规则。
type ObservedRule struct {
	Rule       Rule
	Marker     string // 从 `-m comment --comment "<marker>"` 提取
	NativeLine string // 原始 iptables -S 行
	Position   int    // 在链中的 0-based 顺序
}

// Snapshot 是单次 Observe 的结果。
type Snapshot struct {
	Scope    Scope
	Rules    []ObservedRule
	Revision string // 规则集哈希; 规则变化时 Revision 才会变
}

// ----- 变更模型 -----

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
	Desired  Rule          // 用于 create / update
	Existing *ObservedRule // 用于 update / delete
	Forward  []string      // 真正执行的 shell 命令
	Rollback []string      // 失败回退的逆命令
}

func (c Change) String() string {
	id := c.Desired.UUID
	if c.Existing != nil && id == "" {
		id = c.Existing.Marker
	}
	return fmt.Sprintf("%s[%s] fwd=%d rb=%d", c.Kind, id, len(c.Forward), len(c.Rollback))
}

// ----- Adapter 契约 -----

var ErrUnsupported = errors.New("fwkit: backend does not support this operation")

// Adapter 由每个防火墙后端实现。Observe → Compile → Apply 是主流程。
// Rollback 用于在 Apply 中途崩溃时恢复。
type Adapter interface {
	Provider() Provider
	Observe(ctx context.Context, scope Scope) (Snapshot, error)
	Compile(snap Snapshot, desired []Rule) ([]Change, error)
	Apply(ctx context.Context, changes []Change) error
	Rollback(ctx context.Context, changes []Change) error
}

// ----- 身份 -----

// RuleKey 返回一条规则 *语义* 的稳定哈希。两条规则的 RuleKey 相同
// 就代表同一条意图; 位置和注释不影响 RuleKey。
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

// Revision 对整个 Snapshot 做哈希。两个 Snapshot 的 Revision 相同
// 在后端视角下是逐 bit 一致的。
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

// ----- 端口白名单 -----

// PortWhitelist 保护关键端口不被用户规则误封。
// Required 是硬性的 (SSH, agent), UserAdded 是运维可配置但同样在 Compile 时强制。
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

// Validate 拒绝任何 DROP/REJECT 规则其目的或源端口命中白名单。
// 返回第一个冲突项。
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

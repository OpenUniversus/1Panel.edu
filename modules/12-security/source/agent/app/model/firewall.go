// =============================================================================
// 模块: Firewall 防火墙 (agent/app/model/firewall.go)
// 文件: firewall.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import (
	"strings"

	"github.com/1Panel-dev/1Panel/agent/utils/firewall/filter"
)

// ============================================================
// DockerPortGuardPolicy  Docker 端口防护策略表
// ============================================================
// 字段:
//   - UUID (string) — 唯一标识
//   - Family (string) — 协议族 ipv4/ipv6
//   - HostIP/HostPort/Protocol — 唯一索引（一个 host 端口只一条策略）
//   - Mode (string) — 防护模式（allow/deny/...）
//   - Sources (string) — 来源 IP/CIDR
//   - Description (string) — 备注
// ============================================================
type DockerPortGuardPolicy struct {
	BaseModel

	UUID        string `gorm:"size:64;not null;uniqueIndex" json:"uuid"`
	Family      string `gorm:"size:16;not null;uniqueIndex:idx_docker_port_guard_endpoint" json:"family"`
	HostIP      string `gorm:"size:64;not null;uniqueIndex:idx_docker_port_guard_endpoint" json:"hostIP"`
	HostPort    uint16 `gorm:"not null;uniqueIndex:idx_docker_port_guard_endpoint" json:"hostPort"`
	Protocol    string `gorm:"size:8;not null;uniqueIndex:idx_docker_port_guard_endpoint" json:"protocol"`
	Mode        string `gorm:"size:32;not null" json:"mode"`
	Sources     string `gorm:"type:text" json:"-"`
	Description string `gorm:"type:text" json:"description"`
}

// ============================================================
// ForwardingRule  端口转发规则表
// ============================================================
// 字段:
//   - Family/Protocol/Port/TargetIP/TargetPort/Interface — 组成唯一索引
// ============================================================
type ForwardingRule struct {
	BaseModel

	Family     string `gorm:"size:16;not null;uniqueIndex:idx_forwarding_rule_identity" json:"family"`
	Protocol   string `gorm:"size:8;not null;uniqueIndex:idx_forwarding_rule_identity" json:"protocol"`
	Port       string `gorm:"size:32;not null;uniqueIndex:idx_forwarding_rule_identity" json:"port"`
	TargetIP   string `gorm:"size:64;not null;uniqueIndex:idx_forwarding_rule_identity" json:"targetIP"`
	TargetPort string `gorm:"size:32;not null;uniqueIndex:idx_forwarding_rule_identity" json:"targetPort"`
	Interface  string `gorm:"size:32;not null;default:'';uniqueIndex:idx_forwarding_rule_identity" json:"interface"`
}

// ============================================================
// FirewallRule  统一防火墙 v2 规则表（持久化）
// ============================================================
// 字段:
//   - UUID (string) — 主键
//   - ScopeKey/Provider/Family/Location — 范围信息
//   - NativeKind/Protocol/Action/... — 规则内容
//   - RuleKey/Origin/Owner/MatchKey/Revision — 匹配/归属/版本
// ============================================================
type FirewallRule struct {
	UUID     string `gorm:"size:64;primaryKey" json:"uuid"`
	ScopeKey string `gorm:"size:255;not null;index" json:"scopeKey"`
	Provider string `gorm:"size:32;not null" json:"provider"`
	Family   string `gorm:"size:16;not null" json:"family"`
	Location string `gorm:"size:128;not null" json:"location"`

	NativeKind         string `gorm:"size:64;not null" json:"nativeKind"`
	Protocol           string `gorm:"size:32;not null" json:"protocol"`
	SourceAddress      string `gorm:"size:255" json:"sourceAddress"`
	SourcePort         string `gorm:"size:64" json:"sourcePort"`
	DestinationAddress string `gorm:"size:255" json:"destinationAddress"`
	DestinationPort    string `gorm:"size:64" json:"destinationPort"`
	Interface          string `gorm:"size:128" json:"interface"`
	ConnectionStates   string `gorm:"type:text" json:"connectionStates"`
	Action             string `gorm:"size:32;not null" json:"action"`
	Priority           *int   `json:"priority"`
	OrderIndex         *int64 `json:"orderIndex"`
	OrderBucket        string `gorm:"size:64" json:"orderBucket"`
	Description        string `gorm:"type:text" json:"description"`

	RuleKey  string `gorm:"size:80;not null" json:"ruleKey"`
	Origin   string `gorm:"size:32;not null" json:"origin"`
	Owner    string `gorm:"size:320;not null" json:"owner"`
	MatchKey string `gorm:"size:320" json:"matchKey"`
	Revision uint   `gorm:"not null;default:1" json:"revision"`
}

// ============================================================
// FirewallRuleOwner  工具：组拼"规则所有者" (kind:id)
// ============================================================
func FirewallRuleOwner(sourceKind, sourceID string) string {
	sourceKind = strings.TrimSpace(sourceKind)
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return sourceKind
	}
	return sourceKind + ":" + sourceID
}

// ============================================================
// FirewallRuleFromDomain  把 filter 域对象转成持久化结构
// ============================================================
// 流程:
//   1. 标准化 + 算 RuleKey
//   2. 把 Scope 的 location (zone/chain) 提取出来
//   3. 包装成 model.FirewallRule
// ============================================================
func FirewallRuleFromDomain(rule filter.FirewallRule) (FirewallRule, error) {
	normalized, err := filter.NormalizeRule(rule)
	if err != nil {
		return FirewallRule{}, err
	}
	ruleKey, err := filter.RuleKey(normalized)
	if err != nil {
		return FirewallRule{}, err
	}
	return FirewallRule{
		ScopeKey:           normalized.Scope.Key(),
		Provider:           string(normalized.Scope.Provider),
		Family:             string(normalized.Scope.Family),
		Location:           firewallRuleLocation(normalized.Scope),
		NativeKind:         string(normalized.NativeKind),
		Protocol:           normalized.Protocol,
		SourceAddress:      normalized.SourceAddress,
		SourcePort:         normalized.SourcePort,
		DestinationAddress: normalized.DestinationAddress,
		DestinationPort:    normalized.DestinationPort,
		Interface:          normalized.Interface,
		ConnectionStates:   strings.Join(normalized.ConnectionStates, ","),
		Action:             string(normalized.Action),
		Priority:           normalized.Priority,
		OrderIndex:         persistedFirewallOrderIndex(normalized),
		OrderBucket:        normalized.OrderBucket,
		Description:        normalized.Description,
		RuleKey:            ruleKey,
	}, nil
}

// ============================================================
// persistedFirewallOrderIndex  iptables/nftables/ufw 不持久化 OrderIndex
// ============================================================
func persistedFirewallOrderIndex(rule filter.FirewallRule) *int64 {
	switch rule.Scope.Provider {
	case filter.ProviderIptables, filter.ProviderNftables, filter.ProviderUFW:
		return nil
	default:
		return rule.OrderIndex
	}
}

// ============================================================
// firewallRuleLocation  工具：firewalld 用 zone，其他用 chain
// ============================================================
func firewallRuleLocation(scope filter.Scope) string {
	switch scope.Provider {
	case filter.ProviderFirewalld:
		return scope.Zone
	case filter.ProviderIptables, filter.ProviderNftables, filter.ProviderUFW:
		return scope.Chain
	default:
		return ""
	}
}

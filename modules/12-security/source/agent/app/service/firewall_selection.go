// =============================================================================
// 模块: Firewall 防火墙 (agent/app/service/firewall_selection.go)
// 文件: firewall_selection.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"strings"

	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/firewall/lifecycle"
)

// ============================================================
// selectedDockerFirewallBackend  决定 Docker 防火墙后端（iptables/nftables）
// ============================================================
// 优先级: DB 设置 > 自动 fallback
// ============================================================
func selectedDockerFirewallBackend(fallback string) string {
	selected := ""
	if global.DB != nil {
		selected, _ = settingRepo.GetValueByKey(constant.FirewallDockerBackendKey)
	}
	selected = strings.ToLower(strings.TrimSpace(selected))
	if selected == constant.FirewallProviderIptables || selected == constant.FirewallProviderNftables {
		return selected
	}
	fallback = strings.ToLower(strings.TrimSpace(fallback))
	if fallback == constant.FirewallProviderNftables {
		return fallback
	}
	return constant.FirewallProviderIptables
}

// ============================================================
// selectedSystemFirewallClient  拿系统防火墙 client (firewalld/ufw/iptables)
// ============================================================
func selectedSystemFirewallClient() (lifecycle.Client, error) {
	if provider, _ := settingRepo.GetValueByKey(constant.FirewallSystemBackendKey); strings.TrimSpace(provider) != "" {
		return lifecycle.NewClientFor(strings.TrimSpace(provider))
	}
	client, err := lifecycle.NewClient()
	if err != nil {
		return nil, err
	}
	_ = settingRepo.UpdateOrCreate(constant.FirewallSystemBackendKey, client.Name())
	return client, nil
}

func NewSelectedSystemFirewallClient() (lifecycle.Client, error) {
	return selectedSystemFirewallClient()
}

// ============================================================
// selectedSystemFirewallProvider  拿系统防火墙后端名
// ============================================================
func selectedSystemFirewallProvider() (string, error) {
	client, err := selectedSystemFirewallClient()
	if err != nil {
		return "", err
	}
	return client.Name(), nil
}

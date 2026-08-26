// =============================================================================
// 模块: AI Agent 智能体 (agent/app/dto/response/mcp_server.go)
// 文件: mcp_server.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import (
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/app/model"
)

// McpServersRes (struct)
type McpServersRes struct {
	Items []McpServerDTO `json:"items"`
	Total int64          `json:"total"`
}

type McpServerDTO struct {
	model.McpServer
	Environments []request.Environment `json:"environments"`
	Volumes      []request.Volume      `json:"volumes"`
}

type McpServerStatusDTO struct {
	ID      uint   `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type McpBindDomainRes struct {
	Domain        string   `json:"domain"`
	SSLID         uint     `json:"sslID"`
	AcmeAccountID uint     `json:"acmeAccountID"`
	AllowIPs      []string `json:"allowIPs"`
	WebsiteID     uint     `json:"websiteID"`
	ConnUrl       string   `json:"connUrl"`
}

type McpServerConnectionTestRes struct {
	Success         bool   `json:"success"`
	Endpoint        string `json:"endpoint"`
	OutputTransport string `json:"outputTransport"`
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	Message         string `json:"message"`
}

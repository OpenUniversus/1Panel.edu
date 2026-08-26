// =============================================================================
// 模块: AI Agent 智能体 (agent/app/dto/request/mcp_server.go)
// 文件: mcp_server.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package request

import "github.com/1Panel-dev/1Panel/agent/app/dto"

// McpServerSearch (struct)
type McpServerSearch struct {
	dto.PageInfo
	Name string `json:"name"`
	Sync bool   `json:"sync"`
}

type McpServerCreate struct {
	Name               string        `json:"name" validate:"required"`
	Command            string        `json:"command" validate:"required"`
	Environments       []Environment `json:"environments"`
	Volumes            []Volume      `json:"volumes"`
	Port               int           `json:"port" validate:"required"`
	ContainerName      string        `json:"containerName"`
	BaseURL            string        `json:"baseUrl"`
	SsePath            string        `json:"ssePath"`
	HostIP             string        `json:"hostIP"`
	StreamableHttpPath string        `json:"streamableHttpPath"`
	OutputTransport    string        `json:"outputTransport" validate:"required"`
	Type               string        `json:"type" validate:"required"`
	GatewayImage       string        `json:"gatewayImage"`
	ProtocolVersion    string        `json:"protocolVersion"`
	GatewayArgs        string        `json:"gatewayArgs" validate:"max=4096"`
	TaskID             string        `json:"taskID"`
}

type McpServerUpdate struct {
	ID uint `json:"id" validate:"required"`
	McpServerCreate
}

type McpServerDelete struct {
	ID uint `json:"id" validate:"required"`
}

// McpServerDetail (struct)
type McpServerDetail struct {
	ID uint `json:"id" validate:"required"`
}

type McpServerStatusSync struct {
	IDs []uint `json:"ids"`
}

type McpServerOperate struct {
	ID      uint   `json:"id" validate:"required"`
	Operate string `json:"operate" validate:"required"`
}

type McpServerConnectionTest struct {
	ID uint `json:"id" validate:"required"`
}

type McpBindDomain struct {
	Domain string `json:"domain" validate:"required"`
	SSLID  uint   `json:"sslID"`
	IPList string `json:"ipList"`
}

type McpBindDomainUpdate struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	SSLID     uint   `json:"sslID"`
	IPList    string `json:"ipList"`
}

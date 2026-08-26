// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/mcp_server.go)
// 文件: mcp_server.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// McpServer (struct)
type McpServer struct {
	BaseModel
	Name               string `json:"name"`
	DockerCompose      string `json:"dockerCompose"`
	Command            string `json:"command"`
	ContainerName      string `json:"containerName"`
	Message            string `json:"message"`
	Port               int    `json:"port"`
	Status             string `json:"status"`
	Env                string `json:"env"`
	BaseURL            string `json:"baseUrl"`
	SsePath            string `json:"ssePath"`
	WebsiteID          int    `json:"websiteID"`
	Dir                string `json:"dir"`
	HostIP             string `json:"hostIP"`
	StreamableHttpPath string `json:"streamableHttpPath"`
	OutputTransport    string `json:"outputTransport"`
	Type               string `json:"type"`
	GatewayImage       string `json:"gatewayImage"`
	ProtocolVersion    string `json:"protocolVersion"`
	GatewayArgs        string `json:"gatewayArgs"`
}

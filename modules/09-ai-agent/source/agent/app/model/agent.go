// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/agent.go)
// 文件: agent.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// Agent (struct)
type Agent struct {
	BaseModel
	Name          string `json:"name" gorm:"not null;unique"`
	Remark        string `json:"remark"`
	AgentType     string `json:"agentType" gorm:"default:openclaw"`
	Provider      string `json:"provider"`
	Model         string `json:"model"`
	APIType       string `json:"apiType"`
	MaxTokens     int    `json:"maxTokens"`
	ContextWindow int    `json:"contextWindow"`
	BaseURL       string `json:"baseUrl"`
	APIKey        string `json:"apiKey"`
	Token         string `json:"token"`
	Status        string `json:"status"`
	Message       string `json:"message"`
	AppInstallID  uint   `json:"appInstallId"`
	WebsiteID     uint   `json:"websiteId"`
	AccountID     uint   `json:"accountId"`
	ConfigPath    string `json:"configPath"`
}

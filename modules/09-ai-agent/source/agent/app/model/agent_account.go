// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/agent_account.go)
// 文件: agent_account.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AgentAccount (struct)
type AgentAccount struct {
	BaseModel
	Provider        string `json:"provider"`
	Name            string `json:"name"`
	APIKey          string `json:"apiKey"`
	BaseURL         string `json:"baseUrl"`
	APIType         string `json:"apiType"`
	AuthMode        string `json:"authMode"`
	VerifyModel     string `json:"verifyModel"`
	RememberAPIKey  bool   `json:"rememberApiKey"`
	Verified        bool   `json:"verified"`
	Remark          string `json:"remark"`
	MasterAccountID uint   `json:"masterAccountId" gorm:"index"`
}

func (AgentAccount) TableName() string {
	return "agent_provider_accounts"
}

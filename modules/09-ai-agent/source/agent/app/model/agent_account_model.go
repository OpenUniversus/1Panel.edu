// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/agent_account_model.go)
// 文件: agent_account_model.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AgentAccountModel (struct)
type AgentAccountModel struct {
	BaseModel
	AccountID uint   `json:"accountId" gorm:"index"`
	Model     string `json:"model" gorm:"index"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder" gorm:"index"`
}

func (AgentAccountModel) TableName() string {
	return "agent_account_models"
}

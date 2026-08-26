// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/ai.go)
// 文件: ai.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// OllamaModel (struct)
type OllamaModel struct {
	BaseModel

	Name    string `json:"name"`
	Size    string `json:"size"`
	From    string `json:"from"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

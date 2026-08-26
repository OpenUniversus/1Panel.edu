// =============================================================================
// 模块: Website 网站管理 (agent/app/model/compose_template.go)
// 文件: compose_template.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// ComposeTemplate (struct)
type ComposeTemplate struct {
	BaseModel

	Name        string `gorm:"not null;unique" json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

type Compose struct {
	BaseModel

	Name     string `json:"name"`
	Path     string `json:"path"`
	IsPinned bool   `json:"isPinned"`
}

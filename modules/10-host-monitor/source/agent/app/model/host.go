// =============================================================================
// 模块: Monitor 主机监控 (agent/app/model/host.go)
// 文件: host.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// Host (struct)
type Host struct {
	BaseModel

	GroupID          uint   `gorm:"not null" json:"group_id"`
	Name             string `gorm:"not null" json:"name"`
	Addr             string `gorm:"not null" json:"addr"`
	Port             int    `gorm:"not null" json:"port"`
	User             string `gorm:"not null" json:"user"`
	AuthMode         string `gorm:"not null" json:"authMode"`
	Password         string `json:"password"`
	PrivateKey       string `json:"privateKey"`
	PassPhrase       string `json:"passPhrase"`
	RememberPassword bool   `json:"rememberPassword"`

	Description string `json:"description"`
}

// =============================================================================
// 模块: Website 网站管理 (agent/app/model/website_ca.go)
// 文件: website_ca.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// WebsiteCA (struct)
type WebsiteCA struct {
	BaseModel
	CSR        string `gorm:"not null;" json:"csr"`
	Name       string `gorm:"not null;" json:"name"`
	PrivateKey string `gorm:"not null" json:"privateKey"`
	KeyType    string `gorm:"not null;default:RSA2048" json:"keyType"`
}

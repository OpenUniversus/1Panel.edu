// =============================================================================
// 模块: Website 网站管理 (agent/app/model/website_dns_account.go)
// 文件: website_dns_account.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// WebsiteDnsAccount (struct)
type WebsiteDnsAccount struct {
	BaseModel
	Name          string `gorm:"not null" json:"name"`
	Type          string `gorm:"not null" json:"type"`
	Authorization string `gorm:"not null" json:"-"`
}

func (w WebsiteDnsAccount) TableName() string {
	return "website_dns_accounts"
}

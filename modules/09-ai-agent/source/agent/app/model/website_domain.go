// =============================================================================
// 模块: AI Agent 智能体 (agent/app/model/website_domain.go)
// 文件: website_domain.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// WebsiteDomain (struct)
type WebsiteDomain struct {
	BaseModel
	WebsiteID uint   `gorm:"column:website_id;not null;" json:"websiteId"`
	Domain    string `gorm:"not null" json:"domain"`
	SSL       bool   `json:"ssl"`
	Port      int    `json:"port"`
}

func (w WebsiteDomain) TableName() string {
	return "website_domains"
}

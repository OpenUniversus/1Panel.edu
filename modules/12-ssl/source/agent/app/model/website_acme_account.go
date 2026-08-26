// =============================================================================
// 模块: SSL 证书管理 (agent/app/model/website_acme_account.go)
// 文件: website_acme_account.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// WebsiteAcmeAccount (struct)
type WebsiteAcmeAccount struct {
	BaseModel
	Email      string `gorm:"not null" json:"email"`
	URL        string `gorm:"not null" json:"url"`
	PrivateKey string `gorm:"not null" json:"-"`
	Type       string `gorm:"not null;default:letsencrypt" json:"type"`
	EabKid     string `json:"eabKid"`
	EabHmacKey string `json:"eabHmacKey"`
	KeyType    string `gorm:"not null;default:RSA2048" json:"keyType"`
	UseProxy   bool   `gorm:"default:false" json:"useProxy"`
	CaDirURL   string `json:"caDirURL"`
	UseEAB     bool   `json:"useEAB"`
}

func (w WebsiteAcmeAccount) TableName() string {
	return "website_acme_accounts"
}

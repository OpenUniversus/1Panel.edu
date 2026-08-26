// =============================================================================
// 模块: Website 网站管理 (agent/app/model/website.go)
// 文件: website.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

import "time"

// Website (struct)
type Website struct {
	BaseModel
	Protocol      string    `gorm:"not null" json:"protocol"`
	PrimaryDomain string    `gorm:"not null" json:"primaryDomain"`
	Type          string    `gorm:"not null" json:"type"`
	Alias         string    `gorm:"not null" json:"alias"`
	Remark        string    `json:"remark"`
	Status        string    `gorm:"not null" json:"status"`
	HttpConfig    string    `gorm:"not null" json:"httpConfig"`
	ExpireDate    time.Time `json:"expireDate"`

	Proxy         string `json:"proxy"`
	ProxyType     string `json:"proxyType"`
	SiteDir       string `json:"siteDir"`
	ErrorLog      bool   `json:"errorLog"`
	AccessLog     bool   `json:"accessLog"`
	DefaultServer bool   `json:"defaultServer"`
	IPV6          bool   `json:"IPV6"`
	Rewrite       string `json:"rewrite"`

	WebsiteGroupID  uint `json:"webSiteGroupId"`
	WebsiteSSLID    uint `json:"webSiteSSLId"`
	RuntimeID       uint `json:"runtimeID"`
	AppInstallID    uint `json:"appInstallId"`
	FtpID           uint `json:"ftpId"`
	ParentWebsiteID uint `json:"parentWebsiteID"`

	User  string `json:"user"`
	Group string `json:"group"`

	DbType string `json:"dbType"`
	DbID   uint   `json:"dbID"`

	Favorite bool `json:"favorite"`

	StreamPorts string `json:"streamPorts"`

	Domains    []WebsiteDomain `json:"domains" gorm:"-:migration"`
	WebsiteSSL WebsiteSSL      `json:"webSiteSSL" gorm:"-:migration"`
}

func (w Website) TableName() string {
	return "websites"
}

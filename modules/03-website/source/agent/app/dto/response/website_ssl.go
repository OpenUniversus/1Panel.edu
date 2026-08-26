// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/response/website_ssl.go)
// 文件: website_ssl.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import "github.com/1Panel-dev/1Panel/agent/app/model"

// WebsiteSSLDTO (struct)
type WebsiteSSLDTO struct {
	model.WebsiteSSL
	LogPath string `json:"logPath"`
}

type WebsiteDNSRes struct {
	Key    string `json:"resolve"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Err    string `json:"err"`
}

type WebsiteAcmeAccountDTO struct {
	model.WebsiteAcmeAccount
}

type WebsiteDnsAccountDTO struct {
	model.WebsiteDnsAccount
	Authorization map[string]string `json:"authorization"`
}

type WebsiteCADTO struct {
	model.WebsiteCA
	CommonName       string `json:"commonName" `
	Country          string `json:"country"`
	Organization     string `json:"organization"`
	OrganizationUint string `json:"organizationUint"`
	Province         string `json:"province" `
	City             string `json:"city"`
}

// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/response/website.go)
// 文件: website.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import (
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
)

// WebsiteDTO (struct)
type WebsiteDTO struct {
	model.Website
	ErrorLogPath  string `json:"errorLogPath"`
	AccessLogPath string `json:"accessLogPath"`
	SitePath      string `json:"sitePath"`
	AppName       string `json:"appName"`
	RuntimeName   string `json:"runtimeName"`
	RuntimeType   string `json:"runtimeType"`
	SiteDir       string `json:"siteDir"`
	OpenBaseDir   bool   `json:"openBaseDir"`
	Algorithm     string `json:"algorithm"`
	UDP           bool   `json:"udp"`

	Servers []dto.NginxUpstreamServer `json:"servers"`
}

type WebsiteRes struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	Protocol      string    `json:"protocol"`
	PrimaryDomain string    `json:"primaryDomain"`
	Type          string    `json:"type"`
	Alias         string    `json:"alias"`
	Remark        string    `json:"remark"`
	Status        string    `json:"status"`
	ExpireDate    time.Time `json:"expireDate"`
	SitePath      string    `json:"sitePath"`
	AppName       string    `json:"appName"`
	RuntimeName   string    `json:"runtimeName"`
	SSLExpireDate time.Time `json:"sslExpireDate"`
	SSLStatus     string    `json:"sslStatus"`
	AppInstallID  uint      `json:"appInstallId"`
	ChildSites    []string  `json:"childSites"`
	ParentSite    string    `json:"parentSite"`
	RuntimeType   string    `json:"runtimeType"`
	Favorite      bool      `json:"favorite"`
	IPV6          bool      `json:"IPV6"`
}

// WebsiteOption (struct)
type WebsiteOption struct {
	ID            uint   `json:"id"`
	PrimaryDomain string `json:"primaryDomain"`
	Alias         string `json:"alias"`
}

type WebsitePreInstallCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	AppName string `json:"appName"`
}

type WebsiteNginxConfig struct {
	Enable bool         `json:"enable"`
	Params []NginxParam `json:"params"`
}

type WebsiteHTTPS struct {
	Enable                bool             `json:"enable"`
	HttpConfig            string           `json:"httpConfig"`
	SSL                   model.WebsiteSSL `json:"SSL"`
	SSLProtocol           []string         `json:"SSLProtocol"`
	Algorithm             string           `json:"algorithm"`
	Hsts                  bool             `json:"hsts"`
	HstsIncludeSubDomains bool             `json:"hstsIncludeSubDomains"`
	HttpsPorts            []int            `json:"httpsPorts"`
	HttpsPort             string           `json:"httpsPort"`
	Http3                 bool             `json:"http3"`
}

type WebsiteLog struct {
	Enable  bool   `json:"enable"`
	Content string `json:"content"`
	End     bool   `json:"end"`
	Path    string `json:"path"`
}

// PHPConfig (struct)
type PHPConfig struct {
	Params           map[string]string `json:"params"`
	DisableFunctions []string          `json:"disableFunctions"`
	UploadMaxSize    string            `json:"uploadMaxSize"`
	MaxExecutionTime string            `json:"maxExecutionTime"`
}

type NginxRewriteRes struct {
	Content string `json:"content"`
}

type WebsiteDirConfig struct {
	Dirs      []string `json:"dirs"`
	User      string   `json:"user"`
	UserGroup string   `json:"userGroup"`
	Msg       string   `json:"msg"`
}

type WebsiteHtmlRes struct {
	Content string `json:"content"`
}

type WebsiteRealIP struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	Open      bool   `json:"open"`
	IPFrom    string `json:"ipFrom"`
	IPHeader  string `json:"ipHeader"`
	IPOther   string `json:"ipOther"`
}

type Resource struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	ResourceID uint        `json:"resourceID"`
	Detail     interface{} `json:"detail"`
}

// Database (struct)
type Database struct {
	Name         string `json:"name"`
	DatabaseName string `json:"databaseName"`
	Type         string `json:"type"`
	From         string `json:"from"`
	ID           uint   `json:"id"`
}

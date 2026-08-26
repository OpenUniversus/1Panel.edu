// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/response/nginx.go)
// 文件: nginx.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import "github.com/1Panel-dev/1Panel/agent/app/dto"

// NginxStatus (struct)
type NginxStatus struct {
	Active   int `json:"active"`
	Accepts  int `json:"accepts"`
	Handled  int `json:"handled"`
	Requests int `json:"requests"`
	Reading  int `json:"reading"`
	Writing  int `json:"writing"`
	Waiting  int `json:"waiting"`
}

type NginxParam struct {
	Name   string   `json:"name"`
	Params []string `json:"params"`
}

type NginxAuthRes struct {
	Enable bool            `json:"enable"`
	Items  []dto.NginxAuth `json:"items"`
}

type NginxPathAuthRes struct {
	dto.NginxPathAuth
}

type NginxAntiLeechRes struct {
	Enable      bool     `json:"enable"`
	Extends     string   `json:"extends"`
	Return      string   `json:"return"`
	ServerNames []string `json:"serverNames"`
	Cache       bool     `json:"cache"`
	CacheTime   int      `json:"cacheTime"`
	CacheUint   string   `json:"cacheUint"`
	NoneRef     bool     `json:"noneRef"`
	LogEnable   bool     `json:"logEnable"`
	Blocked     bool     `json:"blocked"`
}

// NginxRedirectConfig (struct)
type NginxRedirectConfig struct {
	WebsiteID    uint     `json:"websiteID"`
	Name         string   `json:"name"`
	Domains      []string `json:"domains"`
	KeepPath     bool     `json:"keepPath"`
	Enable       bool     `json:"enable"`
	Type         string   `json:"type"`
	Redirect     string   `json:"redirect"`
	Path         string   `json:"path"`
	Target       string   `json:"target"`
	FilePath     string   `json:"filePath"`
	Content      string   `json:"content"`
	RedirectRoot bool     `json:"redirectRoot"`
}

type NginxFile struct {
	Content string `json:"content"`
}

type NginxProxyCache struct {
	Open            bool    `json:"open"`
	CacheLimit      float64 `json:"cacheLimit"`
	CacheLimitUnit  string  `json:"cacheLimitUnit" `
	ShareCache      int     `json:"shareCache" `
	ShareCacheUnit  string  `json:"shareCacheUnit" `
	CacheExpire     int     `json:"cacheExpire" `
	CacheExpireUnit string  `json:"cacheExpireUnit" `
}

type NginxModule struct {
	Name        string                    `json:"name"`
	Custom      bool                      `json:"custom"`
	Script      string                    `json:"script"`
	Packages    string                    `json:"packages"`
	Params      string                    `json:"params"`
	Enable      bool                      `json:"enable"`
	BuildMode   string                    `json:"buildMode"`
	Provider    string                    `json:"provider"`
	LoadOrder   int                       `json:"loadOrder"`
	BuildStatus string                    `json:"buildStatus"`
	LoadStatus  string                    `json:"loadStatus"`
	Artifacts   []dto.NginxModuleArtifact `json:"artifacts"`
	LastError   string                    `json:"lastError"`
}

// NginxBuildConfig (struct)
type NginxBuildConfig struct {
	Mirror           string        `json:"mirror"`
	DynamicSupported bool          `json:"dynamicSupported"`
	Modules          []NginxModule `json:"modules"`
}

type NginxConfigRes struct {
	Https              bool `json:"https"`
	SSLRejectHandshake bool `json:"sslRejectHandshake"`
}

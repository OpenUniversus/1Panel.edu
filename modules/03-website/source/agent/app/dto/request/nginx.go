// =============================================================================
// 模块: Website 网站管理 (agent/app/dto/request/nginx.go)
// 文件: nginx.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package request

import "github.com/1Panel-dev/1Panel/agent/app/dto"

// NginxConfigFileUpdate (struct)
type NginxConfigFileUpdate struct {
	Content string `json:"content" validate:"required"`
	Backup  bool   `json:"backup"`
}

type NginxScopeReq struct {
	Scope     dto.NginxKey `json:"scope" validate:"required"`
	WebsiteID uint         `json:"websiteId"`
}

type NginxConfigUpdate struct {
	Scope     dto.NginxKey `json:"scope"`
	Operate   string       `json:"operate" validate:"required,oneof=add update delete"`
	WebsiteID uint         `json:"websiteId"`
	Params    interface{}  `json:"params"`
}

type NginxRewriteReq struct {
	WebsiteID uint   `json:"websiteId" validate:"required"`
	Name      string `json:"name" validate:"required"`
}

type CustomRewriteOperate struct {
	Operate string `json:"operate" validate:"required,oneof=create delete"`
	Content string `json:"content"`
	Name    string `json:"name"`
}

type NginxRewriteUpdate struct {
	WebsiteID uint   `json:"websiteId" validate:"required"`
	Name      string `json:"name" validate:"required"`
	Content   string `json:"content"`
}

// NginxProxyUpdate (struct)
type NginxProxyUpdate struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	Content   string `json:"content" validate:"required"`
	Name      string `json:"name" validate:"required"`
}

type NginxProxyCacheUpdate struct {
	WebsiteID       uint   `json:"websiteID" validate:"required"`
	Open            bool   `json:"open"`
	CacheLimit      int    `json:"cacheLimit" validate:"required"`
	CacheLimitUnit  string `json:"cacheLimitUnit" validate:"required"`
	ShareCache      int    `json:"shareCache" validate:"required"`
	ShareCacheUnit  string `json:"shareCacheUnit" validate:"required"`
	CacheExpire     int    `json:"cacheExpire" validate:"required"`
	CacheExpireUnit string `json:"cacheExpireUnit" validate:"required"`
}

type NginxAuthUpdate struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	Operate   string `json:"operate" validate:"required"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Remark    string `json:"remark"`
}

type NginxPathAuthUpdate struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	Operate   string `json:"operate" validate:"required"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Path      string `json:"path"`
	Remark    string `json:"remark"`
}

// NginxAuthReq (struct)
type NginxAuthReq struct {
	WebsiteID uint `json:"websiteID" validate:"required"`
}

type NginxCommonReq struct {
	WebsiteID uint `json:"websiteID" validate:"required"`
}

type NginxAntiLeechUpdate struct {
	WebsiteID   uint     `json:"websiteID" validate:"required"`
	Extends     string   `json:"extends"`
	Return      string   `json:"return"`
	Enable      bool     `json:"enable" `
	ServerNames []string `json:"serverNames"`
	Cache       bool     `json:"cache"`
	CacheTime   int      `json:"cacheTime"`
	CacheUint   string   `json:"cacheUint"`
	NoneRef     bool     `json:"noneRef"`
	LogEnable   bool     `json:"logEnable"`
	Blocked     bool     `json:"blocked"`
}

type NginxRedirectReq struct {
	Name         string   `json:"name" validate:"required"`
	WebsiteID    uint     `json:"websiteID" validate:"required"`
	Domains      []string `json:"domains"`
	KeepPath     bool     `json:"keepPath"`
	Enable       bool     `json:"enable"`
	Type         string   `json:"type" validate:"required"`
	Redirect     string   `json:"redirect" validate:"required"`
	Path         string   `json:"path"`
	Target       string   `json:"target" validate:"required"`
	Operate      string   `json:"operate" validate:"required"`
	RedirectRoot bool     `json:"redirectRoot"`
}

// NginxRedirectUpdate (struct)
type NginxRedirectUpdate struct {
	WebsiteID uint   `json:"websiteID" validate:"required"`
	Content   string `json:"content" validate:"required"`
	Name      string `json:"name" validate:"required"`
}

type NginxBuildReq struct {
	TaskID  string   `json:"taskID" validate:"required"`
	Mirror  string   `json:"mirror" validate:"required"`
	Modules []string `json:"modules"`
	Force   bool     `json:"force"`
}

type NginxModuleUpdate struct {
	Operate   string `json:"operate" validate:"required,oneof=create delete update"`
	Name      string `json:"name" validate:"required"`
	Script    string `json:"script"`
	Packages  string `json:"packages"`
	Enable    bool   `json:"enable"`
	Params    string `json:"params"`
	BuildMode string `json:"buildMode" validate:"omitempty,oneof=dynamic static"`
	Provider  string `json:"provider" validate:"omitempty,oneof=local prebuilt"`
	LoadOrder int    `json:"loadOrder" validate:"omitempty,min=0,max=9999"`
}

type NginxOperateReq struct {
	Operate string `json:"operate" validate:"required,oneof=enable disable"`
}

type NginxDefaultHTTPSUpdate struct {
	Operate            string `json:"operate" validate:"required,oneof=enable disable"`
	SSLRejectHandshake bool   `json:"sslRejectHandshake"`
}

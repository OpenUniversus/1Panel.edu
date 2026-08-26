// =============================================================================
// 模块: Website 网站管理 (agent/router/ro_website_acme_account.go)
// 文件: ro_website_acme_account.go — 路由注册 (website_acme_account 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// WebsiteAcmeAccountRouter (struct)
type WebsiteAcmeAccountRouter struct {
}

func (a *WebsiteAcmeAccountRouter) InitRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("websites/acme")

	baseApi := v2.ApiGroupApp.BaseApi
	{
		groupRouter.POST("/search", baseApi.PageWebsiteAcmeAccount)
		groupRouter.POST("", baseApi.CreateWebsiteAcmeAccount)
		groupRouter.POST("/del", baseApi.DeleteWebsiteAcmeAccount)
		groupRouter.POST("/update", baseApi.UpdateWebsiteAcmeAccount)
	}
}

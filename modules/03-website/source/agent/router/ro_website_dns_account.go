// =============================================================================
// 模块: Website 网站管理 (agent/router/ro_website_dns_account.go)
// 文件: ro_website_dns_account.go — 路由注册 (website_dns_account 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// WebsiteDnsAccountRouter (struct)
type WebsiteDnsAccountRouter struct {
}

func (a *WebsiteDnsAccountRouter) InitRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("websites/dns")

	baseApi := v2.ApiGroupApp.BaseApi
	{
		groupRouter.POST("/search", baseApi.PageWebsiteDnsAccount)
		groupRouter.POST("", baseApi.CreateWebsiteDnsAccount)
		groupRouter.POST("/update", baseApi.UpdateWebsiteDnsAccount)
		groupRouter.POST("/del", baseApi.DeleteWebsiteDnsAccount)
	}
}

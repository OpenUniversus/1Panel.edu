// =============================================================================
// 模块: Website 网站管理 (agent/router/ro_website_ssl.go)
// 文件: ro_website_ssl.go — 路由注册 (website_ssl 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// WebsiteSSLRouter (struct)
type WebsiteSSLRouter struct {
}

func (a *WebsiteSSLRouter) InitRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("websites/ssl")

	baseApi := v2.ApiGroupApp.BaseApi
	{
		groupRouter.POST("/search", baseApi.PageWebsiteSSL)
		groupRouter.POST("/list", baseApi.ListWebsiteSSL)
		groupRouter.POST("", baseApi.CreateWebsiteSSL)
		groupRouter.POST("/resolve", baseApi.GetDNSResolve)
		groupRouter.POST("/del", baseApi.DeleteWebsiteSSL)
		groupRouter.GET("/website/:websiteId", baseApi.GetWebsiteSSLByWebsiteId)
		groupRouter.GET("/:id", baseApi.GetWebsiteSSLById)
		groupRouter.POST("/update", baseApi.UpdateWebsiteSSL)
		groupRouter.POST("/push", baseApi.PushWebsiteSSLToNode)
		groupRouter.POST("/upload", baseApi.UploadWebsiteSSL)
		groupRouter.POST("/obtain", baseApi.ApplyWebsiteSSL)
		groupRouter.POST("/download", baseApi.DownloadWebsiteSSL)
		groupRouter.POST("/import", baseApi.ImportMasterSSL)
		groupRouter.POST("/upload/file", baseApi.UploadSSLFile)
	}
}

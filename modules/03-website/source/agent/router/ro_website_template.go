// =============================================================================
// 模块: Website 网站管理 (agent/router/ro_website_template.go)
// 文件: ro_website_template.go — 路由注册 (website_template 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// WebsiteTemplateRouter (struct)
type WebsiteTemplateRouter struct {
}

func (a *WebsiteTemplateRouter) InitRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("websites/templates")

	baseApi := v2.ApiGroupApp.BaseApi
	{
		groupRouter.POST("/search", baseApi.PageWebsiteTemplate)
		groupRouter.POST("", baseApi.CreateWebsiteTemplate)
		groupRouter.POST("/update", baseApi.UpdateWebsiteTemplate)
		groupRouter.POST("/del", baseApi.DeleteWebsiteTemplate)
		groupRouter.POST("/get", baseApi.GetWebsiteTemplate)
		groupRouter.POST("/upload", baseApi.UploadTemplateZip)
		groupRouter.POST("/preview", baseApi.PreviewWebsiteTemplate)
		groupRouter.POST("/outputs/search", baseApi.PageWebsiteTemplateOutput)
		groupRouter.POST("/outputs", baseApi.CreateWebsiteTemplateOutput)
		groupRouter.POST("/outputs/del", baseApi.DeleteWebsiteTemplateOutput)
		groupRouter.POST("/outputs/get", baseApi.GetWebsiteTemplateOutput)
	}
}

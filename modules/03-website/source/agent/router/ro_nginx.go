// =============================================================================
// 模块: Website 网站管理 (agent/router/ro_nginx.go)
// 文件: ro_nginx.go — 路由注册 (nginx 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// NginxRouter (struct)
type NginxRouter struct {
}

func (a *NginxRouter) InitRouter(Router *gin.RouterGroup) {
	groupRouter := Router.Group("openresty")

	baseApi := v2.ApiGroupApp.BaseApi
	{
		groupRouter.GET("", baseApi.GetNginx)
		groupRouter.POST("/scope", baseApi.GetNginxConfigByScope)
		groupRouter.POST("/update", baseApi.UpdateNginxConfigByScope)
		groupRouter.GET("/status", baseApi.GetNginxStatus)
		groupRouter.POST("/file", baseApi.UpdateNginxFile)
		groupRouter.POST("/build", baseApi.BuildNginx)
		groupRouter.POST("/modules/update", baseApi.UpdateNginxModule)
		groupRouter.GET("/modules", baseApi.GetNginxModules)
		groupRouter.POST("/https", baseApi.OperateDefaultHTTPs)
		groupRouter.GET("/https", baseApi.GetDefaultHTTPsStatus)
	}
}

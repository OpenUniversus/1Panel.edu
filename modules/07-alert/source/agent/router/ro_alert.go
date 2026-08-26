// =============================================================================
// 模块: Alert 告警 (agent/router/ro_alert.go)
// 文件: ro_alert.go — 路由注册 (alert 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// AlertRouter (struct)
type AlertRouter struct {
}

func (a *AlertRouter) InitRouter(Router *gin.RouterGroup) {
	alertRouter := Router.Group("alert")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		alertRouter.POST("", baseApi.CreateAlert)
		alertRouter.POST("/update", baseApi.UpdateAlert)
		alertRouter.POST("/search", baseApi.PageAlert)
		alertRouter.POST("/status", baseApi.UpdateAlertStatus)
		alertRouter.POST("/del", baseApi.DeleteAlert)
		alertRouter.GET("/disks/list", baseApi.GetDisks)
		alertRouter.POST("/logs/search", baseApi.PageAlertLogs)
		alertRouter.POST("/logs/clean", baseApi.CleanAlertLogs)
		alertRouter.GET("/clams/list", baseApi.GetClams)
		alertRouter.POST("/cronjob/list", baseApi.GetCronJobs)

		alertRouter.POST("/config/update", baseApi.UpdateAlertConfig)
		alertRouter.POST("/config/info", baseApi.GetAlertConfig)
		alertRouter.POST("/config/search", baseApi.PageAlertConfig)
		alertRouter.POST("/config/del", baseApi.DeleteAlertConfig)
		alertRouter.POST("/config/test", baseApi.TestAlertConfig)
	}
}

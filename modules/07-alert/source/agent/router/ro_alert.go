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

// ============================================================
// AlertRouter  告警路由
// ============================================================
// 方法:
//   - InitRouter(Router) — 注册 /alert/* 路由
// ============================================================
type AlertRouter struct {
}

// ============================================================
// InitRouter  注册 /alert/* 路由
// ============================================================
// 分组:
//   - /alert — 告警任务 CRUD + 启停 + 查/清日志
//   - /alert/disks/list, /alert/clams/list, /alert/cronjob/list — 关联资源
//   - /alert/config/* — 通知渠道 CRUD + 测试
// ============================================================
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

// =============================================================================
// 模块: CronJob 定时任务 (agent/router/ro_cronjob.go)
// 文件: ro_cronjob.go — 路由注册 (cronjob 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// CronjobRouter (struct)
type CronjobRouter struct{}

func (s *CronjobRouter) InitRouter(Router *gin.RouterGroup) {
	cmdRouter := Router.Group("cronjobs")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		cmdRouter.POST("", baseApi.CreateCronjob)
		cmdRouter.POST("/next", baseApi.LoadNextHandle)
		cmdRouter.POST("/import", baseApi.ImportCronjob)
		cmdRouter.POST("/export", baseApi.ExportCronjob)
		cmdRouter.POST("/load/info", baseApi.LoadCronjobInfo)
		cmdRouter.GET("/script/options", baseApi.LoadScriptOptions)
		cmdRouter.POST("/del", baseApi.DeleteCronjob)
		cmdRouter.POST("/stop", baseApi.StopCronJob)
		cmdRouter.POST("/update", baseApi.UpdateCronjob)
		cmdRouter.POST("/group/update", baseApi.UpdateCronjobGroup)
		cmdRouter.POST("/status", baseApi.UpdateCronjobStatus)
		cmdRouter.POST("/handle", baseApi.HandleOnce)
		cmdRouter.POST("/search", baseApi.SearchCronjob)
		cmdRouter.POST("/search/records", baseApi.SearchJobRecords)
		cmdRouter.POST("/records/log", baseApi.LoadRecordLog)
		cmdRouter.POST("/records/clean", baseApi.CleanRecord)
	}
}

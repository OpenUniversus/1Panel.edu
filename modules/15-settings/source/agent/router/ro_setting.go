// =============================================================================
// 模块: Settings 系统设置 (agent/router/ro_setting.go)
// 文件: ro_setting.go — 路由注册 (setting 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// SettingRouter (struct)
type SettingRouter struct{}

func (s *SettingRouter) InitRouter(Router *gin.RouterGroup) {
	settingRouter := Router.Group("settings")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		settingRouter.POST("/search", baseApi.GetSettingInfo)
		settingRouter.POST("/terminal/ai/search", baseApi.GetTerminalAISettingInfo)
		settingRouter.POST("/files/ai/search", baseApi.GetFileManageAISettingInfo)
		settingRouter.POST("/file-history/search", baseApi.GetFileHistorySettingInfo)
		settingRouter.GET("/website/dir", baseApi.LoadWebsiteDir)
		settingRouter.GET("/search/available", baseApi.GetSystemAvailable)
		settingRouter.POST("/update", baseApi.UpdateSetting)
		settingRouter.POST("/terminal/ai/update", baseApi.UpdateTerminalAISetting)
		settingRouter.POST("/files/ai/update", baseApi.UpdateFileManageAISetting)
		settingRouter.POST("/file-history/update", baseApi.UpdateFileHistorySetting)

		settingRouter.POST("/description/save", baseApi.SaveDescription)

		settingRouter.GET("/snapshot/load", baseApi.LoadSnapshotData)
		settingRouter.POST("/snapshot", baseApi.CreateSnapshot)
		settingRouter.POST("/snapshot/recreate", baseApi.RecreateSnapshot)
		settingRouter.POST("/snapshot/search", baseApi.SearchSnapshot)
		settingRouter.POST("/snapshot/import", baseApi.ImportSnapshot)
		settingRouter.POST("/snapshot/del", baseApi.DeleteSnapshot)
		settingRouter.POST("/snapshot/recover", baseApi.RecoverSnapshot)
		settingRouter.POST("/snapshot/rollback", baseApi.RollbackSnapshot)
		settingRouter.POST("/snapshot/description/update", baseApi.UpdateSnapDescription)

		settingRouter.GET("/basedir", baseApi.LoadBaseDir)

		settingRouter.POST("/ssh/check", baseApi.CheckLocalConn)
		settingRouter.GET("/ssh/conn", baseApi.LoadLocalConn)
		settingRouter.POST("/ssh/default", baseApi.SetDefaultIsConn)
		settingRouter.POST("/ssh", baseApi.SaveLocalConn)
		settingRouter.POST("/ssh/check/info", baseApi.CheckLocalConnByInfo)
	}
}

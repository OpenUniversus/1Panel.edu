// =============================================================================
// 模块: Settings 系统设置 (core/router/ro_setting.go)
// 文件: ro_setting.go — 路由注册 (setting 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/core/app/api/v2"
	"github.com/1Panel-dev/1Panel/core/middleware"
	"github.com/gin-gonic/gin"
)

// ============================================================
// SettingRouter  core 端系统设置路由
// ============================================================
// 方法: InitRouter(Router) — 注册 /settings/* 路由
// ============================================================
type SettingRouter struct{}

// ============================================================
// InitRouter  注册 /core/settings/* 路由（含 session/密码过期中间件）
// ============================================================
func (s *SettingRouter) InitRouter(Router *gin.RouterGroup) {
	baseApi := v2.ApiGroupApp.BaseApi

	authRouter := Router.Group("settings").
		Use(middleware.SessionAuth())
	{
		authRouter.POST("/search/base", baseApi.GetSettingBaseInfo)
	}

	settingRouter := Router.Group("settings").
		Use(middleware.SessionAuth()).
		Use(middleware.PasswordExpired())
	{
		settingRouter.POST("/search", baseApi.GetSettingInfo)
		settingRouter.POST("/terminal/search", baseApi.GetTerminalSettingInfo)
		settingRouter.GET("/search/available", baseApi.GetSystemAvailable)
		settingRouter.POST("/update", baseApi.UpdateSetting)
		settingRouter.POST("/terminal/update", baseApi.UpdateTerminalSetting)
		settingRouter.GET("/interface", baseApi.LoadInterfaceAddr)
		settingRouter.POST("/menu/update", baseApi.UpdateMenu)
		settingRouter.POST("/menu/default", baseApi.DefaultMenu)
		settingRouter.POST("/proxy/update", baseApi.UpdateProxy)
		settingRouter.POST("/bind/update", baseApi.UpdateBindInfo)
		settingRouter.POST("/port/update", baseApi.UpdatePort)
		settingRouter.POST("/ssl/update", baseApi.UpdateSSL)
		settingRouter.GET("/ssl/info", baseApi.LoadFromCert)
		settingRouter.POST("/ssl/download", baseApi.DownloadSSL)
		settingRouter.POST("/upgrade", baseApi.Upgrade)
		settingRouter.POST("/upgrade/notes", baseApi.GetNotesByVersion)
		settingRouter.GET("/upgrade/releases", baseApi.LoadRelease)
		settingRouter.GET("/upgrade", baseApi.GetUpgradeInfo)
		settingRouter.POST("/apps/store/update", baseApi.UpdateAppstoreConfig)
		settingRouter.GET("/apps/store/config", baseApi.GetAppstoreConfig)
		settingRouter.GET("/memo", baseApi.GetMemo)
		settingRouter.POST("/memo", baseApi.UpdateMemo)
	}

	internalRouter := Router.Group("settings")
	{
		internalRouter.POST("/ssl/reload", baseApi.ReloadSSL)
	}
}

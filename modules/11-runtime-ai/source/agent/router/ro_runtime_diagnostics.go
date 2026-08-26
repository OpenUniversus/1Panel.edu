// =============================================================================
// 模块: Runtime AI 运行时 (agent/router/ro_runtime_diagnostics.go)
// 文件: ro_runtime_diagnostics.go — 路由注册 (runtime_diagnostics 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// RuntimeDiagnosticsRouter (struct)
type RuntimeDiagnosticsRouter struct{}

func (s *RuntimeDiagnosticsRouter) InitRouter(Router *gin.RouterGroup) {
	diagnosticsRouter := Router.Group("hosts/diagnostics")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		diagnosticsRouter.GET("/summary", baseApi.LoadRuntimeDiagnosticsSummary)
		diagnosticsRouter.GET("/goroutines", baseApi.LoadRuntimeGoroutines)
		diagnosticsRouter.POST("/profiles", baseApi.CreateRuntimeProfile)
	}
}

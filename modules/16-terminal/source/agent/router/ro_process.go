// =============================================================================
// 模块: Terminal 终端 (agent/router/ro_process.go)
// 文件: ro_process.go — 路由注册 (process 模块)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package router

import (
	v2 "github.com/1Panel-dev/1Panel/agent/app/api/v2"
	"github.com/gin-gonic/gin"
)

// ============================================================
// ProcessRouter  进程管理路由（WebSocket + REST）
// ============================================================
// 方法:
//   - InitRouter(Router) — 注册 /process/* 路由
// ============================================================
type ProcessRouter struct {
}

// ============================================================
// InitRouter  注册 /process 路由组下的 4 个接口
// ============================================================
// 参数:
//   - Router (*gin.RouterGroup) — 父路由组
// 注册:
//   - GET  /process/ws — 进程 WebSocket 推送
//   - POST /process/stop — 杀进程
//   - POST /process/listening — 监听端口进程
//   - GET  /process/:pid — 进程详情
// ============================================================
func (f *ProcessRouter) InitRouter(Router *gin.RouterGroup) {
	processRouter := Router.Group("process")
	baseApi := v2.ApiGroupApp.BaseApi
	{
		processRouter.GET("/ws", baseApi.ProcessWs)
		processRouter.POST("/stop", baseApi.StopProcess)
		processRouter.POST("/listening", baseApi.GetListeningProcess)
		processRouter.GET("/:pid", baseApi.GetProcessInfoByPID)
	}
}

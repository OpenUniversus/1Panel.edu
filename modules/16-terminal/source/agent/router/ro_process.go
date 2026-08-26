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

// ProcessRouter (struct)
type ProcessRouter struct {
}

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

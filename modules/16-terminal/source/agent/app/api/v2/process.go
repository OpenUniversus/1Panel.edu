// =============================================================================
// 模块: Terminal 终端 (agent/app/api/v2/process.go)
// 文件: process.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	websocket2 "github.com/1Panel-dev/1Panel/agent/utils/websocket"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// ============================================================
// ProcessWs  打开 WebSocket 通道，让前端可以实时看到服务器进程列表
// ============================================================
// 作用:
//   - 把 HTTP 升级成 WebSocket 长连接
//   - 启动后台协程，循环读取/推送进程数据
// 参数:
//   - c (*gin.Context) — Gin 框架的请求上下文（包含 HTTP 请求和响应）
// 返回:
//   - 无；通过 WebSocket 持续推数据
// 流程:
//   1. 校验请求头，确认是 WebSocket 握手
//   2. 把 HTTP 连接升级为 WebSocket
//   3. 用 NewWsClient 包装连接并起读/写两个后台协程
// 调用: 前端 WebSocket 客户端 -> this; this -> websocket2.NewWsClient
// ============================================================

// @Tags Process
// @Summary Process ws
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /process/ws [get]
func (b *BaseApi) ProcessWs(c *gin.Context) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		helper.Success(c)
		return
	}
	ws, err := wsUpgrade.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	wsClient := websocket2.NewWsClient("processClient", ws)
	go wsClient.Read()
	go wsClient.Write()
}

// @Tags Process
// @Summary Stop Process
// @Param request body request.ProcessReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /process/stop [post]
// ============================================================
// StopProcess  根据进程 PID 强制结束一个进程
// ============================================================
// 参数:
//   - c (*gin.Context) — HTTP 请求上下文，body 里是请求体（含 PID）
// 流程:
//   1. 解析并校验请求体（ProcessReq）
//   2. 调 processService.StopProcess 真正去杀进程
//   3. 成功返 200，失败返 400
// 调用: 前端 "结束进程" 按钮 -> this; this -> processService.StopProcess
// ============================================================

// @x-panel-log {"bodyKeys":["PID"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"结束进程 [PID]","formatEN":"结束进程 [PID]"}
func (b *BaseApi) StopProcess(c *gin.Context) {
	var req request.ProcessReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := processService.StopProcess(req); err != nil {
		helper.BadRequest(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Process
// @Summary Get Process Info By PID
// @Param pid path int true "PID"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetProcessInfoByPID  按 PID 查一个进程的详细信息（CPU/内存/启动时间等）
// ============================================================
// 参数:
//   - c (*gin.Context) — URL 路径中带 :pid
// 流程:
//   1. 从 URL 里取 pid（int32）
//   2. 调 service 查 psutil 信息
//   3. 把结果返给前端
// 调用: 前端详情面板 -> this; this -> processService.GetProcessInfoByPID
// ============================================================

// @Router /process/{pid} [get]
func (b *BaseApi) GetProcessInfoByPID(c *gin.Context) {
	pid, err := helper.GetParamInt32("pid", c)
	if err != nil {
		helper.BadRequest(c, err)
		return
	}
	data, err := processService.GetProcessInfoByPID(pid)
	if err != nil {
		helper.BadRequest(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags Process
// @Summary Get Listening Process
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetListeningProcess  列出所有正在监听端口的进程（哪些程序占用了端口）
// ============================================================
// 参数:
//   - c (*gin.Context) — HTTP 请求上下文（可能带查询条件）
// 流程:
//   1. 调 processService.GetListeningProcess 读系统端口表
//   2. 把进程+端口信息打包返给前端
// 调用: 前端"端口占用"页 -> this; this -> processService.GetListeningProcess
// ============================================================

// @Router /process/listening [post]
func (b *BaseApi) GetListeningProcess(c *gin.Context) {
	procs, err := processService.GetListeningProcess(c)
	if err != nil {
		helper.BadRequest(c, err)
		return
	}
	helper.SuccessWithData(c, procs)
}

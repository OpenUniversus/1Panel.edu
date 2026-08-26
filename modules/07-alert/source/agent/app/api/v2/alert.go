// =============================================================================
// 模块: Alert 告警 (agent/app/api/v2/alert.go)
// 文件: alert.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"errors"
	"net/url"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/gin-gonic/gin"
)

const defaultAuditUser = "system"

// @Tags Alert
// @Summary Page alert
// @Accept json
// @Param request body dto.AlertSearch true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// PageAlert  分页查询告警任务列表
// ============================================================
// 参数:
//   - c (*gin.Context) — body 是 dto.AlertSearch
// 流程:
//   1. 校验请求体
//   2. 调 alertService.PageAlert 拿 (总数, 数据)
//   3. 包成 PageResult 返回
// 调用: 前端"告警列表" -> this; this -> alertService.PageAlert
// ============================================================

// @Router /alert/search [post]
func (b *BaseApi) PageAlert(c *gin.Context) {
	var req dto.AlertSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, alerts, err := alertService.PageAlert(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: alerts,
	})
}

// ============================================================
// GetAlerts  拉取所有告警任务（不分页）
// ============================================================
// 流程:
//   1. 调 alertService.GetAlerts
//   2. 直接把列表返给前端
// 调用: 前端下拉/选择器 -> this; this -> alertService.GetAlerts
// ============================================================
func (b *BaseApi) GetAlerts(c *gin.Context) {
	alerts, err := alertService.GetAlerts()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, alerts)
}

// @Tags Alert
// @Summary Create alert
// @Accept json
// @Param request body dto.AlertCreate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert [post]
// ============================================================
// CreateAlert  创建一个新的告警任务
// ============================================================
// 参数:
//   - c — body 是 dto.AlertCreate
// 流程:
//   1. 校验请求体
//   2. 拿操作人 (loadAuditUser)
//   3. 调 alertService.CreateAlert
// 调用: 前端"新建告警" -> this; this -> alertService.CreateAlert
// ============================================================

// @x-panel-log {"bodyKeys":["title"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"创建告警任务 [title]","formatEN":"create alert [title]"}
func (b *BaseApi) CreateAlert(c *gin.Context) {
	var req dto.AlertCreate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	err := alertService.CreateAlert(req, loadAuditUser(c))
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Alert
// @Summary Delete alert
// @Accept json
// @Param request body dto.DeleteRequest true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/del [post]
// ============================================================
// DeleteAlert  按 ID 删除一个告警任务
// ============================================================
// 流程:
//   1. 校验请求体（含 ID）
//   2. 调 alertService.DeleteAlert
// 调用: 前端"删除告警" -> this; this -> alertService.DeleteAlert
// ============================================================

// @x-panel-log {"bodyKeys":["id"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除告警任务 [id]","formatEN":"delete alert [id]"}
func (b *BaseApi) DeleteAlert(c *gin.Context) {
	var req dto.DeleteRequest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	err := alertService.DeleteAlert(req.ID)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Alert
// @Summary Update alert
// @Accept json
// @Param request body dto.AlertUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/update [post]
// ============================================================
// UpdateAlert  更新告警任务
// ============================================================
// 流程:
//   1. 校验请求体
//   2. 拿操作人
//   3. 调 alertService.UpdateAlert
// 调用: 前端"编辑告警" -> this; this -> alertService.UpdateAlert
// ============================================================

// @x-panel-log {"bodyKeys":["title"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新告警任务 [title]","formatEN":"update alert [title]"}
func (b *BaseApi) UpdateAlert(c *gin.Context) {
	var req dto.AlertUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := alertService.UpdateAlert(req, loadAuditUser(c)); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// GetAlert  按 ID 查单个告警任务详情
// ============================================================
// 流程:
//   1. 从 URL 拿 ID
//   2. 调 alertService.GetAlert
//   3. 把详情返给前端
// 调用: 前端"告警详情" -> this; this -> alertService.GetAlert
// ============================================================
func (b *BaseApi) GetAlert(c *gin.Context) {
	id, err := helper.GetParamID(c)
	if err != nil {
		helper.BadRequest(c, errors.New("no such id in request param"))
		return
	}
	alert, err := alertService.GetAlert(id)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, alert)
}

// @Tags Alert
// @Summary Update alert status
// @Accept json
// @Param request body dto.AlertUpdateStatus true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/status [post]
// ============================================================
// UpdateAlertStatus  启用/禁用一个告警任务
// ============================================================
// 流程:
//   1. 校验请求体（ID + status）
//   2. 调 alertService.UpdateStatus
// 调用: 前端"启停告警" -> this; this -> alertService.UpdateStatus
// ============================================================

// @x-panel-log {"bodyKeys":["id","status"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新告警任务 [id] 状态 [status]","formatEN":"update alert [id] status [status]"}
func (b *BaseApi) UpdateAlertStatus(c *gin.Context) {
	var req dto.AlertUpdateStatus
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := alertService.UpdateStatus(req.ID, req.Status); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Alert
// @Summary Get disks
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetDisks  列出所有磁盘（用于磁盘空间告警的"选择监控对象"）
// ============================================================
// 流程:
//   1. 调 alertService.GetDisks 拿系统磁盘列表
//   2. 返给前端选择器
// 调用: 前端"新建磁盘告警" -> this; this -> alertService.GetDisks
// ============================================================

// @Router /alert/disks/list [get]
func (b *BaseApi) GetDisks(c *gin.Context) {
	alerts, err := alertService.GetDisks()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, alerts)
}

// @Tags Alert
// @Summary Page alert logs
// @Accept json
// @Param request body dto.AlertLogSearch true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// PageAlertLogs  分页查询告警日志（每次告警触发都记一条）
// ============================================================
// 流程:
//   1. 校验请求体
//   2. 调 alertService.PageAlertLogs 拿 (总数, 日志)
//   3. 包成 PageResult
// 调用: 前端"告警日志" -> this; this -> alertService.PageAlertLogs
// ============================================================

// @Router /alert/logs/search [post]
func (b *BaseApi) PageAlertLogs(c *gin.Context) {
	var req dto.AlertLogSearch
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, alertLogs, err := alertService.PageAlertLogs(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: alertLogs,
	})
}

// @Tags Alert
// @Summary Clean alert logs
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/logs/clean [post]
// ============================================================
// CleanAlertLogs  清空告警日志
// ============================================================
// 流程:
//   1. 调 alertService.CleanAlertLogs
//   2. 成功返 200
// 调用: 前端"清空告警日志" -> this; this -> alertService.CleanAlertLogs
// ============================================================

// @x-panel-log {"bodyKeys":[],"paramKeys":[],"BeforeFunctions":[],"formatZH":"清空告警日志","formatEN":"clean alert logs"}
func (b *BaseApi) CleanAlertLogs(c *gin.Context) {
	if err := alertService.CleanAlertLogs(); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Alert
// @Summary Get clams
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetClams  列出 ClamAV 杀毒任务（用于"病毒告警"的来源选择）
// ============================================================
// 流程:
//   1. 调 alertService.GetClams
//   2. 返给前端
// 调用: 前端"新建病毒告警" -> this; this -> alertService.GetClams
// ============================================================

// @Router /alert/clams/list [get]
func (b *BaseApi) GetClams(c *gin.Context) {
	clams, err := alertService.GetClams()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, clams)
}

// @Tags Alert
// @Summary Get cron jobs
// @Accept json
// @Param request body dto.CronJobReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetCronJobs  列出可关联到告警的定时任务
// ============================================================
// 流程:
//   1. 校验请求体 (CronJobReq)
//   2. 调 alertService.GetCronJobs
// 调用: 前端"新建任务告警" -> this; this -> alertService.GetCronJobs
// ============================================================

// @Router /alert/cronjob/list [post]
func (b *BaseApi) GetCronJobs(c *gin.Context) {
	var req dto.CronJobReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	cronJobs, err := alertService.GetCronJobs(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, cronJobs)
}

// @Tags Alert
// @Summary Get alert config
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetAlertConfig  按 ID 查告警通知渠道配置
// ============================================================
// 流程:
//   1. 校验请求体 (AlertConfigQuery)
//   2. 调 alertService.GetAlertConfig
// 调用: 前端"通知渠道详情" -> this; this -> alertService.GetAlertConfig
// ============================================================

// @Router /alert/config/info [post]
func (b *BaseApi) GetAlertConfig(c *gin.Context) {
	var req dto.AlertConfigQuery
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	config, err := alertService.GetAlertConfig(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, config)
}

// @Tags Alert
// @Summary Page alert config
// @Accept json
// @Param request body dto.AlertConfigPageReq true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// PageAlertConfig  分页查询告警通知渠道配置
// ============================================================
// 流程:
//   1. 校验请求体
//   2. 调 alertService.PageAlertConfig
//   3. 包成 PageResult
// 调用: 前端"通知渠道列表" -> this; this -> alertService.PageAlertConfig
// ============================================================

// @Router /alert/config/search [post]
func (b *BaseApi) PageAlertConfig(c *gin.Context) {
	var req dto.AlertConfigPageReq
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	total, configs, err := alertService.PageAlertConfig(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, dto.PageResult{
		Total: total,
		Items: configs,
	})
}

// @Tags Alert
// @Summary Update alert config
// @Accept json
// @Param request body dto.AlertConfigUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/config/update [post]
// ============================================================
// UpdateAlertConfig  更新告警通知渠道配置
// ============================================================
// 流程:
//   1. 校验请求体
//   2. 拿操作人
//   3. 调 alertService.UpdateAlertConfig
// 调用: 前端"编辑通知渠道" -> this; this -> alertService.UpdateAlertConfig
// ============================================================

// @x-panel-log {"bodyKeys":["id","displayName"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"更新告警配置 [id][displayName]","formatEN":"update alert config [id][displayName]"}
func (b *BaseApi) UpdateAlertConfig(c *gin.Context) {
	var req dto.AlertConfigUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := alertService.UpdateAlertConfig(req, loadAuditUser(c)); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// loadAuditUser  从请求头 X-Panel-User 拿"操作人"用于审计日志
// ============================================================
// 参数:
//   - c (*gin.Context) — HTTP 请求上下文
// 返回:
//   - string — 操作人用户名（找不到就返 "system"）
// 流程:
//   1. 读 X-Panel-User
//   2. 空白时返默认 "system"
//   3. URL 解码（因为用户名可能含中文）
// ============================================================
func loadAuditUser(c *gin.Context) string {
	userName := strings.TrimSpace(c.GetHeader("X-Panel-User"))
	if userName == "" {
		return defaultAuditUser
	}
	if decoded, err := url.QueryUnescape(userName); err == nil {
		return decoded
	}
	return userName
}

// @Tags Alert
// @Summary Delete alert config
// @Accept json
// @Param request body dto.DeleteRequest true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /alert/config/del [post]
// ============================================================
// DeleteAlertConfig  按 ID 删除一个告警通知渠道
// ============================================================
// 流程:
//   1. 校验请求体（含 ID）
//   2. 调 alertService.DeleteAlertConfig
// 调用: 前端"删除通知渠道" -> this; this -> alertService.DeleteAlertConfig
// ============================================================

// @x-panel-log {"bodyKeys":["id"],"paramKeys":[],"BeforeFunctions":[],"formatZH":"删除告警配置 [id]","formatEN":"delete alert config [id]"}
func (b *BaseApi) DeleteAlertConfig(c *gin.Context) {
	var req dto.DeleteRequest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	err := alertService.DeleteAlertConfig(req.ID)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Alert
// @Summary Test alert config
// @Accept json
// @Param request body dto.AlertConfigTest true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// TestAlertConfig  测试一个告警通知渠道（发一条测试消息）
// ============================================================
// 流程:
//   1. 校验请求体
//   2. 调 alertService.TestAlertConfig
//   3. 把"测试是否成功"返给前端
// 调用: 前端"测试通知" -> this; this -> alertService.TestAlertConfig
// ============================================================

// @Router /alert/config/test [post]
func (b *BaseApi) TestAlertConfig(c *gin.Context) {
	var req dto.AlertConfigTest
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	flag, err := alertService.TestAlertConfig(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, flag)
}

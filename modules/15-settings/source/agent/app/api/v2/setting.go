// =============================================================================
// 模块: Settings 系统设置 (agent/app/api/v2/setting.go)
// 文件: setting.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/dto/request"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/ssh"
	"github.com/gin-gonic/gin"
)

// @Tags System Setting
// @Summary Load system setting info
// @Success 200 {object} dto.SettingInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetSettingInfo  拿系统设置详情
// ============================================================
func (b *BaseApi) GetSettingInfo(c *gin.Context) {
	setting, err := settingService.GetSettingInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// @Tags System Setting
// @Summary Get terminal AI setting info
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetTerminalAISettingInfo  拿终端 AI 配置
// ============================================================
func (b *BaseApi) GetTerminalAISettingInfo(c *gin.Context) {
	setting, err := settingService.GetTerminalAIInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// @Tags System Setting
// @Summary Load system available status
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetSystemAvailable  健康检查（永远返 200）
// ============================================================
func (b *BaseApi) GetSystemAvailable(c *gin.Context) {
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update system setting
// @Accept json
// @Param request body dto.AgentSettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /settings/update [post]
// ============================================================
// UpdateSetting  更新系统配置项（key/value）
// ============================================================
func (b *BaseApi) UpdateSetting(c *gin.Context) {
	var req dto.AgentSettingUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.Update(req.Key, req.Value); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update terminal AI setting
// @Accept json
// @Param request body dto.TerminalAIInfo true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /settings/terminal/ai/update [post]
// ============================================================
// UpdateTerminalAISetting  更新终端 AI 配置
// ============================================================
func (b *BaseApi) UpdateTerminalAISetting(c *gin.Context) {
	var req dto.TerminalAIInfo
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.UpdateTerminalAI(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Get file manage AI setting info
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetFileManageAISettingInfo  拿文件管理 AI 配置
// ============================================================
func (b *BaseApi) GetFileManageAISettingInfo(c *gin.Context) {
	setting, err := settingService.GetFileManageAIInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// @Tags System Setting
// @Summary Update file manage AI setting
// @Accept json
// @Param request body dto.FileManageAIInfo true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /settings/files/ai/update [post]
// ============================================================
// UpdateFileManageAISetting  更新文件管理 AI 配置
// ============================================================
func (b *BaseApi) UpdateFileManageAISetting(c *gin.Context) {
	var req dto.FileManageAIInfo
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := settingService.UpdateFileManageAI(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Load file history setting info
// @Success 200 {object} response.FileHistorySettingInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetFileHistorySettingInfo  拿文件历史记录设置
// ============================================================
func (b *BaseApi) GetFileHistorySettingInfo(c *gin.Context) {
	setting, err := settingService.GetFileHistorySettingInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// @Tags System Setting
// @Summary Update file history setting
// @Accept json
// @Param request body request.FileHistorySettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// UpdateFileHistorySetting  更新文件历史记录设置
// ============================================================
func (b *BaseApi) UpdateFileHistorySetting(c *gin.Context) {
	var req request.FileHistorySettingUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := settingService.UpdateFileHistorySetting(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Load website dir
// @Success 200 {string} path
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadWebsiteDir  拿网站根目录
// ============================================================
func (b *BaseApi) LoadWebsiteDir(c *gin.Context) {
	helper.SuccessWithData(c, settingService.GetWebsiteDir())
}

// @Tags System Setting
// @Summary Load local backup dir
// @Success 200 {string} path
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadBaseDir  拿 1Panel 数据根目录
// ============================================================
func (b *BaseApi) LoadBaseDir(c *gin.Context) {
	helper.SuccessWithData(c, global.Dir.DataDir)
}

// @Tags System Setting
// @Summary Load local conn
// @Success 200 {object} dto.SSHConnData
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadLocalConn  拿本机 SSH 连接信息
// ============================================================
func (b *BaseApi) LoadLocalConn(c *gin.Context) {
	helper.SuccessWithData(c, settingService.GetLocalConn())
}

// @Tags System Setting
// @Summary Check local conn
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// CheckLocalConn  测试本机 SSH 连接是否可用
// ============================================================
func (b *BaseApi) CheckLocalConn(c *gin.Context) {
	client, err := loadLocalConn()
	if err == nil && client != nil {
		client.Close()
	}
	helper.SuccessWithData(c, err == nil)
}

// @Tags System Setting
// @Summary Update local is conn
// @Accept json
// @Param request body dto.SSHDefaultConn true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /settings/ssh/default [post]
// ============================================================
// SetDefaultIsConn  设定"默认是否直连本机"标志
// ============================================================
func (b *BaseApi) SetDefaultIsConn(c *gin.Context) {
	var req dto.SSHDefaultConn
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.SetDefaultIsConn(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Check local conn info
// @Success 200 {boolean} isOk
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// CheckLocalConnByInfo  按指定连接信息测试 SSH 连通性
// ============================================================
func (b *BaseApi) CheckLocalConnByInfo(c *gin.Context) {
	var req dto.SSHConnData
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	helper.SuccessWithData(c, settingService.TestConnByInfo(req))
}

// @Tags System Setting
// @Summary Save local conn info
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// SaveLocalConn  保存本机 SSH 连接信息
// ============================================================
func (b *BaseApi) SaveLocalConn(c *gin.Context) {
	var req dto.SSHConnData
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.SaveConnInfo(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// loadLocalConn  工具：拿 DB 里的本机连接信息并建 SSH 客户端
// ============================================================
func loadLocalConn() (*ssh.SSHClient, error) {
	connInDB, err := settingService.GetLocalConnForSSH()
	if err != nil {
		return nil, err
	}
	sshInfo := ssh.ConnInfo{
		Addr:       connInDB.Addr,
		Port:       int(connInDB.Port),
		User:       connInDB.User,
		AuthMode:   connInDB.AuthMode,
		Password:   connInDB.Password,
		PrivateKey: []byte(connInDB.PrivateKey),
		PassPhrase: []byte(connInDB.PassPhrase),
	}
	return ssh.NewClient(sshInfo)
}

// @Tags System Setting
// @Summary Save common description
// @Accept json
// @Param request body dto.CommonDescription true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// SaveDescription  保存常用描述（个人简介、面板说明等）
// ============================================================
func (b *BaseApi) SaveDescription(c *gin.Context) {
	var req dto.CommonDescription
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.SaveDescription(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

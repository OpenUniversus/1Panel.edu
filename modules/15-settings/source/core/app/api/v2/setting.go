// =============================================================================
// 模块: Settings 系统设置 (core/app/api/v2/setting.go)
// 文件: setting.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package v2

import (
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	appauth "github.com/1Panel-dev/1Panel/core/app/auth"
	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1Panel-dev/1Panel/core/buserr"
	"github.com/1Panel-dev/1Panel/core/constant"
	"github.com/1Panel-dev/1Panel/core/global"
	"github.com/1Panel-dev/1Panel/core/utils/common"
	"github.com/gin-gonic/gin"
)

// @Tags System Setting
// @Summary Load system setting info
// @Success 200 {object} dto.SettingInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetSettingInfo  拿系统设置详情（core 端）
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
// @Summary Load base system setting info
// @Success 200 {object} dto.SettingBaseInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetSettingBaseInfo  拿基础设置（不含敏感字段）
// ============================================================
func (b *BaseApi) GetSettingBaseInfo(c *gin.Context) {
	setting, err := settingService.GetSettingBaseInfo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, setting)
}

// @Tags System Setting
// @Summary Load system terminal setting info
// @Success 200 {object} dto.TerminalInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetTerminalSettingInfo  拿终端设置
// ============================================================
func (b *BaseApi) GetTerminalSettingInfo(c *gin.Context) {
	setting, err := settingService.GetTerminalInfo()
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
// GetSystemAvailable  健康检查
// ============================================================
func (b *BaseApi) GetSystemAvailable(c *gin.Context) {
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update system setting
// @Accept json
// @Param request body dto.SettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/update [post]
// ============================================================
// UpdateSetting  更新系统配置（含值范围/格式校验）
// ============================================================
func (b *BaseApi) UpdateSetting(c *gin.Context) {
	var req dto.SettingUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if req.Key == "SecurityEntrance" {
		if !checkEntrancePattern(req.Value) {
			helper.ErrorWithDetail(c, http.StatusBadRequest, "ErrInvalidParams", buserr.WithName("ErrEntranceFormat", req.Value))
			return
		}
	}
	if !checkSettingValueRange(req.Key, req.Value) {
		helper.ErrorWithDetail(c, http.StatusBadRequest, "ErrInvalidParams", buserr.WithName("ErrInvalidParams", req.Value))
		return
	}
	if req.Key == "PasskeyTrustedProxies" {
		value, err := normalizePasskeyTrustedProxies(req.Value)
		if err != nil {
			helper.BadRequest(c, err)
			return
		}
		req.Value = value
	}
	if req.Key == "AllowIPTrustedProxies" {
		value, err := common.NormalizeTrustedProxies(req.Value)
		if err != nil {
			helper.BadRequest(c, err)
			return
		}
		req.Value = value
	}

	if err := settingService.Update(c, req.Key, req.Value); err != nil {
		helper.InternalServer(c, err)
		return
	}
	if req.Key == "SecurityEntrance" {
		appauth.SetSecurityEntranceCookie(c, req.Value)
	}
	helper.Success(c)
}

// ============================================================
// checkSettingValueRange  工具：按 key 校验值范围（SessionTimeout/ExpirationDays）
// ============================================================
func checkSettingValueRange(key, value string) bool {
	switch key {
	case "SessionTimeout":
		valueNum, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		return valueNum >= 300 && valueNum <= 864000
	case "ExpirationDays":
		valueNum, err := strconv.Atoi(value)
		if err != nil {
			return false
		}
		return valueNum >= 0 && valueNum <= 60
	default:
		return true
	}
}

// @Tags System Setting
// @Summary Update system terminal setting
// @Accept json
// @Param request body dto.TerminalInfo true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/terminal/update [post]
// ============================================================
// UpdateTerminalSetting  更新终端配置
// ============================================================
func (b *BaseApi) UpdateTerminalSetting(c *gin.Context) {
	var req dto.TerminalInfo
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.UpdateTerminal(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update proxy setting
// @Accept json
// @Param request body dto.ProxyUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/proxy/update [post]
// ============================================================
// UpdateProxy  更新服务器代理设置
// ============================================================
func (b *BaseApi) UpdateProxy(c *gin.Context) {
	var req dto.ProxyUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if len(req.ProxyPasswd) != 0 && len(req.ProxyType) != 0 {
		pass, err := base64.StdEncoding.DecodeString(req.ProxyPasswd)
		if err != nil {
			helper.BadRequest(c, err)
			return
		}
		req.ProxyPasswd = string(pass)
	}

	if err := settingService.UpdateProxy(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update system setting
// @Accept json
// @Param request body dto.SettingUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/menu/update [post]
// ============================================================
// UpdateMenu  隐藏/显示高级功能菜单
// ============================================================
func (b *BaseApi) UpdateMenu(c *gin.Context) {
	var req dto.SettingUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.Update(c, req.Key, req.Value); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags Menu Setting
// @Summary Default menu
// @Accept json
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/menu/default [post]
// ============================================================
// DefaultMenu  初始化菜单（首次使用）
// ============================================================
func (b *BaseApi) DefaultMenu(c *gin.Context) {
	if err := settingService.DefaultMenu(); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update system ssl
// @Accept json
// @Param request body dto.SSLUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/ssl/update [post]
// ============================================================
// UpdateSSL  更新系统 SSL 证书
// ============================================================
func (b *BaseApi) UpdateSSL(c *gin.Context) {
	var req dto.SSLUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.UpdateSSL(c, req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Load system cert info
// @Success 200 {object} dto.SSLInfo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadFromCert  从系统证书文件加载 SSL 信息
// ============================================================
func (b *BaseApi) LoadFromCert(c *gin.Context) {
	info, err := settingService.LoadFromCert()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, info)
}

// @Tags System Setting
// @Summary Download system cert
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// DownloadSSL  下载系统证书文件
// ============================================================
func (b *BaseApi) DownloadSSL(c *gin.Context) {
	pathItem := path.Join(global.CONF.Base.InstallDir, "1panel/secret/server.crt")
	if _, err := os.Stat(pathItem); err != nil {
		helper.InternalServer(c, err)
		return
	}

	c.File(pathItem)
}

// @Tags System Setting
// @Summary Load system address
// @Accept json
// @Success 200 {array} string
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// LoadInterfaceAddr  拿本机所有网络接口地址
// ============================================================
func (b *BaseApi) LoadInterfaceAddr(c *gin.Context) {
	data, err := settingService.LoadInterfaceAddr()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, data)
}

// @Tags System Setting
// @Summary Update system bind info
// @Accept json
// @Param request body dto.BindInfo true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/bind/update [post]
// ============================================================
// UpdateBindInfo  更新系统监听地址/启用 ipv6
// ============================================================
func (b *BaseApi) UpdateBindInfo(c *gin.Context) {
	var req dto.BindInfo
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.UpdateBindInfo(req); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Update system port
// @Accept json
// @Param request body dto.PortUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/port/update [post]
// ============================================================
// UpdatePort  修改系统监听端口
// ============================================================
func (b *BaseApi) UpdatePort(c *gin.Context) {
	var req dto.PortUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}

	if err := settingService.UpdatePort(req.ServerPort); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags System Setting
// @Summary Reload SSL
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/ssl/reload [post]
// ============================================================
// ReloadSSL  重载系统 SSL（仅允许本机调用）
// ============================================================
func (b *BaseApi) ReloadSSL(c *gin.Context) {
	clientIP := c.ClientIP()
	if ip := net.ParseIP(clientIP); ip == nil || !ip.IsLoopback() {
		helper.InternalServer(c, errors.New("only localhost can reload ssl"))
		return
	}
	if err := settingService.UpdateSystemSSL(); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags App
// @Summary Update appstore config
// @Accept json
// @Param request body dto.AppstoreUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// UpdateAppstoreConfig  更新应用商店地址
// ============================================================
func (b *BaseApi) UpdateAppstoreConfig(c *gin.Context) {
	var req dto.AppstoreUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	err := settingService.UpdateAppstoreConfig(req)
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// @Tags App
// @Summary Get appstore config
// @Success 200 {object} dto.AppstoreConfig
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetAppstoreConfig  拿应用商店配置
// ============================================================
func (b *BaseApi) GetAppstoreConfig(c *gin.Context) {
	res, err := settingService.GetAppstoreConfig()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, res)
}

// @Tags System Setting
// @Summary Load dashboard memo
// @Success 200 {string} memo
// @Security ApiKeyAuth
// @Security Timestamp
// ============================================================
// GetMemo  拿仪表盘备忘录
// ============================================================
func (b *BaseApi) GetMemo(c *gin.Context) {
	memo, err := settingService.GetMemo()
	if err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.SuccessWithData(c, memo)
}

// @Tags System Setting
// @Summary Update dashboard memo
// @Accept json
// @Param request body dto.MemoUpdate true "request"
// @Success 200
// @Security ApiKeyAuth
// @Security Timestamp
// @Router /core/settings/memo [post]
// ============================================================
// UpdateMemo  更新仪表盘备忘录
// ============================================================
func (b *BaseApi) UpdateMemo(c *gin.Context) {
	var req dto.MemoUpdate
	if err := helper.CheckBindAndValidate(&req, c); err != nil {
		return
	}
	if err := settingService.UpdateMemo(req.Content); err != nil {
		helper.InternalServer(c, err)
		return
	}
	helper.Success(c)
}

// ============================================================
// checkEntrancePattern  校验安全入口（5-116 位字母数字，且不与系统路径冲突）
// ============================================================
func checkEntrancePattern(val string) bool {
	if len(val) == 0 {
		return true
	}
	result, _ := regexp.MatchString("^[a-zA-Z0-9]{5,116}$", val)
	if !result {
		return false
	}
	lowerVal := strings.ToLower(val)
	for key := range constant.WebUrlMap {
		if key == "/" || !strings.HasPrefix(key, "/") {
			continue
		}
		if strings.Count(key, "/") != 1 {
			continue
		}
		segment := strings.ToLower(strings.TrimPrefix(key, "/"))
		if len(segment) < 5 {
			continue
		}
		if lowerVal == segment {
			return false
		}
	}
	assetsList := [2]string{"public", "assets"}
	for _, item := range assetsList {
		if lowerVal == item {
			return false
		}
	}
	return true
}

// ============================================================
// normalizePasskeyTrustedProxies  标准化 Passkey 可信代理（多行 IP/CIDR）
// ============================================================
func normalizePasskeyTrustedProxies(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	lines := strings.Split(value, "\n")
	validLines := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		validLines = append(validLines, line)
	}
	if len(validLines) == 0 {
		return "", nil
	}
	normalized := strings.Join(validLines, "\n")
	if _, err := common.HandleIPList(normalized); err != nil {
		return "", err
	}
	return normalized, nil
}

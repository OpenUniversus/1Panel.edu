// =============================================================================
// 模块: Alert 告警 (agent/app/service/alert_sender.go)
// 文件: alert_sender.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"strconv"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	alertUtil "github.com/1Panel-dev/1Panel/agent/utils/alert"
	"github.com/1Panel-dev/1Panel/agent/utils/xpack"
)

// ============================================================
// AlertSender  告警发送器（按渠道类型分发到 SMS/Email/Bark/Webhook）
// ============================================================
// 字段:
//   - alert (dto.AlertDTO) — 要发的告警
//   - quotaType (string) — 阈值类型（disk/node-error 等）
// 方法:
//   - Send / ResourceSend — 入口
//   - sendXxxWithConfig / sendXxx — 各渠道实现
//   - canSendAlert / canResourceSendAlert — 频次控制
// ============================================================
type AlertSender struct {
	alert     dto.AlertDTO
	quotaType string
}

// ============================================================
// NewAlertSender  构造发送器
// ============================================================
func NewAlertSender(alert dto.AlertDTO, quotaType string) *AlertSender {
	return &AlertSender{
		alert:     alert,
		quotaType: quotaType,
	}
}

// ============================================================
// Send  发送"基础类"告警（SSL/网站过期等，按总次数限制）
// ============================================================
func (s *AlertSender) Send(quota string, params []dto.Param) {
	s.sendByConfigIds(s.alert.Method, quota, params, false)
}

// ============================================================
// ResourceSend  发送"资源类"告警（CPU/内存/磁盘等，按今日次数限制）
// ============================================================
func (s *AlertSender) ResourceSend(quota string, params []dto.Param) {
	s.sendByConfigIds(s.alert.Method, quota, params, true)
}

// ============================================================
// sendByConfigIds  解析 method 字符串里的 config id 列表，逐个发
// ============================================================
func (s *AlertSender) sendByConfigIds(methodStr string, quota string, params []dto.Param, isResource bool) {
	alertRepo := repo.NewIAlertRepo()
	configIds := strings.Split(methodStr, ",")
	for _, idStr := range configIds {
		idStr = strings.TrimSpace(idStr)
		configId, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			s.sendByLegacyMethod(idStr, quota, params, isResource)
			continue
		}
		config, err := alertRepo.GetConfigById(uint(configId))
		if err != nil {
			global.LOG.Errorf("alert config not found for id %d: %v", configId, err)
			continue
		}
		s.sendByConfig(config, quota, params, isResource)
	}
}

// ============================================================
// sendByConfig  按渠道类型分发到具体 sendXxx
// ============================================================
func (s *AlertSender) sendByConfig(config model.AlertConfig, quota string, params []dto.Param, isResource bool) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	switch config.Type {
	case constant.SMS:
		if isResource {
			s.sendResourceSMSWithConfig(config, quota, params)
		} else {
			s.sendSMSWithConfig(config, quota, params)
		}
	case constant.Email:
		if isResource {
			s.sendResourceEmailWithConfig(config, quota, params)
		} else {
			s.sendEmailWithConfig(config, quota, params)
		}
	case constant.Bark:
		if isResource {
			s.sendResourceBarkWithConfig(config, quota, params)
		} else {
			s.sendBarkWithConfig(config, quota, params)
		}
	case constant.WeCom, constant.DingTalk, constant.FeiShu:
		if isResource {
			s.sendResourceWebhookWithConfig(config, quota, params)
		} else {
			s.sendWebhookWithConfig(config, quota, params)
		}
	}
}

// ============================================================
// sendByLegacyMethod  兼容旧版 method 名（mail/bark/sms）
// ============================================================
func (s *AlertSender) sendByLegacyMethod(method string, quota string, params []dto.Param, isResource bool) {
	alertRepo := repo.NewIAlertRepo()
	typeMap := map[string]string{"mail": constant.Email, constant.Bark: constant.Bark, constant.SMS: constant.SMS}
	configType := method
	if mapped, ok := typeMap[method]; ok {
		configType = mapped
	}
	config, err := alertRepo.GetConfig(alertRepo.WithByType(configType))
	if err != nil {
		return
	}
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	s.sendByConfig(config, quota, params, isResource)
}

// ============================================================
// sendSMSWithConfig  用指定渠道发基础类 SMS
// ============================================================
func (s *AlertSender) sendSMSWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	method := strconv.Itoa(int(config.ID))
	if !alertUtil.CheckSMSSendLimit(config, method) {
		return
	}

	totalCount, isValid := s.canSendAlert(method)
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:  constant.AlertSuccess,
		Count:   totalCount + 1,
		AlertId: s.alert.ID,
		Type:    s.alert.Type,
		Method:  method,
	}

	err := xpack.AlertProvider.CreateSMSAlertLog(s.alert.Type, s.alert, create, quota, params, config, method)
	if err != nil {
		global.LOG.Errorf("%s alert sms push failed: %v", s.alert.Type, err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, method)
}

// ============================================================
// sendSMS  按"系统默认 SMS 渠道"发基础类
// ============================================================
func (s *AlertSender) sendSMS(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.SMS))
	if err != nil {
		return
	}
	s.sendSMSWithConfig(config, quota, params)
}

// ============================================================
// sendEmailWithConfig  用指定渠道发基础类邮件
// ============================================================
func (s *AlertSender) sendEmailWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	totalCount, isValid := s.canSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:      constant.AlertSuccess,
		Count:       totalCount + 1,
		AlertId:     s.alert.ID,
		Type:        s.alert.Type,
		AlertRule:   alertUtil.ProcessAlertRule(s.alert),
		AlertDetail: alertUtil.ProcessAlertDetail(s.alert, quota, params, constant.Email),
		Method:      strconv.Itoa(int(config.ID)),
	}

	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	err := alertUtil.CreateEmailAlertLog(create, s.alert, params, transport, agentInfo, config)
	if err != nil {
		global.LOG.Errorf("%s alert email push failed: %v", s.alert.Type, err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendResourceEmailWithConfig  用指定渠道发资源类邮件
// ============================================================
func (s *AlertSender) sendResourceEmailWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	todayCount, isValid := s.canResourceSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:      constant.AlertSuccess,
		Count:       todayCount + 1,
		AlertId:     s.alert.ID,
		Type:        s.alert.Type,
		AlertRule:   alertUtil.ProcessAlertRule(s.alert),
		AlertDetail: alertUtil.ProcessAlertDetail(s.alert, quota, params, constant.Email),
		Method:      strconv.Itoa(int(config.ID)),
	}

	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	if err := alertUtil.CreateEmailAlertLog(create, s.alert, params, transport, agentInfo, config); err != nil {
		global.LOG.Errorf("failed to send Email alert: %v", err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendEmail  按"系统默认 Email 渠道"发基础类
// ============================================================
func (s *AlertSender) sendEmail(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.EmailConfig))
	if err != nil {
		return
	}
	s.sendEmailWithConfig(config, quota, params)
}

// ============================================================
// sendResourceEmail  按系统默认 Email 渠道发资源类
// ============================================================
func (s *AlertSender) sendResourceEmail(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.EmailConfig))
	if err != nil {
		return
	}
	s.sendResourceEmailWithConfig(config, quota, params)
}

// ============================================================
// sendBarkWithConfig  用指定渠道发基础类 Bark
// ============================================================
func (s *AlertSender) sendBarkWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	totalCount, isValid := s.canSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:      constant.AlertSuccess,
		Count:       totalCount + 1,
		AlertId:     s.alert.ID,
		Type:        s.alert.Type,
		AlertRule:   alertUtil.ProcessAlertRule(s.alert),
		AlertDetail: alertUtil.ProcessAlertDetail(s.alert, quota, params, constant.Bark),
		Method:      strconv.Itoa(int(config.ID)),
	}

	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	err := alertUtil.CreateBarkAlertLog(create, s.alert, params, transport, agentInfo, config)
	if err != nil {
		global.LOG.Errorf("%s alert bark push failed: %v", s.alert.Type, err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendResourceBarkWithConfig  用指定渠道发资源类 Bark
// ============================================================
func (s *AlertSender) sendResourceBarkWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	todayCount, isValid := s.canResourceSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:      constant.AlertSuccess,
		Count:       todayCount + 1,
		AlertId:     s.alert.ID,
		Type:        s.alert.Type,
		AlertRule:   alertUtil.ProcessAlertRule(s.alert),
		AlertDetail: alertUtil.ProcessAlertDetail(s.alert, quota, params, constant.Bark),
		Method:      strconv.Itoa(int(config.ID)),
	}

	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	if err := alertUtil.CreateBarkAlertLog(create, s.alert, params, transport, agentInfo, config); err != nil {
		global.LOG.Errorf("failed to send Bark alert: %v", err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendBark  按系统默认 Bark 渠道发基础类
// ============================================================
func (s *AlertSender) sendBark(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.Bark))
	if err != nil {
		return
	}
	s.sendBarkWithConfig(config, quota, params)
}

// ============================================================
// sendResourceBark  按系统默认 Bark 渠道发资源类
// ============================================================
func (s *AlertSender) sendResourceBark(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.Bark))
	if err != nil {
		return
	}
	s.sendResourceBarkWithConfig(config, quota, params)
}

// ============================================================
// sendWebhookWithConfig  用指定渠道发基础类 Webhook（企微/钉钉/飞书）
// ============================================================
func (s *AlertSender) sendWebhookWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	totalCount, isValid := s.canSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:  constant.AlertSuccess,
		Count:   totalCount + 1,
		AlertId: s.alert.ID,
		Type:    s.alert.Type,
		Method:  strconv.Itoa(int(config.ID)),
	}
	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	err := xpack.AlertProvider.CreateWebhookAlertLog(s.alert.Type, s.alert, create, quota, params, config, transport, agentInfo)
	if err != nil {
		global.LOG.Errorf("%s alert %s webhook push failed: %v", s.alert.Type, config.Type, err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendResourceWebhookWithConfig  用指定渠道发资源类 Webhook
// ============================================================
func (s *AlertSender) sendResourceWebhookWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	todayCount, isValid := s.canResourceSendAlert(strconv.Itoa(int(config.ID)))
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:  constant.AlertSuccess,
		Count:   todayCount + 1,
		AlertId: s.alert.ID,
		Type:    s.alert.Type,
		Method:  strconv.Itoa(int(config.ID)),
	}
	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
	if err := xpack.AlertProvider.CreateWebhookAlertLog(s.alert.Type, s.alert, create, quota, params, config, transport, agentInfo); err != nil {
		global.LOG.Errorf("%s alert %s webhook push failed: %v", s.alert.Type, config.Type, err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, strconv.Itoa(int(config.ID)))
}

// ============================================================
// sendWebhook  按 type 找 Webhook 渠道发基础类
// ============================================================
func (s *AlertSender) sendWebhook(quota string, params []dto.Param, method string) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(method))
	if err != nil {
		return
	}
	s.sendWebhookWithConfig(config, quota, params)
}

// ============================================================
// sendResourceWebhook  按 type 找 Webhook 渠道发资源类
// ============================================================
func (s *AlertSender) sendResourceWebhook(quota string, params []dto.Param, method string) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(method))
	if err != nil {
		return
	}
	s.sendResourceWebhookWithConfig(config, quota, params)
}

// ============================================================
// sendResourceSMSWithConfig  用指定渠道发资源类 SMS
// ============================================================
func (s *AlertSender) sendResourceSMSWithConfig(config model.AlertConfig, quota string, params []dto.Param) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	method := strconv.Itoa(int(config.ID))
	if !alertUtil.CheckSMSSendLimit(config, method) {
		return
	}

	todayCount, isValid := s.canResourceSendAlert(method)
	if !isValid {
		return
	}

	create := dto.AlertLogCreate{
		Status:  constant.AlertSuccess,
		Count:   todayCount + 1,
		AlertId: s.alert.ID,
		Type:    s.alert.Type,
		Method:  method,
	}

	if err := xpack.AlertProvider.CreateSMSAlertLog(s.alert.Type, s.alert, create, quota, params, config, method); err != nil {
		global.LOG.Errorf("failed to send SMS alert: %v", err)
		return
	}
	alertUtil.CreateNewAlertTask(quota, s.alert.Type, s.quotaType, method)
}

// ============================================================
// sendResourceSMS  按系统默认 SMS 渠道发资源类
// ============================================================
func (s *AlertSender) sendResourceSMS(quota string, params []dto.Param) {
	alertRepo := repo.NewIAlertRepo()
	config, err := alertRepo.GetConfig(alertRepo.WithByType(constant.SMSConfig))
	if err != nil {
		return
	}
	s.sendResourceSMSWithConfig(config, quota, params)
}

// ============================================================
// canSendAlert  基础类发送判断：今天/总数都未超 sendCount
// ============================================================
func (s *AlertSender) canSendAlert(method string) (uint, bool) {
	todayCount, totalCount, err := alertRepo.LoadTaskCount(s.alert.Type, s.quotaType, method)
	if err != nil {
		global.LOG.Errorf("error getting task count: %v", err)
		return totalCount, false
	}

	if todayCount >= 1 || s.alert.SendCount <= totalCount {
		return totalCount, false
	}
	return totalCount, true
}

// ============================================================
// canResourceSendAlert  资源类发送判断：今日次数未超 sendCount
// ============================================================
func (s *AlertSender) canResourceSendAlert(method string) (uint, bool) {
	todayCount, _, err := alertRepo.LoadTaskCount(s.alert.Type, s.quotaType, method)
	if err != nil {
		global.LOG.Errorf("error getting task count: %v", err)
		return todayCount, false
	}
	if s.alert.SendCount <= todayCount {
		return todayCount, false
	}
	return todayCount, true
}

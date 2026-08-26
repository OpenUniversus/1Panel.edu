// =============================================================================
// 模块: Alert 告警 (agent/app/service/alert_helper.go)
// 文件: alert_helper.go — 辅助函数集 (alert)
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	alertUtil "github.com/1Panel-dev/1Panel/agent/utils/alert"
	"github.com/1Panel-dev/1Panel/agent/utils/common"
	"github.com/1Panel-dev/1Panel/agent/utils/psutil"
	versionUtil "github.com/1Panel-dev/1Panel/agent/utils/version"
	"github.com/1Panel-dev/1Panel/agent/utils/xpack"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
)

const (
	ResourceAlertInterval = 30
	CheckIntervalSec      = 3
	LoadCheckIntervalMin  = 5
)

// ============================================================
// AlertTaskHelper  告警调度器（持有 cron 任务、CPU/内存滑动窗口）
// ============================================================
// 字段:
//   - DiskIO (chan) — 磁盘 IO 数据通道
//   - NetIO (chan)  — 网络 IO 数据通道
// 方法:
//   - StartTask / StopTask / ResetTask / InitTask
// ============================================================
type AlertTaskHelper struct {
	DiskIO chan []disk.IOCountersStat
	NetIO  chan []gnet.IOCountersStat
}

type IAlertTaskHelper interface {
	StopTask()
	StartTask()
	ResetTask()
	InitTask(alertType string)
}

var cpuLoad1, cpuLoad5, cpuLoad15 []float64
var memoryLoad1, memoryLoad5, memoryLoad15 []float64
var alertTaskMu sync.Mutex

var baseTypes = map[string]bool{"ssl": true, "siteEndTime": true, "panelPwdEndTime": true, "panelUpdate": true}
var resourceTypes = map[string]bool{"cpu": true, "memory": true, "disk": true, "load": true, "panelLogin": true, "sshLogin": true, "nodeException": true, "licenseException": true}

// ============================================================
// NewIAlertTaskHelper  构造 IAlertTaskHelper（带缓冲通道）
// ============================================================
func NewIAlertTaskHelper() IAlertTaskHelper {
	return &AlertTaskHelper{
		DiskIO: make(chan []disk.IOCountersStat, 1),
		NetIO:  make(chan []gnet.IOCountersStat, 1),
	}
}
// ============================================================
// StartTask  加锁后启动所有告警任务（base + resource）
// ============================================================
func (m *AlertTaskHelper) StartTask() {
	alertTaskMu.Lock()
	defer alertTaskMu.Unlock()
	m.startTaskLocked()
}

func (m *AlertTaskHelper) startTaskLocked() {
	baseAlert, resourceAlert := m.getClassifiedAlerts()
	if len(baseAlert) == 0 && len(resourceAlert) == 0 {
		return
	}
	handleBaseAlertsLocked(baseAlert)
	handleResourceAlertsLocked(resourceAlert)
}

// ============================================================
// StopTask  停掉所有告警任务
// ============================================================
func (m *AlertTaskHelper) StopTask() {
	alertTaskMu.Lock()
	defer alertTaskMu.Unlock()
	stopBaseJobLocked()
	stopResourceJobLocked()
}

// ============================================================
// ResetTask  重启所有告警任务（停 + 启）
// ============================================================
func (m *AlertTaskHelper) ResetTask() {
	alertTaskMu.Lock()
	defer alertTaskMu.Unlock()
	stopBaseJobLocked()
	stopResourceJobLocked()
	m.startTaskLocked()
}

// ============================================================
// InitTask  按 type 增量初始化（重置状态 + 重启相关任务）
// ============================================================
func (m *AlertTaskHelper) InitTask(alertType string) {
	alertTaskMu.Lock()
	defer alertTaskMu.Unlock()
	m.initTaskLocked(alertType)
}

func (m *AlertTaskHelper) initTaskLocked(alertType string) {
	resetAlertState(alertType)
	if baseTypes[alertType] {
		stopBaseJobLocked()
	} else if resourceTypes[alertType] {
		stopResourceJobLocked()
	}
	m.startTaskLocked()
}

func resetAlertState(alertType string) {
	switch alertType {
	case "cpu":
		cpuLoad1 = []float64{}
		cpuLoad5 = []float64{}
		cpuLoad15 = []float64{}
	case "memory":
		memoryLoad1 = []float64{}
		memoryLoad5 = []float64{}
		memoryLoad15 = []float64{}
	}
}

// ============================================================
// getClassifiedAlerts  把所有告警按 type 分成 base（ssl/...）和 resource（cpu/...）
// ============================================================
func (m *AlertTaskHelper) getClassifiedAlerts() (baseAlerts, resourceAlerts []dto.AlertDTO) {
	alertList, _ := NewIAlertService().GetAlerts()
	for _, alert := range alertList {
		if baseTypes[alert.Type] {
			baseAlerts = append(baseAlerts, alert)
		} else if resourceTypes[alert.Type] {
			resourceAlerts = append(resourceAlerts, alert)
		}
	}
	return
}

// ============================================================
// handleBaseAlertsLocked  注册"基础"告警的 cron 任务（30 分钟一次）
// ============================================================
func handleBaseAlertsLocked(baseAlerts []dto.AlertDTO) {
	if len(baseAlerts) == 0 {
		stopBaseJobLocked()
		return
	}
	if global.AlertBaseJobID == 0 {
		baseTask(baseAlerts)
		jobID, err := global.Cron.AddFunc("@every 30m", func() {
			baseTask(baseAlerts)
		})
		if err != nil {
			global.LOG.Errorf("alert base job start failed: %v", err)
			return
		}
		global.AlertBaseJobID = jobID
		global.LOG.Info("start alert base job")
	}
}

// ============================================================
// handleResourceAlertsLocked  注册"资源"告警的 cron 任务（每分钟一次）
// ============================================================
func handleResourceAlertsLocked(resourceAlerts []dto.AlertDTO) {
	if len(resourceAlerts) == 0 {
		stopResourceJobLocked()
		return
	}
	if global.AlertResourceJobID == 0 {
		jobID, err := global.Cron.AddFunc("*/1 * * * *", func() {
			resourceTask(resourceAlerts)
		})
		if err != nil {
			global.LOG.Errorf("alert resource job start failed: %v", err)
			return
		}
		global.AlertResourceJobID = jobID
		global.LOG.Info("start alert resource job")
	}
}

func stopBaseJobLocked() {
	if global.AlertBaseJobID != 0 {
		global.Cron.Remove(global.AlertBaseJobID)
		global.AlertBaseJobID = 0
		global.LOG.Info("stop alert base job")
	}
}

func stopResourceJobLocked() {
	if global.AlertResourceJobID != 0 {
		global.Cron.Remove(global.AlertResourceJobID)
		global.AlertResourceJobID = 0
		global.LOG.Info("stop alert resource job")
	}
}

// ============================================================
// baseTask  base 告警主循环：按 type 调不同 loader
// ============================================================
func baseTask(baseAlert []dto.AlertDTO) {
	for _, alert := range baseAlert {
		if !alertUtil.CheckSendTimeRange(alert.Type) {
			continue
		}
		switch alert.Type {
		case "ssl":
			loadSSLInfo(alert)
		case "siteEndTime":
			loadWebsiteInfo(alert)
		case "panelPwdEndTime":
			if global.CONF.Base.IsEnterprise {
				continue
			}
			if global.IsMaster {
				loadPanelPwd(alert)
			}
		case "panelUpdate":
			if global.CONF.Base.IsEnterprise {
				continue
			}
			if global.IsMaster {
				loadPanelUpdate(alert)
			}
		}
	}
}

// ============================================================
// resourceTask  resource 告警主循环：每分钟调不同 loader
// ============================================================
func resourceTask(resourceAlert []dto.AlertDTO) {
	minute := time.Now().Minute()
	for _, alert := range resourceAlert {
		if !alertUtil.CheckSendTimeRange(alert.Type) {
			continue
		}
		execute := minute%LoadCheckIntervalMin == 0
		switch alert.Type {
		case "cpu":
			loadCPUUsage(alert)
		case "memory":
			loadMemUsage(alert)
		case "load":
			loadLoadInfo(alert)
		case "disk":
			loadDiskUsage(alert)
		case "panelLogin":
			loadPanelLogin(alert)
		case "sshLogin":
			loadSSHLogin(alert)
		case "nodeException":
			if execute && global.IsMaster {
				loadNodeException(alert)
			}
		case "licenseException":
			if global.CONF.Base.IsEnterprise {
				continue
			}
			if execute && global.IsMaster {
				loadLicenseException(alert)
			}
		}
	}
}

// ============================================================
// loadSSLInfo  检查证书快过期（剩余天数 <= cycle）
// ============================================================
// 流程:
//   1. 拉所有 SSL
//   2. 算剩余天数
//   3. 过滤即将过期
//   4. 触发告警
// ============================================================
func loadSSLInfo(alert dto.AlertDTO) {
	opts := getRepoOptionsByProject(alert.Project)
	sslList, _ := repo.NewISSLRepo().List(opts...)
	if len(sslList) == 0 {
		return
	}
	daysDiffMap, projectMap := calculateSSLExpiryDays(sslList, alert.Cycle)
	projectJSON := serializeAndSortProjects(projectMap)
	if projectJSON == "" || len(daysDiffMap) == 0 {
		return
	}
	sender := NewAlertSender(alert, projectJSON)
	minDays := math.MaxInt
	maxDays := 0
	allDomains := make([]string, 0)
	for daysDiff, domains := range daysDiffMap {
		allDomains = append(allDomains, domains...)
		if daysDiff < minDays {
			minDays = daysDiff
		}
		if daysDiff > maxDays {
			maxDays = daysDiff
		}
	}
	if len(allDomains) == 0 {
		return
	}
	var daysStr string
	if len(allDomains) == 1 {
		daysStr = strconv.Itoa(minDays)
	} else {
		daysStr = strconv.Itoa(minDays) + "-" + strconv.Itoa(maxDays)
	}
	domainStr := strings.Join(allDomains, ",")
	params := createAlertBaseParams(strconv.Itoa(len(daysDiffMap)), daysStr)
	sender.Send(domainStr, params)
}

// ============================================================
// loadWebsiteInfo  检查站点快过期（同 SSL 逻辑但用 website 表）
// ============================================================
func loadWebsiteInfo(alert dto.AlertDTO) {
	opts := getRepoOptionsByProject(alert.Project)
	websiteList, _ := websiteRepo.List(opts...)
	if len(websiteList) == 0 {
		return
	}

	daysDiffMap, projectMap := calculateWebsiteExpiryDays(websiteList, alert.Cycle)
	projectJSON := serializeAndSortProjects(projectMap)
	if projectJSON == "" || len(daysDiffMap) == 0 {
		return
	}
	sender := NewAlertSender(alert, projectJSON)
	minDays := math.MaxInt
	maxDays := 0
	allDomains := make([]string, 0)
	for daysDiff, domains := range daysDiffMap {
		allDomains = append(allDomains, domains...)
		if daysDiff < minDays {
			minDays = daysDiff
		}
		if daysDiff > maxDays {
			maxDays = daysDiff
		}
	}
	if len(allDomains) == 0 {
		return
	}
	var daysStr string
	if len(allDomains) == 1 {
		daysStr = strconv.Itoa(minDays)
	} else {
		daysStr = strconv.Itoa(minDays) + "-" + strconv.Itoa(maxDays)
	}
	domainStr := strings.Join(allDomains, ",")
	params := createAlertBaseParams(strconv.Itoa(len(daysDiffMap)), daysStr)
	sender.Send(domainStr, params)
}

// ============================================================
// loadPanelPwd  检查面板密码快到期
// ============================================================
func loadPanelPwd(alert dto.AlertDTO) {
	// only master alert
	expDays, err := getSettingValue("ExpirationDays")
	if err != nil || expDays == "0" {
		global.LOG.Info("panel password expiration setting not enabled, skip")
		return
	}

	expTimeStr, err := getSettingValue("ExpirationTime")
	if err != nil {
		return
	}
	expTime, _ := time.Parse(constant.DateTimeLayout, expTimeStr)
	daysDiff := calculateDaysDifference(expTime)
	if daysDiff >= 0 && int(alert.Cycle) >= daysDiff {
		params := createAlertPwdParams(strconv.Itoa(daysDiff))
		sender := NewAlertSender(alert, expTimeStr)
		sender.Send(strconv.Itoa(daysDiff), params)
	}
}

// ============================================================
// loadPanelUpdate  检查面板有新版本
// ============================================================
func loadPanelUpdate(alert dto.AlertDTO) {
	// only master alert
	info, err := versionUtil.GetUpgradeVersionInfo()
	if err != nil {
		global.LOG.Errorf("error getting version info: %s", err)
		return
	}

	version := getValidVersion(info)
	if version == "" {
		return
	}

	sender := NewAlertSender(alert, version)
	sender.Send(version, []dto.Param{})
}

// 获取 CPU 使用率数据并发送到通道
// ============================================================
// loadCPUUsage  CPU 使用率检查（用 1/5/15 分钟滑动窗口取平均）
// ============================================================
func loadCPUUsage(alert dto.AlertDTO) {
	percent, err := cpu.Percent(time.Duration(CheckIntervalSec)*time.Second, false)
	if err != nil {
		global.LOG.Errorf("error getting cpu usage, err: %v", err)
		return
	}

	if len(percent) > 0 {
		var usageLoad *[]float64
		var threshold int

		switch alert.Cycle {
		case 1:
			usageLoad = &cpuLoad1
			threshold = 1
		case 5:
			usageLoad = &cpuLoad5
			threshold = 5
		case 15:
			usageLoad = &cpuLoad15
			threshold = 15
		}
		shouldSendResourceAlert(alert, percent[0], usageLoad, threshold)
	}
}

// 获取内存使用情况数据并发送到通道
// ============================================================
// loadMemUsage  内存使用率检查（同 CPU 滑动窗口逻辑）
// ============================================================
func loadMemUsage(alert dto.AlertDTO) {
	memStat, err := mem.VirtualMemory()
	if err != nil {
		global.LOG.Errorf("error getting memory usage, err: %v", err)
		return
	}

	percent := memStat.UsedPercent
	var memoryLoad *[]float64
	var threshold int

	switch alert.Cycle {
	case 1:
		memoryLoad = &memoryLoad1
		threshold = 1
	case 5:
		memoryLoad = &memoryLoad5
		threshold = 5
	case 15:
		memoryLoad = &memoryLoad15
		threshold = 15
	}
	shouldSendResourceAlert(alert, percent, memoryLoad, threshold)
}

// 获取系统负载数据并发送到通道
// ============================================================
// loadLoadInfo  系统负载检查（按 CPU 核数归一化）
// ============================================================
func loadLoadInfo(alert dto.AlertDTO) {
	avgStat, err := load.Avg()
	if err != nil {
		global.LOG.Errorf("error getting load usage, err: %v", err)
		return
	}
	var loadValue float64
	CPUTotal, _ := psutil.CPUInfo.GetLogicalCores(false)
	switch alert.Cycle {
	case 1:
		loadValue = avgStat.Load1 / (float64(CPUTotal*2) * 0.75) * 100
	case 5:
		loadValue = avgStat.Load5 / (float64(CPUTotal*2) * 0.75) * 100
	case 15:
		loadValue = avgStat.Load15 / (float64(CPUTotal*2) * 0.75) * 100
	default:
		return
	}
	if loadValue < float64(alert.Count) {
		return
	}
	newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
	if err != nil {
		global.LOG.Errorf("task log record not found, err: %v", err)
	}
	if isAlertDue(newDate) {
		sendResourceAlert(alert, loadValue)
	}
}

// ============================================================
// loadDiskUsage  磁盘使用率检查（支持 all 或单盘）
// ============================================================
func loadDiskUsage(alert dto.AlertDTO) {
	newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
	if err != nil {
		global.LOG.Errorf("record not found, err: %v", err)
		return
	}
	if isAlertDue(newDate) {
		if strings.Contains(alert.Project, "all") {
			_ = processAllDisks(alert)
		} else {
			_ = processSingleDisk(alert)
		}
	}
}

// ============================================================
// loadPanelLogin  面板登录异常检查（失败次数/陌生 IP）
// ============================================================
func loadPanelLogin(alert dto.AlertDTO) {
	count, isAlert, err := alertUtil.CountRecentFailedLoginLogs(alert.Cycle, alert.Count)
	if err != nil {
		global.LOG.Errorf("Failed to count recent failed login logs: %v", err)
	}
	if isAlert {
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: "",
			},
			{
				Index: "2",
				Key:   "project",
				Value: "",
			},
		}
		sendAlerts(alert, "panelLogin", strconv.Itoa(count), "panelLogin", params)
	}

	whitelist := strings.Split(strings.TrimSpace(alert.AdvancedParams), "\n")
	records, err := alertUtil.FindRecentSuccessLoginsNotInWhitelist(30, whitelist)
	if err != nil {
		global.LOG.Errorf("Failed to check recent failed ip login logs: %v", err)
	}
	records = filterLoginLogsNotInWhitelist(records, whitelist)
	if len(records) > 0 {
		quota := strings.Join(func() []string {
			var ips []string
			for _, r := range records {
				ips = append(ips, r.IP)
			}
			return ips
		}(), "\n")
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: "",
			},
			{
				Index: "2",
				Key:   "project",
				Value: " IP ",
			},
		}
		sendAlerts(alert, "panelIpLogin", quota, "panelIpLogin", params)
	}
}

// ============================================================
// loadSSHLogin  SSH 登录异常检查（失败次数/陌生 IP）
// ============================================================
func loadSSHLogin(alert dto.AlertDTO) {
	count, isAlert, err := alertUtil.CountRecentFailedSSHLog(alert.Cycle, alert.Count)
	if err != nil {
		global.LOG.Errorf("Failed to count recent failed ssh login logs: %v", err)
	}
	if isAlert {
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: " SSH ",
			},
			{
				Index: "2",
				Key:   "project",
				Value: "",
			},
		}
		sendAlerts(alert, "sshLogin", strconv.Itoa(count), "sshLogin", params)
	}
	whitelist := strings.Split(strings.TrimSpace(alert.AdvancedParams), "\n")
	records, err := alertUtil.FindRecentSuccessLoginNotInWhitelist(30, whitelist)
	if err != nil {
		global.LOG.Errorf("Failed to check recent failed ip ssh login logs: %v", err)
	}
	records = filterSSHLoginEntriesNotInWhitelist(records, whitelist)
	if len(records) > 0 {
		quota := strings.Join(records, "\n")
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: " SSH ",
			},
			{
				Index: "2",
				Key:   "project",
				Value: " IP ",
			},
		}
		sendAlerts(alert, "sshIpLogin", quota, "sshIpLogin", params)
	}
}

func filterLoginLogsNotInWhitelist(records []model.LoginLog, whitelist []string) []model.LoginLog {
	filtered := make([]model.LoginLog, 0, len(records))
	for _, record := range records {
		if !isIPInWhitelist(record.IP, whitelist) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func filterSSHLoginEntriesNotInWhitelist(records []string, whitelist []string) []string {
	filtered := make([]string, 0, len(records))
	for _, record := range records {
		ip := record
		if idx := strings.Index(record, "-"); idx >= 0 {
			ip = record[:idx]
		}
		if !isIPInWhitelist(ip, whitelist) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func isIPInWhitelist(ip string, whitelist []string) bool {
	targetIP := net.ParseIP(strings.TrimSpace(ip))
	if targetIP == nil {
		return false
	}
	for _, item := range whitelist {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == ip {
			return true
		}
		if whiteIP := net.ParseIP(item); whiteIP != nil {
			if whiteIP.Equal(targetIP) {
				return true
			}
			continue
		}
		_, ipNet, err := net.ParseCIDR(item)
		if err != nil {
			continue
		}
		if ipNet.Contains(targetIP) {
			return true
		}
	}
	return false
}

// ============================================================
// loadNodeException  节点异常告警（多节点场景，主节点专属）
// ============================================================
func loadNodeException(alert dto.AlertDTO) {
	// only master alert
	failCount, err := xpack.AlertProvider.GetNodeErrorAlert()
	if err != nil {
		global.LOG.Errorf("error getting node, err: %s", err)
		return
	}
	if failCount > 0 {
		quotaType := "node-error"
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: strconv.Itoa(int(failCount)),
			},
		}
		newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
		if err != nil {
			global.LOG.Errorf("record not found, err: %v", err)
			return
		}
		if isAlertDue(newDate) {
			sender := NewAlertSender(alert, quotaType)
			sender.ResourceSend(strconv.Itoa(int(failCount)), params)
		}
	}

}

// ============================================================
// loadLicenseException  License 异常告警（企业版专属，主节点专属）
// ============================================================
func loadLicenseException(alert dto.AlertDTO) {
	// only master alert
	failCount, err := xpack.AlertProvider.GetLicenseErrorAlert()
	if err != nil {
		global.LOG.Errorf("error getting license, err: %s", err)
		return
	}
	if failCount > 0 {
		quotaType := "license-error"
		params := []dto.Param{
			{
				Index: "1",
				Key:   "cycle",
				Value: strconv.Itoa(int(failCount)),
			},
		}
		newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
		if err != nil {
			global.LOG.Errorf("record not found, err: %v", err)
			return
		}
		if isAlertDue(newDate) {
			sender := NewAlertSender(alert, quotaType)
			sender.ResourceSend(strconv.Itoa(int(failCount)), params)
		}
	}
}

// ============================================================
// sendAlerts  通用告警发送（带冷却时间）
// ============================================================
func sendAlerts(alert dto.AlertDTO, alertType, quota, quotaType string, params []dto.Param) {
	methods := strings.Split(alert.Method, ",")
	newDate, err := alertRepo.GetTaskLog(alertType, alert.ID)
	if err != nil {
		global.LOG.Errorf("task log record not found, err: %v", err)
	}
	if newDate.IsZero() || calculateMinutesDifference(newDate) > ResourceAlertInterval {
		for _, m := range methods {
			m = strings.TrimSpace(m)
			if configId, err := strconv.ParseUint(m, 10, 64); err == nil {
				sendAlertsByConfigId(alert, alertType, quota, quotaType, params, uint(configId))
			} else {
				sendAlertsByLegacyMethod(alert, alertType, quota, quotaType, params, m)
			}
		}
	}
}

// ============================================================
// sendAlertsByConfigId  按 config id 找渠道发送
// ============================================================
func sendAlertsByConfigId(alert dto.AlertDTO, alertType, quota, quotaType string, params []dto.Param, configId uint) {
	config, err := alertRepo.GetConfigById(configId)
	if err != nil {
		global.LOG.Errorf("alert config not found for id %d: %v", configId, err)
		return
	}
	doSendAlert(alert, alertType, quota, quotaType, params, config)
}

// ============================================================
// sendAlertsByLegacyMethod  按遗留 method 名（mail/bark/sms）发
// ============================================================
func sendAlertsByLegacyMethod(alert dto.AlertDTO, alertType, quota, quotaType string, params []dto.Param, method string) {
	typeMap := map[string]string{
		"mail":        constant.Email,
		constant.Bark: constant.Bark,
		constant.SMS:  constant.SMS,
	}
	configType, ok := typeMap[method]
	if !ok {
		configType = method
	}
	config, err := alertRepo.GetConfig(alertRepo.WithByType(configType))
	if err != nil {
		return
	}
	doSendAlert(alert, alertType, quota, quotaType, params, config)
}

// ============================================================
// doSendAlert  按渠道类型真正发送（SMS/Email/Bark/Webhook）
// ============================================================
// 流程:
//   1. 检查渠道启用
//   2. 检查今日发送次数
//   3. 调对应渠道的 sender
//   4. 记 AlertTask 计数
// ============================================================
func doSendAlert(alert dto.AlertDTO, alertType, quota, quotaType string, params []dto.Param, config model.AlertConfig) {
	if !alertUtil.IsAlertConfigEnabled(config) {
		return
	}
	methodStr := strconv.Itoa(int(config.ID))
	switch config.Type {
	case constant.SMS:
		if !alertUtil.CheckSMSSendLimit(config, methodStr) {
			return
		}
		todayCount, isValid := canSendAlertToday(alertType, quotaType, alert.SendCount, methodStr)
		if !isValid {
			return
		}
		create := dto.AlertLogCreate{
			Type:    alertType,
			AlertId: alert.ID,
			Count:   todayCount + 1,
			Method:  methodStr,
		}
		alertErr := xpack.AlertProvider.CreateSMSAlertLog(alertType, alert, create, quotaType, params, config, methodStr)
		if alertErr != nil {
			global.LOG.Infof("%s alert sms push faild, err: %v", alertType, alertErr.Error())
			return
		}
		alertUtil.CreateNewAlertTask(quota, alertType, quotaType, methodStr)

	case constant.Email:
		todayCount, isValid := canSendAlertToday(alertType, quotaType, alert.SendCount, methodStr)
		if !isValid {
			return
		}
		create := dto.AlertLogCreate{
			Type:    alertType,
			AlertId: alert.ID,
			Count:   todayCount + 1,
			Method:  methodStr,
		}
		alertInfo := alert
		alertInfo.Type = alertType
		create.AlertRule = alertUtil.ProcessAlertRule(alert)
		create.AlertDetail = alertUtil.ProcessAlertDetail(alertInfo, quotaType, params, constant.Email)
		transport := xpack.MultiNodeProvider.LoadRequestTransport()
		agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
		alertErr := alertUtil.CreateEmailAlertLog(create, alertInfo, params, transport, agentInfo, config)
		if alertErr != nil {
			global.LOG.Infof("%s alert email push faild, err: %v", alertType, alertErr.Error())
			return
		}
		alertUtil.CreateNewAlertTask(quota, alertType, quotaType, methodStr)

	case constant.Bark:
		todayCount, isValid := canSendAlertToday(alertType, quotaType, alert.SendCount, methodStr)
		if !isValid {
			return
		}
		create := dto.AlertLogCreate{
			Type:    alertType,
			AlertId: alert.ID,
			Count:   todayCount + 1,
			Method:  methodStr,
		}
		alertInfo := alert
		alertInfo.Type = alertType
		create.AlertRule = alertUtil.ProcessAlertRule(alert)
		create.AlertDetail = alertUtil.ProcessAlertDetail(alertInfo, quotaType, params, constant.Bark)
		transport := xpack.MultiNodeProvider.LoadRequestTransport()
		agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
		alertErr := alertUtil.CreateBarkAlertLog(create, alertInfo, params, transport, agentInfo, config)
		if alertErr != nil {
			global.LOG.Infof("%s alert %s push failed, err: %v", alertType, methodStr, alertErr.Error())
			return
		}
		alertUtil.CreateNewAlertTask(quota, alertType, quotaType, methodStr)

	case constant.WeCom, constant.DingTalk, constant.FeiShu:
		todayCount, isValid := canSendAlertToday(alertType, quotaType, alert.SendCount, methodStr)
		if !isValid {
			return
		}
		create := dto.AlertLogCreate{
			Type:    alertType,
			AlertId: alert.ID,
			Count:   todayCount + 1,
			Method:  methodStr,
		}
		transport := xpack.MultiNodeProvider.LoadRequestTransport()
		agentInfo, _ := xpack.MultiNodeProvider.GetAgentInfo()
		alertErr := xpack.AlertProvider.CreateWebhookAlertLog(alertType, alert, create, quotaType, params, config, transport, agentInfo)
		if alertErr != nil {
			global.LOG.Infof("%s alert webhook %s push faild, err: %v", alertType, methodStr, alertErr)
			return
		}
		alertUtil.CreateNewAlertTask(quota, alertType, quotaType, methodStr)
	}
}

// ------------------------------
// ============================================================
// getRepoOptionsByProject  把 project 解析成 DBOption（"all" 则空过滤）
// ============================================================
func getRepoOptionsByProject(project string) []repo.DBOption {
	var opts []repo.DBOption
	if project != "all" {
		itemID, _ := strconv.Atoi(project)
		opts = append(opts, repo.WithByID(uint(itemID)))
	}
	return opts
}

func serializeAndSortProjects(projectMap map[uint][]time.Time) string {
	if len(projectMap) == 0 {
		return ""
	}
	keys := make([]int, 0, len(projectMap))
	for k := range projectMap {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	projectJSON, err := json.Marshal(projectMap)
	if err != nil {
		global.LOG.Errorf("Failed to serialize projectMap: %v", err)
		return ""
	}

	return string(projectJSON)
}

// ============================================================
// calculateSSLExpiryDays  算每个证书剩余天数，过滤 cycle 内的
// ============================================================
// 返回:
//   - (map[剩余天数][]域名, map[ssl id][]到期时间)
// ============================================================
func calculateSSLExpiryDays(sslList []model.WebsiteSSL, cycle uint) (map[int][]string, map[uint][]time.Time) {
	currentDate := time.Now()
	daysDiffMap := make(map[int][]string)
	projectMap := make(map[uint][]time.Time)

	for _, ssl := range sslList {
		daysDiff := int(math.Ceil(
			ssl.ExpireDate.Sub(currentDate).Hours() / 24,
		))
		if daysDiff > 0 && int(cycle) >= daysDiff {
			daysDiffMap[daysDiff] = append(daysDiffMap[daysDiff], ssl.PrimaryDomain)
			projectMap[ssl.ID] = append(projectMap[ssl.ID], ssl.ExpireDate)
		}
	}
	return daysDiffMap, projectMap
}

// ============================================================
// calculateWebsiteExpiryDays  算每个站点剩余天数（同 SSL 逻辑）
// ============================================================
func calculateWebsiteExpiryDays(websites []model.Website, cycle uint) (map[int][]string, map[uint][]time.Time) {
	currentDate := time.Now()
	daysDiffMap := make(map[int][]string)
	projectMap := make(map[uint][]time.Time)

	for _, website := range websites {
		daysDiff := int(math.Ceil(
			website.ExpireDate.Sub(currentDate).Hours() / 24,
		))
		if daysDiff > 0 && int(cycle) >= daysDiff {
			daysDiffMap[daysDiff] = append(daysDiffMap[daysDiff], website.PrimaryDomain)
			projectMap[website.ID] = append(projectMap[website.ID], website.ExpireDate)
		}
	}
	return daysDiffMap, projectMap
}

// ============================================================
// getSettingValue  从 setting 表拿配置项
// ============================================================
func getSettingValue(key string) (string, error) {
	var setting model.Setting
	if err := global.CoreDB.Model(&model.Setting{}).Where("key = ?", key).First(&setting).Error; err != nil {
		global.LOG.Errorf("load %s from db setting failed: %v", key, err)
		return "", err
	}
	return setting.Value, nil
}

// ============================================================
// getValidVersion  从升级信息里挑一个可用版本
// ============================================================
// 优先级: NewVersion > TestVersion > LatestVersion
// ============================================================
func getValidVersion(info *dto.UpgradeInfo) string {
	if info.NewVersion != "" {
		return info.NewVersion
	} else if info.TestVersion != "" {
		return info.TestVersion
	} else if info.LatestVersion != "" {
		return info.LatestVersion
	}
	return ""
}

// ============================================================
// shouldSendResourceAlert  CPU/内存 告警判定：滑动窗口平均超阈值才发
// ============================================================
func shouldSendResourceAlert(alert dto.AlertDTO, currentUsage float64, usageLoad *[]float64, threshold int) {
	newDate, err := alertRepo.GetTaskLog(alert.Type, alert.ID)
	if err != nil {
		global.LOG.Errorf("record not found, err: %v", err)
	}
	if isAlertDue(newDate) {
		*usageLoad = append(*usageLoad, currentUsage)
		if len(*usageLoad) > threshold {
			*usageLoad = (*usageLoad)[1:]
		}
		if len(*usageLoad) == threshold {
			avgUsage := average(*usageLoad)
			if avgUsage >= float64(alert.Count) {
				sendResourceAlert(alert, avgUsage)
			}
		}
	}
}

// ============================================================
// isAlertDue  判断"冷却时间"是否已过（避免告警风暴）
// ============================================================
func isAlertDue(lastAlertTime time.Time) bool {
	if lastAlertTime.IsZero() {
		return true
	}
	return calculateMinutesDifference(lastAlertTime) > ResourceAlertInterval
}

// ============================================================
// sendResourceAlert  发资源类告警（CPU/内存/负载）
// ============================================================
func sendResourceAlert(alert dto.AlertDTO, value float64) {
	valueStr := common.FormatPercent(value)
	module := getModuleName(alert.Type)
	params := createAlertAvgParams(strconv.Itoa(int(alert.Cycle)), module, valueStr)
	sender := NewAlertSender(alert, strconv.Itoa(int(alert.Cycle)))
	sender.ResourceSend(valueStr, params)
}

func getModuleName(alertType string) string {
	var module string
	switch alertType {
	case "cpu":
		module = " CPU "
	case "memory":
		module = "内存"
	case "load":
		module = "负载"
	default:
	}
	return module
}

// ============================================================
// canSendAlertToday  判断今天是否还能再发（未超过 sendCount）
// ============================================================
func canSendAlertToday(alertType, quotaType string, sendCount uint, method string) (uint, bool) {
	todayCount, _, err := alertRepo.LoadTaskCount(alertType, quotaType, method)
	if err != nil {
		global.LOG.Errorf("error getting task info, err: %v", err)
		return todayCount, false
	}
	if todayCount >= sendCount {
		return todayCount, false
	}

	return todayCount, true
}

func average(arr []float64) float64 {
	total := 0.0
	for _, v := range arr {
		total += v
	}
	return total / float64(len(arr))
}

func createAlertBaseParams(project, cycle string) []dto.Param {
	return []dto.Param{
		{
			Index: "1",
			Key:   "project",
			Value: project,
		},
		{
			Index: "2",
			Key:   "cycle",
			Value: cycle,
		},
	}
}

func createAlertPwdParams(cycle string) []dto.Param {
	return []dto.Param{
		{
			Index: "1",
			Key:   "cycle",
			Value: cycle,
		},
	}
}

func createAlertAvgParams(cycle, module, count string) []dto.Param {
	return []dto.Param{
		{
			Index: "1",
			Key:   "cycle",
			Value: cycle,
		},
		{
			Index: "2",
			Key:   "module",
			Value: module,
		},
		{
			Index: "3",
			Key:   "count",
			Value: count,
		},
	}
}

func createAlertDiskParams(project, count string) []dto.Param {
	return []dto.Param{
		{
			Index: "1",
			Key:   "project",
			Value: project,
		},
		{
			Index: "2",
			Key:   "count",
			Value: count,
		},
	}
}

// ============================================================
// processAllDisks  遍历所有磁盘检查（alert.Project="all"）
// ============================================================
func processAllDisks(alert dto.AlertDTO) error {
	diskList, err := NewIAlertService().GetDisks()
	if err != nil {
		global.LOG.Errorf("error getting disk list, err: %v", err)
		return err
	}
	var errMsgs []string
	for _, item := range diskList {
		err := checkAndCreateDiskAlert(alert, item.Path)
		if err != nil {
			errMsg := fmt.Sprintf("disk path %s process failed: %v", item.Path, err)
			errMsgs = append(errMsgs, errMsg)
			global.LOG.Errorf("%s", errMsg)
			continue
		}
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf("batch process disks failed, error count: %d, details: %s", len(errMsgs), strings.Join(errMsgs, "; "))
	}
	return nil
}

// ============================================================
// processSingleDisk  检查单个磁盘
// ============================================================
func processSingleDisk(alert dto.AlertDTO) error {
	err := checkAndCreateDiskAlert(alert, alert.Project)
	if err != nil {
		global.LOG.Errorf("%s", err.Error())
		return err
	}
	return nil
}

// ============================================================
// checkAndCreateDiskAlert  检查并发送磁盘告警
// ============================================================
// 阈值单位:
//   - cycle=1 → GB
//   - 其他 → 百分比
// ============================================================
func checkAndCreateDiskAlert(alert dto.AlertDTO, path string) error {
	usageStat, err := psutil.DISK.GetUsage(path, false)
	if err != nil {
		global.LOG.Errorf("error getting disk usage for %s, err: %v", path, err)
		return err
	}

	usedTotal, usedStr := calculateUsedTotal(alert.Cycle, usageStat)
	commonTotal := float64(alert.Count)
	if alert.Cycle == 1 {
		commonTotal *= 1024 * 1024 * 1024
	}
	if usedTotal < commonTotal {
		return nil
	}
	params := createAlertDiskParams(path, usedStr)
	sender := NewAlertSender(alert, alert.Project)
	sender.ResourceSend(path, params)
	return nil
}

// ============================================================
// calculateUsedTotal  按 cycle 单位返回磁盘用量（字节 vs 百分比）
// ============================================================
func calculateUsedTotal(cycle uint, usageStat *disk.UsageStat) (float64, string) {
	if cycle == 1 {
		return float64(usageStat.Used), common.FormatBytes(usageStat.Used)
	}
	return usageStat.UsedPercent, common.FormatPercent(usageStat.UsedPercent)
}

// ============================================================
// calculateDaysDifference  算两个时间相隔天数（向下取整）
// ============================================================
func calculateDaysDifference(expirationTime time.Time) int {
	currentDate := time.Now()
	formattedTime := currentDate.Format(constant.DateTimeLayout)
	parsedTime, _ := time.Parse(constant.DateTimeLayout, formattedTime)
	timeGap := expirationTime.Sub(parsedTime).Milliseconds()
	if timeGap < 0 {
		return -1
	}
	daysDifference := int(math.Floor(float64(timeGap) / (3600 * 1000 * 24)))
	return daysDifference
}

func calculateMinutesDifference(newDate time.Time) int {
	now := time.Now()
	if newDate.After(now) {
		return -1
	}
	minutesDifference := int(now.Sub(newDate).Minutes())
	return minutesDifference
}

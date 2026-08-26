// =============================================================================
// 模块: Alert 告警 (agent/app/service/alert.go)
// 文件: alert.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package service

import (
	"encoding/json"
	"fmt"
	"mime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/dto"
	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/app/repo"
	"github.com/1Panel-dev/1Panel/agent/buserr"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/i18n"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	"github.com/1Panel-dev/1Panel/agent/utils/copier"
	"github.com/1Panel-dev/1Panel/agent/utils/email"
	"github.com/1Panel-dev/1Panel/agent/utils/xpack"
	"github.com/shirou/gopsutil/v4/disk"
)

// ============================================================
// AlertService  告警业务服务（CRUD + 调度 + 通知渠道）
// ============================================================
// 方法:
//   - PageAlert / GetAlerts / CreateAlert / UpdateAlert / DeleteAlert / GetAlert / UpdateStatus
//   - ExternalUpdateAlert (外部触发)
//   - GetDisks / PageAlertLogs / CleanAlertLogs / GetClams / GetCronJobs
//   - GetAlertConfig / PageAlertConfig / UpdateAlertConfig / DeleteAlertConfig / TestAlertConfig
// ============================================================
type AlertService struct{}

var eeHiddenAlertTypes = []string{"licenseException", "panelUpdate", "panelPwdEndTime"}
var communityAlertMethodTypeNames = map[string]string{
	constant.WeCom:    "WeCom",
	constant.DingTalk: "DingTalk",
	constant.FeiShu:   "FeiShu",
	constant.SMS:      "SMS",
}

// ============================================================
// IAlertService  AlertService 的接口
// ============================================================
// 主要方法: PageAlert / GetAlerts / CreateAlert / UpdateAlert / DeleteAlert / GetAlert
//           UpdateStatus / ExternalUpdateAlert
//           GetDisks / PageAlertLogs / CleanAlertLogs / GetClams / GetCronJobs
//           GetAlertConfig / PageAlertConfig / UpdateAlertConfig / DeleteAlertConfig / TestAlertConfig
// ============================================================
type IAlertService interface {
	PageAlert(req dto.AlertSearch) (int64, []dto.AlertDTO, error)
	GetAlerts() ([]dto.AlertDTO, error)
	CreateAlert(create dto.AlertCreate, operator string) error
	UpdateAlert(req dto.AlertUpdate, operator string) error
	DeleteAlert(id uint) error
	GetAlert(id uint) (dto.AlertDTO, error)
	UpdateStatus(id uint, status string) error
	ExternalUpdateAlert(req dto.AlertCreate, operator string) error

	GetDisks() ([]dto.DiskDTO, error)
	PageAlertLogs(req dto.AlertLogSearch) (int64, []dto.AlertLogDTO, error)
	CleanAlertLogs() error
	GetClams() ([]dto.ClamDTO, error)
	GetCronJobs(req dto.CronJobReq) ([]dto.CronJobDTO, error)

	GetAlertConfig(req dto.AlertConfigQuery) ([]model.AlertConfig, error)
	PageAlertConfig(req dto.AlertConfigPageReq) (int64, []model.AlertConfig, error)
	UpdateAlertConfig(req dto.AlertConfigUpdate, operator string) error
	DeleteAlertConfig(id uint) error
	TestAlertConfig(req dto.AlertConfigTest) (bool, error)
}

// ============================================================
// NewIAlertService  构造 IAlertService 默认实现
// ============================================================
func NewIAlertService() IAlertService {
	return &AlertService{}
}

// ============================================================
// PageAlert  分页查告警任务
// ============================================================
// 流程:
//   1. 按 Status/Type 拼 DBOption
//   2. 企业版隐藏 licenseException / panelUpdate 等敏感类型
//   3. 调 alertRepo.Page
//   4. 把 model 拷到 DTO
// 调用: api/v2.PageAlert -> this
// ============================================================
func (a AlertService) PageAlert(search dto.AlertSearch) (int64, []dto.AlertDTO, error) {
	var (
		opts   []repo.DBOption
		result []dto.AlertDTO
	)
	if global.CONF.Base.IsEnterprise {
		opts = append(opts, alertRepo.WithByTypeNotIn(eeHiddenAlertTypes))
	}
	if search.Status != "" {
		opts = append(opts, repo.WithByStatus(search.Status))
	}
	if search.Type != "" {
		opts = append(opts, alertRepo.WithByType(search.Type))
	}
	opts = append(opts, repo.WithOrderDesc("created_at"))

	total, alerts, err := alertRepo.Page(search.Page, search.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	for _, item := range alerts {

		result = append(result, dto.AlertDTO{
			ID:             item.ID,
			Type:           item.Type,
			Cycle:          item.Cycle,
			Count:          item.Count,
			Method:         item.Method,
			Title:          item.Title,
			Project:        item.Project,
			Status:         item.Status,
			SendCount:      item.SendCount,
			AdvancedParams: item.AdvancedParams,
			CreateUser:     item.CreateUser,
			UpdateUser:     item.UpdateUser,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}

	return total, result, err
}

// ============================================================
// GetAlerts  拿所有"已启用"告警任务
// ============================================================
// 用途: 给告警调度器（alert_helper）拉最新的"待监控"列表
// ============================================================
func (a AlertService) GetAlerts() ([]dto.AlertDTO, error) {
	var (
		opts   []repo.DBOption
		result []dto.AlertDTO
	)
	opts = append(opts, repo.WithByStatus(constant.AlertEnable))
	alerts, err := alertRepo.List(opts...)
	if err != nil {
		return nil, err
	}
	for _, item := range alerts {

		result = append(result, dto.AlertDTO{
			ID:             item.ID,
			Type:           item.Type,
			Cycle:          item.Cycle,
			Count:          item.Count,
			Method:         item.Method,
			Title:          item.Title,
			Project:        item.Project,
			Status:         item.Status,
			SendCount:      item.SendCount,
			AdvancedParams: item.AdvancedParams,
			CreateUser:     item.CreateUser,
			UpdateUser:     item.UpdateUser,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}

	return result, err
}

// ============================================================
// CreateAlert  创建/更新告警任务（已存在同 type/project 就改；否则新增）
// ============================================================
// 流程:
//   1. 校验社区版通知方式
//   2. 按 (type, project) 查是否已存在
//   3. 存在则 update，不存在则 create
//   4. 触发 InitTask 启动后台调度
// 调用: api/v2.CreateAlert -> this
// ============================================================
func (a AlertService) CreateAlert(create dto.AlertCreate, operator string) error {
	if err := a.validateCommunityAlertMethod(create.Method); err != nil {
		return err
	}
	var alertID uint
	var alertInfo model.Alert
	if create.Project != "" {
		alertInfo, _ := alertRepo.Get(alertRepo.WithByType(create.Type), alertRepo.WithByProject(create.Project))
		alertID = alertInfo.ID
	} else {
		alertInfo, _ := alertRepo.Get(alertRepo.WithByType(create.Type))
		alertID = alertInfo.ID
	}

	if alertID != 0 {
		var upAlert dto.AlertUpdate
		if err := copier.Copy(&upAlert, &create); err != nil {
			return buserr.WithErr("ErrStructTransform", err)
		}
		upAlert.ID = alertID
		err := a.UpdateAlert(upAlert, operator)
		if err != nil {
			return err
		}
	} else {
		alertInfo.Status = constant.AlertEnable
		if err := copier.Copy(&alertInfo, &create); err != nil {
			return buserr.WithErr("ErrStructTransform", err)
		}
		alertInfo.CreateUser = operator
		alertInfo.UpdateUser = operator

		if err := alertRepo.Create(&alertInfo); err != nil {
			return err
		}
		NewIAlertTaskHelper().InitTask(alertInfo.Type)
	}

	return nil
}

// ============================================================
// UpdateAlert  更新告警任务（按 ID）
// ============================================================
func (a AlertService) UpdateAlert(req dto.AlertUpdate, operator string) error {
	if err := a.validateCommunityAlertMethod(req.Method); err != nil {
		return err
	}

	upMap := make(map[string]interface{})
	upMap["id"] = req.ID
	upMap["type"] = req.Type
	upMap["cycle"] = req.Cycle
	upMap["count"] = req.Count
	upMap["method"] = req.Method
	upMap["title"] = req.Title
	upMap["project"] = req.Project
	upMap["status"] = req.Status
	upMap["send_count"] = req.SendCount
	upMap["advanced_params"] = req.AdvancedParams
	upMap["update_user"] = operator

	if err := alertRepo.Update(upMap, repo.WithByID(req.ID)); err != nil {
		return err
	}
	NewIAlertTaskHelper().InitTask(req.Type)
	return nil
}

// ============================================================
// DeleteAlert  按 ID 删除告警任务
// ============================================================
// 流程:
//   1. 校验存在
//   2. 删除
//   3. 重新拿所有启用告警; 还有就重启 InitTask，否则 StopTask
// ============================================================
func (a AlertService) DeleteAlert(id uint) error {
	alertInfo, _ := alertRepo.Get(repo.WithByID(id))
	if alertInfo.ID == 0 {
		return buserr.New("ErrRecordNotFound")
	}
	err := alertRepo.Delete(repo.WithByID(id))
	if err != nil {
		return err
	}
	alerts, err := a.GetAlerts()
	if err != nil {
		return err
	}
	if len(alerts) > 0 {
		NewIAlertTaskHelper().InitTask(alertInfo.Type)
	} else {
		NewIAlertTaskHelper().StopTask()
	}
	return nil
}

// ============================================================
// GetAlert  按 ID 查告警并转 DTO
// ============================================================
func (a AlertService) GetAlert(id uint) (dto.AlertDTO, error) {
	var res dto.AlertDTO
	alertInfo, err := alertRepo.Get(repo.WithByID(id))
	if err != nil {
		return res, err
	}
	_ = copier.Copy(&res, &alertInfo)
	return res, nil
}

// ============================================================
// UpdateStatus  改告警状态（启用/停用）并同步后台任务
// ============================================================
func (a AlertService) UpdateStatus(id uint, status string) error {
	alertInfo, _ := alertRepo.Get(repo.WithByID(id))
	if alertInfo.ID == 0 {
		return buserr.New("ErrRecordNotFound")
	}
	err := alertRepo.Update(map[string]interface{}{"status": status}, repo.WithByID(alertInfo.ID))
	if err != nil {
		return err
	}
	alerts, err := a.GetAlerts()
	if err != nil {
		return err
	}
	if len(alerts) > 0 {
		NewIAlertTaskHelper().InitTask(alertInfo.Type)
	} else {
		NewIAlertTaskHelper().StopTask()
	}
	return nil
}

// ============================================================
// GetDisks  拿所有可用磁盘（用于磁盘空间告警的"选监控对象"）
// ============================================================
// 流程:
//   1. 跑 df -hT 拿挂载点
//   2. 过滤掉 /boot /tmpfs /docker /snap 等
//   3. 并发调 gopsutil disk.Usage 读每个挂载点的使用率（5s 超时）
//   4. 按路径排序
// 调用: api/v2.GetDisks -> this
// ============================================================
func (a AlertService) GetDisks() ([]dto.DiskDTO, error) {
	var disks []dto.DiskDTO
	excludes := map[string]struct{}{
		"/mnt/cdrom": {}, "/boot": {}, "/boot/efi": {}, "/dev": {}, "/dev/shm": {},
		"/run/lock": {}, "/run": {}, "/run/shm": {}, "/run/user": {},
	}
	stdout, err := executeDiskCommand()
	if err != nil {
		return disks, nil
	}

	lines := strings.Split(stdout, "\n")
	var mounts []dto.AlertDiskInfo

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}
		mountPoint := strings.Join(fields[6:], " ")
		if shouldExclude(fields, mountPoint, excludes) {
			continue
		}
		mounts = append(mounts, dto.AlertDiskInfo{Type: fields[1], Device: fields[0], Mount: mountPoint})

	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	wg.Add(len(mounts))
	for i := 0; i < len(mounts); i++ {
		go func(timeoutCh <-chan time.Time, mount dto.AlertDiskInfo) {
			defer wg.Done()

			var itemData dto.DiskDTO
			itemData.Path = mount.Mount
			itemData.Type = mount.Type
			itemData.Device = mount.Device
			select {
			case <-timeoutCh:
				mu.Lock()
				disks = append(disks, itemData)
				mu.Unlock()
				global.LOG.Errorf("load disk info from %s failed, err: timeout", mount.Mount)
			default:
				state, err := disk.Usage(mount.Mount)
				if err != nil {
					mu.Lock()
					disks = append(disks, itemData)
					mu.Unlock()
					global.LOG.Errorf("load disk info from %s failed, err: %v", mount.Mount, err)
					return
				}
				itemData.Total = state.Total
				itemData.Free = state.Free
				itemData.Used = state.Used
				itemData.UsedPercent = state.UsedPercent
				itemData.InodesTotal = state.InodesTotal
				itemData.InodesUsed = state.InodesUsed
				itemData.InodesFree = state.InodesFree
				itemData.InodesUsedPercent = state.InodesUsedPercent
				mu.Lock()
				disks = append(disks, itemData)
				mu.Unlock()
			}
		}(time.After(5*time.Second), mounts[i])
	}
	wg.Wait()

	sort.Slice(disks, func(i, j int) bool {
		return disks[i].Path < disks[j].Path
	})
	return disks, nil
}

// ============================================================
// executeDiskCommand  执行 df 拿磁盘信息（先 hT 再回退 lhT）
// ============================================================
// 流程:
//   1. df -hT -P （标准 POSIX 格式）
//   2. 失败就回退 df -lhT -P
//   3. 过滤 tmpfs / snap / udev
// ============================================================
func executeDiskCommand() (string, error) {
	cmdMgr := cmd.NewCommandMgr(cmd.WithTimeout(2 * time.Second))
	stdout, err := cmdMgr.RunWithStdout("df", "-hT", "-P")
	if err != nil {
		cmdMgr2 := cmd.NewCommandMgr(cmd.WithTimeout(1 * time.Second))
		stdout, err = cmdMgr2.RunWithStdout("df", "-lhT", "-P")
	}
	if err != nil {
		return stdout, err
	}
	var lines []string
	for _, line := range strings.Split(stdout, "\n") {
		if !strings.Contains(line, "/") || strings.Contains(line, "tmpfs") || strings.Contains(line, "snap/core") || strings.Contains(line, "udev") {
			continue
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "", nil
	}
	return strings.Join(lines, "\n"), nil
}

// ============================================================
// shouldExclude  判断某行磁盘是否要排除
// ============================================================
// 规则: /snap 前缀、tmpfs、含 K（小于 1MB）、含 docker、深度过深、在 excludes 列表中
// ============================================================
func shouldExclude(fields []string, mountPoint string, excludes map[string]struct{}) bool {
	if strings.HasPrefix(mountPoint, "/snap") || len(strings.Split(mountPoint, "/")) > 10 {
		return true
	}
	if strings.TrimSpace(fields[1]) == "tmpfs" {
		return true
	}
	if strings.Contains(fields[2], "K") {
		return true
	}
	if strings.Contains(mountPoint, "docker") {
		return true
	}
	_, excluded := excludes[mountPoint]
	return excluded
}

// ============================================================
// PageAlertLogs  分页查告警日志 + 解析 JSON 字段
// ============================================================
// 流程:
//   1. 拼 DBOption
//   2. 调 alertRepo.PageLog
//   3. 逐条调 parseAlertLog 反序列化 AlertDetail/AlertRule
// 调用: api/v2.PageAlertLogs -> this
// ============================================================
func (a AlertService) PageAlertLogs(search dto.AlertLogSearch) (int64, []dto.AlertLogDTO, error) {
	var (
		opts   []repo.DBOption
		result []dto.AlertLogDTO
	)
	if search.Status != "" {
		opts = append(opts, repo.WithByStatus(search.Status))
	}
	if search.Count != 0 {
		opts = append(opts, alertRepo.WithByCount(search.Count))
	}
	if !search.StartTime.IsZero() && !search.EndTime.IsZero() {
		opts = append(opts, repo.WithByCreatedAt(search.StartTime, search.EndTime))
	}
	opts = append(opts, repo.WithOrderDesc("created_at"))

	total, alerts, err := alertRepo.PageLog(search.Page, search.PageSize, opts...)
	if err != nil {
		return 0, nil, err
	}

	for _, item := range alerts {
		alertLogDTO, err := a.parseAlertLog(item)
		if err != nil {
			return 0, nil, err
		}
		result = append(result, alertLogDTO)
	}

	return total, result, err
}

// ============================================================
// parseAlertLog  把 AlertLog 解析成 DTO（展开 JSON 详情/规则字段）
// ============================================================
func (a AlertService) parseAlertLog(item model.AlertLog) (dto.AlertLogDTO, error) {
	var alertDetail dto.AlertDetail
	var alertRule dto.AlertRule

	if err := unmarshalAlertInfo(item.AlertDetail, &alertDetail); err != nil {
		return dto.AlertLogDTO{}, err
	}
	if err := unmarshalAlertInfo(item.AlertRule, &alertRule); err != nil {
		return dto.AlertLogDTO{}, err
	}
	return dto.AlertLogDTO{
		ID:          item.ID,
		Count:       item.Count,
		Type:        item.Type,
		Status:      item.Status,
		Method:      item.Method,
		Message:     item.Message,
		AlertId:     item.AlertId,
		AlertDetail: alertDetail,
		AlertRule:   alertRule,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, nil
}

// ============================================================
// unmarshalAlertInfo  通用 JSON 反序列化（包装错误信息）
// ============================================================
func unmarshalAlertInfo(data string, v interface{}) error {
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return fmt.Errorf("unmarshal alert info vars failed, err: %v", err)
	}
	return nil
}

// ============================================================
// CleanAlertLogs  清空告警日志
// ============================================================
func (a AlertService) CleanAlertLogs() error {
	return alertRepo.CleanAlertLogs()
}

// ============================================================
// GetClams  拿 ClamAV 杀毒任务列表（用于"病毒告警"选来源）
// ============================================================
func (a AlertService) GetClams() ([]dto.ClamDTO, error) {
	var clams []dto.ClamDTO
	clamList, err := clamRepo.List()

	for _, clam := range clamList {
		var clamDTO dto.ClamDTO
		clamDTO.ID = clam.ID
		clamDTO.Name = clam.Name
		clamDTO.Path = clam.Path
		clamDTO.Status = clam.Status
		clamDTO.UpdatedAt = clam.UpdatedAt
		clamDTO.CreatedAt = clam.CreatedAt
		clams = append(clams, clamDTO)
	}
	return clams, err
}

// ============================================================
// GetCronJobs  拿可关联告警的定时任务列表
// ============================================================
func (a AlertService) GetCronJobs(req dto.CronJobReq) ([]dto.CronJobDTO, error) {
	var cronJobs []dto.CronJobDTO
	var (
		opts []repo.DBOption
	)
	if req.Status != "" {
		opts = append(opts, repo.WithByStatus(req.Status))
	}
	if req.Type != "" {
		opts = append(opts, repo.WithByType(req.Type))
	}
	cronjobList, err := cronjobRepo.List(opts...)

	for _, cronJob := range cronjobList {
		var cronJobDTO dto.CronJobDTO
		cronJobDTO.ID = cronJob.ID
		cronJobDTO.Name = cronJob.Name
		cronJobDTO.Status = cronJob.Status
		cronJobDTO.Type = cronJob.Type
		cronJobDTO.UpdatedAt = cronJob.UpdatedAt
		cronJobDTO.CreatedAt = cronJob.CreatedAt
		cronJobs = append(cronJobs, cronJobDTO)
	}
	return cronJobs, err
}

// ============================================================
// GetAlertConfig  按排除类型查"启用"通知渠道
// ============================================================
// 用途: 告警任务新建时选择通知方式
// ============================================================
func (a AlertService) GetAlertConfig(req dto.AlertConfigQuery) ([]model.AlertConfig, error) {
	var (
		opts    []repo.DBOption
		configs []model.AlertConfig
	)
	if len(req.ExcludeTypes) > 0 {
		opts = append(opts, alertRepo.WithByTypeNotIn(req.ExcludeTypes))
	}
	opts = append(opts, repo.WithByStatus(constant.AlertEnable))
	configs, err := alertRepo.AlertConfigList(opts...)
	return configs, err
}

// ============================================================
// PageAlertConfig  分页查通知渠道（默认排除 common）
// ============================================================
func (a AlertService) PageAlertConfig(req dto.AlertConfigPageReq) (int64, []model.AlertConfig, error) {
	opts := []repo.DBOption{
		alertRepo.WithByTypeNotIn([]string{"common"}),
		repo.WithOrderDesc("created_at"),
	}
	if len(req.ExcludeTypes) > 0 {
		opts = append(opts, alertRepo.WithByTypeNotIn(req.ExcludeTypes))
	}
	return alertRepo.PageAlertConfig(req.Page, req.PageSize, opts...)
}

// ============================================================
// UpdateAlertConfig  更新或新增通知渠道（含唯一性校验）
// ============================================================
// 校验:
//   - validateCommunityAlertConfigType (社区版限制)
//   - displayName 唯一
//   - SMS 手机号唯一
// ============================================================
func (a AlertService) UpdateAlertConfig(req dto.AlertConfigUpdate, operator string) error {
	if err := a.validateCommunityAlertConfigType(req.Type); err != nil {
		return err
	}
	if err := a.checkAlertConfigDisplayNameUnique(req); err != nil {
		return err
	}
	if err := a.checkAlertConfigSMSPhoneUnique(req); err != nil {
		return err
	}
	if req.ID != 0 {
		upMap := make(map[string]interface{})
		upMap["id"] = req.ID
		upMap["type"] = req.Type
		upMap["title"] = req.Title
		upMap["status"] = req.Status
		upMap["config"] = req.Config
		upMap["update_user"] = operator
		if err := alertRepo.UpdateAlertConfig(upMap, repo.WithByID(req.ID)); err != nil {
			return err
		}
	} else {
		var alertConfig model.AlertConfig
		if err := copier.Copy(&alertConfig, &req); err != nil {
			return buserr.WithErr("ErrStructTransform", err)
		}
		alertConfig.CreateUser = operator
		alertConfig.UpdateUser = operator
		if err := alertRepo.CreateAlertConfig(&alertConfig); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// checkAlertConfigSMSPhoneUnique  校验 SMS 手机号不重复
// ============================================================
func (a AlertService) checkAlertConfigSMSPhoneUnique(req dto.AlertConfigUpdate) error {
	if req.Type != constant.SMSConfig {
		return nil
	}

	phone := alertConfigSMSPhone(req.Config)
	configs, err := alertRepo.AlertConfigList(alertRepo.WithByType(req.Type))
	if err != nil {
		return err
	}

	for _, config := range configs {
		if req.ID != 0 && config.ID == req.ID {
			continue
		}
		if alertConfigSMSPhone(config.Config) == phone {
			return buserr.New("ErrAlertConfigPhoneExist")
		}
	}

	return nil
}

// ============================================================
// checkAlertConfigDisplayNameUnique  校验通知渠道"显示名"不重复
// ============================================================
func (a AlertService) checkAlertConfigDisplayNameUnique(req dto.AlertConfigUpdate) error {
	displayName := alertConfigDisplayName(req.Type, req.Config)
	if displayName == "" {
		return nil
	}

	configs, err := alertRepo.AlertConfigList(alertRepo.WithByType(req.Type))
	if err != nil {
		return err
	}

	for _, config := range configs {
		if req.ID != 0 && config.ID == req.ID {
			continue
		}
		if alertConfigDisplayName(config.Type, config.Config) == displayName {
			return buserr.New("ErrNameIsExist")
		}
	}

	return nil
}

// ============================================================
// validateCommunityAlertMethod  社区版通知方式限制（不允许 WeCom/DingTalk/FeiShu/SMS）
// ============================================================
func (a AlertService) validateCommunityAlertMethod(method string) error {
	if global.CONF.Base.IsEnterprise || global.CONF.Base.Edition == "cn" {
		return nil
	}
	if strings.TrimSpace(method) == "" {
		return nil
	}

	for _, item := range strings.Split(method, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if configID, err := strconv.ParseUint(item, 10, 64); err == nil {
			config, err := alertRepo.GetConfigById(uint(configID))
			if err != nil {
				return err
			}
			if _, ok := communityAlertMethodTypeNames[config.Type]; ok {
				return buserr.WithErr("ErrAlertMethodNotSupported", nil)
			}
			continue
		}
		if _, ok := communityAlertMethodTypeNames[item]; ok {
			return buserr.WithErr("ErrAlertMethodNotSupported", nil)
		}
	}

	return nil
}

// ============================================================
// validateCommunityAlertConfigType  社区版通知渠道类型限制
// ============================================================
func (a AlertService) validateCommunityAlertConfigType(configType string) error {
	if global.CONF.Base.IsEnterprise || global.CONF.Base.Edition == "cn" {
		return nil
	}
	if _, ok := communityAlertMethodTypeNames[configType]; ok {
		return buserr.WithErr("ErrAlertMethodNotSupported", nil)
	}
	return nil
}

// ============================================================
// alertConfigDisplayName  从渠道 config JSON 里提"显示名"（用于唯一性校验）
// ============================================================
func alertConfigDisplayName(configType, configData string) string {
	switch configType {
	case constant.Email, constant.WeCom, constant.DingTalk, constant.FeiShu, constant.Bark, constant.SMS:
		var cfg struct {
			DisplayName string `json:"displayName"`
		}
		if err := json.Unmarshal([]byte(configData), &cfg); err != nil {
			return ""
		}
		return strings.TrimSpace(cfg.DisplayName)
	default:
		return ""
	}
}

// ============================================================
// alertConfigSMSPhone  从 SMS config JSON 里提"手机号"
// ============================================================
func alertConfigSMSPhone(configData string) string {
	var cfg struct {
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal([]byte(configData), &cfg); err != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Phone)
}

// ============================================================
// DeleteAlertConfig  删除通知渠道（含引用检查）
// ============================================================
// 流程:
//   1. 检查渠道存在
//   2. 查"是否有告警任务的 method 含此 id"，有就拒绝
//   3. 删除
// ============================================================
func (a AlertService) DeleteAlertConfig(id uint) error {
	_, err := alertRepo.GetConfigById(id)
	if err != nil {
		return err
	}
	usedAlerts, err := alertRepo.List(alertRepo.WithByAlertMethodContainsConfigID(id))
	if err != nil {
		return err
	}
	if len(usedAlerts) > 0 {
		return buserr.New("ErrAlertConfigInUse")
	}
	return alertRepo.DeleteAlertConfig(repo.WithByID(id))
}

// ============================================================
// TestAlertConfig  测通知渠道（发一封测试邮件）
// ============================================================
// 流程:
//   1. 组装 SMTP 配置（含 displayName 编码）
//   2. 拿多节点 transport
//   3. 调 email.SendMail 发一封 i18n 消息
//   4. 返是否成功
// ============================================================
func (a AlertService) TestAlertConfig(req dto.AlertConfigTest) (bool, error) {
	username := req.UserName
	if username == "" {
		username = req.Sender
	}
	encodedDisplayName := mime.BEncoding.Encode("UTF-8", req.DisplayName)
	cfg := email.SMTPConfig{
		Host:       req.Host,
		Port:       req.Port,
		Sender:     req.Sender,
		Username:   username,
		Password:   req.Password,
		From:       fmt.Sprintf(`"%s" <%s>`, encodedDisplayName, req.Sender),
		Encryption: req.Encryption,
		Recipient:  req.Recipient,
	}

	msg := email.EmailMessage{
		Subject: i18n.GetMsgByKey("TestAlertTitle"),
		Body:    i18n.GetMsgByKey("TestAlert"),
		IsHTML:  false,
	}
	transport := xpack.MultiNodeProvider.LoadRequestTransport()
	if err := email.SendMail(cfg, msg, transport); err != nil {
		return false, err
	}
	return true, nil
}

// ============================================================
// ExternalUpdateAlert  外部触发更新告警（按 sendCount 决定启停）
// ============================================================
// 流程:
//   1. 校验社区版通知方式
//   2. sendCount==0 时禁用，否则启用
//   3. 按 (type, project) 查; 存在则差量更新; 不存在则创建
// 用途: 授权计费/license 系统回调用
// ============================================================
func (a AlertService) ExternalUpdateAlert(updateAlert dto.AlertCreate, operator string) error {
	if err := a.validateCommunityAlertMethod(updateAlert.Method); err != nil {
		return err
	}
	upMap := make(map[string]interface{})
	var newStatus string
	if updateAlert.SendCount == 0 {
		newStatus = constant.AlertDisable
	} else {
		newStatus = constant.AlertEnable
		upMap["send_count"] = updateAlert.SendCount
		if updateAlert.Method != "" {
			upMap["method"] = updateAlert.Method
		}
	}
	upMap["status"] = newStatus

	alertInfo, _ := alertRepo.Get(
		alertRepo.WithByType(updateAlert.Type),
		alertRepo.WithByProject(updateAlert.Project),
	)

	if alertInfo.ID > 0 {
		shouldUpdate := false

		if alertInfo.Status != newStatus {
			shouldUpdate = true
		}
		if val, ok := upMap["send_count"]; ok && val != alertInfo.SendCount {
			shouldUpdate = true
		}
		if val, ok := upMap["method"]; ok && val != "" && val != alertInfo.Method {
			shouldUpdate = true
		}

		if shouldUpdate {
			if err := alertRepo.Update(
				upMap,
				alertRepo.WithByProject(updateAlert.Project),
				alertRepo.WithByType(updateAlert.Type),
			); err != nil {
				return err
			}
		}
	} else {
		if updateAlert.Method != "" && updateAlert.Title != "" {
			updateAlert.Status = newStatus
			if err := a.CreateAlert(updateAlert, operator); err != nil {
				return err
			}
		}
	}

	return nil
}

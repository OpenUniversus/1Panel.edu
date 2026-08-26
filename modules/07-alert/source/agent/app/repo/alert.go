// =============================================================================
// 模块: Alert 告警 (agent/app/repo/alert.go)
// 文件: alert.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package repo

import (
	"encoding/json"
	"strings"

	"github.com/1Panel-dev/1Panel/agent/app/model"
	"github.com/1Panel-dev/1Panel/agent/constant"
	"github.com/1Panel-dev/1Panel/agent/global"
	"google.golang.org/genproto/googleapis/type/date"
	"gorm.io/gorm"
	"strconv"
	"time"
)

// ============================================================
// AlertRepo  告警数据库访问对象（封装 GORM 操作）
// ============================================================
// 方法:
//   - WithXxx(...) — 一组查询条件构造器（链式 DBOption）
//   - Create / Save / Get / Page / List / Update / Delete — 告警表 CRUD
//   - GetLog / CreateLog / UpdateLog / PageLog / ListLog / DeleteLog / CleanAlertLogs — 告警日志 CRUD
//   - CreateAlertTask / DeleteAlertTask / GetAlertTask / LoadTaskCount / GetTaskLog / GetLicensePushCount — 任务统计
//   - GetConfig / GetConfigById / AlertConfigList / UpdateAlertConfig / CreateAlertConfig / DeleteAlertConfig / PageAlertConfig / SyncAll — 通知渠道配置
// ============================================================
type AlertRepo struct{}

// ============================================================
// IAlertRepo  告警数据库访问接口（便于注入/测试）
// ============================================================
// 主要方法:
//   - WithByXxx — 各种 DBOption 条件
//   - Alert CRUD: Create/Save/Get/Page/List/Update/Delete
//   - AlertLog CRUD + CleanAlertLogs
//   - AlertTask + 统计查询
//   - AlertConfig CRUD + SyncAll（全量同步）
// ============================================================
type IAlertRepo interface {
	WithByType(alertType string) DBOption
	WithByStatusIn(status []string) DBOption
	WithByProject(project string) DBOption
	WithByCount(count uint) DBOption
	WithByAlertId(alertId uint) DBOption
	WithByCreateAt(date *date.Date) DBOption
	WithByLicenseId(licenseId string) DBOption
	WithByRecordId(recordId uint) DBOption
	WithByAlertMethodContainsConfigID(id uint) DBOption
	WithByMethodConfigIDs(ids []uint) DBOption

	Create(alert *model.Alert) error
	Get(opts ...DBOption) (model.Alert, error)
	Page(page, size int, opts ...DBOption) (int64, []model.Alert, error)
	List(opts ...DBOption) ([]model.Alert, error)
	Delete(opts ...DBOption) error
	Save(alert *model.Alert) error
	Update(maps map[string]interface{}, opts ...DBOption) error

	GetLog(opts ...DBOption) (model.AlertLog, error)
	CreateLog(alertLog *model.AlertLog) error
	PageLog(limit, offset int, opts ...DBOption) (int64, []model.AlertLog, error)
	ListLog(opts ...DBOption) ([]model.AlertLog, error)
	UpdateLog(id uint, maps map[string]interface{}) error
	BatchUpdateLogBy(maps map[string]interface{}, opts ...DBOption) error
	DeleteLog(opts ...DBOption) error
	CleanAlertLogs() error

	CreateAlertTask(alertTaskBase *model.AlertTask) error
	DeleteAlertTask(opts ...DBOption) error
	GetAlertTask(opts ...DBOption) (model.AlertTask, error)
	LoadTaskCount(alertType string, project string, method string) (uint, uint, error)
	GetTaskLog(alertType string, alertId uint) (time.Time, error)
	GetLicensePushCount(method string) (uint, error)

	GetConfig(opts ...DBOption) (model.AlertConfig, error)
	GetConfigById(id uint) (model.AlertConfig, error)
	AlertConfigList(opts ...DBOption) ([]model.AlertConfig, error)
	UpdateAlertConfig(maps map[string]interface{}, opts ...DBOption) error
	CreateAlertConfig(config *model.AlertConfig) error
	DeleteAlertConfig(opts ...DBOption) error

	WithByTypeNotIn(types []string) DBOption
	PageAlertConfig(page, size int, opts ...DBOption) (int64, []model.AlertConfig, error)

	SyncAll(data []model.AlertConfig) error
}

// ============================================================
// NewIAlertRepo  构造 IAlertRepo 默认实现
// ============================================================
func NewIAlertRepo() IAlertRepo {
	return &AlertRepo{}
}

// ============================================================
// WithByType  构造"按 type 过滤"的 DBOption
// ============================================================
func (a *AlertRepo) WithByType(alertType string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("`type` = ?", alertType)
	}
}

// ============================================================
// WithByStatusIn  构造"按 status in (...)"过滤的 DBOption
// ============================================================
func (a *AlertRepo) WithByStatusIn(status []string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("status in (?)", status)
	}
}

// ============================================================
// WithByCount  构造"按 count 过滤"的 DBOption
// ============================================================
func (a *AlertRepo) WithByCount(count uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("count = ?", count)
	}
}

// ============================================================
// WithByProject  构造"按 project 过滤"的 DBOption
// ============================================================
func (a *AlertRepo) WithByProject(project string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("project = ?", project)
	}
}

// ============================================================
// WithByAlertId  构造"按 alert_id 过滤"的 DBOption（日志用）
// ============================================================
func (a *AlertRepo) WithByAlertId(alertId uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("alert_id = ?", alertId)
	}
}

// ============================================================
// WithByLicenseId  构造"按 license_id 过滤"的 DBOption
// ============================================================
func (a *AlertRepo) WithByLicenseId(licenseId string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("license_id = ?", licenseId)
	}
}

// ============================================================
// WithByRecordId  构造"按 record_id 过滤"的 DBOption
// ============================================================
func (a *AlertRepo) WithByRecordId(recordId uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("record_id = ?", recordId)
	}
}

// ============================================================
// WithByAlertMethodContainsConfigID  构造"method 字段含该 config id"的过滤
// ============================================================
// 作用:
//   - 因为 method 字段是逗号分隔的 config id 列表，需要 4 种 LIKE 匹配
// ============================================================
func (a *AlertRepo) WithByAlertMethodContainsConfigID(id uint) DBOption {
	method := strconv.Itoa(int(id))
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("(method = ? OR method LIKE ? OR method LIKE ? OR method LIKE ?)", method, method+",%", "%,"+method, "%,"+method+",%")
	}
}

// ============================================================
// WithByMethodConfigIDs  构造"method IN [id1,id2,...]"的过滤
// ============================================================
func (a *AlertRepo) WithByMethodConfigIDs(ids []uint) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		methods := make([]string, 0, len(ids))
		for _, id := range ids {
			methods = append(methods, strconv.Itoa(int(id)))
		}
		return g.Where("method IN ?", methods)
	}
}

// ============================================================
// WithByCreateAt  构造"按创建日期过滤"的 DBOption（同一天）
// ============================================================
func (a *AlertRepo) WithByCreateAt(createAt *date.Date) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("DATE(created_at) = DATE(?)", createAt)
	}
}

// ============================================================
// Create  新增一条告警任务
// ============================================================
func (a *AlertRepo) Create(alert *model.Alert) error {
	return global.AlertDB.Model(&model.Alert{}).Create(alert).Error
}

// ============================================================
// Save  保存告警（全量 upsert）
// ============================================================
func (a *AlertRepo) Save(alert *model.Alert) error {
	return global.AlertDB.Save(alert).Error
}

// ============================================================
// Get  按条件取一条告警
// ============================================================
func (a *AlertRepo) Get(opts ...DBOption) (model.Alert, error) {
	var alert model.Alert
	db, _ := getAlertDB(opts...)
	err := db.First(&alert).Error
	return alert, err
}

// ============================================================
// Page  分页查告警 + 返回总数
// ============================================================
func (a *AlertRepo) Page(page, size int, opts ...DBOption) (int64, []model.Alert, error) {
	var alerts []model.Alert
	alertDb, _ := getAlertDB(opts...)
	db := alertDb.Model(&model.Alert{})
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&alerts).Error
	return count, alerts, err
}

// ============================================================
// List  查所有符合条件告警（不分页）
// ============================================================
func (a *AlertRepo) List(opts ...DBOption) ([]model.Alert, error) {
	var alert []model.Alert
	db, _ := getAlertDB(opts...)
	err := db.Find(&alert).Error
	return alert, err
}

// ============================================================
// Update  按条件批量更新告警字段
// ============================================================
func (a *AlertRepo) Update(maps map[string]interface{}, opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Model(&model.Alert{}).Updates(maps).Error
}

// ============================================================
// Delete  按条件删除告警
// ============================================================
func (a *AlertRepo) Delete(opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Delete(&model.Alert{}).Error
}

// ============================================================
// GetLog  按条件取一条告警日志
// ============================================================
func (a *AlertRepo) GetLog(opts ...DBOption) (model.AlertLog, error) {
	var alertLog model.AlertLog
	db, _ := getAlertDB(opts...)
	err := db.First(&alertLog).Error
	return alertLog, err
}

// ============================================================
// CreateLog  新增一条告警日志
// ============================================================
func (a *AlertRepo) CreateLog(log *model.AlertLog) error {
	return global.AlertDB.Model(&model.AlertLog{}).Create(&log).Error
}

// ============================================================
// UpdateLog  按 ID 更新告警日志
// ============================================================
func (a *AlertRepo) UpdateLog(id uint, maps map[string]interface{}) error {
	return global.AlertDB.Model(&model.AlertLog{}).Where("id = ?", id).Updates(maps).Error
}

// ============================================================
// BatchUpdateLogBy  按条件批量更新告警日志
// ============================================================
func (a *AlertRepo) BatchUpdateLogBy(maps map[string]interface{}, opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	if len(opts) == 0 {
		db = db.Where("1=1")
	}
	return db.Model(&model.AlertLog{}).Updates(&maps).Error
}

// ============================================================
// PageLog  分页查告警日志（按 created_at desc）
// ============================================================
func (a *AlertRepo) PageLog(page, size int, opts ...DBOption) (int64, []model.AlertLog, error) {
	var alerts []model.AlertLog
	db := global.AlertDB.Model(&model.AlertLog{})
	for _, opt := range opts {
		db = opt(db)
	}
	count := int64(0)
	db = db.Order("created_at desc").Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&alerts).Error
	return count, alerts, err
}

// ============================================================
// ListLog  查所有符合条件的告警日志
// ============================================================
func (a *AlertRepo) ListLog(opts ...DBOption) ([]model.AlertLog, error) {
	var alertLog []model.AlertLog
	db, _ := getAlertDB(opts...)
	err := db.Find(&alertLog).Error
	return alertLog, err
}

// ============================================================
// DeleteLog  按条件删除告警日志
// ============================================================
func (a *AlertRepo) DeleteLog(opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Delete(&model.AlertLog{}).Error
}

// ============================================================
// CleanAlertLogs  清空所有告警日志
// ============================================================
func (a *AlertRepo) CleanAlertLogs() error {
	return global.AlertDB.Where("1 = 1").Delete(&model.AlertLog{}).Error
}

// ============================================================
// CreateAlertTask  新增一条 AlertTask（系统内置的"今日推送"统计记录）
// ============================================================
func (a *AlertRepo) CreateAlertTask(alertTaskBase *model.AlertTask) error {
	return global.AlertDB.Model(&model.AlertTask{}).Create(&alertTaskBase).Error
}

// ============================================================
// DeleteAlertTask  按条件删除 AlertTask
// ============================================================
func (a *AlertRepo) DeleteAlertTask(opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Delete(&model.AlertTask{}).Error
}

// ============================================================
// GetAlertTask  按条件取一条 AlertTask
// ============================================================
func (a *AlertRepo) GetAlertTask(opts ...DBOption) (model.AlertTask, error) {
	var data model.AlertTask
	db, _ := getAlertDB(opts...)
	err := db.First(&data).Error
	return data, err
}

// ============================================================
// LoadTaskCount  拿 (今日次数, 总次数) — 用于告警频率限制
// ============================================================
// 流程:
//   1. 总次数 = AlertTask 全部 count
//   2. 今日次数 = 今日 0 点到次日 0 点范围内
// ============================================================
func (a *AlertRepo) LoadTaskCount(alertType string, project string, method string) (uint, uint, error) {
	var (
		todayCount int64
		totalCount int64
	)
	_ = global.AlertDB.Model(&model.AlertTask{}).Where("type = ? AND quota_type = ? AND method = ?", alertType, project, method).Count(&totalCount).Error

	now := time.Now()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowMidnight := todayMidnight.Add(24 * time.Hour)
	err := global.AlertDB.Model(&model.AlertTask{}).Where("type =  ? AND quota_type = ?  AND method = ? AND created_at > ? AND created_at < ?", alertType, project, method, todayMidnight, tomorrowMidnight).Count(&todayCount).Error
	return uint(todayCount), uint(totalCount), err
}

// ============================================================
// GetTaskLog  拿某个告警今日最近一次成功推送的时间
// ============================================================
// 作用:
//   - 用于"告警冷却"判断（不到冷却时间不再发）
// ============================================================
func (a *AlertRepo) GetTaskLog(alertType string, alertId uint) (time.Time, error) {
	var newDate time.Time
	status := []string{constant.AlertSuccess, constant.AlertPushSuccess, constant.AlertSyncError, constant.AlertPushing}
	now := time.Now()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowMidnight := todayMidnight.Add(24 * time.Hour)
	err := global.AlertDB.Model(&model.AlertLog{}).
		Where("type = ? AND alert_id = ? AND status in ? AND created_at > ? AND created_at < ?", alertType, alertId, status, todayMidnight, tomorrowMidnight).
		Order("created_at DESC").
		Limit(1).
		Pluck("created_at", &newDate).Error
	if err != nil {
		return time.Time{}, err
	}

	if newDate.IsZero() {
		return time.Time{}, nil
	}

	return newDate, nil
}

// ============================================================
// getAlertDB  工具方法：把 DBOption 链式应用到全局 AlertDB
// ============================================================
func getAlertDB(opts ...DBOption) (*gorm.DB, error) {
	var db *gorm.DB
	db = global.AlertDB
	for _, opt := range opts {
		db = opt(db)
	}
	return db, nil
}

// ============================================================
// GetLicensePushCount  今日某个 method 的推送计数
// ============================================================
func (a *AlertRepo) GetLicensePushCount(method string) (uint, error) {
	var (
		todayCount int64
	)
	now := time.Now()
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	tomorrowMidnight := todayMidnight.Add(24 * time.Hour)
	err := global.AlertDB.Model(&model.AlertTask{}).Where("created_at > ? AND created_at < ? AND method = ?", todayMidnight, tomorrowMidnight, method).Count(&todayCount).Error
	return uint(todayCount), err
}

// ============================================================
// AlertConfigList  查所有通知渠道配置
// ============================================================
func (a *AlertRepo) AlertConfigList(opts ...DBOption) ([]model.AlertConfig, error) {
	var config []model.AlertConfig
	db, _ := getAlertDB(opts...)
	err := db.Find(&config).Error
	return config, err
}

// ============================================================
// UpdateAlertConfig  按条件更新通知渠道
// ============================================================
func (a *AlertRepo) UpdateAlertConfig(maps map[string]interface{}, opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Model(&model.AlertConfig{}).Updates(maps).Error
}

// ============================================================
// CreateAlertConfig  新增通知渠道
// ============================================================
func (a *AlertRepo) CreateAlertConfig(config *model.AlertConfig) error {
	return global.AlertDB.Model(&model.AlertConfig{}).Create(config).Error
}

// ============================================================
// DeleteAlertConfig  按条件删除通知渠道
// ============================================================
func (a *AlertRepo) DeleteAlertConfig(opts ...DBOption) error {
	db, _ := getAlertDB(opts...)
	return db.Delete(&model.AlertConfig{}).Error
}

// ============================================================
// GetConfig  按条件取一条通知渠道
// ============================================================
func (a *AlertRepo) GetConfig(opts ...DBOption) (model.AlertConfig, error) {
	var alertConfig model.AlertConfig
	db, _ := getAlertDB(opts...)
	err := db.First(&alertConfig).Error
	return alertConfig, err
}

// ============================================================
// GetConfigById  按主键 ID 取通知渠道
// ============================================================
func (a *AlertRepo) GetConfigById(id uint) (model.AlertConfig, error) {
	var config model.AlertConfig
	err := global.AlertDB.First(&config, id).Error
	return config, err
}

// ============================================================
// WithByTypeNotIn  构造"type NOT IN (...)"的 DBOption
// ============================================================
func (a *AlertRepo) WithByTypeNotIn(types []string) DBOption {
	return func(g *gorm.DB) *gorm.DB {
		return g.Where("`type` NOT IN (?)", types)
	}
}

// ============================================================
// PageAlertConfig  分页查通知渠道 + 总数
// ============================================================
func (a *AlertRepo) PageAlertConfig(page, size int, opts ...DBOption) (int64, []model.AlertConfig, error) {
	var configs []model.AlertConfig
	db := global.AlertDB.Model(&model.AlertConfig{})
	for _, opt := range opts {
		db = opt(db)
	}
	count := int64(0)
	db = db.Count(&count)
	err := db.Limit(size).Offset(size * (page - 1)).Find(&configs).Error
	return count, configs, err
}

// ============================================================
// singletonTypes  单例类型的渠道（全局唯一，不能新增多个）
// ============================================================
// 当前: CommonConfig（通用配置）
// ============================================================
var singletonTypes = map[string]bool{
	constant.CommonConfig: true,
}

// ============================================================
// SyncAll  全量同步通知渠道（按"唯一键"匹配，删除多余）
// ============================================================
// 参数:
//   - data — 新配置列表
// 流程:
//   1. 拉老配置
//   2. 单例类按 type 复用；其他按 (type+config) 匹配
//   3. 匹配到则 Save，否则 Create
//   4. 没被消耗的老配置 + 没被任何告警引用的 删除
// ============================================================
func (a *AlertRepo) SyncAll(data []model.AlertConfig) error {
	tx := global.AlertDB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var oldConfigs []model.AlertConfig
	if err := tx.Find(&oldConfigs).Error; err != nil {
		tx.Rollback()
		return err
	}

	usedConfigIDs, err := loadUsedAlertConfigIDs(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	oldConfigMap := make(map[string]uint)
	oldConfigByType := make(map[string][]model.AlertConfig)
	oldConfigByKey := make(map[string][]model.AlertConfig)
	consumedConfigIDs := make(map[uint]struct{})
	for _, item := range oldConfigs {
		if singletonTypes[item.Type] {
			oldConfigMap[item.Type] = item.ID
			continue
		}
		oldConfigByType[item.Type] = append(oldConfigByType[item.Type], item)
		oldConfigByKey[alertConfigSyncKey(item)] = append(oldConfigByKey[alertConfigSyncKey(item)], item)
	}
	for _, item := range data {
		if singletonTypes[item.Type] {
			if val, ok := oldConfigMap[item.Type]; ok {
				item.ID = val
				delete(oldConfigMap, item.Type)
				consumedConfigIDs[item.ID] = struct{}{}
			} else {
				item.ID = 0
			}
			if item.ID == 0 {
				if err := tx.Create(&item).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else if err := tx.Save(&item).Error; err != nil {
				tx.Rollback()
				return err
			}
			continue
		}

		key := alertConfigSyncKey(item)
		if matched, ok := popAlertConfigByKey(oldConfigByKey, key); ok {
			item.ID = matched.ID
			consumedConfigIDs[item.ID] = struct{}{}
			if err := tx.Save(&item).Error; err != nil {
				tx.Rollback()
				return err
			}
			deleteAlertConfigByID(oldConfigByType, matched.ID)
			continue
		}

		if matched, ok := popUnusedAlertConfigByType(oldConfigByType, usedConfigIDs, item.Type); ok {
			item.ID = matched.ID
			consumedConfigIDs[item.ID] = struct{}{}
			if err := tx.Save(&item).Error; err != nil {
				tx.Rollback()
				return err
			}
			continue
		}

		item.ID = 0
		if err := tx.Create(&item).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	for _, item := range oldConfigs {
		if _, used := usedConfigIDs[item.ID]; used {
			continue
		}
		if _, kept := consumedConfigIDs[item.ID]; kept {
			continue
		}
		if err := tx.Where("id = ?", item.ID).Delete(&model.AlertConfig{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// ============================================================
// loadUsedAlertConfigIDs  拿所有"被某条告警任务引用"的 config id 集合
// ============================================================
// 作用:
//   - 同步时只删"未被引用"的老配置
// ============================================================
func loadUsedAlertConfigIDs(tx *gorm.DB) (map[uint]struct{}, error) {
	var alerts []model.Alert
	if err := tx.Select("method").Find(&alerts).Error; err != nil {
		return nil, err
	}

	usedIDs := make(map[uint]struct{})
	for _, alert := range alerts {
		for _, item := range strings.Split(alert.Method, ",") {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			id, err := strconv.ParseUint(item, 10, 64)
			if err != nil {
				continue
			}
			usedIDs[uint(id)] = struct{}{}
		}
	}
	return usedIDs, nil
}

// ============================================================
// alertConfigSyncKey  算一条渠道的"同步键" (type + 标准化后的 config JSON)
// ============================================================
func alertConfigSyncKey(item model.AlertConfig) string {
	return item.Type + "::" + normalizeAlertConfigJSON(item.Config)
}

// ============================================================
// normalizeAlertConfigJSON  把 config 字符串重新序列化以"去空格稳定"
// ============================================================
// 流程:
//   1. 尝试 JSON 解析
//   2. 成功就 Marshal（标准化格式）
//   3. 失败就返原 trimmed 串
// ============================================================
func normalizeAlertConfigJSON(config string) string {
	trimmed := strings.TrimSpace(config)
	if trimmed == "" {
		return ""
	}

	var data any
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return trimmed
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return trimmed
	}
	return string(buf)
}

// ============================================================
// popAlertConfigByKey  从 key->列表 中弹出一项（同步算法辅助）
// ============================================================
func popAlertConfigByKey(configMap map[string][]model.AlertConfig, key string) (model.AlertConfig, bool) {
	items := configMap[key]
	if len(items) == 0 {
		return model.AlertConfig{}, false
	}

	item := items[0]
	if len(items) == 1 {
		delete(configMap, key)
	} else {
		configMap[key] = items[1:]
	}
	return item, true
}

// ============================================================
// popUnusedAlertConfigByType  弹出一个"未使用且同 type"的老配置
// ============================================================
func popUnusedAlertConfigByType(configMap map[string][]model.AlertConfig, usedConfigIDs map[uint]struct{}, configType string) (model.AlertConfig, bool) {
	items := configMap[configType]
	if len(items) == 0 {
		return model.AlertConfig{}, false
	}

	for idx, item := range items {
		if _, used := usedConfigIDs[item.ID]; used {
			continue
		}
		if idx == 0 {
			if len(items) == 1 {
				delete(configMap, configType)
			} else {
				configMap[configType] = items[1:]
			}
		} else {
			configMap[configType] = append(items[:idx], items[idx+1:]...)
		}
		return item, true
	}
	return model.AlertConfig{}, false
}

// ============================================================
// deleteAlertConfigByID  从嵌套 map 中删掉一个指定 ID
// ============================================================
func deleteAlertConfigByID(configMap map[string][]model.AlertConfig, id uint) {
	for key, items := range configMap {
		for idx, item := range items {
			if item.ID != id {
				continue
			}
			if len(items) == 1 {
				delete(configMap, key)
			} else {
				configMap[key] = append(items[:idx], items[idx+1:]...)
			}
			return
		}
	}
}

// =============================================================================
// 模块: Alert 告警 (agent/app/model/alert.go)
// 文件: alert.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// ============================================================
// Alert  告警任务表（持久化到数据库）
// ============================================================
// 字段:
//   - Title (string) — 告警任务名
//   - Type (string) — 告警类型 (disk/cpu/site/cron 等)
//   - Cycle (uint) — 触发周期（分钟）
//   - Count (uint) — 触发次数
//   - Project (string) — 关联项目
//   - Status (string) — 启用/停用
//   - Method (string) — 通知方式 JSON
//   - SendCount (uint) — 累计发送次数
//   - AdvancedParams (string) — 高级参数 JSON
//   - CreateUser / UpdateUser (string) — 创建/更新人
// ============================================================
type Alert struct {
	BaseModel

	Title          string `gorm:"type:varchar(256);not null" json:"title"`
	Type           string `gorm:"type:varchar(64);not null" json:"type"`
	Cycle          uint   `gorm:"type:integer;not null" json:"cycle"`
	Count          uint   `gorm:"type:integer;not null" json:"count"`
	Project        string `gorm:"type:varchar(64)" json:"project"`
	Status         string `gorm:"type:varchar(64);not null" json:"status"`
	Method         string `gorm:"type:text;not null" json:"method"`
	SendCount      uint   `gorm:"type:integer" json:"sendCount"`
	AdvancedParams string `gorm:"type:longText" json:"advancedParams"`
	CreateUser     string `gorm:"type:varchar(256)" json:"createUser"`
	UpdateUser     string `gorm:"type:varchar(256)" json:"updateUser"`
}

// ============================================================
// AlertTask  告警任务触发计划（系统内置）
// ============================================================
// 字段:
//   - Type (string) — 任务类型
//   - Quota (string) — 阈值
//   - QuotaType (string) — 阈值单位
//   - Method (string) — 默认通知方式 sms
// ============================================================
type AlertTask struct {
	BaseModel
	Type      string `gorm:"type:varchar(64);not null" json:"type"`
	Quota     string `gorm:"type:varchar(64)" json:"quota"`
	QuotaType string `gorm:"type:varchar(64)" json:"quotaType"`
	Method    string `gorm:"type:varchar(128);not null;default:'sms'" json:"method"`
}

// ============================================================
// AlertLog  告警日志表（每次触发都记一条）
// ============================================================
// 字段:
//   - Type / Status (string) — 告警类型 / 处理状态
//   - AlertId (uint) — 关联的告警任务 ID
//   - AlertDetail / AlertRule (string) — 详情/规则
//   - Count (uint) — 触发次数
//   - Message (string) — 通知消息
//   - RecordId (uint) — 关联的子记录 ID
//   - LicenseId (string) — 授权 ID
//   - Method (string) — 通知方式
// ============================================================
type AlertLog struct {
	BaseModel

	Type        string `gorm:"type:varchar(64);not null" json:"type"`
	Status      string `gorm:"type:varchar(64);not null" json:"status"`
	AlertId     uint   `gorm:"type:integer;not null" json:"alertId"`
	AlertDetail string `gorm:"type:varchar(256);not null" json:"alertDetail"`
	AlertRule   string `gorm:"type:varchar(256);not null" json:"alertRule"`
	Count       uint   `gorm:"type:integer;not null" json:"count"`
	Message     string `gorm:"type:varchar(256);" json:"message"`
	RecordId    uint   `gorm:"type:integer;" json:"recordId"`
	LicenseId   string `gorm:"type:varchar(256);not null;" json:"licenseId" `
	Method      string `gorm:"type:varchar(128);not null;default:'sms'" json:"method"`
}

// ============================================================
// AlertConfig  告警通知渠道配置表（钉钉/企微/邮件/SMS 等）
// ============================================================
// 字段:
//   - Type (string) — 渠道类型
//   - Title (string) — 渠道显示名
//   - Status (string) — 启用/停用
//   - Config (string) — 渠道具体配置（webhook URL/手机号 等）
//   - CreateUser / UpdateUser (string) — 创建/更新人
// ============================================================
type AlertConfig struct {
	BaseModel
	Type       string `gorm:"type:varchar(64);not null" json:"type"`
	Title      string `gorm:"type:varchar(64);not null" json:"title"`
	Status     string `gorm:"type:varchar(64);not null" json:"status"`
	Config     string `gorm:"type:varchar(256);not null" json:"config"`
	CreateUser string `gorm:"type:varchar(256)" json:"createUser"`
	UpdateUser string `gorm:"type:varchar(256)" json:"updateUser"`
}

// ============================================================
// LoginLog  登录日志表（也常被告警模块查询）
// ============================================================
// 字段:
//   - IP (string) — 登录 IP
//   - Address (string) — IP 物理地址
//   - Agent (string) — 浏览器/客户端 UA
//   - Status (string) — 成功/失败
//   - Message (string) — 备注
// ============================================================
type LoginLog struct {
	BaseModel
	IP      string `json:"ip"`
	Address string `json:"address"`
	Agent   string `json:"agent"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

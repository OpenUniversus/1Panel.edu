// =============================================================================
// 模块: Alert 告警 (agent/app/model/alert.go)
// 文件: alert.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// Alert (struct)
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

type AlertTask struct {
	BaseModel
	Type      string `gorm:"type:varchar(64);not null" json:"type"`
	Quota     string `gorm:"type:varchar(64)" json:"quota"`
	QuotaType string `gorm:"type:varchar(64)" json:"quotaType"`
	Method    string `gorm:"type:varchar(128);not null;default:'sms'" json:"method"`
}

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

// AlertConfig (struct)
type AlertConfig struct {
	BaseModel
	Type       string `gorm:"type:varchar(64);not null" json:"type"`
	Title      string `gorm:"type:varchar(64);not null" json:"title"`
	Status     string `gorm:"type:varchar(64);not null" json:"status"`
	Config     string `gorm:"type:varchar(256);not null" json:"config"`
	CreateUser string `gorm:"type:varchar(256)" json:"createUser"`
	UpdateUser string `gorm:"type:varchar(256)" json:"updateUser"`
}

type LoginLog struct {
	BaseModel
	IP      string `json:"ip"`
	Address string `json:"address"`
	Agent   string `json:"agent"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

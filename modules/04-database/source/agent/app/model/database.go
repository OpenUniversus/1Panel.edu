// =============================================================================
// 模块: Database 数据库 (agent/app/model/database.go)
// 文件: database.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// Database (struct)
type Database struct {
	BaseModel
	AppInstallID uint   `json:"appInstallID"`
	Name         string `json:"name" gorm:"not null;unique"`
	Type         string `json:"type" gorm:"not null"`
	Version      string `json:"version" gorm:"not null"`
	From         string `json:"from" gorm:"not null"`
	Address      string `json:"address" gorm:"not null"`
	Port         uint   `json:"port" gorm:"not null"`
	InitialDB    string `json:"initialDB"`
	Username     string `json:"username"`
	Password     string `json:"password"`

	SSL        bool   `json:"ssl"`
	RootCert   string `json:"rootCert"`
	ClientKey  string `json:"clientKey"`
	ClientCert string `json:"clientCert"`
	SkipVerify bool   `json:"skipVerify"`

	Timeout     uint   `json:"timeout"`
	Description string `json:"description"`
}

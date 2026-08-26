// =============================================================================
// 模块: Backup 备份 (agent/app/model/backup.go)
// 文件: backup.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// BackupAccount (struct)
type BackupAccount struct {
	BaseModel
	Name       string `gorm:"not null;default:''" json:"name"`
	Type       string `gorm:"not null;default:''" json:"type"`
	IsPublic   bool   `json:"isPublic"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"accessKey"`
	Credential string `json:"credential"`
	BackupPath string `json:"backupPath"`
	Vars       string `json:"vars"`

	RememberAuth bool `json:"rememberAuth"`
}

type BackupRecord struct {
	BaseModel
	From              string `json:"from"`
	CronjobID         uint   `json:"cronjobID"`
	SourceAccountIDs  string `json:"sourceAccountIDs"`
	DownloadAccountID uint   `json:"downloadAccountID"`

	Type       string `gorm:"not null;default:''" json:"type"`
	Name       string `gorm:"not null;default:''" json:"name"`
	DetailName string `json:"detailName"`
	FileDir    string `json:"fileDir"`
	FileName   string `json:"fileName"`

	TaskID      string `json:"taskID"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Description string `json:"description"`
	Args        string `gorm:"not null;default:''" json:"args"`
}

// =============================================================================
// 模块: File 文件管理 (agent/app/model/file_share.go)
// 文件: file_share.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// FileShare (struct)
type FileShare struct {
	BaseModel
	Path          string `gorm:"not null;uniqueIndex" json:"path"`
	Token         string `gorm:"not null;uniqueIndex" json:"token"`
	FileName      string `gorm:"not null" json:"fileName"`
	ExpiresUnix   int64  `json:"expiresUnix"`
	PasswordEnc   string `json:"-"`
	PasswordSalt  string `json:"passwordSalt"`
	PasswordHash  string `json:"passwordHash"`
	MaxDownloads  int    `json:"maxDownloads"`
	DownloadCount int    `json:"downloadCount"`
}

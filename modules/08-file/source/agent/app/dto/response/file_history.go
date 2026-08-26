// =============================================================================
// 模块: File 文件管理 (agent/app/dto/response/file_history.go)
// 文件: file_history.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package response

import "time"

// FileHistoryInfo (struct)
type FileHistoryInfo struct {
	ID             uint      `json:"id"`
	FileID         string    `json:"fileId"`
	Path           string    `json:"path"`
	CurrentPath    string    `json:"currentPath"`
	PreviousID     uint      `json:"previousId"`
	SourcePath     string    `json:"sourcePath"`
	TargetPath     string    `json:"targetPath"`
	FileName       string    `json:"fileName"`
	Extension      string    `json:"extension"`
	FileMode       string    `json:"fileMode"`
	Operation      string    `json:"operation"`
	Deleted        bool      `json:"deleted"`
	ContentSize    int64     `json:"contentSize"`
	ContentSHA     string    `json:"contentSHA"`
	StoragePath    string    `json:"storagePath,omitempty"`
	Content        string    `json:"content,omitempty"`
	CurrentContent string    `json:"currentContent,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type FileHistorySettingInfo struct {
	Enable      string `json:"enable"`
	MaxPerPath  int    `json:"maxPerPath"`
	DiskQuotaMB int    `json:"diskQuotaMB"`
}

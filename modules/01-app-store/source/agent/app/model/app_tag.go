// =============================================================================
// 模块: App 应用商店 (agent/app/model/app_tag.go)
// 文件: app_tag.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AppTag (struct)
type AppTag struct {
	BaseModel
	AppId uint `json:"appId" gorm:"not null"`
	TagId uint `json:"tagId" gorm:"not null"`
}

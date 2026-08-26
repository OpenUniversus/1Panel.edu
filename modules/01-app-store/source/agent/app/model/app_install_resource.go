// =============================================================================
// 模块: App 应用商店 (agent/app/model/app_install_resource.go)
// 文件: app_install_resource.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AppInstallResource (struct)
type AppInstallResource struct {
	BaseModel
	AppInstallId uint   `json:"appInstallId" gorm:"not null;"`
	LinkId       uint   `json:"linkId"  gorm:"not null;"`
	ResourceId   uint   `json:"resourceId"`
	Key          string `json:"key" gorm:"not null"`
	From         string `json:"from" gorm:"not null;default:local"`
}

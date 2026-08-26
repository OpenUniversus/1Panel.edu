// =============================================================================
// 模块: App 应用商店 (agent/app/model/app_ignore_upgrade.go)
// 文件: app_ignore_upgrade.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AppIgnoreUpgrade (struct)
type AppIgnoreUpgrade struct {
	BaseModel
	AppID       uint   `json:"appID"`
	AppDetailID uint   `json:"appDetailID"`
	Scope       string `json:"scope"`
}

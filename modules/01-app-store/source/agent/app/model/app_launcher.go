// =============================================================================
// 模块: App 应用商店 (agent/app/model/app_launcher.go)
// 文件: app_launcher.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// AppLauncher (struct)
type AppLauncher struct {
	BaseModel
	Key string `json:"key"`
}

type QuickJump struct {
	BaseModel
	Name      string `json:"name"`
	Alias     string `json:"alias"`
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Recommend int    `json:"recommend"`
	IsShow    bool   `json:"isShow"`
	Router    string `json:"router"`
}

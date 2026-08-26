// =============================================================================
// 模块: Settings 系统设置 (agent/app/model/setting.go)
// 文件: setting.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// Setting (struct)
type Setting struct {
	BaseModel
	Key   string `json:"key" gorm:"not null;"`
	Value string `json:"value"`
	About string `json:"about"`
}

type CommonDescription struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DetailType  string `json:"detailType"`
	IsPinned    bool   `json:"isPinned"`
	Description string `json:"description"`
}

type NodeInfo struct {
	Scope     string `json:"scope"`
	BaseDir   string `json:"baseDir"`
	NodePort  uint   `json:"nodePort"`
	Version   string `json:"version"`
	ServerCrt string `json:"serverCrt"`
	ServerKey string `json:"serverKey"`
}

type LocalConnInfo struct {
	Addr       string `json:"addr"`
	Port       uint   `json:"port"`
	User       string `json:"user"`
	AuthMode   string `json:"authMode"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	PassPhrase string `json:"passPhrase"`
}

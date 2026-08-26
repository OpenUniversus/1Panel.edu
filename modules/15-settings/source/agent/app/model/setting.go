// =============================================================================
// 模块: Settings 系统设置 (agent/app/model/setting.go)
// 文件: setting.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// ============================================================
// Setting  系统设置表（key-value 存储）
// ============================================================
// 字段:
//   - Key (string) — 配置项 key（唯一）
//   - Value (string) — 配置项 value
//   - About (string) — 配置项说明
// ============================================================
type Setting struct {
	BaseModel
	Key   string `json:"key" gorm:"not null;"`
	Value string `json:"value"`
	About string `json:"about"`
}

// ============================================================
// CommonDescription  常用描述（用于面板内说明文字）
// ============================================================
type CommonDescription struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	DetailType  string `json:"detailType"`
	IsPinned    bool   `json:"isPinned"`
	Description string `json:"description"`
}

// ============================================================
// NodeInfo  节点信息（多节点部署场景）
// ============================================================
type NodeInfo struct {
	Scope     string `json:"scope"`
	BaseDir   string `json:"baseDir"`
	NodePort  uint   `json:"nodePort"`
	Version   string `json:"version"`
	ServerCrt string `json:"serverCrt"`
	ServerKey string `json:"serverKey"`
}

// ============================================================
// LocalConnInfo  本机 SSH 连接信息
// ============================================================
type LocalConnInfo struct {
	Addr       string `json:"addr"`
	Port       uint   `json:"port"`
	User       string `json:"user"`
	AuthMode   string `json:"authMode"`
	Password   string `json:"password"`
	PrivateKey string `json:"privateKey"`
	PassPhrase string `json:"passPhrase"`
}

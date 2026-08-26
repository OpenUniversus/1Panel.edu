// =============================================================================
// 模块: Monitor 主机监控 (agent/app/model/ssh.go)
// 文件: ssh.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package model

// RootCert (struct)
type RootCert struct {
	BaseModel
	Name           string `json:"name" gorm:"not null;"`
	EncryptionMode string `json:"encryptionMode"`
	PassPhrase     string `json:"passPhrase"`
	PublicKeyPath  string `json:"publicKeyPath"`
	PrivateKeyPath string `json:"privateKeyPath"`
	Description    string `json:"description"`
}

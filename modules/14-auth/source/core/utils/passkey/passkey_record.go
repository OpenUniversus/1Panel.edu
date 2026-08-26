// =============================================================================
// 模块: Auth 登录认证 (core/utils/passkey/passkey_record.go)
// 文件: passkey_record.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package passkey

import "github.com/go-webauthn/webauthn/webauthn"

// PasskeyCredentialRecord (struct)
type PasskeyCredentialRecord struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	CreatedAt  string              `json:"createdAt"`
	LastUsedAt string              `json:"lastUsedAt"`
	FlagsValue uint8               `json:"flagsValue"`
	Credential webauthn.Credential `json:"credential"`
}

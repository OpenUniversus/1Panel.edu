// =============================================================================
// 模块: Auth 登录认证 (core/utils/passkey/passkey_user.go)
// 文件: passkey_user.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package passkey

import "github.com/go-webauthn/webauthn/webauthn"

// PasskeyUser (struct)
type PasskeyUser struct {
	ID          []byte
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

func (u PasskeyUser) WebAuthnID() []byte {
	return u.ID
}

func (u PasskeyUser) WebAuthnName() string {
	return u.Name
}

func (u PasskeyUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u PasskeyUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// =============================================================================
// 模块: Auth 登录认证 (core/utils/mfa/mfa.go)
// 文件: mfa.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package mfa

import (
	"bytes"
	"encoding/base64"
	"time"

	"github.com/skip2/go-qrcode"
	"github.com/xlzd/gotp"
)

const secretLength = 16

// Otp (struct)
type Otp struct {
	Secret  string `json:"secret"`
	QrImage string `json:"qrImage"`
}

func GetOtp(username, title string, interval int) (otp Otp, err error) {
	secret := gotp.RandomSecret(secretLength)
	otp.Secret = secret
	totp := gotp.NewTOTP(secret, 6, interval, nil)
	uri := totp.ProvisioningUri(username, title)
	subImg, err := qrcode.Encode(uri, qrcode.Medium, 256)
	dist := make([]byte, 3000)
	base64.StdEncoding.Encode(dist, subImg)
	index := bytes.IndexByte(dist, 0)
	baseImage := dist[0:index]
	otp.QrImage = "data:image/png;base64," + string(baseImage)
	return
}

func ValidCode(interval int, code, secret string) bool {
	totp := gotp.NewTOTP(secret, 6, interval, nil)
	now := time.Now().Unix()
	prevTime := now - int64(interval)
	return totp.Verify(code, now) || totp.Verify(code, prevTime)
}

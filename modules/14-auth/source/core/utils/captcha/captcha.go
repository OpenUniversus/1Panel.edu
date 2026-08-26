// =============================================================================
// 模块: Auth 登录认证 (core/utils/captcha/captcha.go)
// 文件: captcha.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package captcha

import (
	"image/color"
	"strings"

	"github.com/1Panel-dev/1Panel/core/app/dto"
	"github.com/1panel-dev/base64Captcha"
)

var store = base64Captcha.DefaultMemStore

const (
	captchaWidth  = 160
	captchaHeight = 48
	captchaNoise  = 0
)

var captchaFonts = []string{
	base64Captcha.FontRitaSmith,
}

func VerifyCode(codeID string, code string) string {
	vv := store.Get(codeID, true)
	vv = strings.TrimSpace(vv)
	code = strings.TrimSpace(code)
	if codeID == "" || code == "" {
		return "ErrCaptchaCode"
	}
	if strings.EqualFold(vv, code) {
		return ""
	}
	return "ErrCaptchaCode"
}

func CreateCaptcha() (*dto.CaptchaResponse, error) {
	driver := base64Captcha.NewDriverMath(
		captchaHeight,
		captchaWidth,
		captchaNoise,
		0,
		&color.RGBA{R: 246, G: 248, B: 251, A: 255},
		nil,
		captchaFonts,
	)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := c.Generate()
	if err != nil {
		return nil, err
	}
	return &dto.CaptchaResponse{
		CaptchaID: id,
		ImagePath: b64s,
	}, nil
}

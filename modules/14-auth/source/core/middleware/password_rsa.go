// =============================================================================
// 模块: Auth 登录认证 (core/middleware/password_rsa.go)
// 文件: password_rsa.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package middleware

import (
	"encoding/base64"
	"github.com/1Panel-dev/1Panel/core/app/repo"
	"github.com/gin-gonic/gin"
)

func SetPasswordPublicKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookieKey, _ := c.Cookie("panel_public_key")
		settingRepo := repo.NewISettingRepo()
		key, _ := settingRepo.GetValueByKey("PASSWORD_PUBLIC_KEY")
		base64Key := base64.StdEncoding.EncodeToString([]byte(key))
		if base64Key == cookieKey {
			c.Next()
			return
		}
		c.SetCookie("panel_public_key", base64Key, 7*24*60*60, "/", "", false, false)
		c.Next()
	}
}

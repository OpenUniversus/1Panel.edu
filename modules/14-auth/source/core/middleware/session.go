// =============================================================================
// 模块: Auth 登录认证 (core/middleware/session.go)
// 文件: session.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package middleware

import (
	"strconv"

	"github.com/1Panel-dev/1Panel/core/app/api/v2/helper"
	baseRepo "github.com/1Panel-dev/1Panel/core/app/repo"
	"github.com/1Panel-dev/1Panel/core/buserr"
	"github.com/1Panel-dev/1Panel/core/constant"
	"github.com/1Panel-dev/1Panel/core/global"
	psessionUtils "github.com/1Panel-dev/1Panel/core/init/session/psession"
	"github.com/gin-gonic/gin"
)

func SessionAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiReq := c.GetBool("API_AUTH")
		if isAnonymousAuthPath(c.Request.URL.Path) || apiReq || c.GetBool("LOCAL_REQUEST") {
			c.Next()
			return
		}

		psession, err := global.SESSION.Get(c)
		if err != nil {
			errItem := err.Error()
			if errItem == "ErrSessionDataFormat" || errItem == "ErrSessionDataNotFound" {
				helper.BadAuth(c, "ErrNotLogin", buserr.New(errItem))
				return
			}
			helper.BadAuth(c, "ErrNotLogin", err)
			return
		}
		if len(psession.Name) == 0 || len(psession.ID) == 0 {
			helper.BadAuth(c, "ErrNotLogin", err)
			return
		}
		c.Set(psessionUtils.GinContextSessionUserKey, psession)
		sessionTimeout, err := baseRepo.NewISettingRepo().GetValueByKey("SessionTimeout")
		if err != nil {
			global.LOG.Errorf("get session timeout failed, err: %v", err)
			helper.InternalServer(c, err)
			c.Abort()
			return
		}
		lifeTime, _ := strconv.Atoi(sessionTimeout)

		if _, err := global.SESSION.RefreshIfNeeded(c, psession, global.CONF.Conn.SSL == constant.StatusEnable, lifeTime); err != nil {
			errItem := err.Error()
			if errItem == "ErrSessionDataFormat" || errItem == "ErrSessionDataNotFound" {
				helper.BadAuth(c, "ErrNotLogin", buserr.New(errItem))
				return
			}
			global.LOG.Warnf("refresh session failed, path=%s, err=%v", c.Request.URL.Path, err)
			helper.BadAuth(c, "ErrNotLogin", err)
			return
		}
		c.Next()
	}
}

func isAnonymousAuthPath(path string) bool {
	switch path {
	case "/api/v2/core/auth/captcha",
		"/api/v2/core/auth/passkey/begin",
		"/api/v2/core/auth/passkey/finish",
		"/api/v2/core/auth/mfalogin",
		"/api/v2/core/auth/login",
		"/api/v2/core/auth/logout",
		"/api/v2/core/auth/setting",
		"/api/v2/core/auth/welcome":
		return true
	default:
		return false
	}
}

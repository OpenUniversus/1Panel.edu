// =============================================================================
// 模块: SSL 证书管理 (agent/middleware/certificate.go)
// 文件: certificate.go — 主代码
// 说明: 本文件为 1Panel 上游源码拷贝 + 中文注解, 源码 commit: dev-v2
//       注解只增加 // 注释, 不改变 Go 语义, 文件仍可直接用 go build 编译
// =============================================================================

package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/agent/app/api/v2/helper"
	"github.com/1Panel-dev/1Panel/agent/global"
	"github.com/1Panel-dev/1Panel/agent/utils/cmd"
	"github.com/1Panel-dev/1Panel/agent/utils/xpack"
	"github.com/gin-gonic/gin"
)

func Certificate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if global.IsMaster {
			c.Next()
			return
		}
		if !xpack.MultiNodeProvider.ValidateCertificate(c) {
			CloseDirectly(c)
			return
		}
		conn := c.Request.Header.Get("Connection")
		if conn == "Upgrade" {
			c.Next()
			return
		}
		masterProxyID := c.Request.Header.Get("Proxy-Id")
		proxyID, err := cmd.NewCommandMgr(cmd.WithTimeout(20*time.Second)).RunWithStdout("cat", "/etc/1panel/.nodeProxyID")
		if err == nil && len(proxyID) != 0 && strings.TrimSpace(proxyID) != strings.TrimSpace(masterProxyID) {
			helper.InternalServer(c, fmt.Errorf("err proxy id"))
			return
		}
		c.Next()
	}
}

func CloseDirectly(c *gin.Context) {
	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}
	_ = conn.(*net.TCPConn).SetLinger(0)
	conn.Close()
}

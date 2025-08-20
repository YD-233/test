package middleware

import (
	"BackendTemplate/pkg/config"
	"encoding/base64"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware Basic认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")

		if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
			// 返回WWW-Authenticate头，触发浏览器的弹框
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		encodedCreds := authHeader[len("Basic "):]
		creds, err := base64.StdEncoding.DecodeString(encodedCreds)
		if err != nil {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		credParts := strings.SplitN(string(creds), ":", 2)
		if len(credParts) != 2 {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}
		user, pass := credParts[0], credParts[1]

		// 使用配置文件验证用户名和密码
		if user != config.Username || pass != config.Password {
			c.Header("WWW-Authenticate", `Basic realm="Restricted"`)
			c.AbortWithStatus(401)
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
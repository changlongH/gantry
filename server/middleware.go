package server

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 白名单路径（不需要鉴权的路径）
var whitelistPaths = map[string]bool{
	"/health":       true,
	"/alarm/notify": true,
}

// Gin 中间件：负责统一的 X-Secret 鉴权和请求体大小限制
func (s *Server) authAndLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 鉴权校验
		if whitelistPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		secret := s.cfgMgr.Get().Server.Secret
		if secret == "" {
			log.Printf("Server secret not configured, rejecting request")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Server secret not configured"})
			return
		}

		reqSecret := c.GetHeader("X-Secret")
		if reqSecret != secret {
			log.Printf("Invalid server path:%s secret=%s, rejecting request", c.Request.URL.Path, reqSecret)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		// 2. 限制请求体大小为 10MB（取代原先的 http.MaxBytesReader）
		// Gin 的 c.ShouldBindJSON 内部会自动遵循 c.Request.Body 的大小限制
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

		// 校验通过，继续执行后续的 Handler
		c.Next()
	}
}

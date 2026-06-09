package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Gin 中间件：负责统一的 X-Secret 鉴权和请求体大小限制
func (s *Server) authAndLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 鉴权校验
		secret := s.cfgMgr.Get().Server.Secret
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Server secret not configured"})
			return
		}

		reqSecret := c.GetHeader("X-Secret")
		if reqSecret != secret {
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

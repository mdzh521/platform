package middleware

import (
	"net/http"
	"strings"

	"backend-center/internal/infra/web/response"

	"github.com/gin-gonic/gin"
)

func StaticBearer(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, "missing internal bearer token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if token == "" || token != strings.TrimSpace(secret) {
			response.Error(c, http.StatusUnauthorized, "invalid internal bearer token")
			c.Abort()
			return
		}
		c.Next()
	}
}

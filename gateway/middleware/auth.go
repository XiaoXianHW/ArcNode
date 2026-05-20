package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func BearerAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") && strings.TrimPrefix(header, "Bearer ") == token {
			c.Next()
			return
		}
		// Browser <a href download> can't attach an Authorization header,
		// so allow ?token=... as a fallback on GET requests.
		if c.Request.Method == http.MethodGet && c.Query("token") == token {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

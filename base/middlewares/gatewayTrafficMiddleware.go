package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func GatewayTrafficMiddleware(gatewaySecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientSecret := c.Request.Header.Get("X-Gateway-Secret")
		if clientSecret != gatewaySecret {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid gateway secret"})
			return
		}
		c.Next()
	}
}

package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing request ID first
		reqID := c.GetHeader("X-Request-ID")

		// Only generate if not provided
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Set it in both context and response header
		c.Set("RequestID", reqID)
		c.Writer.Header().Set("X-Request-ID", reqID)

		c.Next()
	}
}

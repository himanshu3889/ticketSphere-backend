package middlewares

import (
	"strconv"
	"time"

	baseMetrics "github.com/himanshu3889/ticketSphere-backend/base/metric"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Capture these BEFORE c.Next() in case they get modified
		method := c.Request.Method
		path := c.FullPath()
		if path == "" {
			path = "unknown"
		}

		c.Next()

		// Now capture status after request is processed
		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		// Record metrics asynchronously
		baseMetrics.HttpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
		baseMetrics.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
	}
}

package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func LatencyLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Log it
		status := c.Writer.Status()
		fields := logrus.Fields{
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"status":  status,
			"latency": duration.Milliseconds(),
		}

		// Capture API errors if any
		if len(c.Errors) > 0 {
			fields["error"] = c.Errors.Last().Error()
		}

		// Decide log level
		// switch {
		// case status >= 500:
		// 	logrus.WithFields(fields).Error("API Request")
		// case status >= 400:
		// 	logrus.WithFields(fields).Warn("API Request")
		// default:
		// 	logrus.WithFields(fields).Info("API Request")
		// }
	}
}

package baseApi

import (
	"github.com/gin-gonic/gin"
)

func registerCommonRoutes(rg *gin.RouterGroup) {
	rg.GET("/health-check", healthCheck)
	rg.GET("/ready", ready)
}

func healthCheck(c *gin.Context) {
	c.String(200, "Service OK")
}

func ready(c *gin.Context) {
	c.String(200, "Ready")
}

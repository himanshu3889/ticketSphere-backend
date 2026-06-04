package baseApi

import "github.com/gin-gonic/gin"

func RegisterBaseRoutes(rg *gin.RouterGroup) {
	// core := rg.Group("")
	// core routes
	registerCommonRoutes(rg)
}

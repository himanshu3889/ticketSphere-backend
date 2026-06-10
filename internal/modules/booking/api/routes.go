package bookingApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(routeEngine *gin.Engine, handler *ApiHandler) {
	// Base routes
	baseGroup := routeEngine.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)

	// Api routes
	handler.RegisterAuthRoutes(baseGroup) // Auth
}

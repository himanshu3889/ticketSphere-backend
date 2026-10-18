package bookingApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(baseApiGroup *gin.RouterGroup, apiHandler *ApiHandler) {
	// Base routes
	baseApi.RegisterBaseRoutes(baseApiGroup)

	// Auth routes
	apiHandler.RegisterAuthRoutes(baseApiGroup)

	// Booking routes
	apiHandler.RegisterBookingRoutes(baseApiGroup)
}

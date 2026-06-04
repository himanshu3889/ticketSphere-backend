package bookingApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(r *gin.Engine, h *ApiHandler) {
	// Base routes
	baseGroup := r.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)

	// Api routes
	// api := r.Group("/api")

}

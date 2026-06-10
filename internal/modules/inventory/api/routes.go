package inventoryApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(r *gin.Engine, h *ApiHandler) {
	baseGroup := r.Group("")
	baseApi.RegisterBaseRoutes(baseGroup)

}

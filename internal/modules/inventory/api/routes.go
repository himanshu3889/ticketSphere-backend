package inventoryApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(baseApiGroup *gin.RouterGroup, apiHandler *ApiHandler) {
	baseApi.RegisterBaseRoutes(baseApiGroup)

	// Venue routes
	apiHandler.RegisterVenueRoutes(baseApiGroup)

	// Event routes
	apiHandler.RegisterEventsRoutes(baseApiGroup)

	// Section routes
	apiHandler.RegisterSectionRoutes(baseApiGroup)

	// Seat routes
	apiHandler.RegisterSeatsRoutes(baseApiGroup)
}

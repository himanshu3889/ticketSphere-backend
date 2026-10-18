package inventoryApi

import (
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterSeatsRoutes(rg *gin.RouterGroup) {
	// private seat routes (gateway auth)
	seatPrivateRg := rg.Group("/seats", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))

	seatPrivateRg.GET("/:seat_id", apiHandler.GetSeat)

}

// GetSeat returns one seat by id.
func (apiHandler *ApiHandler) GetSeat(ctx *gin.Context) {
	seatIdStr := ctx.Param("seat_id")
	seatId, err := utils.ValidSnowflakeID(seatIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid seat id")
		return
	}
	seat, appErr := apiHandler.store.GetSeatByID(ctx, seatId)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}
	utils.RespondWithSuccess(ctx, 200, seat)
}

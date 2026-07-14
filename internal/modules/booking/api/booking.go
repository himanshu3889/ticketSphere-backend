package bookingApi

import (
	"net/http"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"

	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterBookingRoutes(rg *gin.RouterGroup) {
	// Booking router group
	bookingRg := rg.Group("", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))

	// Event seat booking
	bookingRg.POST("/events/:event_id", apiHandler.eventSeatsBooking)

}

// Create a payload struct that accepts strings
type BookingRequestPayload struct {
	SeatIDs []string `json:"seats" binding:"required,min=1,max=10,dive,required"`
}

// Event seat booking
func (apiHandler *ApiHandler) eventSeatsBooking(ctx *gin.Context) {
	userID, _, isOk := utils.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid request")
		return
	}

	// Validate event id
	eventIdStr := ctx.Param("event_id")
	eventId, err := utils.ValidSnowflakeID(eventIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid event id")
		return
	}

	var BookingRequestPayload BookingRequestPayload

	// Bind the json
	if err := ctx.ShouldBindJSON(&BookingRequestPayload); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Validate all the booking seats ids are snowflake ids and build array of snowflake ids
	seatIds := make([]snowflake.ID, len(BookingRequestPayload.SeatIDs))
	for seatIdx, bookingId := range BookingRequestPayload.SeatIDs {
		seatId, err := utils.ValidSnowflakeID(bookingId)
		if err != nil {
			utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid seat id")
			return
		}
		seatIds[seatIdx] = seatId
	}

	bookingRequest := &models.BookingRequest{
		EventID: eventId,
		SeatIDs: seatIds,
		UserID:  userID,
	}

	// Headers idempotency key
	rawIdempotencyKeyStr := ctx.GetHeader("Idempotency-Key")
	if rawIdempotencyKeyStr == "" {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	// Booking seats pessimistically
	booking, appErr := apiHandler.cacheStore.BookSeats(ctx, rawIdempotencyKeyStr, bookingRequest)

	// booking, appErr := apiHandler.store.BookSeatsPessimistic(ctx, bookingRequest)
	// booking, appErr := apiHandler.store.BookSeatsOptimistic(ctx, bookingRequest)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"message": "Booking Pending",
		"booking": booking,
	}
	utils.RespondWithSuccess(ctx, http.StatusOK, apiData)
}

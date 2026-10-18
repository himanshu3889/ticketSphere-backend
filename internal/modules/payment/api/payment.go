package paymentApi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterPaymentRoutes(rg *gin.RouterGroup) {
	paymentRg := rg.Group("", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))

	paymentRg.POST("/initiate", apiHandler.initiatePayment)
}

// InitiatePaymentRequest is the raw initiate-payment request body.
type InitiatePaymentRequest struct {
	BookingID string `json:"booking_id" binding:"required"`
}

// initiate a payment for a pending booking
func (apiHandler *ApiHandler) initiatePayment(ctx *gin.Context) {
	userID, _, isOk := utils.GetContextUserIDEmail(ctx)
	if !isOk {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid request")
		return
	}

	var req InitiatePaymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	bookingId, err := utils.ValidSnowflakeID(req.BookingID)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid booking id")
		return
	}

	rawIdempotencyKeyStr := ctx.GetHeader("Idempotency-Key")
	if rawIdempotencyKeyStr == "" {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	paymentResult, appErr := apiHandler.cacheStore.InitiateBookingPayment(ctx, rawIdempotencyKeyStr, userID, bookingId)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"message":    paymentResult.Message,
		"payment_id": paymentResult.ID,
		"data":       paymentResult.Data,
	}
	utils.RespondWithSuccess(ctx, paymentResult.Status, apiData)
}

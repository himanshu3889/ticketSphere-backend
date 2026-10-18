package paymentApi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/himanshu3889/ticketSphere-backend/base/structures"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterPaymentGatewayWebhookRoutes(baseRg *gin.RouterGroup) {
	webhookRg := baseRg.Group("/webhook/payment-gateway")
	webhookRg.POST("", apiHandler.paymentGatewayWebhook)
}

// verify the gateway signature, enqueue the payload, then 200
func (apiHandler *ApiHandler) paymentGatewayWebhook(ctx *gin.Context) {
	// need the raw bytes for signature verification
	body, err := ctx.GetRawData()
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Failed to read body")
		return
	}

	signature := ctx.GetHeader("X-Webhook-Signature")
	if signature == "" {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Missing signature")
		return
	}

	if !verifyWebhookSignature(body, signature, apiHandler.paymentGatewayConfigs.WebhookSecret) {
		utils.RespondWithError(ctx, http.StatusUnauthorized, "Invalid signature")
		return
	}

	var payload structures.PaymentGatewayWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// queue first so a 200 means the event made it to the broker
	err = apiHandler.cacheStore.EventBus.PublishPaymentWebhookEvent(ctx, &payload)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, "Failed to process webhook")
		return
	}

	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{"message": "Payment webhook received successfully"})
}

// verifyWebhookSignature compares HMAC-SHA256 hex over the raw body, constant time
func verifyWebhookSignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

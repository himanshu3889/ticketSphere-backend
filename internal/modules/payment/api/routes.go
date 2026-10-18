package paymentApi

import (
	"github.com/gin-gonic/gin"
	baseApi "github.com/himanshu3889/ticketSphere-backend/base/api"
)

func SetupRoutes(baseApiGroup *gin.RouterGroup, apiHandler *ApiHandler) {
	baseApi.RegisterBaseRoutes(baseApiGroup)
	apiHandler.RegisterPaymentRoutes(baseApiGroup)
	apiHandler.RegisterPaymentGatewayWebhookRoutes(baseApiGroup)
}

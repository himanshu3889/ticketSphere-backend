package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

// Gateway authentication middleware
func GatewayAuthMiddleware(gatewaySecret string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Verify this request came from our gateway
		headerGatewaySecret := ctx.GetHeader("X-Gateway-Secret")
		if headerGatewaySecret != gatewaySecret {
			utils.RespondWithError(ctx, http.StatusForbidden, "Direct access not allowed")
			ctx.Abort()
			return
		}

		// Read the headers that gateway jwtAuth middleware injected
		userIDStr := ctx.GetHeader("X-User-ID")
		email := ctx.GetHeader("X-User-Email")

		if userIDStr == "" {
			utils.RespondWithError(ctx, http.StatusUnauthorized, "Missing user custom headers")
			ctx.Abort()
			return
		}

		// Convert string ID to snowflake
		userID, err := utils.ValidSnowflakeID(userIDStr)
		if err != nil {
			utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid user id")
			ctx.Abort()
			return
		}

		// Set the contexts from jwt
		ctx.Set(utils.UserEmailKey, email)
		ctx.Set(utils.UserIDKey, userID)

		ctx.Next()
	}
}

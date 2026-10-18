package inventoryApi

import (
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterSectionRoutes(rg *gin.RouterGroup) {
	// private section routes (gateway auth)
	sectionPrivateRg := rg.Group("/sections", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))

	sectionPrivateRg.GET("/:section_id", apiHandler.GetSectionByID)

	sectionPrivateRg.GET("/:section_id/seats/avail", apiHandler.GetSectionAvailableSeats)

	sectionPrivateRg.GET("/:section_id/seats/avail-held", apiHandler.GetSectionAvailableOrHeldSeats)

}

// GetSectionByID returns one section by id.
func (apiHandler *ApiHandler) GetSectionByID(ctx *gin.Context) {
	sectionIdStr := ctx.Param("section_id")
	sectionId, err := utils.ValidSnowflakeID(sectionIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid section id")
		return
	}

	section, appErr := apiHandler.store.GetSectionByID(ctx, sectionId)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"section": section,
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

func (apiHandler *ApiHandler) GetSectionAvailableSeats(ctx *gin.Context) {
	sectionIdStr := ctx.Param("section_id")
	sectionId, err := utils.ValidSnowflakeID(sectionIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid section id")
		return
	}

	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}

	seats, appErr := apiHandler.store.GetSectionAvailableSeats(ctx, sectionId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"section_id": sectionId,
		"seats":      seats,
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

func (apiHandler *ApiHandler) GetSectionAvailableOrHeldSeats(ctx *gin.Context) {
	sectionIdStr := ctx.Param("section_id")
	sectionId, err := utils.ValidSnowflakeID(sectionIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid section id")
		return
	}

	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}

	seats, appErr := apiHandler.store.GetSectionAvailableOrHeldSeats(ctx, sectionId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"section_id": sectionId,
		"seats":      seats,
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

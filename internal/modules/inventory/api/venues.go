package inventoryApi

import (
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterVenueRoutes(rg *gin.RouterGroup) {
	// public venue routes
	venueRg := rg.Group("/public/venues")
	venueRg.GET("", apiHandler.GetAllVenues)
	venueRg.GET("/:venue_id", apiHandler.GetVenue)
	venueRg.GET("/:venue_id/events/upcoming-or-live", apiHandler.GetVenueUpcomingOrLiveEvents)

}

// GetAllVenues lists venues, newest first.
func (apiHandler *ApiHandler) GetAllVenues(ctx *gin.Context) {
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}
	venues, appErr := apiHandler.store.GetVenues(ctx, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"venues": venues,
		"meta":   nil,
	}
	if len(venues) > 0 {
		apiData["meta"] = utils.BuildCursorMeta(limit, cursor, &venues[len(venues)-1].ID)
	}

	utils.RespondWithSuccess(ctx, 200, apiData)
}

// GetVenue returns one venue by id.
func (apiHandler *ApiHandler) GetVenue(ctx *gin.Context) {
	venueIdStr := ctx.Param("venue_id")
	venueId, err := utils.ValidSnowflakeID(venueIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid venue id")
		return
	}
	venue, appErr := apiHandler.store.GetVenue(ctx, venueId)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"venue": venue,
	}

	utils.RespondWithSuccess(ctx, 200, apiData)
}

// GetVenueUpcomingOrLiveEvents lists a venue's upcoming or live events.
func (apiHandler *ApiHandler) GetVenueUpcomingOrLiveEvents(ctx *gin.Context) {
	venueIdStr := ctx.Param("venue_id")
	venueId, err := utils.ValidSnowflakeID(venueIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid venue id")
		return
	}
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}
	events, appErr := apiHandler.store.GetVenueUpcomingOrLiveEvents(ctx, venueId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}
	apiData := gin.H{
		"events": events,
		"meta":   nil,
	}
	if len(events) > 0 {
		apiData["meta"] = utils.BuildCursorMeta(limit, cursor, &events[len(events)-1].ID)
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

package inventoryApi

import (
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterEventsRoutes(rg *gin.RouterGroup) {
	// public event routes
	eventPublicRg := rg.Group("/public/events")
	eventPublicRg.GET("", apiHandler.GetAllEvents)
	eventPublicRg.GET("/:event_id", apiHandler.GetEvent)
	eventPublicRg.GET("/upcoming-or-live", apiHandler.GetUpcomingOrLiveEvents)
	eventPublicRg.GET("/:event_id/sections", apiHandler.GetEventSections)

	// private event routes (gateway auth)
	eventPrivateRg := rg.Group("/events", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))
	eventPrivateRg.GET("/:event_id/seats/avail-held", apiHandler.GetEventAvailableOrHeldSeats)
	eventPrivateRg.GET("/:event_id/seats/avail", apiHandler.GetEventAvailableSeats)

}

// GetAllEvents lists events, newest first.
func (apiHandler *ApiHandler) GetAllEvents(ctx *gin.Context) {
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}
	events, appErr := apiHandler.store.GetAllEvents(ctx, limit, cursor)
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

// GetEvent returns one event by id.
func (apiHandler *ApiHandler) GetEvent(ctx *gin.Context) {
	eventIdStr := ctx.Param("event_id")
	eventId, err := utils.ValidSnowflakeID(eventIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid event id")
		return
	}
	event, appErr := apiHandler.store.GetEvent(ctx, eventId)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}
	apiData := gin.H{
		"event": event,
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

// GetUpcomingOrLiveEvents lists events that haven't ended or been cancelled.
func (apiHandler *ApiHandler) GetUpcomingOrLiveEvents(ctx *gin.Context) {
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}
	events, appErr := apiHandler.store.GetUpcomingOrLiveEvents(ctx, limit, cursor)
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

// GetEventSections lists an event's sections.
func (apiHandler *ApiHandler) GetEventSections(ctx *gin.Context) {
	eventIdStr := ctx.Param("event_id")
	eventId, err := utils.ValidSnowflakeID(eventIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid event id")
		return
	}
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}
	sections, appErr := apiHandler.store.GetEventAllSections(ctx, eventId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}
	apiData := gin.H{
		"sections": sections,
		"meta":     nil,
	}
	if len(sections) > 0 {
		apiData["meta"] = utils.BuildCursorMeta(limit, cursor, &sections[len(sections)-1].ID)
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

// GetEventAvailableOrHeldSeats lists an event's available or held seats.
func (apiHandler *ApiHandler) GetEventAvailableOrHeldSeats(ctx *gin.Context) {
	eventIdStr := ctx.Param("event_id")
	eventId, err := utils.ValidSnowflakeID(eventIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid event id")
		return
	}
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}

	sections, appErr := apiHandler.store.GetEventAvailableOrHeldSeats(ctx, eventId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"event_id": eventId,
		"sections": sections,
		"meta":     nil,
	}
	if len(sections) > 0 {
		apiData["meta"] = utils.BuildCursorMeta(limit, cursor, &sections[len(sections)-1].ID)
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

// GetEventAvailableSeats lists an event's available seats.
func (apiHandler *ApiHandler) GetEventAvailableSeats(ctx *gin.Context) {
	eventIdStr := ctx.Param("event_id")
	eventId, err := utils.ValidSnowflakeID(eventIdStr)
	if err != nil {
		utils.RespondWithErrorDetail(ctx, 400, "Invalid event id")
		return
	}
	limit, cursor, ok := utils.GetCursorPagination(ctx)
	if !ok {
		return
	}

	sections, appErr := apiHandler.store.GetEventAvailableSeats(ctx, eventId, limit, cursor)
	if appErr != nil {
		utils.RespondWithError(ctx, int(appErr.Code), appErr.Message)
		return
	}

	apiData := gin.H{
		"event_id": eventId,
		"sections": sections,
		"meta":     nil,
	}
	if len(sections) > 0 {
		apiData["meta"] = utils.BuildCursorMeta(limit, cursor, &sections[len(sections)-1].ID)
	}
	utils.RespondWithSuccess(ctx, 200, apiData)
}

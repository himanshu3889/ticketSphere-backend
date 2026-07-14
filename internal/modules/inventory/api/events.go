package inventoryApi

import (
	"github.com/gin-gonic/gin"
	"github.com/himanshu3889/ticketSphere-backend/base/middlewares"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
)

func (apiHandler *ApiHandler) RegisterEventsRoutes(rg *gin.RouterGroup) {
	// ---- Event public router group ----
	eventPublicRg := rg.Group("/public/events")
	// Get all events
	eventPublicRg.GET("", apiHandler.GetAllEvents)
	// Get event
	eventPublicRg.GET("/:event_id", apiHandler.GetEvent)
	// Get upcoming or live events
	eventPublicRg.GET("/upcoming-or-live", apiHandler.GetUpcomingOrLiveEvents)
	// Get event sections
	eventPublicRg.GET("/:event_id/sections", apiHandler.GetEventSections)

	// ---- Event Private routes ----
	eventPrivateRg := rg.Group("/events", middlewares.GatewayAuthMiddleware(apiHandler.authConfigs.GatewaySecret))
	// Get event available or held seats
	eventPrivateRg.GET("/:event_id/seats/avail-held", apiHandler.GetEventAvailableOrHeldSeats)
	// Get event available seats
	eventPrivateRg.GET("/:event_id/seats/avail", apiHandler.GetEventAvailableSeats)

}

// Get all events
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

// Get event
func (apiHandler *ApiHandler) GetEvent(ctx *gin.Context) {
	// Validate the event_id
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

// Get upcoming or live events
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

// Get event sections
func (apiHandler *ApiHandler) GetEventSections(ctx *gin.Context) {
	// Validate the event_id
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

// Get event available or held seats
func (apiHandler *ApiHandler) GetEventAvailableOrHeldSeats(ctx *gin.Context) {
	// Validate the event_id
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

// Get event available seats
func (apiHandler *ApiHandler) GetEventAvailableSeats(ctx *gin.Context) {
	// Validate the event_id
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

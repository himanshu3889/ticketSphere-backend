package utils

import (
	"strconv"

	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

// GetPagination extracts limit and cursor, handles validation,
// and sends error responses automatically.
func GetCursorPagination(ctx *gin.Context) (limit int, cursor snowflake.ID, ok bool) {
	limitStr := ctx.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		RespondWithError(ctx, 400, "Invalid limit")
		return 0, 0, false
	}

	cursorStr := ctx.Param("cursor")
	if cursorStr != "" {
		cursor, err = ValidSnowflakeID(cursorStr)
		if err != nil {
			RespondWithError(ctx, 400, "Invalid cursor")
			return 0, 0, false
		}
	}
	return limit, cursor, true
}

// BuildCursorMeta builds cursor-based metadata for API responses.
func BuildCursorMeta(limit int, cursor snowflake.ID, lastItemId *snowflake.ID) gin.H {
	var nextCursor snowflake.ID
	if lastItemId != nil && *lastItemId > 0 && *lastItemId != cursor {
		nextCursor = *lastItemId
	}

	meta := gin.H{
		"limit":       limit,
		"has_next":    false,
		"next_cursor": nil,
	}

	if nextCursor != 0 {
		meta["has_next"] = true
		meta["next_cursor"] = nextCursor.String()
	}

	return meta
}

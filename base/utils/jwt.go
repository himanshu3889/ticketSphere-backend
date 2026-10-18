package utils

import (
	"github.com/bwmarrin/snowflake"
	"github.com/gin-gonic/gin"
)

type jwtContextKey string

const UserIDKey jwtContextKey = "userID"
const UserEmailKey jwtContextKey = "email"

func GetContextValue[T any](ctx *gin.Context, key jwtContextKey) (T, bool) {
	if val, exists := ctx.Get(key); exists {
		if typed, ok := val.(T); ok {
			return typed, true
		}
	}
	var zero T
	return zero, false
}

func GetContextUserIDEmail(ctx *gin.Context) (userID snowflake.ID, email string, ok bool) {
	userID, ok1 := GetContextValue[snowflake.ID](ctx, UserIDKey)
	email, ok2 := GetContextValue[string](ctx, UserEmailKey)

	if ok1 && ok2 {
		return userID, email, true
	}
	return 0, "", false
}

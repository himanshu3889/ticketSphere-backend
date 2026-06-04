package utils

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

func RespondWithError(c *gin.Context, code int, message string) {
	c.Error(errors.New(message))
	c.JSON(code, gin.H{"error": message})
	c.Abort()
}

func RespondWithErrorDetail(c *gin.Context, code int, err interface{}) {
	var e error

	switch v := err.(type) {
	case error:
		e = v
	case string:
		e = errors.New(v)
	default:
		e = fmt.Errorf("%v", v)
	}

	c.Error(e)
	c.JSON(code, gin.H{"error": err})
	c.Abort()
}

func RespondWithSuccess(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}

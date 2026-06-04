package appError

import "net/http"

// strictly enforce valid HTTP statuses
type ErrorCode int

const (
	StatusBadRequest          ErrorCode = http.StatusBadRequest          // 400
	StatusUnauthorized        ErrorCode = http.StatusUnauthorized        // 401
	StatusForbidden           ErrorCode = http.StatusForbidden           // 403
	StatusNotFound            ErrorCode = http.StatusNotFound            // 404
	StatusConflict            ErrorCode = http.StatusConflict            // 409
	StatusGone                ErrorCode = http.StatusGone                // 410
	StatusInternalServerError ErrorCode = http.StatusInternalServerError // 500
)

type Error struct {
	Message string
	Code    ErrorCode
}

// enforces a 400 status
func NewBadRequest(message string) *Error {
	return &Error{
		Code:    StatusBadRequest,
		Message: message,
	}
}

// enforces a 404 status
func NewNotFound(message string) *Error {
	return &Error{
		Code:    StatusNotFound,
		Message: message,
	}
}

// enforces a 500
func NewInternal(message string) *Error {
	return &Error{
		Code:    StatusInternalServerError,
		Message: message,
	}
}

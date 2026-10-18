package appError

import (
	"fmt"
	"net/http"
	"runtime"
)

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
	Message         string
	Code            ErrorCode
	InternalMessage *string
	err             error
	stack           []uintptr
}

// Error implements the standard error interface.
func (e *Error) Error() string {
	return e.Message
}

// Unwrap returns the wrapped cause, enabling errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.err
}

// StackTrace returns the captured stack as "function\tfile:line" lines.
// Returns nil when no stack was captured.
func (e *Error) StackTrace() []string {
	if len(e.stack) == 0 {
		return nil
	}
	frames := runtime.CallersFrames(e.stack)
	var lines []string
	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			break
		}
		lines = append(lines, fmt.Sprintf("%s\t%s:%d", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	return lines
}

// ErrorOption configures an Error.
type ErrorOption func(*Error)

// WithCause wraps an underlying error and captures a stack trace.
func WithCause(err error) ErrorOption {
	return func(e *Error) {
		e.err = err
		if e.stack == nil {
			e.stack = callers(5)
		}
	}
}

// WithInternal sets the internal diagnostic message used in telemetry.
func WithInternal(msg string) ErrorOption {
	return func(e *Error) {
		e.InternalMessage = &msg
	}
}

// WithStack captures the current stack trace for this error.
func WithStack() ErrorOption {
	return func(e *Error) {
		e.stack = callers(5)
	}
}

func callers(skip int) []uintptr {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(skip, pcs[:])
	return pcs[:n]
}

func newError(code ErrorCode, message string, opts ...ErrorOption) *Error {
	e := &Error{
		Code:    code,
		Message: message,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// NewBadRequest enforces a 400 status.
func NewBadRequest(message string, opts ...ErrorOption) *Error {
	return newError(StatusBadRequest, message, opts...)
}

// NewNotFound enforces a 404 status.
func NewNotFound(message string, opts ...ErrorOption) *Error {
	return newError(StatusNotFound, message, opts...)
}

// NewInternal enforces a 500 status.
func NewInternal(message string, opts ...ErrorOption) *Error {
	return newError(StatusInternalServerError, message, opts...)
}

// NewConflict enforces a 409 status.
func NewConflict(message string, opts ...ErrorOption) *Error {
	return newError(StatusConflict, message, opts...)
}

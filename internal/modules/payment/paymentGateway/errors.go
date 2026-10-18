package paymentGateway

import (
	"errors"
	"fmt"
)

// sentinel errors matched with errors.Is
var (
	ErrPaymentNotFound         = errors.New("payment not found")
	ErrIdempotencyConflict     = errors.New("idempotency key conflict")
	ErrBadRequest              = errors.New("bad request")
	ErrInvalidStateTransition  = errors.New("invalid state transition")
	ErrInsufficientFunds       = errors.New("insufficient funds")
	ErrExpiredCard             = errors.New("expired card")
	ErrInvalidCard             = errors.New("invalid card")
	ErrCaptureExceedsAvailable = errors.New("capture exceeds available amount")
	ErrRefundExceedsAvailable  = errors.New("refund exceeds available amount")
	ErrGatewayError            = errors.New("gateway internal server error")
)

// GatewayError wraps an HTTP error response from the gateway.
type GatewayError struct {
	Code      int
	ErrorCode string
	Message   string
	Payment   *Payment
	Err       error
}

// Error formats the gateway error with code and message.
func (e *GatewayError) Error() string {
	return fmt.Sprintf("gateway error %d (%s): %s", e.Code, e.ErrorCode, e.Message)
}

// Unwrap exposes the sentinel for errors.Is(err, ErrPaymentNotFound) and friends.
func (e *GatewayError) Unwrap() error {
	return e.Err
}

// NewGatewayError classifies a gateway failure: error_code first, HTTP status as fallback.
func NewGatewayError(code int, errorCode, message string, payment *Payment) error {
	var sentinel error
	switch errorCode {
	case "payment_not_found":
		sentinel = ErrPaymentNotFound
	case "idempotency_conflict":
		sentinel = ErrIdempotencyConflict
	case "insufficient_funds":
		sentinel = ErrInsufficientFunds
	case "expired_card":
		sentinel = ErrExpiredCard
	case "invalid_card":
		sentinel = ErrInvalidCard
	case "invalid_state_transition":
		sentinel = ErrInvalidStateTransition
	case "capture_exceeds_available":
		sentinel = ErrCaptureExceedsAvailable
	case "refund_exceeds_available":
		sentinel = ErrRefundExceedsAvailable
	case "invalid_request", "invalid_amount":
		sentinel = ErrBadRequest
	case "internal_error":
		sentinel = ErrGatewayError
	default:
		// unknown error_code: classify by HTTP status
		switch code {
		case 404:
			sentinel = ErrPaymentNotFound
		case 409:
			sentinel = ErrIdempotencyConflict
		case 400:
			sentinel = ErrBadRequest
		case 500:
			sentinel = ErrGatewayError
		default:
			sentinel = ErrGatewayError
		}
	}
	return &GatewayError{
		Code:      code,
		ErrorCode: errorCode,
		Message:   message,
		Payment:   payment,
		Err:       sentinel,
	}
}

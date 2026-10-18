package paymentGateway

import (
	"time"
)

// PaymentStatus mirrors the gateway's state machine.
type PaymentStatus string

const (
	PaymentStatusPending           PaymentStatus = "pending"
	PaymentStatusAuthorized        PaymentStatus = "authorized"
	PaymentStatusCaptured          PaymentStatus = "captured"
	PaymentStatusCancelled         PaymentStatus = "cancelled"
	PaymentStatusRefunded          PaymentStatus = "refunded"
	PaymentStatusPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentStatusFailed            PaymentStatus = "failed"
)

// Payment mirrors the gateway's payment object.
type Payment struct {
	ID              string            `json:"id"`
	Reference       string            `json:"reference"`
	Amount          int64             `json:"amount"`
	Currency        string            `json:"currency"`
	Status          PaymentStatus     `json:"status"`
	CapturedAmount  int64             `json:"captured_amount"`
	RefundedAmount  int64             `json:"refunded_amount"`
	AvailableAmount int64             `json:"available_amount"`
	CardMask        string            `json:"card_mask,omitempty"`
	WebhookURL      string            `json:"webhook_url,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
}

// CreatePaymentRequest is the create-payment payload.
type CreatePaymentRequest struct {
	Reference  string            `json:"reference"`
	Amount     int64             `json:"amount"`
	Currency   string            `json:"currency,omitempty"`
	WebhookURL string            `json:"webhook_url,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// AuthorizeRequest is the authorize payload.
type AuthorizeRequest struct {
	CardNumber string `json:"card_number"`
}

// CaptureRequest is the capture payload.
// Nil Amount captures the full remaining authorized amount.
type CaptureRequest struct {
	Amount *int64 `json:"amount,omitempty"`
}

// RefundRequest is the refund payload.
// Nil Amount refunds the full remaining captured amount.
type RefundRequest struct {
	Amount *int64 `json:"amount,omitempty"`
}

// ListPaymentsRequest carries optional list filters.
type ListPaymentsRequest struct {
	Reference string
	Status    string
	Limit     int
	Offset    int
}

// ListPaymentsResponse is one page of payments.
type ListPaymentsResponse struct {
	Items  []Payment `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}

// HealthResponse is the gateway health body.
type HealthResponse struct {
	Status string `json:"status"`
}

// WebhookLog is one delivery attempt logged by the gateway.
type WebhookLog struct {
	ID           int64     `json:"id"`
	PaymentID    string    `json:"payment_id"`
	EventType    string    `json:"event_type"`
	URL          string    `json:"url"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body"`
	Attempt      int       `json:"attempt"`
	DurationMs   int       `json:"duration_ms"`
	CreatedAt    time.Time `json:"created_at"`
}

// WebhookLogsResponse wraps a payment's delivery logs.
type WebhookLogsResponse struct {
	Items []WebhookLog `json:"items"`
}

// WebhookPayload is what the gateway sends on payment state changes.
type WebhookPayload struct {
	ID        string            `json:"id"`
	PaymentID string            `json:"payment_id"`
	Reference string            `json:"reference"`
	Amount    int64             `json:"amount"`
	Status    PaymentStatus     `json:"status"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// ErrorResponse is the gateway's error body.
type ErrorResponse struct {
	Error     string   `json:"error"`
	ErrorCode string   `json:"error_code"`
	Payment   *Payment `json:"payment,omitempty"`
}

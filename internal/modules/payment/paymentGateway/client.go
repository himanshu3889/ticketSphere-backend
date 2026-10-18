package paymentGateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const defaultTimeout = 30 * time.Second

// Config holds gateway client settings.
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client   // optional override
	Logger     *logrus.Logger // optional
	Tracer     trace.Tracer
}

// Client talks to the payment gateway over HTTP.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	tracer     trace.Tracer
}

// New builds the gateway client from cfg.
func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		return nil, errors.New("payment gateway base URL is required")
	}
	if cfg.Tracer == nil {
		return nil, errors.New("tracer is required")
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: httpClient,
		logger:     cfg.Logger,
		tracer:     cfg.Tracer,
	}, nil
}

// doRequest runs one HTTP call against the gateway. idempotencyKey is only
// sent when non-empty; methods without idempotency pass "".
func (c *Client) doRequest(ctx context.Context, method, path string, body any, idempotencyKey string, result any) (err error) {
	var span trace.Span
	ctx, span = c.tracer.Start(ctx, constants.SpanGatewayRequest)

	// attributes worth filtering on in the tracing backend
	span.SetAttributes(
		attribute.String("http.method", method),
		attribute.String("http.url", path),
		attribute.String("idempotency_key", idempotencyKey),
	)

	defer utils.EndSpan(span, err)

	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("invalid gateway URL: %w", err)
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("X-Request-ID", uuid.Must(uuid.NewRandom()).String())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}

	// c.logger.WithFields(logrus.Fields{
	// 	"method":          method,
	// 	"url":             u.String(),
	// 	"x_request_id":    req.Header.Get("X-Request-ID"),
	// 	"idempotency_key": idempotencyKey,
	// }).Debug("payment gateway request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// c.logger.WithFields(logrus.Fields{
	// 	"method":       method,
	// 	"url":          u.String(),
	// 	"status":       resp.StatusCode,
	// 	"x_request_id": resp.Header.Get("X-Request-ID"),
	// 	"body":         string(respBody),
	// }).Debug("payment gateway response")

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if result != nil && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("unmarshal success response: %w", err)
			}
		}
		return nil
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(respBody, &errResp); err != nil {
		return NewGatewayError(resp.StatusCode, "", string(respBody), nil)
	}

	return NewGatewayError(resp.StatusCode, errResp.ErrorCode, errResp.Error, errResp.Payment)
}

// HealthCheck hits GET /health.
func (c *Client) HealthCheck(ctx context.Context) (*HealthResponse, error) {
	var res HealthResponse
	if err := c.doRequest(ctx, http.MethodGet, "/health", nil, "", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CreatePayment creates a new payment order.
func (c *Client) CreatePayment(ctx context.Context, req CreatePaymentRequest, idempotencyKey string) (*Payment, error) {
	var res Payment
	if err := c.doRequest(ctx, http.MethodPost, "/api/payments", req, idempotencyKey, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPayment fetches one payment by id.
func (c *Client) GetPayment(ctx context.Context, id string) (*Payment, error) {
	var res Payment
	path := fmt.Sprintf("/api/payments/%s", id)
	if err := c.doRequest(ctx, http.MethodGet, path, nil, "", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// ListPayments lists payments, filtered by req.
func (c *Client) ListPayments(ctx context.Context, req ListPaymentsRequest) (*ListPaymentsResponse, error) {
	q := url.Values{}
	if req.Reference != "" {
		q.Set("reference", req.Reference)
	}
	if req.Status != "" {
		q.Set("status", req.Status)
	}
	if req.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", req.Limit))
	}
	if req.Offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", req.Offset))
	}

	path := "/api/payments"
	if len(q) > 0 {
		path = path + "?" + q.Encode()
	}

	var res ListPaymentsResponse
	if err := c.doRequest(ctx, http.MethodGet, path, nil, "", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Authorize authorizes a payment. A GatewayError may still carry the payment state.
func (c *Client) Authorize(ctx context.Context, id string, req AuthorizeRequest, idempotencyKey string) (*Payment, error) {
	var res Payment
	path := fmt.Sprintf("/api/payments/%s/authorize", id)
	if err := c.doRequest(ctx, http.MethodPost, path, req, idempotencyKey, &res); err != nil {
		var gwErr *GatewayError
		if errors.As(err, &gwErr) && gwErr.Payment != nil {
			return gwErr.Payment, err
		}
		return nil, err
	}
	return &res, nil
}

// Capture captures an authorized payment.
func (c *Client) Capture(ctx context.Context, id string, req CaptureRequest, idempotencyKey string) (*Payment, error) {
	var res Payment
	path := fmt.Sprintf("/api/payments/%s/capture", id)
	if err := c.doRequest(ctx, http.MethodPost, path, req, idempotencyKey, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Cancel cancels a payment.
func (c *Client) Cancel(ctx context.Context, id string, idempotencyKey string) (*Payment, error) {
	var res Payment
	path := fmt.Sprintf("/api/payments/%s/cancel", id)
	if err := c.doRequest(ctx, http.MethodPost, path, nil, idempotencyKey, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Refund refunds a captured payment.
func (c *Client) Refund(ctx context.Context, id string, req RefundRequest, idempotencyKey string) (*Payment, error) {
	var res Payment
	path := fmt.Sprintf("/api/payments/%s/refund", id)
	if err := c.doRequest(ctx, http.MethodPost, path, req, idempotencyKey, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetWebhookLogs returns the webhook delivery attempts for a payment.
func (c *Client) GetWebhookLogs(ctx context.Context, paymentID string) (*WebhookLogsResponse, error) {
	var res WebhookLogsResponse
	path := fmt.Sprintf("/api/webhook-logs/%s", url.PathEscape(paymentID))
	if err := c.doRequest(ctx, http.MethodGet, path, nil, "", &res); err != nil {
		return nil, err
	}
	return &res, nil
}

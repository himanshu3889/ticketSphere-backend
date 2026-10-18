package paymentCacheStore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/google/uuid"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	bookingPaymentScripts "github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/lib/redis"
	"github.com/himanshu3889/ticketSphere-backend/internal/modules/payment/paymentGateway"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
)

type PaymentResult struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	ID      string          `json:"id"`
	Data    *models.Payment `json:"data,omitempty"`
}

// splitClaimValue peels the "{paymentId}|" owner prefix off a stored slot value
func splitClaimValue(v string) (string, string) {
	if i := strings.IndexByte(v, '|'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return "", v
}

// InitiateBookingPayment: claim the slot -> create gateway order -> attach txn
// The gateway is only ever called by the request that won the claim
func (cs *CacheStore) InitiateBookingPayment(ctx context.Context, rawIdempotencyKey string, userId snowflake.ID, bookingID snowflake.ID) (payment *PaymentResult, appErr *appError.Error) {
	ctx, span := cs.Tracer.Start(ctx, constants.SpanPaymentInitiate)
	defer utils.EndSpanAppErr(span, appErr)

	if _, err := uuid.Parse(rawIdempotencyKey); err != nil {
		return nil, appError.NewBadRequest("Idempotency-Key must be a valid UUID")
	}
	if bookingID <= 0 {
		return nil, appError.NewBadRequest("Invalid booking id")
	}
	if userId <= 0 {
		return nil, appError.NewBadRequest("Invalid user id")
	}

	bookingAmount, appErr := cs.GetUserPendingBookingAmount(ctx, userId, bookingID)
	if appErr != nil {
		return nil, appErr
	}

	// Grab the booking request: seat ids and event id go into gateway metadata,
	// so webhook consumers can always reconstruct the booking record
	var bookingRequest *models.BookingRequest
	bookingCacheState, err := cs.CacheManager.HGetInto(ctx, "", keys.UserBookingsKey(userId), bookingID.String(), nil, nil, &bookingRequest)
	if err != nil || bookingCacheState != redisDatabase.CacheHit || bookingRequest == nil {
		return nil, appError.NewInternal("Failed to get booking from redis")
	}

	seatIdStrs := make([]string, 0, len(bookingRequest.SeatIDs))
	for _, seatId := range bookingRequest.SeatIDs {
		seatIdStrs = append(seatIdStrs, seatId.String())
	}

	paymentSafeIdempotencyKey := keys.PaymentIdempotencyKey(userId, rawIdempotencyKey)
	userPaymentsKey := keys.UserPaymentsKey(userId)
	paymentIdempotencyTTL := int(keys.PaymentIdempotencyTTL.Seconds())
	userLatestPaymentsTTL := int(keys.UserLatestPaymentsTTL.Seconds())

	scriptKeys := []string{paymentSafeIdempotencyKey, userPaymentsKey}

	// Build the local payment record first — the claim needs nothing from the gateway.
	paymentId := utils.GenerateSnowflakeID(cs.Store.SnowflakeNode)
	paymentRequest := models.Payment{
		ID:        paymentId,
		BookingID: bookingID,
		Amount:    bookingAmount,
		Provider:  models.PaymentProviderInternal,
		Status:    models.PaymentStatusPending,
		CreatedAt: time.Now(),
	}
	paymentReqBytes, err := json.Marshal(&paymentRequest)
	if err != nil {
		return nil, appError.NewInternal("Failed to marshal initiate payment request")
	}
	claimValue := fmt.Sprintf("%d|%s", paymentId, paymentReqBytes)

	span.SetAttributes(
		attribute.String("payment.idempotency_key", paymentSafeIdempotencyKey),
		attribute.Int("payment.user_id", int(userId)),
		attribute.Int("payment.booking_id", int(bookingID)),
		attribute.Int("payment.id", int(paymentId)),
	)

	// Claim the payment slot atomically, before the gateway ever hears about this
	claimRaw, err := cs.CacheManager.RunScript(ctx,
		"payment_claim_slot",
		bookingPaymentScripts.PaymentClaimSlotScript,
		scriptKeys,
		bookingID.String(),
		claimValue,
		paymentIdempotencyTTL,
		userLatestPaymentsTTL,
	)
	if err != nil {
		cs.logger.WithError(err).WithFields(logrus.Fields{"keys": scriptKeys}).Error("Failed to run claim script")
		return nil, appError.NewInternal("Failed to initiate payment")
	}

	claimArr, ok := claimRaw.([]interface{})
	if !ok || len(claimArr) < 2 {
		cs.logger.WithFields(logrus.Fields{"claimRaw": claimRaw}).Error("Unexpected claim script result")
		return nil, appError.NewInternal("Failed to initiate payment")
	}

	claimStatus, _ := claimArr[0].(string)
	if claimStatus == "IDEMPOTENCY_HIT" || claimStatus == "PAYMENT_HIT" {
		raw := claimArr[1]
		storedValue, ok := raw.(string)
		if !ok || storedValue == "" {
			cs.logger.WithFields(logrus.Fields{"raw": raw}).Error("Failed to read stored payment data")
			return nil, appError.NewInternal("Failed to retrieve existing payment")
		}

		_, storedJSON := splitClaimValue(storedValue)
		var storedPayment models.Payment
		if err = json.Unmarshal([]byte(storedJSON), &storedPayment); err != nil {
			cs.logger.WithError(err).WithFields(logrus.Fields{"raw": raw}).Error("Failed to unmarshal stored payment data")
			return nil, appError.NewInternal("Failed to process existing payment")
		}

		// Slot claimed but gateway txn not attached yet — someone is mid-initiation
		if storedPayment.GatewayTransactionID == nil || *storedPayment.GatewayTransactionID == "" {
			return &PaymentResult{
				Status:  http.StatusConflict,
				Message: "Payment initiation already in progress",
				ID:      storedPayment.ID.String(),
			}, nil
		}

		paymentResult := &PaymentResult{
			ID:   storedPayment.ID.String(),
			Data: &storedPayment,
		}
		if claimStatus == "IDEMPOTENCY_HIT" {
			paymentResult.Status = http.StatusFound
			paymentResult.Message = "Payment Idempotency Hit"
		} else {
			paymentResult.Status = http.StatusOK
			paymentResult.Message = "Payment Already Initiated"
		}
		return paymentResult, nil
	}

	if claimStatus != "CLAIMED" {
		cs.logger.WithFields(logrus.Fields{"status": claimStatus, "raw": claimRaw}).Error("Unexpected claim script status")
		return nil, appError.NewInternal("Failed to initiate payment")
	}

	// We own the slot — only this request reaches the gateway
	payementGatewayReq := paymentGateway.CreatePaymentRequest{
		Reference:  paymentId.String(),
		Amount:     bookingAmount,
		Currency:   "INR",
		WebhookURL: cs.paymentGatewayConfigs.WebhookURL,
		Metadata: map[string]string{
			"payment_id": paymentId.String(),
			"booking_id": bookingID.String(),
			"user_id":    userId.String(),
			"event_id":   bookingRequest.EventID.String(),
			"seat_ids":   strings.Join(seatIdStrs, ","),
		},
	}

	gatewayPayment, err := cs.PaymentGateway.CreatePayment(ctx, payementGatewayReq, paymentSafeIdempotencyKey)
	if err != nil {
		cs.logger.WithError(err).WithFields(logrus.Fields{"payment_id": paymentId.String()}).Error("Failed to create payment gateway order, releasing claim")

		// Release the claim so a retry starts clean. Detached from the request
		// context so a disconnecting client can't break the rollback.
		rbCtx, rbCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer rbCancel()
		if _, relErr := cs.CacheManager.RunScript(rbCtx,
			"payment_release_claim",
			bookingPaymentScripts.PaymentReleaseClaimScript,
			scriptKeys,
			bookingID.String(),
			paymentId.String(),
		); relErr != nil {
			cs.logger.WithError(relErr).WithFields(logrus.Fields{"payment_id": paymentId.String()}).Error("Failed to release payment claim")
		}
		return nil, appError.NewInternal("Failed to create payment gateway order")
	}

	// Attach the gateway transaction id to our claim (owner-checked inside the script)
	paymentRequest.GatewayTransactionID = &gatewayPayment.ID
	finalBytes, err := json.Marshal(&paymentRequest)
	if err != nil {
		return nil, appError.NewInternal("Failed to marshal payment request")
	}
	finalValue := fmt.Sprintf("%d|%s", paymentId, finalBytes)

	attachRaw, err := cs.CacheManager.RunScript(ctx,
		"payment_attach_gateway_txn",
		bookingPaymentScripts.PaymentAttachGatewayTxnScript,
		[]string{userPaymentsKey},
		bookingID.String(),
		paymentId.String(),
		finalValue,
	)
	if err != nil {
		cs.logger.WithError(err).WithFields(logrus.Fields{"keys": userPaymentsKey}).Error("Failed to run attach script")
		return nil, appError.NewInternal("Failed to initiate payment")
	}
	attachArr, _ := attachRaw.([]interface{})
	attachStatus, _ := attachArr[0].(string)
	if attachStatus != "ATTACHED" {
		// Gateway order exists but we lost the slot — currently unrecoverable here;
		// the reconciliation loop (by gateway Reference) is the planned follow-up
		cs.logger.WithFields(logrus.Fields{"payment_id": paymentId.String(), "status": attachStatus}).Error("Failed to attach gateway txn to payment slot")
		return nil, appError.NewInternal("Failed to initiate payment")
	}

	paymentResult := &PaymentResult{
		Status:  http.StatusOK,
		Message: "Payment Initiated",
		ID:      paymentId.String(),
		Data:    &paymentRequest,
	}

	// Publish after attach so the event carries the gateway txn id
	err = cs.EventBus.PublishPaymentInitiated(ctx, paymentResult.Data)
	if err != nil {
		cs.logger.WithError(err).WithFields(logrus.Fields{"req": paymentRequest}).Error("Failed to publish payment initiated event")
		return paymentResult, appError.NewInternal("Failed to place payment")
	}

	return paymentResult, nil
}

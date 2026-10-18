package paymentStore

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/jmoiron/sqlx"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
)

// projects payment.created, insert the PENDING payment row.
func (store *Store) UpsertPaymentCreated(ctx context.Context, payment *models.Payment) error {
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Error("Begin transaction failed: Unable to project payment.created")
		return fmt.Errorf("begin payment.created projection tx: %w", err)
	}
	defer tx.Rollback()

	if err := upsertPendingPaymentTx(ctx, tx, payment); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Commit failed: Unable to project payment.created")
		return fmt.Errorf("commit payment.created projection: %w", err)
	}
	return nil
}

// projects payment.authorized: self-heal the PENDING row
// (if payment.created was lost / is re-driven late), then transition it PENDING -> SUCCESS with processed_at set.
// transitioned=false means the payment was already past PENDING (replay or late event).
func (store *Store) ApplyPaymentAuthorized(ctx context.Context, payment *models.Payment, processedAt time.Time) (transitioned bool, err error) {
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Error("Begin transaction failed: Unable to project payment.authorized")
		return false, fmt.Errorf("begin payment.authorized projection tx: %w", err)
	}
	defer tx.Rollback()

	if err := upsertPendingPaymentTx(ctx, tx, payment); err != nil {
		return false, err
	}

	transitioned, err = transitionPaymentStatusTx(ctx, tx, payment.ID, models.PaymentStatusSuccess, processedAt)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Commit failed: Unable to project payment.authorized")
		return false, fmt.Errorf("commit payment.authorized projection: %w", err)
	}
	return transitioned, nil
}

// inserts the payment row as PENDING
func upsertPendingPaymentTx(ctx context.Context, tx *sqlx.Tx, payment *models.Payment) error {
	query := `
		INSERT INTO payments (id, booking_id, amount, provider, gateway_transaction_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, query,
		payment.ID,
		payment.BookingID,
		payment.Amount,
		payment.Provider,
		payment.GatewayTransactionID,
		models.PaymentStatusPending,
		payment.CreatedAt,
	); err != nil {
		return fmt.Errorf("upsert payment %d: %w", payment.ID, err)
	}
	return nil
}

// moves a payment PENDING -> targetStatus;
// TODO: How to guard that it should not go in the lower states ?
func transitionPaymentStatusTx(ctx context.Context, tx *sqlx.Tx, paymentID snowflake.ID, targetStatus models.PaymentStatus, processedAt time.Time) (transitioned bool, err error) {
	query := `
		UPDATE payments
		SET status = $2, processed_at = $3
		WHERE id = $1 AND status = 'PENDING'
	`
	res, err := tx.ExecContext(ctx, query, paymentID, targetStatus, processedAt)
	if err != nil {
		return false, fmt.Errorf("transition payment %d to %s: %w", paymentID, targetStatus, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected for payment %d transition: %w", paymentID, err)
	}
	return rowsAffected > 0, nil
}

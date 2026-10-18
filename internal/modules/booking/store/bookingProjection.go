package bookingStore

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
)

// projects booking.created: insert the PENDING booking row
// and link its seats. Replays and late duplicates no-op on the primary key.
func (store *Store) UpsertBookingCreated(ctx context.Context, bookingRequest *models.BookingRequest) error {
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Error("Begin transaction failed: Unable to project booking.created")
		return fmt.Errorf("begin booking.created projection tx: %w", err)
	}
	defer tx.Rollback()

	if err := upsertPendingBookingTx(ctx, tx, bookingRequest); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Commit failed: Unable to project booking.created")
		return fmt.Errorf("commit booking.created projection: %w", err)
	}
	return nil
}

// projects booking.confirmed: self-heal the PENDING row,
// then transition it PENDING -> CONFIRMED with confirmed_at set.
// transitioned=false means the booking was already past PENDING (replay or late event).
func (store *Store) ApplyBookingConfirmed(ctx context.Context, bookingRequest *models.BookingRequest, confirmedAt time.Time) (transitioned bool, err error) {
	return store.transitionBooking(ctx, bookingRequest, models.BookingStatusConfirmed, sql.NullTime{Time: confirmedAt, Valid: true})
}

// TODO: the schema has no failure enum
// projects booking.failed: self-heal the PENDING row, then
// transition it PENDING -> FAILED .
// transitioned=false means the booking was already past PENDING (replay or late event).
func (store *Store) ApplyBookingFailed(ctx context.Context, bookingRequest *models.BookingRequest) (transitioned bool, err error) {
	return store.transitionBooking(ctx, bookingRequest, models.BookingStatusFailed, sql.NullTime{Valid: false})
}

// runs the shared guarded transition in one
// transaction so the row and its seat links never exist in a half-written state.
func (store *Store) transitionBooking(ctx context.Context, bookingRequest *models.BookingRequest, targetStatus models.BookingStatus, confirmedAt sql.NullTime) (transitioned bool, err error) {
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Error("Begin transaction failed: Unable to project booking transition")
		return false, fmt.Errorf("begin booking %s projection tx: %w", targetStatus, err)
	}
	defer tx.Rollback()

	if err := upsertPendingBookingTx(ctx, tx, bookingRequest); err != nil {
		return false, err
	}

	transitioned, err = transitionBookingStatusTx(ctx, tx, bookingRequest.ID, targetStatus, confirmedAt)
	if err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Commit failed: Unable to project booking transition")
		return false, fmt.Errorf("commit booking %s projection: %w", targetStatus, err)
	}
	return transitioned, nil
}

// inserts the booking row as PENDING together with its
// seat links. An existing row is left completely untouched.
func upsertPendingBookingTx(ctx context.Context, tx *sqlx.Tx, bookingRequest *models.BookingRequest) error {
	queryBooking := `
		INSERT INTO bookings (id, event_id, user_id, status, total_amount, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, queryBooking,
		bookingRequest.ID,
		bookingRequest.EventID,
		bookingRequest.UserID,
		models.BookingStatusPending,
		bookingRequest.TotalAmount,
		bookingRequest.ExpiresAt,
		bookingRequest.CreatedAt,
	); err != nil {
		return fmt.Errorf("upsert booking %d: %w", bookingRequest.ID, err)
	}

	// UNNEST flattens the seat id array into one row per seat
	querySeats := `
		INSERT INTO booking_seats (booking_id, seat_id)
		VALUES ($1, UNNEST($2::bigint[]))
		ON CONFLICT DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, querySeats, bookingRequest.ID, pq.Array(bookingRequest.SeatIDs)); err != nil {
		return fmt.Errorf("link seats for booking %d: %w", bookingRequest.ID, err)
	}
	return nil
}

// moves a booking PENDING -> targetStatus. The PENDING guard is the idempotency mechanism.
func transitionBookingStatusTx(ctx context.Context, tx *sqlx.Tx, bookingID snowflake.ID, targetStatus models.BookingStatus, confirmedAt sql.NullTime) (transitioned bool, err error) {
	// COALESCE leaves confirmed_at untouched for statuses that don't set it (FAILED)
	query := `
		UPDATE bookings
		SET status = $2,
		    confirmed_at = COALESCE($3, confirmed_at)
		WHERE id = $1 AND status = 'PENDING'
	`
	res, err := tx.ExecContext(ctx, query, bookingID, targetStatus, confirmedAt)
	if err != nil {
		return false, fmt.Errorf("transition booking %d to %s: %w", bookingID, targetStatus, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("rows affected for booking %d transition: %w", bookingID, err)
	}
	return rowsAffected > 0, nil
}

// ---------------------------------------------------------------------------
// Batched projection — one transaction per N events, used by the consumer
// batcher. Same guards as the row-wise path: ON CONFLICT no-ops replays and
// the PENDING filter keeps re-ordered events safe.
// ---------------------------------------------------------------------------

// UpsertBookingCreatedBatch projects many booking.created events in one tx.
func (store *Store) UpsertBookingCreatedBatch(ctx context.Context, bookingRequests []*models.BookingRequest) error {
	if len(bookingRequests) == 0 {
		return nil
	}

	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin booking.created batch projection tx: %w", err)
	}
	defer tx.Rollback()

	if err := upsertPendingBookingsBatchTx(ctx, tx, bookingRequests); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit booking.created batch projection (%d rows): %w", len(bookingRequests), err)
	}
	return nil
}

// ApplyBookingConfirmedBatch self-heals missing PENDING rows, then moves them
// PENDING -> CONFIRMED in one tx. Returns how many rows actually transitioned
// (replays and late events no-op on the PENDING guard).
func (store *Store) ApplyBookingConfirmedBatch(ctx context.Context, bookingRequests []*models.BookingRequest, confirmedAts []time.Time) (int64, error) {
	if len(bookingRequests) != len(confirmedAts) {
		return 0, fmt.Errorf("batch transition length mismatch: %d rows vs %d confirmed_at", len(bookingRequests), len(confirmedAts))
	}
	return store.transitionBookingsBatch(ctx, bookingRequests, models.BookingStatusConfirmed, confirmedAts)
}

// ApplyBookingFailedBatch is ApplyBookingConfirmedBatch for PENDING -> FAILED.
func (store *Store) ApplyBookingFailedBatch(ctx context.Context, bookingRequests []*models.BookingRequest) (int64, error) {
	return store.transitionBookingsBatch(ctx, bookingRequests, models.BookingStatusFailed, nil)
}

// upsertPendingBookingsBatchTx is the batch twin of upsertPendingBookingTx:
// one UNNEST-driven INSERT for bookings plus one for seat links. Duplicate
// booking ids inside the same batch are collapsed first — ON CONFLICT only
// guards committed rows, not intra-statement duplicates.
func upsertPendingBookingsBatchTx(ctx context.Context, tx *sqlx.Tx, bookingRequests []*models.BookingRequest) error {
	if len(bookingRequests) == 0 {
		return nil
	}

	seenBookings := make(map[snowflake.ID]struct{}, len(bookingRequests))
	ids := make([]int64, 0, len(bookingRequests))
	eventIDs := make([]int64, 0, len(bookingRequests))
	userIDs := make([]int64, 0, len(bookingRequests))
	totalAmounts := make([]int64, 0, len(bookingRequests))
	expiresAts := make([]time.Time, 0, len(bookingRequests))
	createdAts := make([]time.Time, 0, len(bookingRequests))

	seatBookingIDs := make([]int64, 0, len(bookingRequests))
	seatIDs := make([]int64, 0, len(bookingRequests))
	seenSeats := make(map[[2]int64]struct{}, len(bookingRequests))

	for _, bookingRequest := range bookingRequests {
		if _, dup := seenBookings[bookingRequest.ID]; dup {
			continue
		}
		seenBookings[bookingRequest.ID] = struct{}{}
		ids = append(ids, int64(bookingRequest.ID))
		eventIDs = append(eventIDs, int64(bookingRequest.EventID))
		userIDs = append(userIDs, int64(bookingRequest.UserID))
		totalAmounts = append(totalAmounts, bookingRequest.TotalAmount)
		expiresAts = append(expiresAts, bookingRequest.ExpiresAt)
		createdAts = append(createdAts, bookingRequest.CreatedAt)

		for _, seatID := range bookingRequest.SeatIDs {
			pair := [2]int64{int64(bookingRequest.ID), int64(seatID)}
			if _, dup := seenSeats[pair]; dup {
				continue
			}
			seenSeats[pair] = struct{}{}
			seatBookingIDs = append(seatBookingIDs, int64(bookingRequest.ID))
			seatIDs = append(seatIDs, int64(seatID))
		}
	}

	// status is always PENDING here — a literal, not a param.
	queryBookings := `
		INSERT INTO bookings (id, event_id, user_id, status, total_amount, expires_at, created_at)
		SELECT u.id, u.event_id, u.user_id, 'PENDING', u.total_amount, u.expires_at, u.created_at
		FROM UNNEST($1::bigint[], $2::bigint[], $3::bigint[], $4::bigint[], $5::timestamptz[], $6::timestamptz[])
			AS u(id, event_id, user_id, total_amount, expires_at, created_at)
		ON CONFLICT (id) DO NOTHING
	`
	if _, err := tx.ExecContext(ctx, queryBookings,
		pq.Array(ids), pq.Array(eventIDs), pq.Array(userIDs),
		pq.Array(totalAmounts), pq.Array(expiresAts), pq.Array(createdAts),
	); err != nil {
		return fmt.Errorf("batch upsert bookings (%d rows): %w", len(ids), err)
	}

	if len(seatIDs) > 0 {
		querySeats := `
			INSERT INTO booking_seats (booking_id, seat_id)
			SELECT * FROM UNNEST($1::bigint[], $2::bigint[])
			ON CONFLICT DO NOTHING
		`
		if _, err := tx.ExecContext(ctx, querySeats, pq.Array(seatBookingIDs), pq.Array(seatIDs)); err != nil {
			return fmt.Errorf("batch link seats (%d rows): %w", len(seatIDs), err)
		}
	}
	return nil
}

// transitionBookingsBatch is the batch twin of transitionBooking: self-heal
// PENDING rows first (keeps created/confirmed reordering safe), then one
// guarded UPDATE. The status = 'PENDING' filter is the idempotency guard.
func (store *Store) transitionBookingsBatch(ctx context.Context, bookingRequests []*models.BookingRequest, targetStatus models.BookingStatus, confirmedAts []time.Time) (int64, error) {
	if len(bookingRequests) == 0 {
		return 0, nil
	}

	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin booking %s batch projection tx: %w", targetStatus, err)
	}
	defer tx.Rollback()

	if err := upsertPendingBookingsBatchTx(ctx, tx, bookingRequests); err != nil {
		return 0, err
	}

	seen := make(map[snowflake.ID]struct{}, len(bookingRequests))
	ids := make([]int64, 0, len(bookingRequests))
	ats := make([]time.Time, 0, len(bookingRequests))
	for i, bookingRequest := range bookingRequests {
		if _, dup := seen[bookingRequest.ID]; dup {
			continue
		}
		seen[bookingRequest.ID] = struct{}{}
		ids = append(ids, int64(bookingRequest.ID))
		if confirmedAts != nil {
			ats = append(ats, confirmedAts[i])
		}
	}

	var res sql.Result
	if len(ats) > 0 {
		query := `
			UPDATE bookings b
			SET status = $1, confirmed_at = v.confirmed_at
			FROM UNNEST($2::bigint[], $3::timestamptz[]) AS v(id, confirmed_at)
			WHERE b.id = v.id AND b.status = 'PENDING'
		`
		res, err = tx.ExecContext(ctx, query, targetStatus, pq.Array(ids), pq.Array(ats))
	} else {
		query := `
			UPDATE bookings b
			SET status = $1
			WHERE b.id = ANY($2::bigint[]) AND b.status = 'PENDING'
		`
		res, err = tx.ExecContext(ctx, query, targetStatus, pq.Array(ids))
	}
	if err != nil {
		return 0, fmt.Errorf("batch transition to %s (%d rows): %w", targetStatus, len(ids), err)
	}

	updated, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected for batch transition to %s: %w", targetStatus, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit booking %s batch projection (%d rows): %w", targetStatus, len(ids), err)
	}
	return updated, nil
}

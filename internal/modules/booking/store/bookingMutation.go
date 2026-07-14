package bookingStore

import (
	"context"
	"fmt"
	"time"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// Book Seats pessimistically
func (store *Store) BookSeatsPessimistic(ctx context.Context, req *models.BookingRequest) (*models.Booking, *appError.Error) {
	// Booking request buildup
	req.ID = utils.GenerateSnowflakeID(store.SnowflakeNode)

	// Begin transaction
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Debug("Begin transaction failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}
	defer tx.Rollback()

	// Pessimistic Locking
	// FOR UPDATE: locks the rows so that no other transaction can update or delete them
	queryLock := `
		SELECT id, status, version, price
		FROM seats
		WHERE id = ANY($1)
		FOR UPDATE
	`
	seats := make([]*models.Seat, 0)
	// Use pq.Array to safely pass the slice to PostgreSQL
	if err := tx.SelectContext(ctx, &seats, queryLock, pq.Array(req.SeatIDs)); err != nil {
		store.logger.WithError(err).WithFields(logrus.Fields{"seats": req.SeatIDs}).Error("Locking seats failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	if len(seats) != len(req.SeatIDs) {
		return nil, appError.NewBadRequest("Some seats not found")
	}

	// Build total and check if seat is available to book or not
	total := 0.0
	for _, seat := range seats {
		if seat.Status != models.SeatStatusAvailable {
			errMsg := fmt.Sprintf("seat %d is not available", seat.ID)
			return nil, appError.NewBadRequest(errMsg)
		}
		total += seat.Price
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	// Create Booking
	queryBooking := `
		INSERT INTO bookings (id, event_id, user_id, status, total_amount, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	if _, err := tx.ExecContext(ctx, queryBooking, req.ID, req.EventID, req.UserID, models.BookingStatusPending, total, expiresAt); err != nil {
		store.logger.WithError(err).WithFields(logrus.Fields{"seats": req.SeatIDs}).Error("Booking create failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Link Seats
	// UNNEST() : function that takes an array and flattens or expands it into a set of rows
	queryLink := `
		INSERT INTO booking_seats (booking_id, seat_id)
		VALUES ($1, UNNEST($2::bigint[]))
	`
	if _, err := tx.ExecContext(ctx, queryLink, req.ID, pq.Array(req.SeatIDs)); err != nil {
		store.logger.WithError(err).WithFields(logrus.Fields{"seats": req.SeatIDs}).Error("Booking seats failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Mark Seats Held
	queryUpdate := `
		UPDATE seats
		SET status = $1, version = version + 1
		WHERE id = ANY($2)
	`
	if _, err := tx.ExecContext(ctx, queryUpdate, models.SeatStatusHeld, pq.Array(req.SeatIDs)); err != nil {
		store.logger.WithError(err).WithFields(logrus.Fields{"seats": req.SeatIDs}).Error("Mark seats held failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Commit Transaction
	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Transaction commit failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Return the newly created Booking object
	return &models.Booking{
		ID:          req.ID,
		EventID:     req.EventID,
		UserID:      req.UserID,
		Status:      models.BookingStatusPending,
		TotalAmount: total,
		ExpiresAt:   expiresAt,
	}, nil
}

// Book Seats optimistically
func (store *Store) BookSeatsOptimistic(ctx context.Context, req *models.BookingRequest) (*models.Booking, *appError.Error) {
	// Booking request buildup
	req.ID = utils.GenerateSnowflakeID(store.SnowflakeNode)

	// Begin transaction
	tx, err := store.PostgresDB.BeginTxx(ctx, nil)
	if err != nil {
		store.logger.WithError(err).Debug("Begin transaction failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}
	defer tx.Rollback()

	// Read Seats; for the optimistic locking, to get the version of the seat
	queryRead := `
		SELECT id, status, version, price
		FROM seats
		WHERE id = ANY($1)
	`
	seats := make([]*models.Seat, 0)
	// Use pq.Array to safely pass the slice to PostgreSQL
	if err := tx.SelectContext(ctx, &seats, queryRead, pq.Array(req.SeatIDs)); err != nil {
		store.logger.WithError(err).Error("Read seats failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	if len(seats) != len(req.SeatIDs) {
		return nil, appError.NewInternal("Some seats not found")
	}

	// Build total and check if seat is available to book or not
	total := 0.0
	for _, seat := range seats {
		if seat.Status != models.SeatStatusAvailable {
			errMsg := fmt.Sprintf("Seat %d is not available", seat.ID)
			return nil, appError.NewBadRequest(errMsg)
		}
		total += seat.Price
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	// Create Booking
	queryBooking := `
		INSERT INTO bookings (id, event_id, user_id, status, total_amount, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	if _, err := tx.ExecContext(ctx, queryBooking, req.ID, req.EventID, req.UserID, models.BookingStatusPending, total, expiresAt); err != nil {
		store.logger.WithError(err).Error("Create booking failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Link Seats
	// UNNEST() : function that takes an array and flattens or expands it into a set of rows
	queryLink := `
		INSERT INTO booking_seats (booking_id, seat_id)
		VALUES ($1, UNNEST($2::bigint[]))
	`
	if _, err := tx.ExecContext(ctx, queryLink, req.ID, pq.Array(req.SeatIDs)); err != nil {
		store.logger.WithError(err).Error("Link booking seats failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Mark Seats Held USING VERSION (Optimistic Lock)
	queryUpdate := `
		UPDATE seats
		SET status = $1, version = version + 1
		WHERE id = $2 AND version = $3
	`
	// Update seat version one by one
	for _, seat := range seats {
		res, err := tx.ExecContext(ctx, queryUpdate, models.SeatStatusHeld, seat.ID, seat.Version)
		if err != nil {
			store.logger.WithFields(logrus.Fields{"seats": req.SeatIDs, "seat": seat}).WithError(err).Error("Update seat version failed: Unable to book seats")
			return nil, appError.NewInternal("Unable to book seats")
		}

		// Check if the row was actually updated
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			store.logger.WithFields(logrus.Fields{"seats": req.SeatIDs, "seat": seat}).WithError(err).Error("Error from rows affected: Unable to book seats")
			return nil, appError.NewInternal("Unable to book seats")
		}

		// If 0 rows were affected someone else updated this seat since we read it
		if rowsAffected == 0 {
			store.logger.WithFields(logrus.Fields{"seats": req.SeatIDs, "seat": seat}).WithError(err).Debug("Optimistic seat failed, chage by other user: Unable to book seats")
			return nil, appError.NewInternal("Unable to book seats")
		}
	}

	// Commit Transaction
	if err := tx.Commit(); err != nil {
		store.logger.WithError(err).Error("Commit transaction failed: Unable to book seats")
		return nil, appError.NewInternal("Unable to book seats")
	}

	// Return the newly created Booking object
	return &models.Booking{
		ID:          req.ID,
		EventID:     req.EventID,
		UserID:      req.UserID,
		Status:      models.BookingStatusPending,
		TotalAmount: total,
		ExpiresAt:   expiresAt,
	}, nil

}

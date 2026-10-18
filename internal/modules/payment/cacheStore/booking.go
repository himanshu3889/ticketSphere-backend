package paymentCacheStore

import (
	"context"
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/himanshu3889/ticketSphere-backend/base/constants"
	redisDatabase "github.com/himanshu3889/ticketSphere-backend/base/infrastructure/redis"
	"github.com/himanshu3889/ticketSphere-backend/base/keys"
	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
)

// GetUserBooking returns a pending, unexpired booking from the user's cache hash.
func (cs *CacheStore) GetUserBooking(ctx context.Context, userId snowflake.ID, bookingId snowflake.ID) (*models.Booking, *appError.Error) {
	ctx, span := cs.Tracer.Start(ctx, constants.SpanBookingGetUserBooking)
	defer span.End()

	userBookingsKey := keys.UserBookingsKey(userId)

	booking := &models.Booking{}
	cacheState, err := cs.CacheManager.HGetInto(ctx, constants.BoundedKeyPaymentInitiateBookingGet, userBookingsKey, bookingId.String(), nil, nil, booking)
	if err != nil {
		return nil, appError.NewInternal("Failed to get booking from redis")
	}

	if cacheState == redisDatabase.CacheMiss || cacheState == redisDatabase.CacheNullBlock {
		return nil, appError.NewInternal("Failed to get booking from redis")
	}

	if booking.ID <= 0 {
		return nil, appError.NewInternal("Invalid booking id")
	}
	if booking.UserID != userId {
		return nil, appError.NewInternal("Invalid user id")
	}
	if booking.Status != models.BookingStatusPending {
		return nil, appError.NewInternal("Invalid booking status")
	}
	if booking.ExpiresAt.Before(time.Now()) {
		return nil, appError.NewInternal("Booking has expired")
	}

	return booking, nil
}

// GetUserPendingBookingAmount returns the total for a pending booking.
func (cs *CacheStore) GetUserPendingBookingAmount(ctx context.Context, userId, bookingId snowflake.ID) (int64, *appError.Error) {
	booking, appErr := cs.GetUserBooking(ctx, userId, bookingId)
	if appErr != nil {
		return 0, appErr
	}

	if booking.Status != models.BookingStatusPending {
		return 0, appError.NewInternal("Booking is not in pending state")
	}

	bookingAmount := booking.TotalAmount
	if bookingAmount <= 0 {
		return 0, appError.NewInternal("Invalid booking amount")
	}
	return bookingAmount, nil
}

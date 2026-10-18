package handlers

import (
	"context"

	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	"github.com/sirupsen/logrus"
)

// HandleBookingConfirmedMongoDB projects the booking.confirmed event into mongodb.
func HandleBookingConfirmedMongoDB(ctx context.Context, body []byte, cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger) error {
	// bookingRequest, err := decodeBookingConfirmed(body)
	// if err != nil {
	// 	return err
	// }
	// // TODO: mark the booking document as confirmed in mongodb
	// logger.Infof("TODO mongodb projection for booking.confirmed: %+v", bookingRequest)
	return nil
}

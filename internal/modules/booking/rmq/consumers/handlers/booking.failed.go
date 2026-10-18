package handlers

import (
	"context"

	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	"github.com/sirupsen/logrus"
)

// HandleBookingFailedMongoDB projects the booking.failed event into mongodb.
func HandleBookingFailedMongoDB(ctx context.Context, body []byte, cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger) error {
	// bookingRequest, err := decodeBookingFailed(body)
	// if err != nil {
	// 	return err
	// }
	// // TODO: mark the booking document as failed in mongodb
	// logger.Infof("TODO mongodb projection for booking.failed: %+v", bookingRequest)
	return nil
}

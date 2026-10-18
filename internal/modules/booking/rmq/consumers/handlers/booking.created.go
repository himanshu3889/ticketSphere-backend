package handlers

import (
	"context"

	bookingCacheStore "github.com/himanshu3889/ticketSphere-backend/internal/modules/booking/cacheStore"
	"github.com/sirupsen/logrus"
)

// HandleBookingCreatedMongoDB projects the booking.created event into mongodb.
func HandleBookingCreatedMongoDB(ctx context.Context, body []byte, cacheStore *bookingCacheStore.CacheStore, logger *logrus.Logger) error {
	// bookingRequest, err := decodeBookingCreated(body)
	// if err != nil {
	// 	return err
	// }
	// // TODO: upsert the pending booking document in mongodb
	// logger.Infof("TODO mongodb projection for booking.created: %+v", bookingRequest)
	return nil
}

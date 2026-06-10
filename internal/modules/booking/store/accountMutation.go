package bookingStore

import (
	"context"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"
	"github.com/sirupsen/logrus"
)

// Create a new user
func (store *Store) CreateUser(ctx context.Context, user *models.User) *appError.Error {
	const queryUserInsert = `
		INSERT INTO users (id, username, email, password, name, image_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, Now(), Now())
		RETURNING *
	`

	user.ID = utils.GenerateSnowflakeID(store.SnowflakeNode)
	if err := store.PostgresDB.GetContext(ctx, user, queryUserInsert,
		user.ID,
		user.Email,
		user.Email,
		user.Password,
		user.Name,
		user.ImageUrl,
	); err != nil {
		if utils.IsPgDBUniqueViolationError(err) {
			store.logger.WithField("email", user.Email).Warn("User already exists in database")
			return appError.NewBadRequest("User already exists in database")
		}
		store.logger.WithFields(logrus.Fields{
			"email":           user.Email,
			"password_length": len(user.Password),
			"image_url":       user.ImageUrl,
		}).WithError(err).Error("Failed to create user")
		return appError.NewInternal("Failed to create user")
	}
	return nil
}

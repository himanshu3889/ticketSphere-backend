package bookingStore

import (
	"context"
	"database/sql"
	"errors"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"

	"github.com/bwmarrin/snowflake"
)

// Return the user by id if found
func (store *Store) GetUserByID(ctx context.Context, userID snowflake.ID) (*models.User, *appError.Error) {
	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE id = $1
		LIMIT 1
	`

	err := store.PostgresDB.GetContext(ctx, &user, query, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("User not found", appError.WithCause(err))
		}
		store.logger.WithError(err).Error("Failed to fetch user by ID")
		return nil, appError.NewInternal("Failed to fetch user", appError.WithCause(err))
	}

	return &user, nil
}

// Return the user by the username if found
func (store *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, *appError.Error) {
	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE username = $1
		LIMIT 1
	`

	err := store.PostgresDB.GetContext(ctx, &user, query, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("User not found", appError.WithCause(err))
		}
		store.logger.WithError(err).Error("Failed to fetch user by username")
		return nil, appError.NewInternal("Failed to fetch user", appError.WithCause(err))
	}

	return &user, nil
}

// Return the user by the email if found
func (store *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, *appError.Error) {

	var user models.User

	query := `
		SELECT *
		FROM users
		WHERE email = $1
		LIMIT 1
	`

	err := store.PostgresDB.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, appError.NewNotFound("User not found", appError.WithCause(err))
		}
		store.logger.WithError(err).Error("Failed to fetch user by email")
		return nil, appError.NewInternal("Failed to fetch user", appError.WithCause(err))
	}

	return &user, nil
}

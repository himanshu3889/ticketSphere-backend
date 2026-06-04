package bookingStore

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/himanshu3889/ticketSphere-backend/base/models"
	"github.com/himanshu3889/ticketSphere-backend/base/utils"
	"github.com/himanshu3889/ticketSphere-backend/base/utils/appError"

	"github.com/bwmarrin/snowflake"
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
		if utils.IsDBUniqueViolationError(err) {
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

// Create a new session for user in session
func (store *Store) CreateSession(ctx context.Context, session *models.UserSession) *appError.Error {
	const querySessionInsert = `
        INSERT INTO user_sessions (id, user_id, refresh_token, device_info,
                                   ip_address, expires_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING *`
	err := store.PostgresDB.GetContext(ctx, session, querySessionInsert,
		utils.GenerateSnowflakeID(store.SnowflakeNode),
		session.UserID,
		session.RefreshToken,
		session.DeviceInfo,
		session.IPAddress,
		session.ExpiresAt)
	if err != nil {
		store.logger.WithFields(logrus.Fields{
			"user_id": session.UserID,
		}).WithError(err).Error("Failed to create user session")
		return appError.NewInternal("Failed to create user session")
	}
	return nil
}

// Returns the session only if it belongs to the given user and is still valid.
func (store *Store) GetUserSessionByToken(ctx context.Context, userID snowflake.ID, token string) (*models.UserSession, *appError.Error) {
	const querySessionGet = `
		SELECT *
		FROM user_sessions
		WHERE user_id = $1
		  AND refresh_token = $2
		  AND expires_at > NOW()`
	var s models.UserSession
	err := store.PostgresDB.GetContext(ctx, &s, querySessionGet, userID, token)
	if err != nil {
		store.logger.WithFields(logrus.Fields{
			"user_id": userID,
			"token":   token,
		}).Error("Unable to get user session token")
		return nil, appError.NewInternal(fmt.Sprintf("Unable to get user %s session token", userID))
	}
	return &s, nil
}

// Removes the exact session row for that user/token pair.
func (store *Store) DeleteUserSession(ctx context.Context, userID snowflake.ID, refreshToken string) *appError.Error {
	const queryDeleteSession = `DELETE FROM user_sessions WHERE user_id = $1 AND refresh_token = $2`
	_, err := store.PostgresDB.ExecContext(ctx, queryDeleteSession, userID, refreshToken)
	if err != nil {
		if err == sql.ErrNoRows {
			store.logger.WithFields(logrus.Fields{
				"user_id": userID,
			}).Warn("session not found for delete")
			return appError.NewNotFound("session not found")
		}
		return appError.NewInternal("Unable to delete user session")
	}
	return nil
}

// Delete all user sessions logs the user out everywhere.
func (store *Store) DeleteUserAllSessions(ctx context.Context, userID snowflake.ID) *appError.Error {
	const queryDeleteAllSessions = `DELETE FROM user_sessions WHERE user_id = $1`
	_, err := store.PostgresDB.ExecContext(ctx, queryDeleteAllSessions, userID)
	if err != nil {
		store.logger.WithFields(logrus.Fields{
			"user_id": userID,
		}).WithError(err).Error("Unable to delete the user sessions")
		return appError.NewInternal("Unable to delete the user sessions")
	}
	return nil
}

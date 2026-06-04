package models

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/snowflake"
)

type User struct {
	ID        snowflake.ID `db:"id" json:"id"`
	Username  string       `db:"username" json:"username"`
	Email     string       `db:"email" json:"email"`
	Password  string       `db:"password" json:"-"`
	Name      string       `db:"name" json:"name"`
	ImageUrl  string       `db:"image_url" json:"imageUrl"`
	CreatedAt time.Time    `db:"created_at" json:"-"`
	UpdatedAt time.Time    `db:"updated_at" json:"-"`
	DeletedAt *time.Time   `db:"deleted_at" json:"-"`
}

type UserSession struct {
	ID           snowflake.ID    `db:"id" json:"id"`
	UserID       snowflake.ID    `db:"user_id" json:"userId"`
	RefreshToken string          `db:"refresh_token" json:"refreshToken"`
	DeviceInfo   json.RawMessage `db:"device_info" json:"deviceInfo,omitempty"`
	IPAddress    string          `db:"ip_address" json:"ipAddress,omitempty"`
	CreatedAt    time.Time       `db:"created_at" json:"-"`
	UpdatedAt    time.Time       `db:"updated_at" json:"-"`
	ExpiresAt    time.Time       `db:"expires_at" json:"expiresAt"`
}

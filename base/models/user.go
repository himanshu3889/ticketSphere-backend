package models

import (
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

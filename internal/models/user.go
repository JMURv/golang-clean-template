package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID `db:"id"                json:"id"`
	Name            string    `db:"name"              json:"name"`
	Password        string    `db:"password"          json:"password"`
	Email           string    `db:"email"             json:"email"`
	Avatar          string    `db:"avatar"            json:"avatar"`
	IsActive        bool      `db:"is_active"         json:"isActive"`
	IsEmailVerified bool      `db:"is_email_verified" json:"isEmailVerified"`
	Devices         []Device  `db:"devices"           json:"devices"`
	CreatedAt       time.Time `db:"created_at"        json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at"        json:"updatedAt"`
}

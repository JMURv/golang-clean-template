package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uint64    `db:"id"           json:"id"`
	UserID     uuid.UUID `db:"user_id"      json:"userId"`
	TokenHash  string    `db:"token_hash"   json:"tokenHash"`
	ExpiresAt  time.Time `db:"expires_at"   json:"expiresAt"`
	Revoked    bool      `db:"revoked"      json:"revoked"`
	DeviceID   string    `db:"device_id"    json:"deviceId"`
	LastUsedAt time.Time `db:"last_used_at" json:"lastUsedAt"`
	CreatedAt  time.Time `db:"created_at"   json:"createdAt"`
}

type Device struct {
	ID         string    `db:"id"          json:"id"`
	UserID     uuid.UUID `db:"user_id"     json:"userId"`
	Name       string    `db:"name"        json:"name"`
	DeviceType string    `db:"device_type" json:"deviceType"`
	OS         string    `db:"os"          json:"os"`
	Browser    string    `db:"browser"     json:"browser"`
	UA         string    `db:"user_agent"  json:"ua"`
	IP         string    `db:"ip"          json:"ip"`
	LastActive time.Time `db:"last_active" json:"lastActive"`
	CreatedAt  time.Time `db:"created_at"  json:"createdAt"`
}

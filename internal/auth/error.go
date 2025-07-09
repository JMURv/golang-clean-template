package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	// ErrTokenRevoked is error that indicates token expired.
	ErrTokenRevoked = errors.New("token revoked")
)

package jwt

import "errors"

var (
	// ErrWhileCreatingToken is an error while creating token.
	ErrWhileCreatingToken = errors.New("error while creating token")

	// ErrUnexpectedSignMethod is an error that indicates unexpected sign method.
	ErrUnexpectedSignMethod = errors.New("unexpected signing method")

	// ErrInvalidToken is an error that indicates invalid token.
	ErrInvalidToken = errors.New("invalid token")
)

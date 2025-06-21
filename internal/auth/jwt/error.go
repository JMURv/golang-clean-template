package jwt

import "errors"

var ErrWhileCreatingToken = errors.New("error while creating token")
var ErrUnexpectedSignMethod = errors.New("unexpected signing method")
var ErrInvalidToken = errors.New("invalid token")

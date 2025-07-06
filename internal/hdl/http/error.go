package http

import "errors"

var (
	ErrMethodNotAllowed = errors.New("method not allowed")
	ErrInvalidURL       = errors.New("invalid URL")
	ErrRetrievePathVars = errors.New("cannot retrieve path variables")
)

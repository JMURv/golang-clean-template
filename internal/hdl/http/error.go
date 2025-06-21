package http

import "errors"

var ErrMethodNotAllowed = errors.New("method not allowed")
var ErrInvalidURL = errors.New("invalid URL")
var ErrRetrievePathVars = errors.New("cannot retrieve path variables")

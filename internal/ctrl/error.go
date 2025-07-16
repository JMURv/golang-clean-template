package ctrl

import "errors"

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned when a resource already exists.
var ErrAlreadyExists = errors.New("already exists")

// ErrCodeIsNotValid is returned when login code is not valid.
var ErrCodeIsNotValid = errors.New("code is not valid")

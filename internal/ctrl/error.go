package ctrl

import "errors"

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = errors.New("not found")

// ErrAlreadyExists is returned when a resource already exists.
var ErrAlreadyExists = errors.New("already exists")

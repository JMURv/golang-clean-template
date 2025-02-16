package validation

import "errors"

var ErrUsernameIsRequired = errors.New("username is required")
var ErrPasswordIsRequired = errors.New("password is required")

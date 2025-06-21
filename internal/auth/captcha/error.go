package captcha

import "errors"

var ErrVerificationFailed = errors.New("CAPTCHA verification failed")
var ErrValidationFailed = errors.New("CAPTCHA validation failed")

package captcha

import "errors"

var (
	ErrVerificationFailed = errors.New("CAPTCHA verification failed")
	ErrValidationFailed   = errors.New("CAPTCHA validation failed")
)

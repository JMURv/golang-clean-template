package validation

import "github.com/go-playground/validator/v10"

var V *validator.Validate //nolint:gochecknoglobals

func New() {
	V = validator.New()
}

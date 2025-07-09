package dto

import "github.com/google/uuid"

type TokenRequest struct {
	Token string `json:"token" validate:"required"`
}

type TokenPair struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

type RefreshRequest struct {
	Refresh string `json:"refresh" validate:"required"`
}

type EmailAndPasswordRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Token    string `json:"token"    validate:"required"`
}

type LoginCodeRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
	Token    string `json:"token"    validate:"required"`
}

type CheckLoginCodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  int    `json:"code"  validate:"required"`
}

type CheckEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type CheckEmailResponse struct {
	Exists bool `json:"exists" validate:"required"`
}

type CheckForgotPasswordEmailRequest struct {
	Password string    `json:"password" validate:"required"`
	ID       uuid.UUID `json:"uidb64"   validate:"required"`
	Code     int       `json:"token"    validate:"required"`
}

type SendForgotPasswordEmail struct {
	Email string `json:"email" validate:"required,email"`
	Token string `json:"token" validate:"required"`
}

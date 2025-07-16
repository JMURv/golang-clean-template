package dto

import (
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/google/uuid"
)

type PaginatedUserResponse struct {
	Data        []*md.User `json:"data"`
	Count       int64      `json:"count"`
	TotalPages  int        `json:"totalPages"`
	CurrentPage int        `json:"currentPage"`
	HasNextPage bool       `json:"hasNextPage"`
}

type CreateUserRequest struct {
	Name     string `json:"name"            validate:"required"`
	Email    string `json:"email"           validate:"required,email"`
	Password string `json:"password"        validate:"required"`
	Avatar   string `json:"avatar"`
	IsActive bool   `json:"isActive"`
	IsEmail  bool   `json:"isEmailVerified"`
}

type UpdateUserRequest struct {
	Name     string `json:"name"            validate:"required"`
	Email    string `json:"email"           validate:"required,email"`
	Avatar   string `json:"avatar"`
	IsActive bool   `json:"isActive"`
	IsEmail  bool   `json:"isEmailVerified"`
}

type CreateUserResponse struct {
	ID uuid.UUID `json:"id"`
}

type ExistsUserResponse struct {
	Exists bool `json:"exists"`
}

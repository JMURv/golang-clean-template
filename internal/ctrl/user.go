package ctrl

import (
	"context"
	"errors"
	"fmt"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/dto"
	md "github.com/JMURv/golang-clean-template/internal/models"
	"github.com/JMURv/golang-clean-template/internal/repo"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
)

type userCtrl interface {
	IsUserExist(ctx context.Context, email string) (*dto.ExistsUserResponse, error)
	ListUsers(
		ctx context.Context,
		page, size int,
		filters map[string]any,
	) (*dto.PaginatedUserResponse, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*md.User, error)
	GetUserByEmail(ctx context.Context, email string) (*md.User, error)
	CreateUser(
		ctx context.Context,
		u *dto.CreateUserRequest,
		file *s3.UploadFileRequest,
	) (*dto.CreateUserResponse, error)
	UpdateUser(
		ctx context.Context,
		id uuid.UUID,
		req *dto.UpdateUserRequest,
		file *s3.UploadFileRequest,
	) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

type userRepo interface {
	ListUsers(
		ctx context.Context,
		page, size int,
		filters map[string]any,
	) (*dto.PaginatedUserResponse, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*md.User, error)
	GetUserByEmail(ctx context.Context, email string) (*md.User, error)
	CreateUser(ctx context.Context, req *dto.CreateUserRequest) (uuid.UUID, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req *dto.UpdateUserRequest) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
}

const (
	userCacheKey = "user:%v"
	usersListKey = "users-list:%v:%v:%v"
	userPattern  = "users-*"
)

func (c *Controller) IsUserExist(
	ctx context.Context,
	email string,
) (*dto.ExistsUserResponse, error) {
	const op = "users.IsUserExist.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	res := &dto.ExistsUserResponse{Exists: false}

	_, err := c.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return res, nil
		}
		return nil, err
	}

	res.Exists = true

	return res, nil
}

func (c *Controller) ListUsers(
	ctx context.Context,
	page, size int,
	filters map[string]any,
) (*dto.PaginatedUserResponse, error) {
	const op = "users.ListUsers.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	cached := &dto.PaginatedUserResponse{}
	cacheKey := fmt.Sprintf(usersListKey, page, size, filters)
	if err := c.cache.GetToStruct(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	res, err := c.repo.ListUsers(ctx, page, size, filters)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(res)
	if err == nil {
		c.cache.Set(ctx, config.DefaultCacheTime, cacheKey, bytes)
	}

	return res, nil
}

func (c *Controller) GetUserByID(ctx context.Context, userID uuid.UUID) (*md.User, error) {
	const op = "users.GetUserByID.ctrl"
	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	cached := &md.User{}
	cacheKey := fmt.Sprintf(userCacheKey, userID)
	err := c.cache.GetToStruct(ctx, cacheKey, cached)
	if err == nil {
		return cached, nil
	}

	res, err := c.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	bytes, err := json.Marshal(res)
	if err == nil {
		c.cache.Set(ctx, config.DefaultCacheTime, cacheKey, bytes)
	}

	return res, nil
}

func (c *Controller) GetUserByEmail(ctx context.Context, email string) (*md.User, error) {
	const op = "users.GetUserByEmail.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	cached := &md.User{}
	cacheKey := fmt.Sprintf(userCacheKey, email)

	err := c.cache.GetToStruct(ctx, cacheKey, cached)
	if err == nil {
		return cached, nil
	}

	res, err := c.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	bytes, err := json.Marshal(res)
	if err == nil {
		c.cache.Set(ctx, config.DefaultCacheTime, cacheKey, bytes)
	}

	return res, nil
}

func (c *Controller) CreateUser(
	ctx context.Context,
	u *dto.CreateUserRequest,
	file *s3.UploadFileRequest,
) (*dto.CreateUserResponse, error) {
	const op = "users.CreateUser.ctrl"
	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	var err error

	u.Password, err = c.au.Hash(u.Password)
	if err != nil {
		return nil, err
	}

	if file != nil && len(file.File) > 0 {
		url, err := c.s3.UploadFile(ctx, file)
		if err != nil {
			return nil, err
		}

		u.Avatar = url
	}

	id, err := c.repo.CreateUser(ctx, u)
	if err != nil {
		if errors.Is(err, repo.ErrAlreadyExists) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	go c.cache.InvalidateKeysByPattern(ctx, userPattern)

	return &dto.CreateUserResponse{
		ID: id,
	}, nil
}

func (c *Controller) UpdateUser(
	ctx context.Context,
	id uuid.UUID,
	req *dto.UpdateUserRequest,
	file *s3.UploadFileRequest,
) error {
	const op = "users.UpdateUser.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	if file != nil && len(file.File) > 0 {
		url, err := c.s3.UploadFile(ctx, file)
		if err != nil {
			return err
		}

		req.Avatar = url
	}

	err := c.repo.UpdateUser(ctx, id, req)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	c.cache.Delete(ctx, fmt.Sprintf(userCacheKey, id))

	go c.cache.InvalidateKeysByPattern(ctx, userPattern)

	return nil
}

func (c *Controller) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	const op = "users.DeleteUser.ctrl"

	span, ctx := opentracing.StartSpanFromContext(ctx, op)
	defer span.Finish()

	err := c.repo.DeleteUser(ctx, userID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrNotFound
		}
		return err
	}

	c.cache.Delete(ctx, fmt.Sprintf(userCacheKey, userID))

	go c.cache.InvalidateKeysByPattern(ctx, userPattern)

	return nil
}

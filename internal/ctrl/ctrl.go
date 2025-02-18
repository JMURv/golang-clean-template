package ctrl

import (
	"context"
	"io"
	"time"
)

type AppRepo interface{}

type AppCtrl interface{}

type CacheService interface {
	io.Closer
	GetToStruct(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, t time.Duration, key string, val any)
	Delete(ctx context.Context, key string)
	InvalidateKeysByPattern(ctx context.Context, pattern string)
}

type Controller struct {
	repo  AppRepo
	cache CacheService
}

func New(repo AppRepo, cache CacheService) *Controller {
	return &Controller{
		repo:  repo,
		cache: cache,
	}
}

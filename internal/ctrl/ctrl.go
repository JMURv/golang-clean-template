package ctrl

import (
	"context"
	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"io"
	"time"
)

type AppRepo interface{}

type AppCtrl interface{}

type S3Service interface {
	UploadFile(ctx context.Context, req *s3.UploadFileRequest) (string, error)
}

type CacheService interface {
	io.Closer
	GetToStruct(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, t time.Duration, key string, val any)
	Delete(ctx context.Context, key string)
	InvalidateKeysByPattern(ctx context.Context, pattern string)
}

type EmailService interface{}

type Controller struct {
	au    auth.Core
	repo  AppRepo
	cache CacheService
	s3    S3Service
	smtp  EmailService
}

func New(au auth.Core, repo AppRepo, cache CacheService, s3 S3Service, smtp EmailService) *Controller {
	return &Controller{
		au:    au,
		repo:  repo,
		cache: cache,
		s3:    s3,
		smtp:  smtp,
	}
}

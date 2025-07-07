package s3

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type S3 struct {
	cli    *minio.Client
	bucket string
}

func New(conf config.Config) *S3 {
	ctx := context.Background()
	client, err := minio.New(
		conf.Minio.Addr, &minio.Options{
			Creds:  credentials.NewStaticV4(conf.Minio.AccessKey, conf.Minio.SecretKey, ""),
			Secure: conf.Minio.UseSSL,
		},
	)
	if err != nil {
		zap.L().Fatal("failed to create MinIO client", zap.Error(err))
	}

	exists, err := client.BucketExists(ctx, conf.Minio.Bucket)
	if err != nil {
		zap.L().Fatal("failed to check bucket existence", zap.Error(err))
	}

	if !exists {
		err = client.MakeBucket(ctx, conf.Minio.Bucket, minio.MakeBucketOptions{})
		if err != nil {
			zap.L().Fatal("failed to create bucket", zap.Error(err))
		}
	}

	err = client.SetBucketPolicy(
		ctx, conf.Minio.Bucket, fmt.Sprintf(
			`{
				"Version": "2012-10-17",
				"Statement": [{
					"Effect": "Allow",
					"Principal": {"AWS": ["*"]},
					"Action": ["s3:GetObject"],
					"Resource": ["arn:aws:s3:::%s/*"]
				}]
			}`,
			conf.Minio.Bucket,
		),
	)
	if err != nil {
		zap.L().Fatal("failed to set bucket policy", zap.Error(err))
	}

	zap.L().Info("MinIO connection established", zap.String("addr", conf.Minio.Addr))
	return &S3{
		cli:    client,
		bucket: conf.Minio.Bucket,
	}
}

type UploadFileRequest struct {
	File        []byte
	Filename    string
	ContentType string
}

func (s *S3) UploadFile(ctx context.Context, req *UploadFileRequest) (string, error) {
	uniqueName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), req.Filename)

	_, err := s.cli.PutObject(
		ctx,
		s.bucket,
		uniqueName,
		bytes.NewReader(req.File),
		int64(len(req.File)),
		minio.PutObjectOptions{
			ContentType:  req.ContentType,
			UserMetadata: map[string]string{"x-amz-acl": "public-read"},
		},
	)
	if err != nil {
		zap.L().Error("[S3] failed to upload file", zap.Error(err))
		return "", ErrFailedToUploadFile
	}

	return fmt.Sprintf("s3/%s/%s", s.bucket, uniqueName), nil
}

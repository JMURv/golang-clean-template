package redis

import (
	"context"
	"encoding/json"
	"github.com/JMURv/golang-clean-template/internal/cache"
	cfg "github.com/JMURv/golang-clean-template/internal/config"
	"github.com/go-redis/redis/v8"
	ot "github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
	"log"
	"time"
)

type Cache struct {
	cli *redis.Client
}

func New(conf *cfg.RedisConfig) *Cache {
	cli := redis.NewClient(
		&redis.Options{
			Addr:     conf.Addr,
			Password: conf.Pass,
			DB:       0,
		},
	)

	_, err := cli.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	return &Cache{cli: cli}
}

func (c *Cache) Close() error {
	return c.cli.Close()
}

func (c *Cache) GetToStruct(ctx context.Context, key string, dest any) error {
	const op = "cache.GetToStruct"
	span, ctx := ot.StartSpanFromContext(ctx, op)
	defer span.Finish()

	val, err := c.cli.Get(ctx, key).Bytes()
	if err == redis.Nil {
		zap.L().Debug(
			cache.ErrNotFoundInCache.Error(),
			zap.String("op", op), zap.String("key", key),
		)
		return cache.ErrNotFoundInCache
	} else if err != nil {
		span.SetTag("error", true)
		zap.L().Debug(
			"failed to get from cache",
			zap.String("op", op), zap.String("key", key),
			zap.Error(err),
		)
		return err
	}

	if err = json.Unmarshal(val, dest); err != nil {
		span.SetTag("error", true)
		zap.L().Debug(
			"failed to unmarshal",
			zap.String("op", op),
			zap.String("key", key), zap.Any("dest", dest),
			zap.Error(err),
		)
		return err
	}

	zap.L().Debug("cache hit", zap.String("key", key))
	return nil
}

func (c *Cache) Set(ctx context.Context, t time.Duration, key string, val any) {
	const op = "SetToCache"
	span, ctx := ot.StartSpanFromContext(ctx, op)
	defer span.Finish()

	if err := c.cli.Set(ctx, key, val, t).Err(); err != nil {
		span.SetTag("error", true)
		zap.L().Debug(
			"failed to set to cache",
			zap.String("op", op),
			zap.String("t", t.String()), zap.String("key", key), zap.Any("val", val),
			zap.Error(err),
		)
		return
	}

	zap.L().Debug(
		"successfully set to cache",
		zap.String("key", key),
	)
	return
}

func (c *Cache) Delete(ctx context.Context, key string) {
	const op = "cache.Delete"
	span, ctx := ot.StartSpanFromContext(ctx, op)
	defer span.Finish()

	if err := c.cli.Del(ctx, key).Err(); err != nil {
		span.SetTag("error", true)
		zap.L().Debug(
			"failed to delete from cache",
			zap.String("op", op),
			zap.String("key", key),
			zap.Error(err),
		)
		return
	}
	return
}

func (c *Cache) InvalidateKeysByPattern(ctx context.Context, pattern string) {
	var cursor uint64
	for {
		var err error
		var keys []string

		keys, cursor, err = c.cli.Scan(ctx, cursor, pattern, 100).Result() // 100 keys at a time
		if err != nil {
			zap.L().Debug("failed to scan redis", zap.Error(err))
			break
		}

		if len(keys) > 0 {
			if err = c.cli.Del(ctx, keys...).Err(); err != nil {
				zap.L().Debug("failed to delete keys", zap.Error(err))
				break
			}
		}

		if cursor == 0 {
			break
		}
	}
}

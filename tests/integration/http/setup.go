package http

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JMURv/golang-clean-template/internal/auth"
	"github.com/JMURv/golang-clean-template/internal/cache/redis"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/JMURv/golang-clean-template/internal/ctrl"
	hdl "github.com/JMURv/golang-clean-template/internal/hdl/http"
	"github.com/JMURv/golang-clean-template/internal/repo/db"
	"github.com/JMURv/golang-clean-template/internal/repo/s3"
	"github.com/JMURv/golang-clean-template/internal/smtp"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

const getTables = `
SELECT tablename 
FROM pg_tables 
WHERE schemaname = 'public';
`

var rootDir = filepath.Join("..", "..", "..")

var conf config.Config

var (
	redisC testcontainers.Container
	minioC testcontainers.Container
	pgC    testcontainers.Container
)

func getRedis() testcontainers.Container {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "redis:alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				"6379/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "6379",
					},
				},
			}
		},
	}

	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	return redisC
}

func getPostgres() testcontainers.Container {
	ctx := context.Background()
	pgPort := os.Getenv("POSTGRES_PORT")
	pgPortC := fmt.Sprintf("%s/tcp", pgPort)

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17.4-alpine",
		WaitingFor:   wait.ForHealthCheck(),
		ExposedPorts: []string{pgPortC},
		ConfigModifier: func(conf *container.Config) {
			conf.Healthcheck = &container.HealthConfig{
				Test: []string{
					"CMD-SHELL",
					fmt.Sprintf(
						"pg_isready -U %s -d %s",
						os.Getenv("POSTGRES_USER"),
						os.Getenv("POSTGRES_DB"),
					),
				},
				Interval:    5 * time.Second,
				Timeout:     2 * time.Second,
				Retries:     5,
				StartPeriod: 2 * time.Second,
			}
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				nat.Port(pgPortC): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: pgPort,
					},
				},
			}
		},
		Env: map[string]string{
			"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
			"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
			"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
			"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
			"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		},
	}

	pgC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	return pgC
}

func getMinio() testcontainers.Container {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "minio/minio:RELEASE.2025-06-13T11-33-47Z",
		Cmd:   []string{"server", "/data", "--console-address", ":9001"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("9000/tcp"),
			wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp"),
		),
		ExposedPorts: []string{"9000/tcp", "9001/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":            os.Getenv("MINIO_ROOT_USER"),
			"MINIO_ROOT_PASSWORD":        os.Getenv("MINIO_ROOT_PASSWORD"),
			"MINIO_PROMETHEUS_AUTH_TYPE": "public",
		},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				"9000/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "9000",
					},
				},
				"9001/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: "9001",
					},
				},
			}
		},
	}

	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		panic(err)
	}

	return minioC
}

func setupTestServer() (*httptest.Server, func(t *testing.T)) {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))

	au := auth.New(conf)
	cache := redis.New(conf)
	repo := db.New(conf)
	svc := ctrl.New(au, repo, cache, s3.New(conf), smtp.New(conf))
	h := hdl.New(au, svc)

	ts := httptest.NewServer(h.Router)

	cleanupFunc := func(t *testing.T) {
		ctx := context.Background()
		ts.Close()

		conn, err := pgx.Connect(ctx, fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			conf.DB.User,
			conf.DB.Password,
			conf.DB.Host,
			conf.DB.Port,
			conf.DB.Database,
		))
		if err != nil {
			zap.L().Fatal("Failed to connect to the database", zap.Error(err))
		}

		if err = conn.Ping(ctx); err != nil {
			zap.L().Fatal("Failed to ping the database", zap.Error(err))
		}

		rows, err := conn.Query(ctx, getTables)
		if err != nil {
			zap.L().Fatal("Failed to fetch table names", zap.Error(err))
		}
		defer func(rows pgx.Rows) {
			rows.Close()
		}(rows)

		var tables []string
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				zap.L().Fatal("Failed to scan table name", zap.Error(err))
			}
			tables = append(tables, name)
		}

		if len(tables) == 0 {
			return
		}

		_, err = conn.Exec(
			ctx,
			fmt.Sprintf("TRUNCATE TABLE %v RESTART IDENTITY CASCADE;", strings.Join(tables, ", ")),
		)
		if err != nil {
			zap.L().Fatal("Failed to truncate tables", zap.Error(err))
		}
	}

	return ts, cleanupFunc
}

func init() {
	conf = config.MustLoad(
		filepath.ToSlash(
			filepath.Join(rootDir, "config", ".env.integration"),
		),
	)

	_ = os.Setenv("MIGRATIONS_PATH", filepath.ToSlash(
		filepath.Join(rootDir, "internal", "repo", "db", "migration"),
	))

	wg := &sync.WaitGroup{}
	wg.Add(3)

	go func() {
		redisC = getRedis()
		wg.Done()
	}()

	go func() {
		minioC = getMinio()
		wg.Done()
	}()

	go func() {
		pgC = getPostgres()
		wg.Done()
	}()

	wg.Wait()
}

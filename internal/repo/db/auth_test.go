package db

import (
	"context"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:17.4-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = pgContainer.Terminate(ctx)
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := sqlx.Open("pgx", dsn)
	if err != nil {
		zap.L().Fatal("failed to connect to the database", zap.Error(err))
	}

	err = goose.SetDialect("postgres")
	if err != nil {
		t.Fatal(err)
	}

	err = goose.Up(conn.DB, "../../../migrations")
	if err != nil {
		t.Fatal(err)
	}

	return conn
}

func newTestRepo(t *testing.T) *Repository {
	return &Repository{
		conn: newTestDB(t),
	}
}

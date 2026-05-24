package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/JMURv/golang-clean-template/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
)

type Repository struct {
	conn *sqlx.DB
}

func New(config config.Config) *Repository {
	conn, err := sqlx.Open(
		"pgx", fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			config.DB.User,
			config.DB.Password,
			config.DB.Host,
			config.DB.Port,
			config.DB.Database,
		),
	)
	if err != nil {
		zap.L().Fatal("failed to connect to the database", zap.Error(err))
	}

	if err = conn.Ping(); err != nil {
		zap.L().Fatal("failed to ping the database", zap.Error(err))
	}

	if err = applyMigrations(conn.DB); err != nil {
		zap.L().Fatal("failed to apply migrations", zap.Error(err))
	}

	mustPrecreate(config, conn.DB)
	return &Repository{conn: conn}
}

func (r *Repository) Close(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		done <- r.conn.Close()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

func applyMigrations(db *sql.DB) error {
	err := goose.SetDialect("postgres")
	if err != nil {
		zap.L().Error("failed to set postgres dialect", zap.Error(err))
		return err
	}

	err = goose.Up(db, "migrations")
	if err != nil {
		return err
	}

	zap.L().Info("migrations applied")
	return nil
}

func mustPrecreate(conf config.Config, db *sql.DB) {}

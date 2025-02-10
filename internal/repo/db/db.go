package db

import (
	"database/sql"
	"errors"
	"fmt"
	conf "github.com/JMURv/golang-clean-template/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
	"path/filepath"
)

type Repository struct {
	conn *sql.DB
}

func New(conf *conf.DBConfig) *Repository {
	conn, err := sql.Open(
		"postgres", fmt.Sprintf(
			"postgres://%s:%s@%s:%d/%s?sslmode=disable",
			conf.User,
			conf.Password,
			conf.Host,
			conf.Port,
			conf.Database,
		),
	)
	if err != nil {
		zap.L().Fatal("Failed to connect to the database", zap.Error(err))
	}

	if err = conn.Ping(); err != nil {
		zap.L().Fatal("Failed to ping the database", zap.Error(err))
	}

	if err = applyMigrations(conn, conf); err != nil {
		zap.L().Fatal("Failed to apply migrations", zap.Error(err))
	}

	return &Repository{conn: conn}
}

func (r *Repository) Close() error {
	return r.conn.Close()
}

func applyMigrations(db *sql.DB, conf *conf.DBConfig) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	path, _ := filepath.Abs("internal/repo/db/migration")
	path = filepath.ToSlash(path)

	m, err := migrate.NewWithDatabaseInstance("file://"+path, conf.Database, driver)
	if err != nil {
		return err
	}

	if err = m.Up(); err != nil && errors.Is(err, migrate.ErrNoChange) {
		zap.L().Info("No migrations to apply")
		return nil
	} else if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}

	zap.L().Info("Applied migrations")
	return nil
}

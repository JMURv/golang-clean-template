package db

import (
	"database/sql"
	"errors"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

func applyMigrations(db *sql.DB, conf config.Config) error {
	driver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return err
	}

	path := os.Getenv("MIGRATIONS_PATH")
	if path == "" {
		path = filepath.ToSlash(
			filepath.Join("internal", "repo", "db", "migration"),
		)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+path, conf.DB.Database, driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			zap.L().Info("No migrations to apply")
			return nil
		} else {
			zap.L().Error("Failed to apply migrations", zap.Error(err))
			return err
		}
	}

	zap.L().Info("Applied migrations")
	return nil
}

func mustPrecreate(conf config.Config, db *sql.DB) {}

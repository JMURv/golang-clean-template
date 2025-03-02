package db

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/JMURv/golang-clean-template/internal/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

func applyMigrations(db *sql.DB, conf config.Config) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	rootDir, err := findRootDir()
	if err != nil {
		return err
	}

	path := filepath.ToSlash(
		filepath.Join(rootDir, "internal", "repo", "db", "migration"),
	)

	m, err := migrate.NewWithDatabaseInstance("file://"+path, conf.DB.Database, driver)
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

func findRootDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		if dir == "/" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "", fmt.Errorf("go.mod not found")
}

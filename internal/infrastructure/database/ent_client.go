package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	entdb "github.com/eslsoft/vocnet/internal/infrastructure/database/ent"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// NewEntClient constructs an ent.Client configured for the application's database.
func NewEntClient(cfg *config.Config) (*entdb.Client, func(), error) {
	driver, err := cfg.DatabaseDriver()
	if err != nil {
		return nil, nil, fmt.Errorf("determine database driver: %w", err)
	}

	dsn, err := cfg.DatabaseURL()
	if err != nil {
		return nil, nil, fmt.Errorf("determine database dsn: %w", err)
	}

	switch driver {
	case "postgres":
		return newPostgresEntClient(cfg, dsn)
	case "sqlite3":
		return newSQLiteEntClient(cfg, dsn)
	default:
		return nil, nil, fmt.Errorf("unsupported database driver %q", driver)
	}
}

func newPostgresEntClient(cfg *config.Config, dsn string) (*entdb.Client, func(), error) {
	rawDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open ent sql db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rawDB.PingContext(ctx); err != nil {
		rawDB.Close()
		return nil, nil, fmt.Errorf("ping ent sql db: %w", err)
	}

	driver := entsql.OpenDB(dialect.Postgres, rawDB)
	client := entdb.NewClient(entdb.Driver(driver))
	if cfg.Database.LogSQL {
		client = client.Debug()
	}

	return client, func() {
		_ = client.Close()
	}, nil
}

func newSQLiteEntClient(cfg *config.Config, dsn string) (*entdb.Client, func(), error) {
	rawDB, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite db: %w", err)
	}
	rawDB.SetMaxOpenConns(1)
	rawDB.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rawDB.PingContext(ctx); err != nil {
		rawDB.Close()
		return nil, nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if _, err := rawDB.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		rawDB.Close()
		return nil, nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}

	driver := entsql.OpenDB(dialect.SQLite, rawDB)
	client := entdb.NewClient(entdb.Driver(driver))
	if cfg.Database.LogSQL {
		client = client.Debug()
	}

	return client, func() {
		_ = client.Close()
	}, nil
}

package database

import (
	"fmt"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/eslsoft/vocnet/internal/infrastructure/database/ent"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// NewEntClient constructs an ent.Client configured for the application's database.
func NewEntClient(cfg *config.Config) (*ent.Client, func(), error) {
	driver, err := cfg.DatabaseDriver()
	if err != nil {
		return nil, nil, fmt.Errorf("determine database driver: %w", err)
	}

	dsn, err := cfg.DatabaseURL()
	if err != nil {
		return nil, nil, fmt.Errorf("determine database dsn: %w", err)
	}

	client, err := ent.Open(driver, dsn, ent.Debug())
	return client, func() { client.Close() }, err
}

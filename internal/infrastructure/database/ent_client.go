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
)

// NewEntClient constructs an ent.Client configured for the application's database.
func NewEntClient(cfg *config.Config) (*entdb.Client, func(), error) {
	rawDB, err := sql.Open("postgres", cfg.DatabaseURL())
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

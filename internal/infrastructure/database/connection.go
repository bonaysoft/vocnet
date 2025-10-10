package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

// NewConnection creates a new pgx connection pool
func NewConnection(cfg *config.Config) (*pgxpool.Pool, func(), error) {
	if cfg.DatabaseDriver() != "postgres" {
		return nil, nil, fmt.Errorf("连接池仅支持 PostgreSQL，当前驱动: %s", cfg.DatabaseDriver())
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		return nil, nil, fmt.Errorf("parse pool config: %w", err)
	}
	poolCfg.MaxConns = 10

	if cfg.Database.LogSQL {
		logger := log.New(log.Writer(), "pgx ", log.LstdFlags|log.Lmicroseconds)
		poolCfg.ConnConfig.Tracer = &tracelog.TraceLog{
			Logger: tracelog.LoggerFunc(func(_ context.Context, lvl tracelog.LogLevel, msg string, data map[string]any) {
				logger.Printf("level=%s msg=%s data=%v", lvl, msg, data)
			}),
			LogLevel: tracelog.LogLevelTrace,
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, pool.Close, fmt.Errorf("ping db: %w", err)
	}

	return pool, pool.Close, nil
}

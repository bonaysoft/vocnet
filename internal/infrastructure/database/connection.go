package database

import (
	"database/sql"
	"fmt"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	_ "github.com/mattn/go-sqlite3"
)

// NewConnection creates a new database connection
func NewConnection(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./rockd.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1) // SQLite works best with a single connection
	db.SetMaxIdleConns(1)

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

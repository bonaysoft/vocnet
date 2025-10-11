package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for our application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host     string `mapstructure:"host"`
	GRPCPort int    `mapstructure:"grpc_port"`
	HTTPPort int    `mapstructure:"http_port"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	DSN    string `mapstructure:"dsn"`
	LogSQL bool   `mapstructure:"log_sql"`

	driver      string
	initialized bool
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set default values
	setDefaults()

	if err := bindEnvAliases(); err != nil {
		return nil, fmt.Errorf("bind env aliases: %w", err)
	}

	// Enable reading from environment variables
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := config.Database.ensureInitialized(); err != nil {
		return nil, fmt.Errorf("validate database config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.http_port", 8080)

	// Database defaults
	viper.SetDefault("database.dsn", "file:./data/vocnet.db")
	viper.SetDefault("database.log_sql", false)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
}

func bindEnvAliases() error {
	bindings := map[string][]string{
		"database.dsn":     {"DB_DSN", "DB_URL"},
		"database.log_sql": {"DB_LOG_SQL"},
	}

	for key, envs := range bindings {
		if len(envs) == 0 {
			if err := viper.BindEnv(key); err != nil {
				return err
			}
			continue
		}
		if err := viper.BindEnv(append([]string{key}, envs...)...); err != nil {
			return err
		}
	}
	return nil
}

// DatabaseURL returns the configured database DSN.
func (c *Config) DatabaseURL() (string, error) {
	return c.Database.databaseURL()
}

// DatabaseDriver returns the normalized database driver identifier.
func (c *Config) DatabaseDriver() (string, error) {
	return c.Database.normalizedDriver()
}

func (db *DatabaseConfig) normalizedDriver() (string, error) {
	if err := db.ensureInitialized(); err != nil {
		return "", err
	}
	return db.driver, nil
}

func (db *DatabaseConfig) databaseURL() (string, error) {
	if err := db.ensureInitialized(); err != nil {
		return "", err
	}
	return db.DSN, nil
}

func (db *DatabaseConfig) sqliteDSN(dsn string) string {
	if dsn == "" {
		dsn = "file:./data/vocnet.db"
	}
	if strings.HasPrefix(dsn, "file:") {
		return ensureSQLiteDSNParams(dsn)
	}
	if strings.Contains(dsn, "://") {
		return ensureSQLiteDSNParams(dsn)
	}
	return ensureSQLiteDSNParams("file:" + dsn)
}

func ensureSQLiteDSNParams(base string) string {
	params := []string{}
	if !strings.Contains(base, "_fk=") {
		params = append(params, "_fk=1")
	}
	if !strings.Contains(base, "_busy_timeout") {
		params = append(params, "_busy_timeout=5000")
	}
	if !strings.Contains(base, "_journal") {
		params = append(params, "_journal=WAL")
	}
	if len(params) == 0 {
		return base
	}

	var builder strings.Builder
	builder.WriteString(base)
	switch {
	case strings.HasSuffix(base, "?"), strings.HasSuffix(base, "&"):
		// no extra separator needed
	case strings.Contains(base, "?"):
		builder.WriteString("&")
	default:
		builder.WriteString("?")
	}
	builder.WriteString(strings.Join(params, "&"))
	return builder.String()
}

func (db *DatabaseConfig) ensureInitialized() error {
	if db.initialized {
		return nil
	}

	dsn := strings.TrimSpace(db.DSN)
	if dsn == "" {
		return fmt.Errorf("database dsn is required")
	}
	driver, err := driverFromDSN(dsn)
	if err != nil {
		return err
	}
	switch driver {
	case "sqlite3":
		dsn = db.sqliteDSN(dsn)
	case "postgres":
		// keep DSN as-is for postgres
	default:
		return fmt.Errorf("unsupported database driver %q", driver)
	}

	db.DSN = dsn
	db.driver = driver
	db.initialized = true
	return nil
}

func driverFromDSN(dsn string) (string, error) {
	dsn = strings.TrimSpace(strings.ToLower(dsn))
	switch {
	case dsn == "":
		return "", fmt.Errorf("database dsn is empty")
	case strings.HasPrefix(dsn, "postgres://"),
		strings.HasPrefix(dsn, "postgresql://"),
		strings.HasPrefix(dsn, "postgresql+unix://"):
		return "postgres", nil
	case strings.HasPrefix(dsn, "file:"),
		strings.HasPrefix(dsn, "sqlite://"),
		strings.HasPrefix(dsn, "sqlite3://"):
		return "sqlite3", nil
	}

	if strings.Contains(dsn, "=") {
		switch {
		case strings.Contains(dsn, "host="),
			strings.Contains(dsn, "dbname="),
			strings.Contains(dsn, "user="):
			return "postgres", nil
		}
	}

	if !strings.Contains(dsn, "://") {
		switch {
		case strings.HasSuffix(dsn, ".db"),
			strings.HasSuffix(dsn, ".sqlite"),
			strings.HasSuffix(dsn, ".sqlite3"),
			strings.HasPrefix(dsn, "./"),
			strings.HasPrefix(dsn, "/"):
			return "sqlite3", nil
		}
	}

	return "", fmt.Errorf("unable to determine database driver from DSN %q", dsn)
}

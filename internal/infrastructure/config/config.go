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
	Driver   string `mapstructure:"driver"`
	DSN      string `mapstructure:"dsn"`
	Path     string `mapstructure:"path"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslmode"`
	LogSQL   bool   `mapstructure:"log_sql"`
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

	return &config, nil
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.grpc_port", 9090)
	viper.SetDefault("server.http_port", 8080)

	// Database defaults
	viper.SetDefault("database.driver", "sqlite3")
	viper.SetDefault("database.dsn", "")
	viper.SetDefault("database.path", "./data/vocnet.db")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.name", "rockd")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.log_sql", true)

	// Log defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
}

func bindEnvAliases() error {
	bindings := map[string][]string{
		"database.driver":   {"DB_DRIVER"},
		"database.dsn":      {"DB_DSN", "DB_URL"},
		"database.path":     {"DB_PATH"},
		"database.host":     {"DB_HOST"},
		"database.port":     {"DB_PORT"},
		"database.name":     {"DB_NAME"},
		"database.user":     {"DB_USER"},
		"database.password": {"DB_PASSWORD"},
		"database.sslmode":  {"DB_SSLMODE"},
		"database.log_sql":  {"DB_LOG_SQL"},
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
func (c *Config) DatabaseURL() string {
	switch c.Database.normalizedDriver() {
	case "postgres":
		return c.Database.postgresDSN()
	default:
		return c.Database.sqliteDSN()
	}
}

// DatabaseDriver returns the normalized database driver identifier.
func (c *Config) DatabaseDriver() string {
	return c.Database.normalizedDriver()
}

func (db DatabaseConfig) normalizedDriver() string {
	switch strings.ToLower(strings.TrimSpace(db.Driver)) {
	case "", "sqlite", "sqlite3":
		return "sqlite3"
	case "postgresql", "postgres":
		return "postgres"
	default:
		return strings.ToLower(strings.TrimSpace(db.Driver))
	}
}

func (db DatabaseConfig) sqliteDSN() string {
	if dsn := strings.TrimSpace(db.DSN); dsn != "" {
		return ensureSQLiteDSNParams(dsn)
	}

	path := strings.TrimSpace(db.Path)
	if strings.HasPrefix(path, "file:") {
		return ensureSQLiteDSNParams(path)
	}
	return ensureSQLiteDSNParams("file:" + path)
}

func (db DatabaseConfig) postgresDSN() string {
	if dsn := strings.TrimSpace(db.DSN); dsn != "" {
		return dsn
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.User,
		db.Password,
		db.Host,
		db.Port,
		db.Name,
		db.SSLMode,
	)
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

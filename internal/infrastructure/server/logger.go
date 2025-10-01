package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/sirupsen/logrus"
)

// InterceptorLogger adapts slog logger to interceptor logger.
// This code is simple enough to be copied and not imported.
func InterceptorLogger() logging.Logger {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		logger.Log(ctx, slog.Level(lvl), msg, fields...)
	})
}

// NewLogger builds a configured logrus logger from application config.
func NewLogger(cfg *config.Config) (*logrus.Logger, error) {
	logger := logrus.New()
	level, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}
	logger.SetLevel(level)
	if cfg.Log.Format == "text" {
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	}
	return logger, nil
}

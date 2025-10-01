package app

import (
	"github.com/eslsoft/vocnet/internal/infrastructure/server"
	"github.com/sirupsen/logrus"
)

// Container aggregates the application dependencies produced by Wire.
type Container struct {
	Logger *logrus.Logger
	Server *server.Server
}

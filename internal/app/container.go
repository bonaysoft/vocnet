package app

import (
	entdb "github.com/eslsoft/vocnet/internal/infrastructure/database/ent"
	"github.com/eslsoft/vocnet/internal/infrastructure/server"
	"github.com/sirupsen/logrus"
)

// Container aggregates the application dependencies produced by Wire.
type Container struct {
	Logger    *logrus.Logger
	Server    *server.Server
	EntClient *entdb.Client
}

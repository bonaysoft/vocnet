//go:build wireinject
// +build wireinject

package app

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"

	adaptergrpc "github.com/eslsoft/vocnet/internal/adapter/grpc"
	"github.com/eslsoft/vocnet/internal/adapter/repository"
	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	"github.com/eslsoft/vocnet/internal/infrastructure/database"
	dbpkg "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/eslsoft/vocnet/internal/infrastructure/server"
	"github.com/eslsoft/vocnet/internal/usecase"

	"github.com/eslsoft/vocnet/api/gen/dict/v1/dictv1connect"
	"github.com/eslsoft/vocnet/api/gen/vocnet/v1/vocnetv1connect"
)

var configSet = wire.NewSet(
	config.Load,
)

var databaseSet = wire.NewSet(
	database.NewConnection,
	wire.Bind(new(dbpkg.DBTX), new(*pgxpool.Pool)),
	dbpkg.New,
)

var repositorySet = wire.NewSet(
	repository.NewVocRepository,
	repository.NewUserWordRepository,
)

var usecaseSet = wire.NewSet(
	usecase.NewWordUsecase,
	usecase.NewUserWordUsecase,
)

var serviceSet = wire.NewSet(
	adaptergrpc.NewWordServiceServer,
	adaptergrpc.NewUserWordServiceServer,
	wire.Bind(new(vocnetv1connect.UserWordServiceHandler), new(*adaptergrpc.UserWordServiceServer)),
	wire.Bind(new(dictv1connect.WordServiceHandler), new(*adaptergrpc.WordServiceServer)),
)

var serverSet = wire.NewSet(
	server.NewLogger,
	server.NewServer,
)

// Initialize builds the application container using Wire.
func Initialize() (*Container, func(), error) {
	wire.Build(
		configSet,
		databaseSet,
		repositorySet,
		usecaseSet,
		serviceSet,
		serverSet,
		wire.Struct(new(Container), "Logger", "Server"),
	)
	return nil, nil, nil
}

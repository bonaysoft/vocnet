package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	commonv1 "github.com/eslsoft/vocnet/api/gen/common/v1"
	dictv1 "github.com/eslsoft/vocnet/api/gen/dict/v1"
	adaptergrpc "github.com/eslsoft/vocnet/internal/adapter/grpc"
	"github.com/eslsoft/vocnet/internal/adapter/repository"
	"github.com/eslsoft/vocnet/internal/infrastructure/config"
	dbpkg "github.com/eslsoft/vocnet/internal/infrastructure/database/db"
	"github.com/eslsoft/vocnet/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// Server represents the application server
type Server struct {
	config     *config.Config
	grpcServer *grpc.Server
	httpServer *http.Server
	logger     *logrus.Logger
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger *logrus.Logger, pool *pgxpool.Pool) *Server {
	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create gRPC-Gateway mux
	mux := runtime.NewServeMux()

	// Setup dependencies for VocService
	queries := dbpkg.New(pool)
	wordRepo := repository.NewVocRepository(queries)
	wordUC := usecase.NewWordUsecase(wordRepo, "en")
	wordSvc := adaptergrpc.NewWordServiceServer(wordUC)
	dictv1.RegisterWordServiceServer(grpcServer, wordSvc)

	// Register gateway handler for WordService
	ctx := context.Background()
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	// We assume same host different port for grpc
	endpoint := fmt.Sprintf("localhost:%d", cfg.Server.GRPCPort)
	_ = commonv1.Language_LANGUAGE_UNSPECIFIED // reference to keep imported commonv1 (maybe unused otherwise)
	if err := dictv1.RegisterWordServiceHandlerFromEndpoint(ctx, mux, endpoint, dialOpts); err != nil {
		logger.Errorf("failed to register voc service handler: %v", err)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler: mux,
	}

	return &Server{
		config:     cfg,
		grpcServer: grpcServer,
		httpServer: httpServer,
		logger:     logger,
	}
}

// StartGRPC starts the gRPC server
func (s *Server) StartGRPC() error {
	addr := fmt.Sprintf(":%d", s.config.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.logger.Infof("gRPC server starting on %s", addr)

	if err := s.grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// StartHTTP starts the HTTP gateway server
func (s *Server) StartHTTP() error {
	// Register gRPC-Gateway handlers

	s.logger.Infof("HTTP server starting on %s", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve HTTP: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Errorf("Failed to shutdown HTTP server: %v", err)
	}

	// Shutdown gRPC server
	s.grpcServer.GracefulStop()

	s.logger.Info("Server shutdown complete")
	return nil
}

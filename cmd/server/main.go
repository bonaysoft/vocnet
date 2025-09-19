package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eslsoft/vocnet/internal/infrastructure/config"
)

func main() {
	// Setup logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting rockd server...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server will run on gRPC port %d and HTTP port %d", cfg.Server.GRPCPort, cfg.Server.HTTPPort)

	// Simple HTTP server for testing
	go func() {
		log.Printf("Starting simple HTTP server on port %d", cfg.Server.HTTPPort)
		// TODO: Add proper HTTP server implementation
		select {}
	}()

	// Simple gRPC server for testing
	go func() {
		log.Printf("Starting simple gRPC server on port %d", cfg.Server.GRPCPort)
		// TODO: Add proper gRPC server implementation
		select {}
	}()

	log.Println("Servers started successfully")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		log.Println("Shutdown timeout exceeded")
	default:
		log.Println("Server shutdown complete")
	}
}

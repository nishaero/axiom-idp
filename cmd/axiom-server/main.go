package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/axiom-idp/axiom/internal/logging"
	"github.com/axiom-idp/axiom/internal/server"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Load configuration
	cfg := config.NewConfig()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logger := logging.NewLogger(cfg.LogLevel)
	logger.WithFields(logrus.Fields{
		"version":    Version,
		"buildTime":  BuildTime,
		"environment": cfg.Environment,
	}).Info("Starting Axiom IDP server")

	// Initialize server
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to create server: %v", err)
	}

	// Start server in background
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		logger.Infof("Server listening on %s", addr)
		if err := srv.Start(addr); err != nil {
			logger.Errorf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Shutdown gracefully
	logger.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.MCPTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server shutdown error: %v", err)
	}

	logger.Info("Server stopped")
}

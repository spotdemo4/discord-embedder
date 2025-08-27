package app

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	slogctx "github.com/veqryn/slog-context"
)

type App struct {
	Config

	Logger *slog.Logger
}

func New() (*App, error) {
	slogHandler := slog.NewTextHandler(os.Stdout, nil)
	slogctxHandler := slogctx.NewHandler(slogHandler, nil)
	logger := slog.New(slogctxHandler)

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		logger.Info("Failed to load .env file, using environment variables")
	}

	// Parse config
	cfg, err := config(logger)
	if err != nil {
		return nil, err
	}

	return &App{
		Config: *cfg,
		Logger: logger,
	}, nil
}

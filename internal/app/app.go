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

	// Log config (without sensitive info)
	logger.Info("config",
		"discord_application_id", cfg.DiscordApplicationID,
		"discord_channel_ids", cfg.DiscordChannelIDs,
		"files_dir", cfg.FilesDir,
		"temp_dir", cfg.TempDir,
		"host", cfg.Host,
		"quicksync", cfg.Quicksync,
	)

	return &App{
		Config: *cfg,
		Logger: logger,
	}, nil
}

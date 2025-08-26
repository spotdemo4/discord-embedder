package app

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type App struct {
	Logger               *slog.Logger
	DiscordToken         string
	DiscordApplicationID string
	FilesDir             string
	Host                 string
	Quicksync            bool
}

func New() (*App, error) {
	logger := slog.Default()

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		logger.Info("Failed to load .env file, using environment variables")
	}

	// Get env
	discordToken := os.Getenv("DISCORD_TOKEN")
	if discordToken == "" {
		return nil, errors.New("env DISCORD_TOKEN not set")
	}

	discordApplicationID := os.Getenv("DISCORD_APPLICATION_ID")
	if discordApplicationID == "" {
		return nil, errors.New("env DISCORD_APPLICATION_ID not set")
	}

	filesDir := os.Getenv("FILES_DIR")
	if filesDir == "" {
		filesDir = "./files"
	}

	// Create file path if it doesn't exist
	if _, err = os.Stat(filesDir); os.IsNotExist(err) {
		if err = os.MkdirAll(filesDir, 0750); err != nil {
			logger.Error("could not create file path", "path", filesDir, "error", err)
			return nil, err
		}
	}
	logger.Info("using file path", "path", filesDir)

	host := os.Getenv("HOST")
	if host == "" {
		return nil, errors.New("env HOST not set")
	}

	quicksync := os.Getenv("QUICKSYNC") == "true"

	return &App{
		Logger:               logger,
		DiscordToken:         discordToken,
		DiscordApplicationID: discordApplicationID,
		FilesDir:             filesDir,
		Host:                 host,
		Quicksync:            quicksync,
	}, nil
}

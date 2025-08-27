package app

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	DiscordToken         string   `env:"DISCORD_TOKEN,required"`
	DiscordApplicationID string   `env:"DISCORD_APPLICATION_ID,required"`
	DiscordChannelIDs    []string `env:"DISCORD_CHANNEL_IDS"             envSeparator:","`
	FilesDir             string   `env:"FILES_DIR"`
	TempDir              string   `env:"TMP_DIR"`
	Host                 string   `env:"HOST,required"`
	Port                 int      `env:"PORT"                                             envDefault:"8080"`
	Quicksync            bool     `env:"QUICKSYNC"                                        envDefault:"false"`

	// Logins
	RedditUsername    string `env:"REDDIT_USERNAME"`
	RedditPassword    string `env:"REDDIT_PASSWORD"`
	TikTokUsername    string `env:"TIKTOK_USERNAME"`
	TikTokPassword    string `env:"TIKTOK_PASSWORD"`
	InstagramUsername string `env:"INSTAGRAM_USERNAME"`
	InstagramPassword string `env:"INSTAGRAM_PASSWORD"`
	XUsername         string `env:"X_USERNAME"`
	XPassword         string `env:"X_PASSWORD"`
}

func config(logger *slog.Logger) (*Config, error) {
	// Parse config from environment variables
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		logger.Error("could not parse env", "error", err)
		return nil, err
	}

	// Validate FilesDir
	if cfg.FilesDir == "" {
		var configDir string
		configDir, err = os.UserConfigDir()
		if err != nil {
			logger.Error("could not get user config dir", "error", err)
			return nil, err
		}

		cfg.FilesDir = filepath.Join(configDir, "discord-embedder", "files")
	}
	if _, err = os.Stat(cfg.FilesDir); os.IsNotExist(err) {
		logger.Info("files directory does not exist, creating it", "path", cfg.FilesDir)

		// nosemgrep // 0700 is required for directory traversal
		if err = os.MkdirAll(cfg.FilesDir, 0700); err != nil {
			logger.Error("could not create files directory", "path", cfg.FilesDir, "error", err)
			return nil, err
		}
	}

	// Validate TempDir
	if cfg.TempDir == "" {
		tempDir := os.TempDir()
		if err != nil {
			logger.Error("could not get os temp dir", "error", err)
			return nil, err
		}

		cfg.TempDir = filepath.Join(tempDir, "discord-embedder")
	}
	if _, err = os.Stat(cfg.TempDir); os.IsNotExist(err) {
		logger.Info("temp directory does not exist, creating it", "path", cfg.TempDir)

		// nosemgrep // 0700 is required for directory traversal
		if err = os.MkdirAll(cfg.TempDir, 0700); err != nil {
			logger.Error("could not create temp directory", "path", cfg.TempDir, "error", err)
			return nil, err
		}
	}

	// Log config (without sensitive info)
	logger.Info("got config",
		"discord_application_id", cfg.DiscordApplicationID,
		"discord_channel_ids", cfg.DiscordChannelIDs,
		"files_dir", cfg.FilesDir,
		"temp_dir", cfg.TempDir,
		"host", cfg.Host,
		"port", cfg.Port,
		"quicksync", cfg.Quicksync,
	)

	return &cfg, nil
}

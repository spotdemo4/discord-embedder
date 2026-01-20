package config

import (
	"context"
	"discord-embedder/internal/logger"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
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
	LogLevel             string   `env:"LOG_LEVEL"                                        envDefault:"info"`

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

func New(ctx context.Context) (*Config, error) {
	log := logger.FromContext(ctx)

	// Load environment variables from .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.InfoContext(ctx, "Failed to load .env file, using environment variables")
	}

	// Parse config from environment variables
	var cfg Config
	err = env.Parse(&cfg)
	if err != nil {
		log.ErrorContext(ctx, "could not parse env", "error", err)
		return nil, err
	}

	// Set default FilesDir
	if cfg.FilesDir == "" {
		var configDir string
		configDir, err = os.UserConfigDir()
		if err != nil {
			log.ErrorContext(ctx, "could not get user config dir", "error", err)
			return nil, err
		}

		cfg.FilesDir = filepath.Join(configDir, "discord-embedder", "files")
	}

	// Set default TempDir
	if cfg.TempDir == "" {
		tempDir := os.TempDir()
		cfg.TempDir = filepath.Join(tempDir, "discord-embedder")
	}

	// Set log level
	switch cfg.LogLevel {
	case "debug":
		log.SetLevel(slog.LevelDebug)
	case "info":
		log.SetLevel(slog.LevelInfo)
	case "warn":
		log.SetLevel(slog.LevelWarn)
	case "error":
		log.SetLevel(slog.LevelError)
	default:
		return nil, err
	}

	// Validate FilesDir
	err = validatePath(ctx, cfg.FilesDir)
	if err != nil {
		return nil, err
	}

	// Validate TempDir
	err = validatePath(ctx, cfg.TempDir)
	if err != nil {
		return nil, err
	}

	// Log config (without sensitive info)
	log.InfoContext(ctx, "got config",
		"discord_application_id", cfg.DiscordApplicationID,
		"discord_channel_ids", cfg.DiscordChannelIDs,
		"files_dir", cfg.FilesDir,
		"temp_dir", cfg.TempDir,
		"host", cfg.Host,
		"port", cfg.Port,
		"quicksync", cfg.Quicksync,
		"log_level", cfg.LogLevel,
	)

	return &cfg, nil
}

func validatePath(ctx context.Context, path string) error {
	log := logger.FromContext(ctx)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.InfoContext(ctx, "directory does not exist, creating it", "path", path)

		// nosemgrep // 0700 is required for directory traversal
		if err = os.MkdirAll(path, 0700); err != nil {
			log.ErrorContext(ctx, "could not create directory", "path", path, "error", err)
			return err
		}
	}

	return nil
}

type configKey struct{}

func WithConfig(ctx context.Context, config *Config) context.Context {
	return context.WithValue(ctx, configKey{}, config)
}

func FromContext(ctx context.Context) *Config {
	if ctx == nil {
		return &Config{}
	}

	if l, ok := ctx.Value(configKey{}).(*Config); ok {
		return l
	}

	return &Config{}
}

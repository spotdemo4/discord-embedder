package main

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/discord"
	"discord-embedder/internal/logger"
	"discord-embedder/internal/web"
	"embed"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

//go:embed templates/home.html
var home embed.FS

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Setup logger
	log := logger.New()
	ctx = logger.WithLogger(ctx, log)

	// Get config
	cfg, err := config.New(ctx)
	if err != nil {
		log.Error("could not load config", "error", err)
		os.Exit(1)
	}
	ctx = config.WithConfig(ctx, cfg)

	// Check if yt-dlp is installed
	if _, err = exec.LookPath("yt-dlp"); err != nil {
		log.Error("yt-dlp is not installed", "error", err)
		os.Exit(1)
	}

	// Check if ffmpeg is installed
	if _, err = exec.LookPath("ffmpeg"); err != nil {
		log.Error("ffmpeg is not installed", "error", err)
		os.Exit(1)
	}

	// Check if ffprobe is installed
	if _, err = exec.LookPath("ffprobe"); err != nil {
		log.Error("ffprobe is not installed", "error", err)
		os.Exit(1)
	}

	// Create Discord connection
	wg.Go(func() {
		err = discord.New(ctx)
		if err != nil {
			log.Error("problem with discord", "error", err)
		}
	})

	// Create web server
	wg.Go(func() {
		err = web.New(ctx, home)
		if err != nil {
			log.Error("problem running web server", "error", err)
		}
	})

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		log.Info("exiting", "signal", sig)

		// Cancel context
		cancel()
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	log.Info("done")
}

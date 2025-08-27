package main

import (
	"context"
	"discord-embedder/internal/app"
	"discord-embedder/internal/discord"
	"discord-embedder/internal/web"
	"embed"
	"log"
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

	a, err := app.New()
	if err != nil {
		log.Fatalf("could not initialize app: %s", err)
	}

	// Check if yt-dlp is installed
	if _, err = exec.LookPath("yt-dlp"); err != nil {
		a.Logger.Error("yt-dlp is not installed", "error", err)
		os.Exit(1)
	}

	// Check if ffmpeg is installed
	if _, err = exec.LookPath("ffmpeg"); err != nil {
		a.Logger.Error("ffmpeg is not installed", "error", err)
		os.Exit(1)
	}

	// Check if ffprobe is installed
	if _, err = exec.LookPath("ffprobe"); err != nil {
		a.Logger.Error("ffprobe is not installed", "error", err)
		os.Exit(1)
	}

	// Create Discord connection
	wg.Add(1) // TODO: replace with wg.Go() when moved to Go 1.25
	go func() {
		err = discord.New(ctx, a)
		if err != nil {
			a.Logger.Error("problem with discord", "error", err)
		}

		wg.Done()
	}()

	// Create web server
	wg.Add(1) // TODO: replace with wg.Go() when moved to Go 1.25
	go func() {
		err = web.New(ctx, a, home)
		if err != nil {
			a.Logger.Error("problem running web server", "error", err)
		}

		wg.Done()
	}()

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		a.Logger.Info("exiting", "signal", sig)

		// Cancel context
		cancel()
	}()

	// Wait for all goroutines to finish
	wg.Wait()
	a.Logger.Info("done")
}

package main

import (
	"context"
	"discord-embedder/internal/app"
	"discord-embedder/internal/discord"
	"discord-embedder/internal/web"
	"embed"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

//go:embed templates/home.html
var home embed.FS

func main() {
	ctx, cancel := context.WithCancel(context.Background())

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

	// Create a new Discord instance
	d, err := discord.New(ctx, a)
	if err != nil {
		a.Logger.Error("could not create discord instance", "error", err)
		os.Exit(1)
	}

	// Add server handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", web.NewHomeHandler(ctx, home, a.Host, a.FilesDir))
	mux.HandleFunc("/files/", web.NewFileHandler(a.FilesDir))

	// Create HTTP server
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	// Gracefully shutdown on SIGINT or SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		a.Logger.Info("exiting", "signal", sig)

		// Cancel context
		cancel()

		// Close discord connection
		err = d.Close()
		if err != nil {
			a.Logger.Warn("could not close session gracefully", "error", err)
		}

		// Close webserver
		tctx, tcancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err = server.Shutdown(tctx); err != nil {
			if err = server.Close(); err != nil {
				a.Logger.Warn("could not close server gracefully", "error", err)
			}
		}
		tcancel()
	}()

	// Start http server
	if err = server.ListenAndServe(); err != nil {
		a.Logger.Error("could not start server", "error", err)
	}
}

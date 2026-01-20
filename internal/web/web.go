package web

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"embed"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const timeout = 5 * time.Second

func New(ctx context.Context, home embed.FS) error {
	cfg := config.FromContext(ctx)
	log := logger.FromContext(ctx)
	done := make(chan bool, 1)

	// Create server handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(ctx, home))
	mux.HandleFunc("/files/", fileHandler(cfg))

	// Create HTTP server
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           mux,
		ReadTimeout:       timeout,
		ReadHeaderTimeout: timeout,
		WriteTimeout:      timeout * 2,
		IdleTimeout:       timeout * 2,
	}

	// Stop server on context cancel
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("could not gracefully shutdown server, closing", "error", err)

			if err = server.Close(); err != nil {
				log.Error("could not close server", "error", err)
			}
		}

		done <- true
	}()

	// Start http server
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	// Wait for server to shutdown
	<-done
	return nil
}

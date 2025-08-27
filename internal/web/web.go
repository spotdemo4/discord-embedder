package web

import (
	"context"
	"discord-embedder/internal/app"
	"embed"
	"errors"
	"net/http"
	"time"
)

func New(ctx context.Context, a *app.App, home embed.FS) error {
	done := make(chan bool, 1)

	// Create server handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler(ctx, a, home))
	mux.HandleFunc("/files/", fileHandler(a))

	// Create HTTP server
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	// Stop server on context cancel
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			a.Logger.Error("could not gracefully shutdown server, closing", "error", err)

			if err = server.Close(); err != nil {
				a.Logger.Error("could not close server", "error", err)
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

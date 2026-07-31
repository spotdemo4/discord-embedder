package web

import (
	"context"
	"discord-embedder/internal/config"
	"discord-embedder/internal/logger"
	"discord-embedder/internal/video"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Page struct {
	ImageURL  string
	VideoURL  string
	SourceURL string
	Width     string
	Height    string
}

func homeHandler(ctx context.Context, home embed.FS) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.FromContext(ctx)

		// Handle request
		switch r.Method {
		case http.MethodGet:
			pathItems := strings.Split(r.URL.Path, "/") // /{id}
			if len(pathItems) < 2 {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			// Get file ID
			id := pathItems[1]

			// Find video file
			file, err := video.Get(ctx, id)
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			// Get resolution
			width, height, err := file.Resolution(ctx)
			if err != nil {
				http.Error(w, "Could not get resolution", http.StatusInternalServerError)
				return
			}

			// Get original URL
			sourceURL, err := file.OriginalURL(ctx)
			if err != nil {
				logger.FromContext(ctx).WarnContext(ctx, "could not get original URL", "video", id, "error", err)
				sourceURL = ""
			}

			// Generate page
			page := Page{
				ImageURL:  fmt.Sprintf("%s/files/%s", cfg.Host, file.Thumbnail.Name),
				VideoURL:  fmt.Sprintf("%s/files/%s", cfg.Host, file.Name),
				SourceURL: sourceURL,
				Width:     width,
				Height:    height,
			}
			t, err := template.ParseFS(home, "templates/home.html")
			if err != nil {
				http.Error(w, "Could not use template", http.StatusInternalServerError)
				return
			}

			// Render template
			if err = t.Execute(w, page); err != nil {
				http.Error(w, "Could not render template", http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

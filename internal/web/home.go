package web

import (
	"context"
	"discord-embedder/internal/app"
	"discord-embedder/internal/video"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type Page struct {
	ImageURL string
	VideoURL string
	Width    string
	Height   string
}

func homeHandler(
	ctx context.Context,
	app *app.App,
	home embed.FS,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
			file, err := video.Get(app, id)
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

			// Generate page
			page := Page{
				ImageURL: fmt.Sprintf("%s/files/%s", app.Host, file.Thumbnail.Name),
				VideoURL: fmt.Sprintf("%s/files/%s", app.Host, file.Name),
				Width:    width,
				Height:   height,
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

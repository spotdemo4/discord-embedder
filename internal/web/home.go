package web

import (
	"context"
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

func NewHomeHandler(
	ctx context.Context,
	home embed.FS,
	host string,
	filesDir string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pathItems := strings.Split(r.URL.Path, "/")
			if len(pathItems) < 2 {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			fileID := pathItems[1]

			// Find video file
			file, err := video.Get(fileID, filesDir)
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
				ImageURL: fmt.Sprintf("%s/files/%s.%s", host, fileID, "jpeg"),
				VideoURL: fmt.Sprintf("%s/files/%s", host, file.Name),
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

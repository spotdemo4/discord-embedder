package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewFileHandler(filesDir string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			pathItems := strings.Split(r.URL.Path, "/")
			if len(pathItems) < 3 {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			// Open file
			fileName := pathItems[2]
			file, err := os.Open(filepath.Join(filesDir, fileName))
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			defer file.Close()

			// Set response headers
			w.Header().Set("Access-Control-Allow-Origin", "*")

			// Serve video
			http.ServeContent(w, r, fileName, time.Now(), file)

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
	}
}

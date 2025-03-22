package handlers

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func NewFileHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			pathItems := strings.Split(r.URL.Path, "/")
			if len(pathItems) < 3 {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			// Open file
			fileName := pathItems[2]
			file, err := os.Open(fmt.Sprintf("files/%s", fileName))
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}
			defer file.Close()

			// Set response headers
			w.Header().Set("Access-Control-Allow-Origin", "*")

			// Serve video
			http.ServeContent(w, r, fileName, time.Now(), file)
		}
	}
}

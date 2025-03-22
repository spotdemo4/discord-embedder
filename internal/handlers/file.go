package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

			// Get first 512 bytes of file
			start := make([]byte, 512)
			_, err = file.Read(start)
			if err != nil {
				http.Error(w, "Could not get content type", http.StatusInternalServerError)
				return
			}

			// Set response headers
			w.Header().Set("Content-Type", http.DetectContentType(start))
			w.Header().Set("Access-Control-Allow-Origin", "*")

			// Write to output
			io.Copy(w, file)
		}
	}
}

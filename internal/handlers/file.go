package handlers

import (
	"fmt"
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

			fileName := pathItems[2]
			file, err := os.ReadFile(fmt.Sprintf("files/%s", fileName))
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", http.DetectContentType(file))
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Write(file)
		}
	}
}

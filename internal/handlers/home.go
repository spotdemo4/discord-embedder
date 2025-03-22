package handlers

import (
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type Page struct {
	ImageURL string
	VideoURL string
	Width    string
	Height   string
}

func NewHomeHandler(home embed.FS, host string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			pathItems := strings.Split(r.URL.Path, "/")
			if len(pathItems) < 2 {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			fileID := pathItems[1]

			// Find video file
			var file *os.File
			err := filepath.Walk("files", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if strings.HasPrefix(info.Name(), fileID) && !strings.HasSuffix(info.Name(), "jpeg") {
					file, err = os.Open(fmt.Sprintf("files/%s", info.Name()))
					if err != nil {
						return err
					}
				}

				return nil
			})
			if err != nil {
				log.Println(err.Error())
				http.Error(w, "Could not open files", http.StatusInternalServerError)
				return
			}
			if file == nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			width, height, err := getResolution(file.Name())
			if err != nil {
				http.Error(w, "Could not get resolution", http.StatusInternalServerError)
				return
			}

			// Serve page
			page := Page{
				ImageURL: fmt.Sprintf("%s/files/%s.%s", host, fileID, "jpeg"),
				VideoURL: fmt.Sprintf("%s/%s", host, file.Name()),
				Width:    width,
				Height:   height,
			}
			t, err := template.ParseFS(home, "templates/home.html")
			if err != nil {
				http.Error(w, "Could not use template", http.StatusInternalServerError)
				return
			}
			t.Execute(w, page)
		}
	}
}

func getResolution(filename string) (width string, height string, err error) {
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0",
		filename,
	)

	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}

	resolution := strings.Split(string(out), "x")
	if len(resolution) != 2 {
		return "", "", errors.New("could not get resolution")
	}

	return resolution[0], resolution[1], nil
}

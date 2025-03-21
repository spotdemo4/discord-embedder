package handlers

import (
	"net/http"
)

func NewFileHandler() http.Handler {
	return http.FileServer(http.Dir("files"))
}

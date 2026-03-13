package web

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:out
var staticFiles embed.FS

func FileServer() http.Handler {
	frontendFS, err := fs.Sub(staticFiles, "out")
	if err != nil {
		log.Fatalf("failed to prepare static files: %v", err)
	}
	return http.FileServer(http.FS(frontendFS))
}

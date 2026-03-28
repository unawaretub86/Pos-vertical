//go:generate goversioninfo -icon=app.ico -o resource.syso

package main

import (
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/handler"
	"github.com/andreyo/pos/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Initialize database
	db, err := database.New("data.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	h := handler.New(db)
	r.Mount("/api", h.Routes())

	// Serve frontend static files
	r.Handle("/*", web.FileServer())

	addr := "localhost:8080"
	log.Printf("POS server starting at http://%s", addr)

	// Open browser after server starts
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://" + addr)
	}()

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}
	exec.Command(cmd, args...).Start()
}

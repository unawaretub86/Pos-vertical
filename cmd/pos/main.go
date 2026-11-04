//go:generate goversioninfo -icon=app.ico -o resource.syso

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/andreyo/pos/internal/database"
	"github.com/andreyo/pos/internal/handler"
	posmw "github.com/andreyo/pos/internal/middleware"
	"github.com/andreyo/pos/internal/repository"
	"github.com/andreyo/pos/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// port is the TCP port the POS listens on; --port or POS_PORT override 8080.
var port = "8080"

// resolvePort picks the flag value, then the environment value, then 8080.
// Anything that is not a port number between 1 and 65535 is ignored.
func resolvePort(flagVal, envVal string) string {
	for _, v := range []string{flagVal, envVal} {
		v = strings.TrimSpace(v)
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 65535 {
			return strconv.Itoa(n)
		}
	}
	return "8080"
}

func main() {
	portFlag := flag.String("port", "", "TCP port to listen on (default 8080, or POS_PORT)")
	flag.Parse()
	port = resolvePort(*portFlag, os.Getenv("POS_PORT"))

	// Initialize database
	db, err := database.New("data.db")
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize JWT secret (US-070)
	setupRepo := repository.NewSetupRepository(db.DB)
	jwtSecret, err := setupRepo.GetOrGenerateJWTSecret()
	if err != nil {
		log.Fatalf("failed to initialize JWT secret: %v", err)
	}
	posmw.SetJWTSecret(jwtSecret)

	// Start automatic daily backup scheduler
	db.StartBackupScheduler()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// API routes
	h := handler.New(db)
	r.Mount("/api", h.Routes())

	// Serve frontend static files
	r.Handle("/*", web.FileServer())

	lan := false
	if cfg, err := setupRepo.Get(); err != nil {
		log.Printf("could not read LAN access config, binding locally: %v", err)
	} else if cfg != nil {
		lan = cfg.LanAccessEnabled
	}
	addr := listenAddr(lan)
	log.Printf("POS server starting at %s (LAN access: %v)", browserURL(), lan)

	// Open browser after server starts
	go func() {
		time.Sleep(500 * time.Millisecond)
		openBrowser(browserURL())
	}()

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// listenAddr returns the bind address: all interfaces only when LAN access is on.
func listenAddr(lan bool) string {
	if lan {
		return "0.0.0.0:" + port
	}
	return "127.0.0.1:" + port
}

// browserURL is always localhost, regardless of the bind address.
func browserURL() string {
	return "http://localhost:" + port
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

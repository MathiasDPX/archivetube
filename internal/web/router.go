package web

import (
	"net/http"
	"path/filepath"

	"github.com/MathiasDPX/archivetube/internal/archive"
	"github.com/MathiasDPX/archivetube/internal/config"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
)

// NewRouter sets up the HTTP routes and returns the top-level handler.
func NewRouter(cfg *config.Config, st *store.Store, archiveSvc *archive.Service, q *queue.Queue, tmpl *Templates, staticDir string) http.Handler {
	mux := http.NewServeMux()

	h := &handlers{
		config:  cfg,
		store:   st,
		archive: archiveSvc,
		queue:   q,
		tmpl:    tmpl,
	}

	// Static files (CSS, JS)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	// Media/data files served from DataDir
	mux.Handle("GET /data/", http.StripPrefix("/data/", http.FileServer(http.Dir(cfg.DataDir))))

	// Auth
	mux.HandleFunc("GET /login", h.handleLoginPage)
	mux.HandleFunc("POST /login", h.handleLoginSubmit)
	mux.HandleFunc("POST /logout", h.handleLogout)

	// Pages
	mux.HandleFunc("GET /{$}", h.handleHome)
	mux.HandleFunc("GET /videos/{id}", h.handleVideo)
	mux.HandleFunc("GET /creators", h.handleCreators)
	mux.HandleFunc("GET /creators/{id}", h.handleCreator)
	mux.HandleFunc("GET /download/{id}", h.handleDownload)
	mux.HandleFunc("POST /videos/{id}/delete", h.requireAuth(h.handleDeleteVideo))
	mux.HandleFunc("GET /archive", h.requireAuth(h.handleArchivePage))
	mux.HandleFunc("POST /archive", h.requireAuth(h.handleArchiveSubmit))
	mux.HandleFunc("GET /api/videos", h.handleAPIVideos)
	mux.HandleFunc("GET /api/creators/{id}/videos", h.handleAPICreatorVideos)
	mux.HandleFunc("GET /api/queue", h.requireAuthAPI(h.handleQueueStatus))
	mux.HandleFunc("POST /archive/clear", h.requireAuth(h.handleQueueClear))
	mux.HandleFunc("GET /api/playlist", h.requireAuthAPI(h.handlePlaylistFetch))
	mux.HandleFunc("POST /archive/batch", h.requireAuthAPI(h.handleArchiveBatch))

	return logRequests(mux)
}

// logRequests is a simple logging middleware.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// WebPaths holds the filesystem paths for web assets, resolved from the project root.
type WebPaths struct {
	TemplateDir string
	StaticDir   string
}

// DefaultWebPaths returns web asset paths relative to the working directory.
func DefaultWebPaths() WebPaths {
	return WebPaths{
		TemplateDir: filepath.Join("web", "templates"),
		StaticDir:   filepath.Join("web", "static"),
	}
}

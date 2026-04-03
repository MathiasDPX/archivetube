package web

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/MathiasDPX/archivetube/internal/archive"
	"github.com/MathiasDPX/archivetube/internal/config"
	"github.com/MathiasDPX/archivetube/internal/metrics"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
)

func NewRouter(cfg *config.Config, st *store.Store, archiveSvc *archive.Service, q *queue.Queue, tmpl *Templates, staticDir string) http.Handler {
	mux := http.NewServeMux()

	h := &handlers{
		config:  cfg,
		store:   st,
		archive: archiveSvc,
		queue:   q,
		tmpl:    tmpl,
	}

	// static files
	mux.Handle("GET /static/", http.StripPrefix("/static", neuter(http.FileServer(http.Dir(staticDir)))))

	// data files served from DataDir
	mux.Handle("GET /data/", http.StripPrefix("/data/", neuter(http.FileServer(http.Dir(cfg.DataDir)))))

	// auth
	mux.HandleFunc("GET /login", h.handleLoginPage)
	mux.HandleFunc("POST /login", h.handleLoginSubmit)
	mux.HandleFunc("POST /logout", h.handleLogout)

	// pages
	mux.HandleFunc("GET /{$}", h.handleHome)
	mux.HandleFunc("GET /videos/{id}", h.handleVideo)
	mux.HandleFunc("GET /creators", h.handleCreators)
	mux.HandleFunc("GET /creators/{id}", h.handleCreator)
	mux.HandleFunc("GET /download/{id}", h.handleDownload)
	mux.HandleFunc("POST /videos/{id}/delete", h.requireAuth(h.handleDeleteVideo))
	mux.HandleFunc("POST /creators/{id}/delete", h.requireAuth(h.handleDeleteCreator))
	mux.HandleFunc("POST /creators/{id}/refresh", h.requireAuth(h.handleRefreshCreator))
	mux.HandleFunc("GET /archive", h.requireAuth(h.handleArchivePage))
	mux.HandleFunc("POST /archive", h.requireAuth(h.handleArchiveSubmit))
	mux.HandleFunc("GET /api/videos", h.handleAPIVideos)
	mux.HandleFunc("GET /api/creators/{id}/videos", h.handleAPICreatorVideos)
	mux.HandleFunc("GET /api/queue", h.requireAuthAPI(h.handleQueueStatus))
	mux.HandleFunc("POST /archive/clear", h.requireAuth(h.handleQueueClear))
	mux.HandleFunc("GET /api/playlist", h.requireAuthAPI(h.handlePlaylistFetch))
	mux.HandleFunc("POST /archive/batch", h.requireAuthAPI(h.handleArchiveBatch))

	// metrics
	mux.Handle("GET /metrics", metrics.Handler())

	return mux
}

// https://www.alexedwards.net/blog/disable-http-fileserver-directory-listings#using-middleware
func neuter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type WebPaths struct {
	TemplateDir string
	StaticDir   string
}

func DefaultWebPaths() WebPaths {
	return WebPaths{
		TemplateDir: filepath.Join("web", "templates"),
		StaticDir:   filepath.Join("web", "static"),
	}
}

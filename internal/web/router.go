package web

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

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

	if cfg.Auth.Mode == "oidc" {
		h.oidc = newOIDCAuth(&cfg.Auth)
	}

	// static files
	mux.Handle("GET /static/", http.StripPrefix("/static", neuter(http.FileServer(http.Dir(staticDir)))))

	// data files served from DataDir
	mux.Handle("GET /data/", http.StripPrefix("/data/", dearrowThumbnail(neuter(http.FileServer(http.Dir(cfg.Archive.DataDir))), h.config.Dearrow)))

	// auth pages (HTML)
	mux.HandleFunc("GET /login", h.handleLoginPage)
	mux.HandleFunc("POST /login", h.handleLoginSubmit)
	mux.HandleFunc("GET /auth/callback", h.handleOIDCCallback)

	// HTML pages
	mux.HandleFunc("GET /{$}", h.handleHome)
	mux.HandleFunc("GET /videos/{id}", h.handleVideo)
	mux.HandleFunc("GET /creators", h.handleCreators)
	mux.HandleFunc("GET /creators/{id}", h.handleCreator)
	mux.HandleFunc("GET /archive", h.requireAuth(h.handleArchivePage))
	mux.HandleFunc("POST /archive", h.requireAuth(h.handleArchiveSubmit))

	// API
	mux.HandleFunc("GET /api/videos", h.handleAPIVideos)
	mux.HandleFunc("GET /api/videos/{id}/download", h.handleDownload)
	mux.HandleFunc("POST /api/videos/{id}/delete", h.requireAuthAPI(h.handleDeleteVideo, config.PermDelete))
	mux.HandleFunc("GET /api/creators/{id}/videos", h.handleAPICreatorVideos)
	mux.HandleFunc("POST /api/creators/{id}/delete", h.requireAuthAPI(h.handleDeleteCreator, config.PermDelete))
	mux.HandleFunc("POST /api/creators/{id}/refresh", h.requireAuthAPI(h.handleRefreshCreator, config.PermRefresh))
	mux.HandleFunc("GET /api/queue", h.requireAuthAPI(h.handleQueueStatus, config.PermArchive))
	mux.HandleFunc("POST /api/queue/clear", h.requireAuth(h.handleQueueClear))
	mux.HandleFunc("GET /api/playlist", h.requireAuthAPI(h.handlePlaylistFetch, config.PermArchive))
	mux.HandleFunc("POST /api/archive/batch", h.requireAuthAPI(h.handleArchiveBatch, config.PermArchive))
	mux.HandleFunc("GET /openapi.yml", h.handleOpenAPI)

	// metrics
	if cfg.Observability.EnablePrometheus {
		mux.Handle("GET /metrics", metrics.Handler())
	}

	return otelMiddleware(corsMiddleware(mux, cfg.Server.CorsHost))
}

func otelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/data/") || strings.HasPrefix(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func corsMiddleware(next http.Handler, origin string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin) // TODO: use config variable
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func dearrowThumbnail(next http.Handler, dearrowConfig config.DearrowConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "media/channels/") && strings.HasSuffix(r.URL.Path, "/video.webp") && dearrowConfig.Enabled {
			parts := strings.Split(r.URL.Path, "/")

			if len(parts) < 5 {
				next.ServeHTTP(w, r)
				return
			}

			videoID := parts[3]

			resp, err := http.Get(dearrowConfig.ThumbApiURL + "/api/v1/getThumbnail?videoID=" + videoID)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			defer resp.Body.Close()

			for k, v := range resp.Header {
				for _, vv := range v {
					w.Header().Add(k, vv)
				}
			}

			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
			return
		}

		next.ServeHTTP(w, r)
	})
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

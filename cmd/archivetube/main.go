package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MathiasDPX/archivetube/internal/archive"
	"github.com/MathiasDPX/archivetube/internal/config"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
	"github.com/MathiasDPX/archivetube/internal/web"
)

func main() {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	dbPath := filepath.Join(cfg.DataDir, "archivetube.db")
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("opening store: %v", err)
	}
	defer st.Close()

	archiveSvc := archive.New(cfg.YtDlpPath, cfg.DataDir, cfg.Proxy, st)
	q := queue.New(archiveSvc.ArchiveURL)

	webPaths := web.DefaultWebPaths()
	tmpl, err := web.NewTemplates(webPaths.TemplateDir)
	if err != nil {
		log.Fatalf("loading templates: %v", err)
	}

	router := web.NewRouter(cfg, st, archiveSvc, q, tmpl, webPaths.StaticDir)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("archivetube listening on %s", cfg.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

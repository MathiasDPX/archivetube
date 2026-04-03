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
	"github.com/MathiasDPX/archivetube/internal/metrics"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
	"github.com/MathiasDPX/archivetube/internal/web"
)

func main() {
	cfg := config.Load("config.toml")

	if err := os.MkdirAll(cfg.Archive.DataDir, 0o755); err != nil {
		log.Fatalf("creating data dir: %v", err)
	}

	dbPath := filepath.Join(cfg.Archive.DataDir, "archivetube.db")
	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("opening store: %v", err)
	}
	defer st.Close()

	if n, err := st.CountVideos(); err == nil {
		metrics.SetVideosTotal(n)
	}
	if n, err := st.CountChannels(); err == nil {
		metrics.SetChannelsTotal(n)
	}

	archiveSvc := archive.New(cfg.Archive.YtDlpPath, cfg.Archive.DataDir, cfg.Archive.Proxy, st)
	q := queue.New(archiveSvc.ArchiveURL)

	if sha := os.Getenv("GIT_SHA"); sha != "" {
		web.SetGitSHA(sha)
	}
	webPaths := web.DefaultWebPaths()
	tmpl, err := web.NewTemplates(webPaths.TemplateDir)
	if err != nil {
		log.Fatalf("loading templates: %v", err)
	}

	router := web.NewRouter(cfg, st, archiveSvc, q, tmpl, webPaths.StaticDir)

	srv := &http.Server{
		Addr:              cfg.Server.ListenAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("archivetube listening on %s", cfg.Server.ListenAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server error: %v", err)
	}
}

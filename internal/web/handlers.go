package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"

	"github.com/MathiasDPX/archivetube/internal/archive"
	"github.com/MathiasDPX/archivetube/internal/config"
	"github.com/MathiasDPX/archivetube/internal/domain"
	"github.com/MathiasDPX/archivetube/internal/embedding"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
)

var tracer = otel.Tracer("github.com/MathiasDPX/archivetube/internal/web")

type handlers struct {
	config  *config.Config
	store   *store.Store
	archive *archive.Service
	queue   *queue.Queue
	tmpl    *Templates
	oidc    *oidcAuth
}

type HomeData struct {
	Videos  []domain.Video
	Query   string
	Page    int
	Total   int
	PerPage int
}

type VideoData struct {
	Video     *domain.Video
	Channel   *domain.Channel
	Chapters  []domain.Chapter
	Subtitles []domain.Subtitle
}

type ArchiveData struct {
	Error        string
	Jobs         []queue.Job
	PrefilledURL string
}

type CreatorsData struct {
	Channels []domain.Channel
}

type CreatorData struct {
	Channel *domain.Channel
	Videos  []domain.Video
	Page    int
	Total   int
	PerPage int
}

type NotFoundData struct {
	Kind string
	URL  string
}

func (h *handlers) handleHome(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleHome")
	defer span.End()

	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	if query != "" && h.config.SmartSearch.Enabled {
		span.AddEvent("Searching with embedding")
		videos, err := h.smartSearch(ctx, query, perPage)
		if err != nil {
			h.serverError(w, err)
			return
		}
		span.AddEvent("Videos searched, rendering")
		h.renderWithRequest(w, r, "home.tmpl", HomeData{
			Videos:  videos,
			Query:   query,
			Page:    1,
			Total:   len(videos),
			PerPage: perPage,
		})
		span.AddEvent("Rendered")
		return
	}

	span.AddEvent("Searching")
	videos, total, err := h.store.ListVideos(ctx, query, "desc", perPage, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}
	span.AddEvent("Videos searched, rendering")

	h.renderWithRequest(w, r, "home.tmpl", HomeData{
		Videos:  videos,
		Query:   query,
		Page:    page,
		Total:   total,
		PerPage: perPage,
	})
	span.AddEvent("Rendered")
}

func (h *handlers) handleVideo(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleVideo")
	defer span.End()

	ytID := r.PathValue("id")

	video, err := h.store.GetVideoByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if video == nil {
		w.WriteHeader(http.StatusNotFound)
		h.renderWithRequest(w, r, "notfound.tmpl", NotFoundData{
			Kind: "video",
			URL:  "https://www.youtube.com/watch?v=" + ytID,
		})
		return
	}

	chapters, err := h.store.GetChapters(ctx, video.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	subtitles, err := h.store.GetSubtitles(ctx, video.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	channel, err := h.store.GetChannelByYoutubeID(ctx, video.ChannelYoutubeID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "video.tmpl", VideoData{
		Video:     video,
		Channel:   channel,
		Chapters:  chapters,
		Subtitles: subtitles,
	})
}

func (h *handlers) handleCreators(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleCreators")
	defer span.End()

	channels, err := h.store.ListChannels(ctx)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "creators.tmpl", CreatorsData{Channels: channels})
}

func (h *handlers) handleCreator(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleCreator")
	defer span.End()

	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if channel == nil {
		w.WriteHeader(http.StatusNotFound)
		h.renderWithRequest(w, r, "notfound.tmpl", NotFoundData{
			Kind: "author",
			URL:  "https://www.youtube.com/channel/" + ytID,
		})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListVideosByChannel(ctx, channel.ID, perPage, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "creator.tmpl", CreatorData{
		Channel: channel,
		Videos:  videos,
		Page:    page,
		Total:   total,
		PerPage: perPage,
	})
}

func (h *handlers) handleDownload(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleDownload")
	defer span.End()

	ytID := r.PathValue("id")

	video, err := h.store.GetVideoByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if video == nil {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(h.config.Archive.DataDir, video.VideoRelPath)
	filename := video.Title + "." + video.VideoExt

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, filePath)
}

func (h *handlers) handleArchivePage(w http.ResponseWriter, r *http.Request) {
	h.renderWithRequest(w, r, "archive.tmpl", ArchiveData{
		Jobs:         h.queue.Jobs(),
		PrefilledURL: r.URL.Query().Get("url"),
	})
}

func (h *handlers) handleArchiveSubmit(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleArchiveSubmit")
	defer span.End()

	if err := r.ParseForm(); err != nil {
		h.serverError(w, err)
		return
	}

	url := r.FormValue("url")
	quality := r.FormValue("quality")
	if url == "" {
		h.renderWithRequest(w, r, "archive.tmpl", ArchiveData{
			Error: "Please provide a URL.",
			Jobs:  h.queue.Jobs(),
		})
		return
	}

	if h.queue.IsURLQueued(url) {
		h.renderWithRequest(w, r, "archive.tmpl", ArchiveData{
			Error:        "This URL is already in the queue.",
			Jobs:         h.queue.Jobs(),
			PrefilledURL: url,
		})
		return
	}

	if ytID, err := archive.ExtractVideoID(url); err == nil {
		if v, _ := h.store.GetVideoByYoutubeID(ctx, ytID); v != nil {
			h.queue.EnqueueAlreadyArchived(url)
			http.Redirect(w, r, "/archive", http.StatusSeeOther)
			return
		}
	}

	h.queue.Enqueue(url, quality)
	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.queue.Jobs())
}

func (h *handlers) handleAPIVideos(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIVideos")
	defer span.End()

	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	if query != "" && h.config.SmartSearch.Enabled {
		videos, err := h.smartSearch(ctx, query, perPage)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"videos":  videos,
			"page":    1,
			"total":   len(videos),
			"perPage": perPage,
		})
		return
	}

	videos, total, err := h.store.ListVideos(ctx, query, "desc", perPage, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"videos":  videos,
		"page":    page,
		"total":   total,
		"perPage": perPage,
	})
}

func (h *handlers) handleAPICreatorVideos(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPICreatorVideos")
	defer span.End()

	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ctx, ytID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if channel == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListVideosByChannel(ctx, channel.ID, perPage, offset)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"videos":  videos,
		"page":    page,
		"total":   total,
		"perPage": perPage,
	})
}

func (h *handlers) handleDeleteVideo(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleDeleteVideo")
	defer span.End()

	ytID := r.PathValue("id")

	video, err := h.store.GetVideoByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if video == nil {
		http.NotFound(w, r)
		return
	}

	for _, rel := range []string{video.VideoRelPath, video.ThumbnailRelPath, video.InfoJSONRelPath} {
		if rel != "" {
			os.Remove(filepath.Join(h.config.Archive.DataDir, rel))
		}
	}

	subtitles, _ := h.store.GetSubtitles(ctx, video.ID)
	for _, sub := range subtitles {
		if sub.RelPath != "" {
			os.Remove(filepath.Join(h.config.Archive.DataDir, sub.RelPath))
		}
	}

	h.store.DeleteVideoVectors(ctx, video.YoutubeVideoID)

	channelID := video.ChannelID

	if err := h.store.DeleteVideo(ctx, video.ID); err != nil {
		h.serverError(w, err)
		return
	}

	count, err := h.store.CountVideosByChannel(ctx, channelID)
	if err == nil && count == 0 {
		h.store.DeleteChannel(ctx, channelID)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *handlers) handleDeleteCreator(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleDeleteCreator")
	defer span.End()

	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if channel == nil {
		http.NotFound(w, r)
		return
	}

	channelDir := filepath.Join(h.config.Archive.DataDir, "media", "channels", channel.YoutubeChannelID)
	for _, prefix := range []string{"avatar", "banner"} {
		for _, ext := range []string{"jpg", "png", "webp"} {
			os.Remove(filepath.Join(channelDir, prefix+"."+ext))
		}
	}

	if err := h.store.ClearChannelImages(ctx, channel.ID); err != nil {
		h.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/creators/"+ytID, http.StatusSeeOther)
}

func (h *handlers) handleRefreshCreator(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleRefreshCreator")
	defer span.End()

	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ctx, ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if channel == nil {
		http.NotFound(w, r)
		return
	}

	if err := h.archive.RefreshChannelMetadata(ctx, channel); err != nil {
		log.Printf("refresh creator metadata: %v", err)
	}

	http.Redirect(w, r, "/creators/"+ytID, http.StatusSeeOther)
}

func (h *handlers) handleQueueClear(w http.ResponseWriter, r *http.Request) {
	h.queue.ClearFinished()
	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handlePlaylistFetch(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handlePlaylistFetch")
	defer span.End()

	url := r.URL.Query().Get("url")
	if url == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url parameter is required"})
		return
	}

	entries, err := h.archive.FetchPlaylistEntries(ctx, url)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *handlers) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("web", "openapi.yml"))
}

func (h *handlers) handleArchiveBatch(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleArchiveBatch")
	defer span.End()

	var body struct {
		URLs    []string `json:"urls"`
		Quality string   `json:"quality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, url := range body.URLs {
		if url == "" {
			continue
		}
		if h.queue.IsURLQueued(url) {
			continue
		}
		if ytID, err := archive.ExtractVideoID(url); err == nil {
			if v, _ := h.store.GetVideoByYoutubeID(ctx, ytID); v != nil {
				h.queue.EnqueueAlreadyArchived(url)
				continue
			}
		}
		h.queue.Enqueue(url, body.Quality)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *handlers) renderWithRequest(w http.ResponseWriter, r *http.Request, name string, data any) {
	if err := h.tmpl.render(w, name, data, isLoggedIn(r), h.authEnabled(), absoluteRequestURL(r)); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func absoluteRequestURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.Split(forwardedProto, ",")[0]
	}

	host := strings.TrimSpace(r.Host)
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = strings.Split(forwardedHost, ",")[0]
	}

	if host == "" {
		return r.URL.RequestURI()
	}
	return scheme + "://" + host + r.URL.RequestURI()
}

func (h *handlers) smartSearch(ctx context.Context, query string, limit int) ([]domain.Video, error) {
	queryVec, err := embedding.GetEmbedding(ctx, &h.config.SmartSearch, query)
	if err != nil {
		return nil, fmt.Errorf("embedding query: %w", err)
	}
	return h.store.SearchVideosSmart(ctx, queryVec, limit)
}

func (h *handlers) serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

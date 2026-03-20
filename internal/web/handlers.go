package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/MathiasDPX/archivetube/internal/archive"
	"github.com/MathiasDPX/archivetube/internal/config"
	"github.com/MathiasDPX/archivetube/internal/domain"
	"github.com/MathiasDPX/archivetube/internal/queue"
	"github.com/MathiasDPX/archivetube/internal/store"
)

type handlers struct {
	config  *config.Config
	store   *store.Store
	archive *archive.Service
	queue   *queue.Queue
	tmpl    *Templates
}

type HomeData struct {
	Videos  []domain.Video
	Query   string
	Page    int
	Total   int
	PerPage int
}

type VideoData struct {
	Video    *domain.Video
	Channel  *domain.Channel
	Chapters []domain.Chapter
	Subtitles []domain.Subtitle
}

type ArchiveData struct {
	Error string
	Jobs  []queue.Job
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

func (h *handlers) handleHome(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListVideos(query, "desc", perPage, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "home.tmpl", HomeData{
		Videos:  videos,
		Query:   query,
		Page:    page,
		Total:   total,
		PerPage: perPage,
	})
}

func (h *handlers) handleVideo(w http.ResponseWriter, r *http.Request) {
	ytID := r.PathValue("id")

	video, err := h.store.GetVideoByYoutubeID(ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if video == nil {
		http.NotFound(w, r)
		return
	}

	chapters, err := h.store.GetChapters(video.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	subtitles, err := h.store.GetSubtitles(video.ID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	channel, err := h.store.GetChannelByYoutubeID(video.ChannelYoutubeID)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "video.tmpl", VideoData{
		Video:    video,
		Channel:  channel,
		Chapters: chapters,
		Subtitles: subtitles,
	})
}

func (h *handlers) handleCreators(w http.ResponseWriter, r *http.Request) {
	channels, err := h.store.ListChannels()
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "creators.tmpl", CreatorsData{Channels: channels})
}

func (h *handlers) handleCreator(w http.ResponseWriter, r *http.Request) {
	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if channel == nil {
		http.NotFound(w, r)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListVideosByChannel(channel.ID, perPage, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, "creator.tmpl", CreatorData{
		Channel: channel,
		Videos:  videos,
		Page:    page,
		Total:   total,
		PerPage: perPage,
	})
}

func (h *handlers) handleDownload(w http.ResponseWriter, r *http.Request) {
	ytID := r.PathValue("id")

	video, err := h.store.GetVideoByYoutubeID(ytID)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if video == nil {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(h.config.DataDir, video.VideoRelPath)
	filename := video.Title + "." + video.VideoExt

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, filePath)
}

func (h *handlers) handleArchivePage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "archive.tmpl", ArchiveData{Jobs: h.queue.Jobs()})
}

func (h *handlers) handleArchiveSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.serverError(w, err)
		return
	}

	url := r.FormValue("url")
	if url == "" {
		h.render(w, "archive.tmpl", ArchiveData{
			Error: "Please provide a URL.",
			Jobs:  h.queue.Jobs(),
		})
		return
	}

	h.queue.Enqueue(url)
	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.queue.Jobs())
}

func (h *handlers) handleQueueClear(w http.ResponseWriter, r *http.Request) {
	h.queue.ClearFinished()
	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handlePlaylistFetch(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "url parameter is required"})
		return
	}

	entries, err := h.archive.FetchPlaylistEntries(r.Context(), url)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *handlers) handleArchiveBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, url := range body.URLs {
		if url != "" {
			h.queue.Enqueue(url)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *handlers) render(w http.ResponseWriter, name string, data any) {
	if err := h.tmpl.Render(w, name, data); err != nil {
		log.Printf("render error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *handlers) serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

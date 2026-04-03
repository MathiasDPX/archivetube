package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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

	h.renderWithRequest(w, r, "home.tmpl", HomeData{
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
		w.WriteHeader(http.StatusNotFound)
		h.renderWithRequest(w, r, "notfound.tmpl", NotFoundData{
			Kind: "video",
			URL:  "https://www.youtube.com/watch?v=" + ytID,
		})
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

	h.renderWithRequest(w, r, "video.tmpl", VideoData{
		Video:     video,
		Channel:   channel,
		Chapters:  chapters,
		Subtitles: subtitles,
	})
}

func (h *handlers) handleCreators(w http.ResponseWriter, r *http.Request) {
	channels, err := h.store.ListChannels()
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "creators.tmpl", CreatorsData{Channels: channels})
}

func (h *handlers) handleCreator(w http.ResponseWriter, r *http.Request) {
	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ytID)
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

	videos, total, err := h.store.ListVideosByChannel(channel.ID, perPage, offset)
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
	h.renderWithRequest(w, r, "archive.tmpl", ArchiveData{
		Jobs:         h.queue.Jobs(),
		PrefilledURL: r.URL.Query().Get("url"),
	})
}

func (h *handlers) handleArchiveSubmit(w http.ResponseWriter, r *http.Request) {
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

	h.queue.Enqueue(url, quality)
	http.Redirect(w, r, "/archive", http.StatusSeeOther)
}

func (h *handlers) handleQueueStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.queue.Jobs())
}

func (h *handlers) handleAPIVideos(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListVideos(query, "desc", perPage, offset)
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
	ytID := r.PathValue("id")

	channel, err := h.store.GetChannelByYoutubeID(ytID)
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

	videos, total, err := h.store.ListVideosByChannel(channel.ID, perPage, offset)
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

	for _, rel := range []string{video.VideoRelPath, video.ThumbnailRelPath, video.InfoJSONRelPath} {
		if rel != "" {
			os.Remove(filepath.Join(h.config.DataDir, rel))
		}
	}

	subtitles, _ := h.store.GetSubtitles(video.ID)
	for _, sub := range subtitles {
		if sub.RelPath != "" {
			os.Remove(filepath.Join(h.config.DataDir, sub.RelPath))
		}
	}

	channelID := video.ChannelID

	if err := h.store.DeleteVideo(video.ID); err != nil {
		h.serverError(w, err)
		return
	}

	count, err := h.store.CountVideosByChannel(channelID)
	if err == nil && count == 0 {
		h.store.DeleteChannel(channelID)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *handlers) handleDeleteCreator(w http.ResponseWriter, r *http.Request) {
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

	channelDir := filepath.Join(h.config.DataDir, "media", "channels", channel.YoutubeChannelID)
	for _, prefix := range []string{"avatar", "banner"} {
		for _, ext := range []string{"jpg", "png", "webp"} {
			os.Remove(filepath.Join(channelDir, prefix+"."+ext))
		}
	}

	if err := h.store.ClearChannelImages(channel.ID); err != nil {
		h.serverError(w, err)
		return
	}

	http.Redirect(w, r, "/creators/"+ytID, http.StatusSeeOther)
}

func (h *handlers) handleRefreshCreator(w http.ResponseWriter, r *http.Request) {
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

	if err := h.archive.RefreshChannelMetadata(r.Context(), channel); err != nil {
		log.Printf("refresh creator metadata: %v", err)
	}

	http.Redirect(w, r, "/creators/"+ytID, http.StatusSeeOther)
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
		URLs    []string `json:"urls"`
		Quality string   `json:"quality"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	for _, url := range body.URLs {
		if url != "" {
			h.queue.Enqueue(url, body.Quality)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *handlers) renderWithRequest(w http.ResponseWriter, r *http.Request, name string, data any) {
	if err := h.tmpl.render(w, name, data, isLoggedIn(r), h.config.PasswordHash != "", absoluteRequestURL(r)); err != nil {
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

func (h *handlers) serverError(w http.ResponseWriter, err error) {
	log.Printf("server error: %v", err)
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

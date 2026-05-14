package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

type PlaylistsData struct {
	Playlists []domain.Playlist
}

type PlaylistData struct {
	Playlist *domain.Playlist
	Videos   []domain.Video
	Page     int
	Total    int
	PerPage  int
}

func (h *handlers) handlePlaylists(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handlePlaylists")
	defer span.End()

	playlists, err := h.store.ListPlaylists(ctx)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "playlists.tmpl", PlaylistsData{Playlists: playlists})
}

func (h *handlers) handlePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handlePlaylist")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	playlist, err := h.store.GetPlaylist(ctx, id)
	if err != nil {
		h.serverError(w, err)
		return
	}
	if playlist == nil {
		w.WriteHeader(http.StatusNotFound)
		h.renderWithRequest(w, r, "notfound.tmpl", NotFoundData{Kind: "playlist"})
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListPlaylistVideos(ctx, id, perPage, offset)
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.renderWithRequest(w, r, "playlist.tmpl", PlaylistData{
		Playlist: playlist,
		Videos:   videos,
		Page:     page,
		Total:    total,
		PerPage:  perPage,
	})
}

// API handlers

func (h *handlers) handleAPIListPlaylists(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIListPlaylists")
	defer span.End()

	playlists, err := h.store.ListPlaylists(ctx)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, playlists)
}

func (h *handlers) handleAPIPlaylistVideos(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIPlaylistVideos")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage := 24
	offset := (page - 1) * perPage

	videos, total, err := h.store.ListPlaylistVideos(ctx, id, perPage, offset)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"videos":  videos,
		"page":    page,
		"total":   total,
		"perPage": perPage,
	})
}

func (h *handlers) handleAPICreatePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPICreatePlaylist")
	defer span.End()

	var body struct {
		Name              string `json:"name"`
		SourceURL         string `json:"source_url"`
		YoutubePlaylistID string `json:"youtube_playlist_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	id, err := h.store.CreatePlaylist(ctx, name, body.SourceURL, body.YoutubePlaylistID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id, "name": name})
}

func (h *handlers) handleAPIRenamePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIRenamePlaylist")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required")
		return
	}

	if err := h.store.RenamePlaylist(ctx, id, name); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "name": name})
}

func (h *handlers) handleAPIDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIDeletePlaylist")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	if err := h.store.DeletePlaylist(ctx, id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handlers) handleAPIAddVideosToPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIAddVideosToPlaylist")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}

	var body struct {
		YoutubeVideoIDs []string `json:"youtube_video_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "bad request")
		return
	}

	added := 0
	for _, ytID := range body.YoutubeVideoIDs {
		ytID = strings.TrimSpace(ytID)
		if ytID == "" {
			continue
		}
		v, err := h.store.GetVideoByYoutubeID(ctx, ytID)
		if err != nil || v == nil {
			continue
		}
		if err := h.store.AddVideoToPlaylist(ctx, id, v.ID); err == nil {
			added++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "added": added})
}

func (h *handlers) handleAPIRemoveVideoFromPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "handleAPIRemoveVideoFromPlaylist")
	defer span.End()

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid playlist id")
		return
	}
	ytID := r.PathValue("vid")

	v, err := h.store.GetVideoByYoutubeID(ctx, ytID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if v == nil {
		writeJSONError(w, http.StatusNotFound, "video not found")
		return
	}

	if err := h.store.RemoveVideoFromPlaylist(ctx, id, v.ID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

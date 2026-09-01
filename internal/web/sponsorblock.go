package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/MathiasDPX/archivetube/internal/config"
)

// SponsorBlock segment categories used to build the skip request.
const (
	sbSponsorCategory   = "sponsor"
	sbPromotionCategory = "selfpromo"
)

// segment is a single SponsorBlock skip segment as returned by the API.
type segment struct {
	Segment  []float64 `json:"segment"`
	Category string    `json:"category"`
	UUID     string    `json:"UUID"`
}

func (h *handlers) handleSponsorBlockSegments(w http.ResponseWriter, r *http.Request) {
	if !h.config.SponsorBlock.HasSegments() {
		http.Error(w, "sponsorblock not enabled", http.StatusNotFound)
		return
	}

	videoID := r.PathValue("id")
	if videoID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]segment{})
		return
	}

	categories := h.sponsorBlockCategories()
	if len(categories) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]segment{})
		return
	}

	apiURL := strings.TrimRight(h.config.SponsorBlock.ApiURL, "/")
	params := url.Values{}
	params.Set("videoID", videoID)
	params.Set("categories", `["`+strings.Join(categories, `","`)+`"]`)

	resp, err := http.Get(apiURL + "/api/skipSegments?" + params.Encode())
	if err != nil {
		http.Error(w, "failed to reach SponsorBlock", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "failed to read SponsorBlock response", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// SponsorBlock returns a 404 with a JSON error body when a video has no
	// segments, and a 400 when nothing matches the requested categories.
	if resp.StatusCode != http.StatusOK {
		w.Write([]byte("[]"))
		return
	}

	w.Write(body)
}

// sponsorBlockCategories returns the category names to request based on the
// configured segment modes. Categories set to "hide" are excluded entirely.
func (h *handlers) sponsorBlockCategories() []string {
	sb := h.config.SponsorBlock
	var cats []string
	if sb.SponsorSegments != config.SegmentModeHide {
		cats = append(cats, sbSponsorCategory)
	}
	if sb.PromotionSegments != config.SegmentModeHide {
		cats = append(cats, sbPromotionCategory)
	}
	return cats
}

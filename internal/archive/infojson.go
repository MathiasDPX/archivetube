package archive

import (
	"encoding/json"
	"fmt"
	"os"
)

type InfoJSON struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Duration       float64 `json:"duration"`
	UploadDate     string  `json:"upload_date"`
	WebpageURL     string  `json:"webpage_url"`
	Channel        string  `json:"channel"`
	ChannelID      string  `json:"channel_id"`
	ChannelURL     string  `json:"channel_url"`
	UploaderID     string  `json:"uploader_id"`
	UploaderURL    string  `json:"uploader_url"`
	Thumbnail      string  `json:"thumbnail"`
	ChannelBanner  string  `json:"channel_banner_url"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FilesizeApprox float64 `json:"filesize_approx"`
	Ext            string  `json:"ext"`
	Language       string  `json:"language"`
	Chapters       []struct {
		Title     string  `json:"title"`
		StartTime float64 `json:"start_time"`
		EndTime   float64 `json:"end_time"`
	} `json:"chapters"`
	RequestedSubtitles map[string]struct {
		Ext      string `json:"ext"`
		Filepath string `json:"filepath"`
	} `json:"requested_subtitles"`
	Formats []FormatInfo `json:"formats"`
}

type FormatInfo struct {
	FormatID   string  `json:"format_id"`
	Ext        string  `json:"ext"`
	Language   string  `json:"language"`
	Vcodec     string  `json:"vcodec"`
	Acodec     string  `json:"acodec"`
	FormatNote string  `json:"format_note"`
	Tbr        float64 `json:"tbr"`
	AudioTrack *struct {
		Name      string `json:"name"`
		ID        string `json:"id"`
		IsDefault bool   `json:"audio_is_default"`
	} `json:"audio_track"`
}

func parseInfoJSON(path string) (*InfoJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading info json %s: %w", path, err)
	}

	var info InfoJSON
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing info json %s: %w", path, err)
	}

	return &info, nil
}

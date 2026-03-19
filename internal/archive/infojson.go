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
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	FilesizeApprox float64 `json:"filesize_approx"`
	Ext            string  `json:"ext"`
	Chapters       []struct {
		Title     string  `json:"title"`
		StartTime float64 `json:"start_time"`
		EndTime   float64 `json:"end_time"`
	} `json:"chapters"`
	RequestedSubtitles map[string]struct {
		Ext      string `json:"ext"`
		Filepath string `json:"filepath"`
	} `json:"requested_subtitles"`
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

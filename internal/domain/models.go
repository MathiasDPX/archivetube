package domain

import "time"

type Channel struct {
	ID               int64
	YoutubeChannelID string
	Handle           string
	Name             string
	URL              string
	Description      string
	ThumbnailPath    string
	BannerPath       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Video struct {
	ID               int64
	YoutubeVideoID   string
	ChannelID        int64
	Title            string
	Description      string
	DurationSeconds  int
	PublishedAt      *time.Time
	ArchivedAt       time.Time
	WebpageURL       string
	VideoRelPath     string
	VideoExt         string
	ThumbnailRelPath string
	InfoJSONRelPath  string
	FileSizeBytes    int64
	Width            int
	Height           int
	// joined fields
	ChannelName      string
	ChannelYoutubeID string
}

type Chapter struct {
	ID           int64
	VideoID      int64
	Position     int
	Title        string
	StartSeconds float64
	EndSeconds   float64
}

type Subtitle struct {
	ID           int64
	VideoID      int64
	LanguageCode string
	LanguageName string
	Ext          string
	RelPath      string
	IsDefault    bool
}

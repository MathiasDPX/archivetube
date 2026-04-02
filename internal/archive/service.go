package archive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/MathiasDPX/archivetube/internal/domain"
	"github.com/MathiasDPX/archivetube/internal/store"
)

type Service struct {
	YtDlpPath string
	DataDir   string
	Proxy     string
	Store     *store.Store
}

func New(ytdlpPath, dataDir, proxy string, st *store.Store) *Service {
	return &Service{
		YtDlpPath: ytdlpPath,
		DataDir:   dataDir,
		Proxy:     proxy,
		Store:     st,
	}
}

// matches a 11-character video ID
var ytVideoIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

func extractVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")

	switch host {
	case "youtube.com", "music.youtube.com":
		// /watch?v=ID
		if v := u.Query().Get("v"); v != "" && ytVideoIDRe.MatchString(v) {
			return v, nil
		}
		// /shorts/ID, /embed/ID, /v/ID, /live/ID
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 2 && ytVideoIDRe.MatchString(parts[1]) {
			switch parts[0] {
			case "shorts", "embed", "v", "live":
				return parts[1], nil
			}
		}
	case "youtu.be":
		id := strings.Trim(u.Path, "/")
		if ytVideoIDRe.MatchString(id) {
			return id, nil
		}
	}

	return "", fmt.Errorf("could not extract YouTube video ID from %q", rawURL)
}

func qualityToFormat(quality string) string {
	switch quality {
	case "360":
		return "bv*[height<=360]+ba/bv*[height<=360]/b[height<=360]/bv*+ba/b"
	case "480":
		return "bv*[height<=480]+ba/bv*[height<=480]/b[height<=480]/bv*+ba/b"
	case "720":
		return "bv*[height<=720]+ba/bv*[height<=720]/b[height<=720]/bv*+ba/b"
	case "1080":
		return "bv*[height<=1080]+ba/bv*[height<=1080]/b[height<=1080]/bv*+ba/b"
	case "1440":
		return "bv*[height<=1440]+ba/bv*[height<=1440]/b[height<=1440]/bv*+ba/b"
	case "2160":
		return "bv*[height<=2160]+ba/bv*[height<=2160]/b[height<=2160]/bv*+ba/b"
	default: // best or empty
		return "bv*[ext=mp4]+ba[ext=m4a]/bv*+ba/b"
	}
}

func (s *Service) ArchiveURL(ctx context.Context, url string, quality string) error {
	ytID, err := extractVideoID(url)
	if err != nil {
		return fmt.Errorf("extracting video ID: %w", err)
	}
	existing, err := s.Store.GetVideoByYoutubeID(ytID)
	if err != nil {
		return fmt.Errorf("checking existing video: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("video %s is already archived", ytID)
	}

	tmpBase := TmpDir(s.DataDir)
	tmpDir, err := os.MkdirTemp(tmpBase, "dl-*")
	if err != nil {
		if mkErr := os.MkdirAll(tmpBase, 0o755); mkErr != nil {
			return fmt.Errorf("creating tmp base dir: %w", mkErr)
		}
		tmpDir, err = os.MkdirTemp(tmpBase, "dl-*")
		if err != nil {
			return fmt.Errorf("creating temp dir: %w", err)
		}
	}
	defer os.RemoveAll(tmpDir)

	// build yt-dlp command
	outputTemplate := filepath.Join(tmpDir, "video.%(ext)s")
	args := []string{
		"-o", outputTemplate,
		"--write-info-json",
		"--write-thumbnail",
		"--write-subs",
		"--no-write-auto-subs",
		"--sub-format", "vtt/best",
		"--sub-langs", "all",
		"--no-write-comments",
		"-f", qualityToFormat(quality),
		"--merge-output-format", "mp4",
		"--remote-components", "ejs:npm",
	}

	if s.Proxy != "" {
		args = append(args, "--proxy", s.Proxy)
	}

	cookiePath := "/app/cookies.txt"
	if _, err := os.Stat(cookiePath); err == nil {
		args = append(args, "--cookies", cookiePath)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %w\n%s", err, string(output))
	}

	infoPath := filepath.Join(tmpDir, "video.info.json")
	info, err := parseInfoJSON(infoPath)
	if err != nil {
		return fmt.Errorf("parsing info json: %w", err)
	}

	videoFile, err := findDownloadedVideo(tmpDir)
	if err != nil {
		return fmt.Errorf("locating downloaded video in %s: %w", tmpDir, err)
	}

	subtitleFiles, err := filepath.Glob(filepath.Join(tmpDir, "video.*.vtt"))
	if err != nil {
		return fmt.Errorf("finding subtitle files: %w", err)
	}

	var thumbnailFile string
	for _, ext := range []string{"jpg", "png", "webp"} {
		candidate := filepath.Join(tmpDir, "video."+ext)
		if _, err := os.Stat(candidate); err == nil {
			thumbnailFile = candidate
			break
		}
	}

	finalDir := MediaDir(s.DataDir, info.ChannelID, info.ID)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return fmt.Errorf("creating media dir %s: %w", finalDir, err)
	}

	videoExt := strings.TrimPrefix(filepath.Ext(videoFile), ".")
	finalVideoPath := filepath.Join(finalDir, "video."+videoExt)
	if err := moveFile(videoFile, finalVideoPath); err != nil {
		return fmt.Errorf("moving video file: %w", err)
	}

	finalInfoPath := filepath.Join(finalDir, "video.info.json")
	if err := moveFile(infoPath, finalInfoPath); err != nil {
		return fmt.Errorf("moving info json: %w", err)
	}
	infoJSONRel, _ := filepath.Rel(s.DataDir, finalInfoPath)
	infoJSONRel = filepath.ToSlash(infoJSONRel)

	var finalThumbnailRel string
	if thumbnailFile != "" {
		thumbName := filepath.Base(thumbnailFile)
		finalThumbPath := filepath.Join(finalDir, thumbName)
		if err := moveFile(thumbnailFile, finalThumbPath); err != nil {
			return fmt.Errorf("moving thumbnail: %w", err)
		}
		finalThumbnailRel, _ = filepath.Rel(s.DataDir, finalThumbPath)
		finalThumbnailRel = filepath.ToSlash(finalThumbnailRel)
	}

	type subtitleEntry struct {
		language string
		relPath  string
	}
	var subtitles []subtitleEntry

	for _, sf := range subtitleFiles {
		name := filepath.Base(sf)
		lang := extractSubtitleLang(name)
		finalSubPath := filepath.Join(finalDir, name)
		if err := moveFile(sf, finalSubPath); err != nil {
			return fmt.Errorf("moving subtitle %s: %w", name, err)
		}
		rel, _ := filepath.Rel(s.DataDir, finalSubPath)
		rel = filepath.ToSlash(rel)
		subtitles = append(subtitles, subtitleEntry{language: lang, relPath: rel})
	}

	videoRel, _ := filepath.Rel(s.DataDir, finalVideoPath)
	videoRel = filepath.ToSlash(videoRel)
	var fileSizeBytes int64
	if fi, err := os.Stat(finalVideoPath); err == nil {
		fileSizeBytes = fi.Size()
	}

	uploadDate, err := time.Parse("20060102", info.UploadDate)
	if err != nil {
		return fmt.Errorf("parsing upload date %q: %w", info.UploadDate, err)
	}

	channelDir := filepath.Join(s.DataDir, "media", "channels", info.ChannelID)
	if err := os.MkdirAll(channelDir, 0o755); err != nil {
		return fmt.Errorf("creating channel dir: %w", err)
	}

	avatarRel, bannerRel := s.fetchChannelImages(ctx, channelDir, info.ChannelURL)

	channel := &domain.Channel{
		YoutubeChannelID: info.ChannelID,
		Name:             info.Channel,
		URL:              info.ChannelURL,
		ThumbnailPath:    avatarRel,
		BannerPath:       bannerRel,
	}
	channelID, err := s.Store.UpsertChannel(channel)
	if err != nil {
		return fmt.Errorf("upserting channel: %w", err)
	}

	video := &domain.Video{
		ChannelID:        channelID,
		YoutubeVideoID:   info.ID,
		Title:            info.Title,
		Description:      info.Description,
		DurationSeconds:  int(info.Duration),
		PublishedAt:      &uploadDate,
		WebpageURL:       info.WebpageURL,
		ThumbnailRelPath: finalThumbnailRel,
		VideoRelPath:     videoRel,
		VideoExt:         videoExt,
		InfoJSONRelPath:  infoJSONRel,
		Width:            info.Width,
		Height:           info.Height,
		FileSizeBytes:    fileSizeBytes,
	}
	videoID, err := s.Store.UpsertVideo(video)
	if err != nil {
		return fmt.Errorf("upserting video: %w", err)
	}

	var chapters []domain.Chapter
	for i, ch := range info.Chapters {
		chapters = append(chapters, domain.Chapter{
			VideoID:      videoID,
			Position:     i,
			Title:        ch.Title,
			StartSeconds: ch.StartTime,
			EndSeconds:   ch.EndTime,
		})
	}
	if err := s.Store.ReplaceChapters(videoID, chapters); err != nil {
		return fmt.Errorf("replacing chapters: %w", err)
	}

	var domainSubs []domain.Subtitle
	for _, sub := range subtitles {
		domainSubs = append(domainSubs, domain.Subtitle{
			VideoID:      videoID,
			LanguageCode: sub.language,
			RelPath:      sub.relPath,
			Ext:          "vtt",
		})
	}
	if err := s.Store.ReplaceSubtitles(videoID, domainSubs); err != nil {
		return fmt.Errorf("replacing subtitles: %w", err)
	}

	return nil
}

// attempts an atomic os.Rename, if it fails with EXDEV (cross-device)
// it falls back to copy + remove so that moves across filesystems (e.g. local -> rclone FUSE) work
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		os.Remove(dst)
		return fmt.Errorf("copying data: %w", err)
	}
	if err := dstFile.Close(); err != nil {
		os.Remove(dst)
		return fmt.Errorf("closing destination: %w", err)
	}
	srcFile.Close()
	return os.Remove(src)
}

// scans dir for the video file downloaded by yt-dlp
func findDownloadedVideo(dir string) (string, error) {
	// extensions that are never the video stream itself
	nonVideoExts := map[string]bool{
		".json": true,
		".vtt":  true,
		".srt":  true,
		".ass":  true,
		".ssa":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
		".part": true,
		".ytdl": true,
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading temp dir %s: %w", dir, err)
	}

	var fallback string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if nonVideoExts[ext] {
			continue
		}
		name := e.Name()
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		if stem == "video" {
			return filepath.Join(dir, name), nil
		}

		if fallback == "" && strings.HasPrefix(stem, "video") {
			fallback = filepath.Join(dir, name)
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no video file found in %s", dir)
}

// extracts the language code from a subtitle filename.
// e.g., "video.en.vtt" -> "en"
func extractSubtitleLang(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "unknown"
}

func (s *Service) fetchChannelImages(ctx context.Context, channelDir, channelURL string) (string, string) {
	if channelURL == "" {
		return "", ""
	}

	avatarRel := findExistingImage(s.DataDir, channelDir, "avatar")
	bannerRel := findExistingImage(s.DataDir, channelDir, "banner")
	if avatarRel != "" && bannerRel != "" {
		return avatarRel, bannerRel
	}

	tmpDir, err := os.MkdirTemp(TmpDir(s.DataDir), "ch-*")
	if err != nil {
		return avatarRel, bannerRel
	}
	defer os.RemoveAll(tmpDir)

	outTmpl := filepath.Join(tmpDir, "%(thumbnail_id)s.%(ext)s")
	args := []string{
		"--write-all-thumbnails",
		"--skip-download",
		"--playlist-items", "0",
		"-o", outTmpl,
	}

	if s.Proxy != "" {
		args = append(args, "--proxy", s.Proxy)
	}

	cookiePath := "/app/cookies.txt"
	if _, err := os.Stat(cookiePath); err == nil {
		args = append(args, "--cookies", cookiePath)
	}

	args = append(args, channelURL)
	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	cmd.CombinedOutput()

	entries, _ := os.ReadDir(tmpDir)
	if bannerRel == "" {
		for _, e := range entries {
			if strings.Contains(e.Name(), "banner") {
				src := filepath.Join(tmpDir, e.Name())
				ext := filepath.Ext(e.Name())
				dst := filepath.Join(channelDir, "banner"+ext)
				if moveFile(src, dst) == nil {
					bannerRel, _ = filepath.Rel(s.DataDir, dst)
					bannerRel = filepath.ToSlash(bannerRel)
				}
				break
			}
		}
	}

	if avatarRel == "" {
		for _, e := range entries {
			if strings.Contains(e.Name(), "avatar") {
				src := filepath.Join(tmpDir, e.Name())
				ext := filepath.Ext(e.Name())
				dst := filepath.Join(channelDir, "avatar"+ext)
				if moveFile(src, dst) == nil {
					avatarRel, _ = filepath.Rel(s.DataDir, dst)
					avatarRel = filepath.ToSlash(avatarRel)
				}
				break
			}
		}
	}

	return avatarRel, bannerRel
}

func (s *Service) RefreshChannelMetadata(ctx context.Context, ch *domain.Channel) error {
	channelDir := filepath.Join(s.DataDir, "media", "channels", ch.YoutubeChannelID)
	if err := os.MkdirAll(channelDir, 0o755); err != nil {
		return fmt.Errorf("creating channel dir: %w", err)
	}

	for _, prefix := range []string{"avatar", "banner"} {
		for _, ext := range []string{"jpg", "png", "webp"} {
			os.Remove(filepath.Join(channelDir, prefix+"."+ext))
		}
	}

	channelURL := ch.URL
	if channelURL == "" {
		channelURL = "https://www.youtube.com/channel/" + ch.YoutubeChannelID
	}

	avatarRel, bannerRel := s.fetchChannelImages(ctx, channelDir, channelURL)

	args := []string{
		"--dump-json",
		"--playlist-items", "0",
		"--no-warnings",
	}
	if s.Proxy != "" {
		args = append(args, "--proxy", s.Proxy)
	}
	cookiePath := "/app/cookies.txt"
	if _, err := os.Stat(cookiePath); err == nil {
		args = append(args, "--cookies", cookiePath)
	}
	args = append(args, channelURL)

	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	output, _ := cmd.Output()
	if len(output) > 0 {
		var info struct {
			Channel    string `json:"channel"`
			ChannelID  string `json:"channel_id"`
			Uploader   string `json:"uploader"`
			UploaderID string `json:"uploader_id"`
		}
		if err := json.Unmarshal(output, &info); err == nil {
			if info.Channel != "" {
				ch.Name = info.Channel
			}
			if info.UploaderID != "" && strings.HasPrefix(info.UploaderID, "@") {
				ch.Handle = info.UploaderID
			}
		}
	}

	ch.ThumbnailPath = avatarRel
	ch.BannerPath = bannerRel

	_, err := s.Store.UpsertChannel(ch)
	return err
}

func findExistingImage(dataDir, dir, prefix string) string {
	for _, ext := range []string{"jpg", "png", "webp"} {
		p := filepath.Join(dir, prefix+"."+ext)
		if _, err := os.Stat(p); err == nil {
			rel, _ := filepath.Rel(dataDir, p)
			return filepath.ToSlash(rel)
		}
	}
	return ""
}

type PlaylistEntry struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Thumbnail   string  `json:"thumbnail"`
	Duration    float64 `json:"duration"`
	URL         string  `json:"url"`
	ReleaseDate string  `json:"release_date"`
	IsShort     bool    `json:"is_short"`
}

func (s *Service) FetchPlaylistEntries(ctx context.Context, url string) ([]PlaylistEntry, error) {
	args := []string{
		"--flat-playlist",
		"--dump-json",
		"--no-warnings",
	}

	if s.Proxy != "" {
		args = append(args, "--proxy", s.Proxy)
	}

	cookiePath := "/app/cookies.txt"
	if _, err := os.Stat(cookiePath); err == nil {
		args = append(args, "--cookies", cookiePath)
	}

	args = append(args, url)

	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp flat-playlist failed: %w", err)
	}

	var entries []PlaylistEntry
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		var raw struct {
			ID         string `json:"id"`
			Title      string `json:"title"`
			Thumbnails []struct {
				URL string `json:"url"`
			} `json:"thumbnails"`
			Thumbnail   string  `json:"thumbnail"`
			Duration    float64 `json:"duration"`
			URL         string  `json:"url"`
			WebpageURL  string  `json:"webpage_url"`
			ReleaseDate string  `json:"release_date"`
			UploadDate  string  `json:"upload_date"`
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		thumbnail := raw.Thumbnail
		if thumbnail == "" && len(raw.Thumbnails) > 0 {
			thumbnail = raw.Thumbnails[len(raw.Thumbnails)-1].URL
		}
		if thumbnail == "" {
			thumbnail = "https://i.ytimg.com/vi/" + raw.ID + "/hqdefault.jpg"
		}

		videoURL := raw.WebpageURL
		if videoURL == "" {
			videoURL = raw.URL
		}
		if videoURL == "" {
			videoURL = "https://www.youtube.com/watch?v=" + raw.ID
		}

		releaseDate := raw.ReleaseDate
		if releaseDate == "" {
			releaseDate = raw.UploadDate
		}

		entries = append(entries, PlaylistEntry{
			ID:          raw.ID,
			Title:       raw.Title,
			Thumbnail:   thumbnail,
			Duration:    raw.Duration,
			URL:         videoURL,
			ReleaseDate: releaseDate,
			IsShort:     strings.Contains(videoURL, "/shorts/"),
		})
	}

	return entries, nil
}

package archive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MathiasDPX/archivetube/internal/domain"
	"github.com/MathiasDPX/archivetube/internal/store"
)

type Service struct {
	YtDlpPath string
	DataDir   string
	Store     *store.Store
}

func New(ytdlpPath, dataDir string, st *store.Store) *Service {
	return &Service{
		YtDlpPath: ytdlpPath,
		DataDir:   dataDir,
		Store:     st,
	}
}

// ArchiveURL downloads a video and stores its metadata.
func (s *Service) ArchiveURL(ctx context.Context, url string) error {
	// 1. Create temp work dir
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

	// 2. Build yt-dlp command
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
		"-f", "bv*[ext=mp4]+ba[ext=m4a]/bv*+ba/b",
		"--merge-output-format", "mp4",
		url,
	}

	// 3. Run the command
	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %w\n%s", err, string(output))
	}

	// 4. Parse info.json
	infoPath := filepath.Join(tmpDir, "video.info.json")
	info, err := parseInfoJSON(infoPath)
	if err != nil {
		return fmt.Errorf("parsing info json: %w", err)
	}

	// Check if video is already archived
	existing, err := s.Store.GetVideoByYoutubeID(info.ID)
	if err != nil {
		return fmt.Errorf("checking existing video: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("video %s is already archived", info.ID)
	}

	// 5. Find the video file
	videoFile := filepath.Join(tmpDir, "video.mp4")
	if _, err := os.Stat(videoFile); err != nil {
		return fmt.Errorf("video file not found at %s: %w", videoFile, err)
	}

	// 6. Find subtitle files (video.*.vtt)
	subtitleFiles, err := filepath.Glob(filepath.Join(tmpDir, "video.*.vtt"))
	if err != nil {
		return fmt.Errorf("finding subtitle files: %w", err)
	}

	// 7. Find thumbnail file
	var thumbnailFile string
	for _, ext := range []string{"jpg", "png", "webp"} {
		candidate := filepath.Join(tmpDir, "video."+ext)
		if _, err := os.Stat(candidate); err == nil {
			thumbnailFile = candidate
			break
		}
	}

	// 8. Determine final media dir
	finalDir := MediaDir(s.DataDir, info.ChannelID, info.ID)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return fmt.Errorf("creating media dir %s: %w", finalDir, err)
	}

	// 9. Move files to final dir
	finalVideoPath := filepath.Join(finalDir, "video.mp4")
	if err := os.Rename(videoFile, finalVideoPath); err != nil {
		return fmt.Errorf("moving video file: %w", err)
	}

	// Move info.json
	finalInfoPath := filepath.Join(finalDir, "video.info.json")
	if err := os.Rename(infoPath, finalInfoPath); err != nil {
		return fmt.Errorf("moving info json: %w", err)
	}
	infoJSONRel, _ := filepath.Rel(s.DataDir, finalInfoPath)
	infoJSONRel = filepath.ToSlash(infoJSONRel)

	var finalThumbnailRel string
	if thumbnailFile != "" {
		thumbName := filepath.Base(thumbnailFile)
		finalThumbPath := filepath.Join(finalDir, thumbName)
		if err := os.Rename(thumbnailFile, finalThumbPath); err != nil {
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
		if err := os.Rename(sf, finalSubPath); err != nil {
			return fmt.Errorf("moving subtitle %s: %w", name, err)
		}
		rel, _ := filepath.Rel(s.DataDir, finalSubPath)
		rel = filepath.ToSlash(rel)
		subtitles = append(subtitles, subtitleEntry{language: lang, relPath: rel})
	}

	// Compute relative video path and file size
	videoRel, _ := filepath.Rel(s.DataDir, finalVideoPath)
	videoRel = filepath.ToSlash(videoRel)
	var fileSizeBytes int64
	if fi, err := os.Stat(finalVideoPath); err == nil {
		fileSizeBytes = fi.Size()
	}

	// Parse upload date
	uploadDate, err := time.Parse("20060102", info.UploadDate)
	if err != nil {
		return fmt.Errorf("parsing upload date %q: %w", info.UploadDate, err)
	}

	// 10. Upsert channel (with avatar + banner)
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

	// 11. Upsert video
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
		VideoExt:         "mp4",
		InfoJSONRelPath:  infoJSONRel,
		Width:            info.Width,
		Height:           info.Height,
		FileSizeBytes:    fileSizeBytes,
	}
	videoID, err := s.Store.UpsertVideo(video)
	if err != nil {
		return fmt.Errorf("upserting video: %w", err)
	}

	// 12. Replace chapters
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

	// 13. Replace subtitles
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

// extractSubtitleLang extracts the language code from a subtitle filename.
// e.g., "video.en.vtt" -> "en", "video.pt-BR.vtt" -> "pt-BR"
func extractSubtitleLang(filename string) string {
	name := strings.TrimSuffix(filename, filepath.Ext(filename)) // remove .vtt
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "unknown"
}

// fetchChannelImages uses yt-dlp to download the channel avatar and banner.
// It returns (avatarRelPath, bannerRelPath) relative to DataDir.
func (s *Service) fetchChannelImages(ctx context.Context, channelDir, channelURL string) (string, string) {
	if channelURL == "" {
		return "", ""
	}

	// Check if already downloaded
	avatarRel := findExistingImage(s.DataDir, channelDir, "avatar")
	bannerRel := findExistingImage(s.DataDir, channelDir, "banner")
	if avatarRel != "" && bannerRel != "" {
		return avatarRel, bannerRel
	}

	// Use yt-dlp to dump channel info JSON (no download)
	tmpDir, err := os.MkdirTemp(TmpDir(s.DataDir), "ch-*")
	if err != nil {
		return avatarRel, bannerRel
	}
	defer os.RemoveAll(tmpDir)

	outTmpl := filepath.Join(tmpDir, "channel.%(ext)s")
	args := []string{
		"--write-thumbnail",
		"--skip-download",
		"--playlist-items", "0",
		"-o", outTmpl,
		channelURL,
	}
	cmd := exec.CommandContext(ctx, s.YtDlpPath, args...)
	cmd.CombinedOutput()

	// yt-dlp writes avatar as channel.jpg/webp and banner as channel.banner_background.jpg/webp
	if avatarRel == "" {
		for _, ext := range []string{"jpg", "png", "webp"} {
			src := filepath.Join(tmpDir, "channel."+ext)
			if _, err := os.Stat(src); err == nil {
				dst := filepath.Join(channelDir, "avatar."+ext)
				if os.Rename(src, dst) == nil {
					avatarRel, _ = filepath.Rel(s.DataDir, dst)
					avatarRel = filepath.ToSlash(avatarRel)
				}
				break
			}
		}
	}

	if bannerRel == "" {
		entries, _ := os.ReadDir(tmpDir)
		for _, e := range entries {
			if strings.Contains(e.Name(), "banner") {
				src := filepath.Join(tmpDir, e.Name())
				ext := filepath.Ext(e.Name())
				dst := filepath.Join(channelDir, "banner"+ext)
				if os.Rename(src, dst) == nil {
					bannerRel, _ = filepath.Rel(s.DataDir, dst)
					bannerRel = filepath.ToSlash(bannerRel)
				}
				break
			}
		}
	}

	return avatarRel, bannerRel
}

// findExistingImage checks if an image with the given prefix already exists in dir.
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

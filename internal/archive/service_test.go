package archive

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindDownloadedVideo_mp4 verifies the common case: yt-dlp outputs video.mp4.
func TestFindDownloadedVideo_mp4(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "video.mp4")
	touch(t, dir, "video.info.json")
	touch(t, dir, "video.jpg")

	got, err := findDownloadedVideo(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "video.mp4"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFindDownloadedVideo_webm verifies that a .webm output (e.g., when ffmpeg is absent) is found.
func TestFindDownloadedVideo_webm(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "video.webm")
	touch(t, dir, "video.info.json")
	touch(t, dir, "video.en.vtt")

	got, err := findDownloadedVideo(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "video.webm"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFindDownloadedVideo_mkv verifies that a .mkv output is found.
func TestFindDownloadedVideo_mkv(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "video.mkv")
	touch(t, dir, "video.info.json")

	got, err := findDownloadedVideo(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := filepath.Join(dir, "video.mkv"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestFindDownloadedVideo_notFound verifies an error is returned when no video file exists.
func TestFindDownloadedVideo_notFound(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "video.info.json")
	touch(t, dir, "video.jpg")
	touch(t, dir, "video.en.vtt")

	_, err := findDownloadedVideo(dir)
	if err == nil {
		t.Fatal("expected error when no video file present, got nil")
	}
}

// TestFindDownloadedVideo_emptyDir verifies an error is returned for an empty directory.
func TestFindDownloadedVideo_emptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := findDownloadedVideo(dir)
	if err == nil {
		t.Fatal("expected error for empty directory, got nil")
	}
}

// TestFindDownloadedVideo_partFileIgnored verifies that .part files are not treated as video.
func TestFindDownloadedVideo_partFileIgnored(t *testing.T) {
	dir := t.TempDir()
	touch(t, dir, "video.mp4.part")
	touch(t, dir, "video.info.json")

	_, err := findDownloadedVideo(dir)
	if err == nil {
		t.Fatal("expected error when only .part file present, got nil")
	}
}

// TestExtractSubtitleLang covers the language extraction helper.
func TestExtractSubtitleLang(t *testing.T) {
	cases := []struct {
		filename string
		want     string
	}{
		{"video.en.vtt", "en"},
		{"video.pt-BR.vtt", "pt-BR"},
		{"video.zh-Hans.vtt", "zh-Hans"},
		{"video.vtt", "unknown"},
		{"noperiod", "unknown"},
	}
	for _, tc := range cases {
		got := extractSubtitleLang(tc.filename)
		if got != tc.want {
			t.Errorf("extractSubtitleLang(%q) = %q, want %q", tc.filename, got, tc.want)
		}
	}
}

// touch creates an empty file in dir with the given name.
func touch(t *testing.T, dir, name string) {
	t.Helper()
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("touch %s: %v", name, err)
	}
	f.Close()
}

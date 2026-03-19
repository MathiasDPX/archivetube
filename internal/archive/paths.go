package archive

import "path/filepath"

// MediaDir returns the path for a video's media directory.
// Pattern: <dataDir>/media/channels/<channelID>/<videoID>/
func MediaDir(dataDir, channelID, videoID string) string {
	return filepath.Join(dataDir, "media", "channels", channelID, videoID)
}

// TmpDir returns the temporary working directory.
func TmpDir(dataDir string) string {
	return filepath.Join(dataDir, "tmp")
}

package archive

import "path/filepath"

func MediaDir(dataDir, channelID, videoID string) string {
	return filepath.Join(dataDir, "media", "channels", channelID, videoID)
}

func TmpDir(dataDir string) string {
	return filepath.Join(dataDir, "tmp")
}

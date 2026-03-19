package config

import "os"

type Config struct {
	ListenAddr string
	DataDir    string
	YtDlpPath  string
}

func Load() *Config {
	c := &Config{
		ListenAddr: ":8080",
		DataDir:    "./data",
		YtDlpPath:  "yt-dlp",
	}
	if v := os.Getenv("ARCHIVETUBE_LISTEN"); v != "" {
		c.ListenAddr = v
	}
	if v := os.Getenv("ARCHIVETUBE_DATA_DIR"); v != "" {
		c.DataDir = v
	}
	if v := os.Getenv("ARCHIVETUBE_YTDLP_PATH"); v != "" {
		c.YtDlpPath = v
	}
	return c
}

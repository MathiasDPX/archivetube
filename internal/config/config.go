package config

import "os"

type Config struct {
	ListenAddr   string
	DataDir      string
	YtDlpPath    string
	PasswordHash string // bcrypt hash from ARCHIVETUBE_PASSWORD env var
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
	if v := os.Getenv("ARCHIVETUBE_PASSWORD"); v != "" {
		c.PasswordHash = v
	}
	return c
}

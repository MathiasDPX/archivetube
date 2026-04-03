package config

import "os"

type Config struct {
	ListenAddr   string
	DataDir      string
	YtDlpPath    string
	Proxy        string
	RealIPHeader string // HTTP Header with the real IP if behind a reverse proxy (like X-Forwarded-For)
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
	if v := os.Getenv("ARCHIVETUBE_PROXY"); v != "" {
		c.Proxy = v
	}
	if v := os.Getenv("ARCHIVETUBE_PASSWORD"); v != "" {
		c.PasswordHash = v
	}
	if v := os.Getenv("ARCHIVETUBE_IPHEADER"); v != "" {
		c.RealIPHeader = v
	}
	return c
}

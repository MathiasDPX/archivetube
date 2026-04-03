package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	ListenAddr   string `toml:"listen_addr"`
	RealIPHeader string `toml:"real_ip_header"`
}

type ArchiveConfig struct {
	YtDlpPath string `toml:"ytdlp_path"`
	DataDir   string `toml:"data_dir"`
	Proxy     string `toml:"proxy"`
}

type AuthConfig struct {
	Mode         string `toml:"mode"`
	PasswordHash string `toml:"password_hash"`

	// OIDC settings (used when mode = "oidc")
	OIDCIssuer       string `toml:"oidc_issuer"`
	OIDCClientID     string `toml:"oidc_client_id"`
	OIDCClientSecret string `toml:"oidc_client_secret"`
	OIDCRedirectURL  string `toml:"oidc_redirect_url"`
}

type Config struct {
	Server  ServerConfig  `toml:"server"`
	Archive ArchiveConfig `toml:"archive"`
	Auth    AuthConfig    `toml:"auth"`
}

func Load(path string) *Config {
	c := &Config{
		Server: ServerConfig{
			ListenAddr: ":8080",
		},
		Archive: ArchiveConfig{
			YtDlpPath: "yt-dlp",
			DataDir:   "./data",
		},
	}

	if _, err := toml.DecodeFile(path, c); err != nil {
		log.Fatalf("loading config file %s: %v", path, err)
	}

	return c
}

package config

import (
	"log"

	"github.com/BurntSushi/toml"
)

type ServerConfig struct {
	ListenAddr   string `toml:"listen_addr"`
	RealIPHeader string `toml:"real_ip_header"`
	CorsHost     string `toml:"cors_host"`
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

type DearrowConfig struct {
	Enabled     bool   `toml:"enable"`
	ApiURL      string `toml:"main_api"`
	ThumbApiURL string `toml:"thumb_api"`
}

type SmartSearchConfig struct {
	Enabled bool   `toml:"enabled"`
	ApiKey  string `toml:"apikey"`
}

type ObservabilityConfig struct {
	EnablePrometheus bool `toml:"prometheus"`
}

type Config struct {
	Server        ServerConfig        `toml:"server"`
	Archive       ArchiveConfig       `toml:"archive"`
	Auth          AuthConfig          `toml:"auth"`
	Observability ObservabilityConfig `toml:"observability"`
	SmartSearch   SmartSearchConfig   `toml:"smart_search"`
	Dearrow       DearrowConfig       `toml:"dearrow"`
}

func Load(path string) *Config {
	c := &Config{
		Server: ServerConfig{
			ListenAddr: ":8080",
			CorsHost:   "*",
		},
		Archive: ArchiveConfig{
			YtDlpPath: "yt-dlp",
			DataDir:   "./data",
		},
		Observability: ObservabilityConfig{
			EnablePrometheus: false,
		},
		SmartSearch: SmartSearchConfig{
			Enabled: false,
			ApiKey:  "archivetube // smart search enabled but no api key set",
		},
		Dearrow: DearrowConfig{
			Enabled:     false,
			ApiURL:      "https://sponsor.ajay.app",
			ThumbApiURL: "https://dearrow-thumb.ajay.app",
		},
	}

	if _, err := toml.DecodeFile(path, c); err != nil {
		log.Fatalf("loading config file %s: %v", path, err)
	}

	return c
}

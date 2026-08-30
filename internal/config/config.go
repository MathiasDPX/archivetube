package config

import (
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
)

type ApiPermission string

const (
	PermAll     ApiPermission = "*"
	PermDelete  ApiPermission = "delete"
	PermArchive ApiPermission = "archive"
	PermRefresh ApiPermission = "refresh"
)

var validPermissions = map[ApiPermission]struct{}{
	PermAll:     {},
	PermDelete:  {},
	PermArchive: {},
	PermRefresh: {},
}

type APIClientConfig struct {
	Key         string            `toml:"key"`
	Permissions apiPermissionList `toml:"-"`
	RawPerms    interface{}       `toml:"permissions"`
}

type apiPermissionList []ApiPermission

func (c *APIClientConfig) HasPermission(p ApiPermission) bool {
	for _, perm := range c.Permissions {
		if perm == PermAll || perm == p {
			return true
		}
	}
	return false
}

type ServerConfig struct {
	ListenAddr   string `toml:"listen_addr"`
	RealIPHeader string `toml:"real_ip_header"`
	CorsHost     string `toml:"cors_host"`
}

type ArchiveConfig struct {
	YtDlpPath         string   `toml:"ytdlp_path"`
	DataDir           string   `toml:"data_dir"`
	Proxy             string   `toml:"proxy"`
	AudioLanguages    []string `toml:"audio_languages"`
	SubtitleLanguages []string `toml:"subtitle_languages"`
}

type AuthConfig struct {
	Mode         string `toml:"mode"`
	PasswordHash string `toml:"password_hash"`

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
	Backend string `toml:"backend"`
	Model   string `toml:"model"`
}

type ObservabilityConfig struct {
	EnablePrometheus bool   `toml:"prometheus"`
	OTelExporter     string `toml:"otel_exporter_otlp_endpoint"`
}

type Config struct {
	Server        ServerConfig                `toml:"server"`
	Archive       ArchiveConfig               `toml:"archive"`
	Auth          AuthConfig                  `toml:"auth"`
	Observability ObservabilityConfig         `toml:"observability"`
	SmartSearch   SmartSearchConfig           `toml:"smart_search"`
	Dearrow       DearrowConfig               `toml:"dearrow"`
	API           map[string]*APIClientConfig `toml:"api"`
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
			Model:   "qwen/qwen3-embedding-8b",
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

	for name, client := range c.API {
		perms, err := parsePermissions(client.RawPerms)
		if err != nil {
			log.Fatalf("api client %q: %v", name, err)
		}
		client.Permissions = perms
	}

	return c
}

func parsePermissions(raw interface{}) ([]ApiPermission, error) {
	switch v := raw.(type) {
	case string:
		p := ApiPermission(v)
		if _, ok := validPermissions[p]; !ok {
			return nil, fmt.Errorf("invalid permission %q", v)
		}
		return []ApiPermission{p}, nil
	case []interface{}:
		perms := make([]ApiPermission, 0, len(v))
		for _, elem := range v {
			s, ok := elem.(string)
			if !ok {
				return nil, fmt.Errorf("permission must be a string, got %T", elem)
			}
			p := ApiPermission(s)
			if _, ok := validPermissions[p]; !ok {
				return nil, fmt.Errorf("invalid permission %q", s)
			}
			perms = append(perms, p)
		}
		return perms, nil
	case nil:
		return nil, nil
	default:
		return nil, fmt.Errorf("permissions must be a string or array, got %T", raw)
	}
}

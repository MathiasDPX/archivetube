# Extra configuration

The code for configuration can be found on <a href="https://github.com/MathiasDPX/archivetube/blob/main/internal/config/config.go" target="_blank">`internal/config/config.go`</a>

_This page assume you have basic <a href="https://toml.io/fr/" target="_blank">TOML<a/> knowledge_

## Proxy

ArchiveTube support HTTP, HTTPS and SOCKS proxy

```toml
[archive]
proxy = "protocol://username:password@ip:port"
```

## Reverse proxy

If you expose your ArchiveTube instance, you may want to put it behind a reverse proxy.

```toml
[server]
real_ip_header = "header"
```

| Service    | Header           |
| ---------- | ---------------- |
| Nginx      | X-Forwarded-For  |
| Cloudflare | CF-Connecting-IP |
| Popular    | X-Real-IP        |


## CORS

If you want to use your archived videos on another site without making your data accessible to everyone, you can setup <a href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS" target="_blank">CORS</a>. Default is set to `*`

```toml
[server]
cors_host = "https://yt.mathiasd.fr"
```

# yt-dlp path

You can change the yt-dlp by adding a ytdlp_path argument to the archive section

```toml
[archive]
ytdlp_path = "/usr/local/bin/yt-dlp"
```

Note: this can be used to add custom argument to yt-dlp by making a middleman bash script. This hasn't been tested tho, custom argument support will be added one day

```bash
yt-dlp --quiet "$@"
```

Here, `--quiet` is added to all yt-dlp commands
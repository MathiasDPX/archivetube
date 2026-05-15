# Extra configuration

The configuration code can be found at <a href="https://github.com/MathiasDPX/archivetube/blob/main/internal/config/config.go" target="_blank">`internal/config/config.go`</a>.

_This page assumes you have basic <a href="https://toml.io/fr/" target="_blank">TOML</a> knowledge._

## Proxy

ArchiveTube supports HTTP, HTTPS, and SOCKS proxies.

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

If you want to use your archived videos on another site without making your data accessible to everyone, you can set up <a href="https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS" target="_blank">CORS</a>. The default is set to `*`.

```toml
[server]
cors_host = "https://yt.mathiasd.fr"
```

# yt-dlp path

You can change the yt-dlp path by adding a `ytdlp_path` argument to the archive section.

```toml
[archive]
ytdlp_path = "/usr/local/bin/yt-dlp"
```

Note: this can be used to add custom arguments to yt-dlp by creating a middleman bash script. This has not been tested, though; custom argument support will be added one day.

```bash
yt-dlp --quiet "$@"
```

Here, `--quiet` is added to all yt-dlp commands.

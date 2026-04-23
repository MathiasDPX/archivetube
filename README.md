<div align="center">
<br>
  <img src="https://raw.githubusercontent.com/MathiasDPX/archivetube/refs/heads/main/web/static/favicon.svg" alt="ArchiveTube Logo" height="200"/>
  
  # ArchiveTube 
  
A self-hosted YouTube archiving application made for high-quality and complete archives
</div>

## Features

- Authentication with password, oidc or none
- Full video archiving (video, thumbnail, subtitles, description...)
- Batch archiving of playlists or channels in one go
- YouTube-like interface
- rclone compatible, local file-based storage works with rclone mount 

## Installation

1. Create a `config.toml`

```toml
[server]
listen_addr = ":8080"

[auth]
mode = "password"
password_hash = "bcrypt-password"
```

2. And that's all, now you can start ArchiveTube with Docker Compose

```yml
services:
  archivetube:
    container_name: archivetube
    image: ghcr.io/mathiasdpx/archivetube:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./config.toml:/app/config.toml
```

You can find the full installation guide in the [documentation](https://mathiasdpx.github.io/archivetube/admin/installation.html)

# Development

Follow the 1st step of the normal installation for setting up the confnig and start 

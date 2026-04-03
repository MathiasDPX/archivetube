# archivetube

A self-hosted YouTube archiving application

## Installation

### Configuration

Env variables:

| Variable                   | Default      | Description                    |
|----------------------------|--------------|--------------------------------|
| `ARCHIVETUBE_LISTEN`       | `:8080`      | Address to listen on           |
| `ARCHIVETUBE_DATA_DIR`     | `/app/data`  | Directory for data and media   |
| `ARCHIVETUBE_YTDLP_PATH`   | `yt-dlp`     | Path or command for yt-dlp     |
| `ARCHIVETUBE_PROXY`        | None         | Proxy URL for yt-dlp           |
| `ARCHIVETUBE_PASSWORD`     | None         | bcrypt password for login      |

Save theses variables inside a `.env` file to save them

### Using Docker

```bash
docker pull ghcr.io/mathiasdpx/archivetube:latest

docker run -d \
  -p 8080:8080 \
  -v archivetube-data:/app/data \
  --name archivetube \
  --env-file ./.env
  ghcr.io/mathiasdpx/archivetube:latest
```

### Using Docker Compose

```yaml
archivetube:
    container_name: archivetube
    image: ghcr.io/mathiasdpx/archivetube:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
#     - ./cookies.txt:/app/cookies.txt # For using your account's cookies with yt-dlp - Not recommended
    env_file: .env
```

Then open [http://localhost:8080](http://localhost:8080)
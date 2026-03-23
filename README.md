# archivetube

A self-hosted YouTube archiving application

## Docker

### Configuration

Env variables:

| Variable                   | Default      | Description                    |
|----------------------------|--------------|--------------------------------|
| `ARCHIVETUBE_LISTEN`       | `:8080`      | Address to listen on           |
| `ARCHIVETUBE_DATA_DIR`     | `/app/data`  | Directory for data and media   |
| `ARCHIVETUBE_YTDLP_PATH`   | `yt-dlp`     | Path or command for yt-dlp     |
| `ARCHIVETUBE_PASSWORD`     | None         | bcrypt password for login      |


```bash
docker pull ghcr.io/mathiasdpx/archivetube:latest

docker run -d \
  -p 8080:8080 \
  -v archivetube-data:/app/data \
  --name archivetube \
  ghcr.io/mathiasdpx/archivetube:latest
```

Then open [http://localhost:8080](http://localhost:8080).
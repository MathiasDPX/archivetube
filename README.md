# archivetube

A self-hosted YouTube archiving application

## Installation

### Configuration

Create a `config.toml` file with this inside:

```toml
[server]
listen_addr = ":8080"
real_ip_header = ""

[archive]
ytdlp_path = "yt-dlp"
data_dir = "./data"
proxy = ""

[auth]
mode = "password" # "password" or "oidc"
password_hash = "bcrypt-generator"

#oidc_issuer = "https://auth.mathiasd.fr"
#oidc_client_id = ""
#oidc_client_secret = ""
#oidc_redirect_url = ""
```

### Using Docker

```bash
docker pull ghcr.io/mathiasdpx/archivetube:latest

docker run -d \
  -p 8080:8080 \
  -v ./data:/app/data \
  -v ./config.toml:/app/config.toml \
  --name archivetube \
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
      - ./config.toml:/app/config.toml
#     - ./cookies.txt:/app/cookies.txt # For using your account's cookies with yt-dlp - Not recommended
```

Then open [http://localhost:8080](http://localhost:8080)
# Installation


## Configuration

Before starting a container, you'll need to write a config file named `config.toml`, here's a minimal configuration, you'll need to change the password_hash to a bcrypt password. See how to manage auth [here](./auth.md#password)

```toml
[server]
listen_addr = ":8080"

[auth]
mode = "password"
password_hash = "bcrypt-password"
```

Now you, need to start a container using Docker. You can find features to extend your configuration in the Admin guide's [features](./) and [extra configuration](./extra-config.md)

## Running

### Docker Compose (recommended)

```yaml
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

### Docker

```bash
docker pull ghcr.io/mathiasdpx/archivetube:latest

docker run -d \
  -p 8080:8080 \
  -v ./data:/app/data \
  -v ./config.toml:/app/config.toml \
  --name archivetube \
  ghcr.io/mathiasdpx/archivetube:latest
```

### Podman

You might be able to run ArchiveTube with [Podman](https://podman.io/) but this has not been tested yet.


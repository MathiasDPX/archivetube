# Build stage
FROM golang:1.27-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends build-essential && rm -rf /var/lib/apt/lists/*

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o archivetube .

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

ARG GIT_SHA
ENV GIT_SHA=$GIT_SHA

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*

RUN apt-get update && apt-get install -y --no-install-recommends \
    python3 python3-pip pipx curl unzip \
    && PIPX_BIN_DIR=/usr/local/bin pipx install yt-dlp \
    && yt-dlp --version \
    && rm -rf /var/lib/apt/lists/*

# Install deno as the local JS runtime for yt-dlp (full YouTube format extraction
# and avoids throttling on server/datacenter IPs). The deno.land install.sh is
# unreliable in Docker, so download the release binary directly from GitHub.
ARG DENO_VERSION=2.9.6
RUN case "$(dpkg --print-architecture)" in \
        amd64)  DENO_ARCH=x86_64 ;; \
        arm64)  DENO_ARCH=aarch64 ;; \
        *)      echo "unsupported arch: $TARGETARCH" >&2; exit 1 ;; \
    esac \
    && curl -fsSL -o /tmp/deno.zip "https://github.com/denoland/deno/releases/download/v${DENO_VERSION}/deno-${DENO_ARCH}-unknown-linux-gnu.zip" \
    && unzip /tmp/deno.zip -d /tmp/deno \
    && install -m 0755 /tmp/deno/deno /usr/local/bin/deno \
    && rm -rf /tmp/deno /tmp/deno.zip
ENV PATH="${PATH}:/usr/local/bin"

COPY --from=builder /build/archivetube /app/archivetube
COPY web/ /app/web/

VOLUME /app/data

ENV ARCHIVETUBE_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/archivetube"]

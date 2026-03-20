# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o archivetube ./cmd/archivetube

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    ffmpeg \
    wget \
    && rm -rf /var/lib/apt/lists/* \
    && case "${TARGETARCH}" in \
        amd64)  wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" ;; \
        arm64)  wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_aarch64" ;; \
        arm*)   wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l" ;; \
        *)      wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" ;; \
    esac \
    && chmod +x /usr/local/bin/yt-dlp

COPY --from=builder /build/archivetube /app/archivetube
COPY web/ /app/web/

VOLUME /app/data

ENV ARCHIVETUBE_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/archivetube"]

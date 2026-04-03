# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o archivetube .

# Final stage
FROM debian:bookworm-slim

WORKDIR /app

ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    unzip \
    wget \
    xz-utils \
    && rm -rf /var/lib/apt/lists/* \
    && case "${TARGETARCH}" in \
        amd64) \
            wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" \
            && wget -O /tmp/ffmpeg.tar.xz "https://github.com/yt-dlp/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-linux64-gpl.tar.xz" \
            && wget -O /tmp/deno.zip "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-unknown-linux-gnu.zip" \
            ;; \
        arm64) \
            wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_aarch64" \
            && wget -O /tmp/ffmpeg.tar.xz "https://github.com/yt-dlp/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-linuxarm64-gpl.tar.xz" \
            && wget -O /tmp/deno.zip "https://github.com/denoland/deno/releases/latest/download/deno-aarch64-unknown-linux-gnu.zip" \
            ;; \
        arm*) \
            wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux_armv7l" \
            && wget -O /tmp/ffmpeg.tar.xz "https://github.com/yt-dlp/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-linuxarm64-gpl.tar.xz" \
            ;; \
        *) \
            wget -O /usr/local/bin/yt-dlp "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp_linux" \
            && wget -O /tmp/ffmpeg.tar.xz "https://github.com/yt-dlp/FFmpeg-Builds/releases/latest/download/ffmpeg-master-latest-linux64-gpl.tar.xz" \
            && wget -O /tmp/deno.zip "https://github.com/denoland/deno/releases/latest/download/deno-x86_64-unknown-linux-gnu.zip" \
            ;; \
    esac \
    && tar -xJf /tmp/ffmpeg.tar.xz -C /tmp \
    && find /tmp -name "ffmpeg" -type f -exec mv {} /usr/local/bin/ffmpeg \; \
    && if [ -f /tmp/deno.zip ]; then unzip -o /tmp/deno.zip -d /usr/local/bin && rm /tmp/deno.zip; fi \
    && chmod +x /usr/local/bin/yt-dlp /usr/local/bin/ffmpeg \
    && rm -rf /tmp/ffmpeg* \
    && apt-get purge -y unzip wget xz-utils \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /build/archivetube /app/archivetube
COPY web/ /app/web/

VOLUME /app/data

ENV ARCHIVETUBE_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/archivetube"]

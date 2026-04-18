# Build stage
FROM golang:1.26-bookworm AS builder

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
    python3 python3-pip pipx \
    && pipx install yt-dlp \
    && rm -rf /var/lib/apt/lists/*
ENV PATH="/root/.local/bin:$PATH"

COPY --from=builder /build/archivetube /app/archivetube
COPY web/ /app/web/

VOLUME /app/data

ENV ARCHIVETUBE_DATA_DIR=/app/data

EXPOSE 8080

ENTRYPOINT ["/app/archivetube"]

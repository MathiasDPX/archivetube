CREATE TABLE IF NOT EXISTS channels (
    id                 INTEGER PRIMARY KEY,
    youtube_channel_id TEXT    UNIQUE NOT NULL,
    handle             TEXT    NOT NULL DEFAULT '',
    name               TEXT    NOT NULL DEFAULT '',
    url                TEXT    NOT NULL DEFAULT '',
    description        TEXT    NOT NULL DEFAULT '',
    thumbnail_path     TEXT    NOT NULL DEFAULT '',
    banner_path        TEXT    NOT NULL DEFAULT '',
    created_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS videos (
    id                  INTEGER PRIMARY KEY,
    youtube_video_id    TEXT    UNIQUE NOT NULL,
    channel_id          INTEGER NOT NULL REFERENCES channels(id),
    title               TEXT    NOT NULL DEFAULT '',
    description         TEXT    NOT NULL DEFAULT '',
    duration_seconds    INTEGER NOT NULL DEFAULT 0,
    published_at        DATETIME,
    archived_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    webpage_url         TEXT    NOT NULL DEFAULT '',
    video_rel_path      TEXT    NOT NULL DEFAULT '',
    video_ext           TEXT    NOT NULL DEFAULT '',
    thumbnail_rel_path  TEXT    NOT NULL DEFAULT '',
    info_json_rel_path  TEXT    NOT NULL DEFAULT '',
    file_size_bytes     INTEGER NOT NULL DEFAULT 0,
    width               INTEGER NOT NULL DEFAULT 0,
    height              INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS video_chapters (
    id            INTEGER PRIMARY KEY,
    video_id      INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    position      INTEGER NOT NULL,
    title         TEXT    NOT NULL DEFAULT '',
    start_seconds REAL    NOT NULL DEFAULT 0,
    end_seconds   REAL    NOT NULL DEFAULT 0,
    UNIQUE(video_id, position)
);

CREATE TABLE IF NOT EXISTS video_subtitles (
    id            INTEGER PRIMARY KEY,
    video_id      INTEGER NOT NULL REFERENCES videos(id) ON DELETE CASCADE,
    language_code TEXT    NOT NULL DEFAULT '',
    language_name TEXT    NOT NULL DEFAULT '',
    ext           TEXT    NOT NULL DEFAULT '',
    rel_path      TEXT    NOT NULL DEFAULT '',
    is_default    INTEGER NOT NULL DEFAULT 0,
    UNIQUE(video_id, language_code)
);

CREATE INDEX IF NOT EXISTS idx_videos_archived_at ON videos(archived_at DESC);
CREATE INDEX IF NOT EXISTS idx_videos_channel_archived ON videos(channel_id, archived_at DESC);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);

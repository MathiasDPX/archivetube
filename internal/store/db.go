package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/MathiasDPX/archivetube/internal/config"
	_ "github.com/duckdb/duckdb-go/v2"
	"go.opentelemetry.io/otel"
	_ "modernc.org/sqlite"
)

var tracer = otel.Tracer("github.com/MathiasDPX/archivetube/internal/store")

const migrationSQL = `
CREATE SEQUENCE IF NOT EXISTS channels_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS videos_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS video_chapters_id_seq START 1;
CREATE SEQUENCE IF NOT EXISTS video_subtitles_id_seq START 1;

CREATE TABLE IF NOT EXISTS channels (
    id                 BIGINT PRIMARY KEY DEFAULT nextval('channels_id_seq'),
    youtube_channel_id TEXT    UNIQUE NOT NULL,
    handle             TEXT    NOT NULL DEFAULT '',
    name               TEXT    NOT NULL DEFAULT '',
    url                TEXT    NOT NULL DEFAULT '',
    description        TEXT    NOT NULL DEFAULT '',
    thumbnail_path     TEXT    NOT NULL DEFAULT '',
    banner_path        TEXT    NOT NULL DEFAULT '',
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS videos (
    id                  BIGINT PRIMARY KEY DEFAULT nextval('videos_id_seq'),
    youtube_video_id    TEXT    UNIQUE NOT NULL,
    channel_id          BIGINT NOT NULL,
    title               TEXT    NOT NULL DEFAULT '',
    description         TEXT    NOT NULL DEFAULT '',
    duration_seconds    INTEGER NOT NULL DEFAULT 0,
    published_at        TIMESTAMP,
    archived_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    webpage_url         TEXT    NOT NULL DEFAULT '',
    video_rel_path      TEXT    NOT NULL DEFAULT '',
    video_ext           TEXT    NOT NULL DEFAULT '',
    thumbnail_rel_path  TEXT    NOT NULL DEFAULT '',
    info_json_rel_path  TEXT    NOT NULL DEFAULT '',
    file_size_bytes     BIGINT  NOT NULL DEFAULT 0,
    width               INTEGER NOT NULL DEFAULT 0,
    height              INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS video_chapters (
    id            BIGINT PRIMARY KEY DEFAULT nextval('video_chapters_id_seq'),
    video_id      BIGINT NOT NULL,
    position      INTEGER NOT NULL,
    title         TEXT    NOT NULL DEFAULT '',
    start_seconds REAL    NOT NULL DEFAULT 0,
    end_seconds   REAL    NOT NULL DEFAULT 0,
    UNIQUE(video_id, position)
);

CREATE TABLE IF NOT EXISTS video_subtitles (
    id            BIGINT PRIMARY KEY DEFAULT nextval('video_subtitles_id_seq'),
    video_id      BIGINT NOT NULL,
    language_code TEXT    NOT NULL DEFAULT '',
    language_name TEXT    NOT NULL DEFAULT '',
    ext           TEXT    NOT NULL DEFAULT '',
    rel_path      TEXT    NOT NULL DEFAULT '',
    is_default    INTEGER NOT NULL DEFAULT 0,
    UNIQUE(video_id, language_code)
);

CREATE TABLE IF NOT EXISTS videos_vectors (
    youtube_video_id TEXT PRIMARY KEY,
    title_vec FLOAT[384],
    description_vec FLOAT[384]
);

CREATE INDEX IF NOT EXISTS idx_videos_archived_at ON videos(archived_at);
CREATE INDEX IF NOT EXISTS idx_videos_channel_archived ON videos(channel_id, archived_at);
CREATE INDEX IF NOT EXISTS idx_channels_name ON channels(name);
`

type Store struct {
	db  *sql.DB
	cfg *config.Config
}

func New(dbPath string, cfg *config.Config) (*Store, error) {
	// If the file exists and is a SQLite database, migrate it to DuckDB in place
	needsMigration := false
	sqliteBackup := dbPath + ".sqlite-backup"
	if _, err := os.Stat(dbPath); err == nil {
		if isSQLiteFile(dbPath) {
			needsMigration = true
			if err := os.Rename(dbPath, sqliteBackup); err != nil {
				return nil, fmt.Errorf("backing up SQLite database: %w", err)
			}
		}
	}

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(migrationSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	s := &Store{db: db, cfg: cfg}

	// Always ensure sequences are ahead of existing max IDs
	// (prevents PK collisions after DB restart which DuckDB treats as
	// DELETE+INSERT, violating foreign key constraints)
	for _, r := range []struct{ seq, table, col string }{
		{"channels_id_seq", "channels", "id"},
		{"videos_id_seq", "videos", "id"},
		{"video_chapters_id_seq", "video_chapters", "id"},
		{"video_subtitles_id_seq", "video_subtitles", "id"},
	} {
		var maxID sql.NullInt64
		if err := db.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s", r.col, r.table)).Scan(&maxID); err == nil && maxID.Valid && maxID.Int64 > 0 {
			db.Exec(fmt.Sprintf("DROP SEQUENCE IF EXISTS %s", r.seq))
			db.Exec(fmt.Sprintf("CREATE SEQUENCE %s START %d", r.seq, maxID.Int64+1))
		}
	}

	if needsMigration {
		log.Printf("found existing SQLite database, migrating to DuckDB...")
		if err := s.migrateFromSQLite(sqliteBackup); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrating from SQLite: %w", err)
		}
		log.Printf("migration from SQLite completed successfully (backup at %s)", sqliteBackup)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// checks if the file starts with the SQLite magic header
func isSQLiteFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	header := make([]byte, 16)
	if _, err := f.Read(header); err != nil {
		return false
	}
	return string(header[:13]) == "SQLite format"
}

// copies all data from an existing SQLite database into DuckDB
func (s *Store) migrateFromSQLite(sqlitePath string) error {
	srcDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return fmt.Errorf("opening SQLite database: %w", err)
	}
	defer srcDB.Close()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Migrate channels
	rows, err := srcDB.Query(`SELECT id, youtube_channel_id, handle, name, url, description, thumbnail_path, COALESCE(banner_path, ''), created_at, updated_at FROM channels`)
	if err != nil {
		return fmt.Errorf("reading channels: %w", err)
	}
	for rows.Next() {
		var id int64
		var ytID, handle, name, url, desc, thumb, banner string
		var createdAt, updatedAt sql.NullString
		if err := rows.Scan(&id, &ytID, &handle, &name, &url, &desc, &thumb, &banner, &createdAt, &updatedAt); err != nil {
			rows.Close()
			return err
		}
		_, err = tx.Exec(`INSERT INTO channels (id, youtube_channel_id, handle, name, url, description, thumbnail_path, banner_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, ytID, handle, name, url, desc, thumb, banner, createdAt, updatedAt)
		if err != nil {
			rows.Close()
			return fmt.Errorf("inserting channel %d: %w", id, err)
		}
	}
	rows.Close()

	// Migrate videos
	rows, err = srcDB.Query(`SELECT id, youtube_video_id, channel_id, title, description, duration_seconds, published_at, archived_at, webpage_url, video_rel_path, video_ext, thumbnail_rel_path, info_json_rel_path, file_size_bytes, width, height FROM videos`)
	if err != nil {
		return fmt.Errorf("reading videos: %w", err)
	}
	for rows.Next() {
		var id, channelID, fileSizeBytes int64
		var ytID, title, desc, webpageURL, videoRelPath, videoExt, thumbRelPath, infoJSONRelPath string
		var durationSeconds, width, height int
		var publishedAt, archivedAt sql.NullString
		if err := rows.Scan(&id, &ytID, &channelID, &title, &desc, &durationSeconds, &publishedAt, &archivedAt, &webpageURL, &videoRelPath, &videoExt, &thumbRelPath, &infoJSONRelPath, &fileSizeBytes, &width, &height); err != nil {
			rows.Close()
			return err
		}
		_, err = tx.Exec(`INSERT INTO videos (id, youtube_video_id, channel_id, title, description, duration_seconds, published_at, archived_at, webpage_url, video_rel_path, video_ext, thumbnail_rel_path, info_json_rel_path, file_size_bytes, width, height) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, ytID, channelID, title, desc, durationSeconds, publishedAt, archivedAt, webpageURL, videoRelPath, videoExt, thumbRelPath, infoJSONRelPath, fileSizeBytes, width, height)
		if err != nil {
			rows.Close()
			return fmt.Errorf("inserting video %d: %w", id, err)
		}
	}
	rows.Close()

	// Migrate chapters
	rows, err = srcDB.Query(`SELECT id, video_id, position, title, start_seconds, end_seconds FROM video_chapters`)
	if err != nil {
		return fmt.Errorf("reading chapters: %w", err)
	}
	for rows.Next() {
		var id, videoID int64
		var position int
		var title string
		var startSeconds, endSeconds float64
		if err := rows.Scan(&id, &videoID, &position, &title, &startSeconds, &endSeconds); err != nil {
			rows.Close()
			return err
		}
		_, err = tx.Exec(`INSERT INTO video_chapters (id, video_id, position, title, start_seconds, end_seconds) VALUES (?, ?, ?, ?, ?, ?)`,
			id, videoID, position, title, startSeconds, endSeconds)
		if err != nil {
			rows.Close()
			return fmt.Errorf("inserting chapter %d: %w", id, err)
		}
	}
	rows.Close()

	// Migrate subtitles
	rows, err = srcDB.Query(`SELECT id, video_id, language_code, language_name, ext, rel_path, is_default FROM video_subtitles`)
	if err != nil {
		return fmt.Errorf("reading subtitles: %w", err)
	}
	for rows.Next() {
		var id, videoID int64
		var languageCode, languageName, ext, relPath string
		var isDefault int
		if err := rows.Scan(&id, &videoID, &languageCode, &languageName, &ext, &relPath, &isDefault); err != nil {
			rows.Close()
			return err
		}
		_, err = tx.Exec(`INSERT INTO video_subtitles (id, video_id, language_code, language_name, ext, rel_path, is_default) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, videoID, languageCode, languageName, ext, relPath, isDefault)
		if err != nil {
			rows.Close()
			return fmt.Errorf("inserting subtitle %d: %w", id, err)
		}
	}
	rows.Close()

	if err := tx.Commit(); err != nil {
		return err
	}

	// Reset sequences to continue after the max imported IDs
	resetSeqs := []struct{ seq, table, col string }{
		{"channels_id_seq", "channels", "id"},
		{"videos_id_seq", "videos", "id"},
		{"video_chapters_id_seq", "video_chapters", "id"},
		{"video_subtitles_id_seq", "video_subtitles", "id"},
	}
	for _, r := range resetSeqs {
		var maxID sql.NullInt64
		if err := s.db.QueryRow(fmt.Sprintf("SELECT MAX(%s) FROM %s", r.col, r.table)).Scan(&maxID); err != nil {
			return err
		}
		if maxID.Valid && maxID.Int64 > 0 {
			// Drop and recreate the sequence starting after the max ID
			s.db.Exec(fmt.Sprintf("DROP SEQUENCE %s", r.seq))
			s.db.Exec(fmt.Sprintf("CREATE SEQUENCE %s START %d", r.seq, maxID.Int64+1))
		}
	}

	return nil
}

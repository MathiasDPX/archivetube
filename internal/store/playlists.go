package store

import (
	"context"
	"database/sql"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) CreatePlaylist(ctx context.Context, name, sourceURL, ytPlaylistID string) (int64, error) {
	ctx, span := tracer.Start(ctx, "store.CreatePlaylist")
	defer span.End()

	// If a playlist with the same youtube_playlist_id already exists, return its ID
	if ytPlaylistID != "" {
		var existing int64
		err := s.db.QueryRow(`SELECT id FROM playlists WHERE youtube_playlist_id = ?`, ytPlaylistID).Scan(&existing)
		if err == nil {
			return existing, nil
		}
		if err != sql.ErrNoRows {
			return 0, err
		}
	}

	if _, err := s.db.Exec(
		`INSERT INTO playlists (name, source_url, youtube_playlist_id, updated_at) VALUES (?, ?, ?, now())`,
		name, sourceURL, ytPlaylistID,
	); err != nil {
		return 0, err
	}

	var id int64
	if err := s.db.QueryRow(
		`SELECT id FROM playlists WHERE name = ? AND source_url = ? AND youtube_playlist_id = ? ORDER BY id DESC LIMIT 1`,
		name, sourceURL, ytPlaylistID,
	).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) RenamePlaylist(ctx context.Context, id int64, name string) error {
	ctx, span := tracer.Start(ctx, "store.RenamePlaylist")
	defer span.End()

	_, err := s.db.Exec(`UPDATE playlists SET name = ?, updated_at = now() WHERE id = ?`, name, id)
	return err
}

func (s *Store) DeletePlaylist(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "store.DeletePlaylist")
	defer span.End()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM playlist_videos WHERE playlist_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM playlists WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetPlaylist(ctx context.Context, id int64) (*domain.Playlist, error) {
	ctx, span := tracer.Start(ctx, "store.GetPlaylist")
	defer span.End()

	p := &domain.Playlist{}
	err := s.db.QueryRow(`
		SELECT id, name, source_url, youtube_playlist_id, created_at, updated_at
		FROM playlists WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.SourceURL, &p.YoutubePlaylistID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var count sql.NullInt64
	s.db.QueryRow(`SELECT COUNT(*) FROM playlist_videos WHERE playlist_id = ?`, id).Scan(&count)
	if count.Valid {
		p.VideoCount = int(count.Int64)
	}
	return p, nil
}

func (s *Store) GetPlaylistByYoutubeID(ctx context.Context, ytPlaylistID string) (*domain.Playlist, error) {
	ctx, span := tracer.Start(ctx, "store.GetPlaylistByYoutubeID")
	defer span.End()

	if ytPlaylistID == "" {
		return nil, nil
	}
	p := &domain.Playlist{}
	err := s.db.QueryRow(`
		SELECT id, name, source_url, youtube_playlist_id, created_at, updated_at
		FROM playlists WHERE youtube_playlist_id = ?`, ytPlaylistID,
	).Scan(&p.ID, &p.Name, &p.SourceURL, &p.YoutubePlaylistID, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) ListPlaylists(ctx context.Context) ([]domain.Playlist, error) {
	ctx, span := tracer.Start(ctx, "store.ListPlaylists")
	defer span.End()

	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.source_url, p.youtube_playlist_id, p.created_at, p.updated_at,
			(SELECT COUNT(*) FROM playlist_videos pv WHERE pv.playlist_id = p.id) AS video_count,
			COALESCE((
				SELECT v.thumbnail_rel_path
				FROM playlist_videos pv
				JOIN videos v ON v.id = pv.video_id
				WHERE pv.playlist_id = p.id
				ORDER BY pv.position ASC, pv.added_at ASC
				LIMIT 1
			), '') AS thumb
		FROM playlists p
		ORDER BY p.updated_at DESC, p.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Playlist
	for rows.Next() {
		var p domain.Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.SourceURL, &p.YoutubePlaylistID,
			&p.CreatedAt, &p.UpdatedAt, &p.VideoCount, &p.ThumbnailRelPath); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) AddVideoToPlaylist(ctx context.Context, playlistID, videoID int64) error {
	ctx, span := tracer.Start(ctx, "store.AddVideoToPlaylist")
	defer span.End()

	// Check if it already exists
	var exists int
	err := s.db.QueryRow(
		`SELECT 1 FROM playlist_videos WHERE playlist_id = ? AND video_id = ?`,
		playlistID, videoID,
	).Scan(&exists)
	if err == nil {
		return nil // already in playlist
	}
	if err != sql.ErrNoRows {
		return err
	}

	// Find next position
	var nextPos sql.NullInt64
	if err := s.db.QueryRow(
		`SELECT COALESCE(MAX(position), 0) + 1 FROM playlist_videos WHERE playlist_id = ?`,
		playlistID,
	).Scan(&nextPos); err != nil {
		return err
	}

	pos := int64(1)
	if nextPos.Valid {
		pos = nextPos.Int64
	}

	if _, err := s.db.Exec(
		`INSERT INTO playlist_videos (playlist_id, video_id, position) VALUES (?, ?, ?)`,
		playlistID, videoID, pos,
	); err != nil {
		return err
	}

	_, err = s.db.Exec(`UPDATE playlists SET updated_at = now() WHERE id = ?`, playlistID)
	return err
}

func (s *Store) RemoveVideoFromPlaylist(ctx context.Context, playlistID, videoID int64) error {
	ctx, span := tracer.Start(ctx, "store.RemoveVideoFromPlaylist")
	defer span.End()

	if _, err := s.db.Exec(
		`DELETE FROM playlist_videos WHERE playlist_id = ? AND video_id = ?`,
		playlistID, videoID,
	); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE playlists SET updated_at = now() WHERE id = ?`, playlistID)
	return err
}

func (s *Store) ListPlaylistVideos(ctx context.Context, playlistID int64, limit, offset int) ([]domain.Video, int, error) {
	ctx, span := tracer.Start(ctx, "store.ListPlaylistVideos")
	defer span.End()

	var total int
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM playlist_videos WHERE playlist_id = ?`, playlistID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM playlist_videos pv
		JOIN videos v ON v.id = pv.video_id
		JOIN channels c ON c.id = v.channel_id
		WHERE pv.playlist_id = ?
		ORDER BY pv.position ASC, pv.added_at ASC
		LIMIT ? OFFSET ?`, playlistID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var videos []domain.Video
	for rows.Next() {
		var v domain.Video
		if err := rows.Scan(&v.ID, &v.YoutubeVideoID, &v.ChannelID, &v.Title, &v.Description,
			&v.DurationSeconds, &v.PublishedAt, &v.ArchivedAt, &v.WebpageURL,
			&v.VideoRelPath, &v.VideoExt, &v.ThumbnailRelPath, &v.InfoJSONRelPath,
			&v.FileSizeBytes, &v.Width, &v.Height,
			&v.ChannelName, &v.ChannelYoutubeID); err != nil {
			return nil, 0, err
		}
		v.Title = s.getDearrowTitle(v.YoutubeVideoID, v.Title)
		videos = append(videos, v)
	}
	return videos, total, rows.Err()
}

// PlaylistsContainingVideo returns the playlist IDs that include the given video ID.
func (s *Store) PlaylistsContainingVideo(ctx context.Context, videoID int64) ([]int64, error) {
	ctx, span := tracer.Start(ctx, "store.PlaylistsContainingVideo")
	defer span.End()

	rows, err := s.db.Query(`SELECT playlist_id FROM playlist_videos WHERE video_id = ?`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CountPlaylists returns the total number of playlists.
func (s *Store) CountPlaylists(ctx context.Context) (int, error) {
	ctx, span := tracer.Start(ctx, "store.CountPlaylists")
	defer span.End()

	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM playlists`).Scan(&n)
	return n, err
}

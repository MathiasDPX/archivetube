package store

import (
	"database/sql"
	"fmt"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) UpsertVideo(v *domain.Video) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO videos (youtube_video_id, channel_id, title, description, duration_seconds,
			published_at, webpage_url, video_rel_path, video_ext, thumbnail_rel_path,
			info_json_rel_path, file_size_bytes, width, height)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(youtube_video_id) DO UPDATE SET
			channel_id         = excluded.channel_id,
			title              = excluded.title,
			description        = excluded.description,
			duration_seconds   = excluded.duration_seconds,
			published_at       = excluded.published_at,
			webpage_url        = excluded.webpage_url,
			video_rel_path     = excluded.video_rel_path,
			video_ext          = excluded.video_ext,
			thumbnail_rel_path = excluded.thumbnail_rel_path,
			info_json_rel_path = excluded.info_json_rel_path,
			file_size_bytes    = excluded.file_size_bytes,
			width              = excluded.width,
			height             = excluded.height`,
		v.YoutubeVideoID, v.ChannelID, v.Title, v.Description, v.DurationSeconds,
		v.PublishedAt, v.WebpageURL, v.VideoRelPath, v.VideoExt, v.ThumbnailRelPath,
		v.InfoJSONRelPath, v.FileSizeBytes, v.Width, v.Height,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		row := s.db.QueryRow("SELECT id FROM videos WHERE youtube_video_id = ?", v.YoutubeVideoID)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) GetVideoByYoutubeID(ytID string) (*domain.Video, error) {
	return s.scanVideo(s.db.QueryRow(`
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM videos v
		JOIN channels c ON c.id = v.channel_id
		WHERE v.youtube_video_id = ?`, ytID))
}

func (s *Store) GetVideoByID(id int64) (*domain.Video, error) {
	return s.scanVideo(s.db.QueryRow(`
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM videos v
		JOIN channels c ON c.id = v.channel_id
		WHERE v.id = ?`, id))
}

func (s *Store) scanVideo(row *sql.Row) (*domain.Video, error) {
	v := &domain.Video{}
	err := row.Scan(&v.ID, &v.YoutubeVideoID, &v.ChannelID, &v.Title, &v.Description,
		&v.DurationSeconds, &v.PublishedAt, &v.ArchivedAt, &v.WebpageURL,
		&v.VideoRelPath, &v.VideoExt, &v.ThumbnailRelPath, &v.InfoJSONRelPath,
		&v.FileSizeBytes, &v.Width, &v.Height,
		&v.ChannelName, &v.ChannelYoutubeID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Store) ListVideos(query string, sort string, limit, offset int) ([]domain.Video, int, error) {
	orderDir := "DESC"
	if sort == "asc" {
		orderDir = "ASC"
	}

	where := ""
	var args []any
	if query != "" {
		where = "WHERE v.title LIKE ? OR c.name LIKE ?"
		like := "%" + query + "%"
		args = append(args, like, like)
	}

	var total int
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM videos v JOIN channels c ON c.id = v.channel_id %s", where)
	if err := s.db.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listSQL := fmt.Sprintf(`
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM videos v
		JOIN channels c ON c.id = v.channel_id
		%s
		ORDER BY v.archived_at %s
		LIMIT ? OFFSET ?`, where, orderDir)

	rows, err := s.db.Query(listSQL, append(args, limit, offset)...)
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
		videos = append(videos, v)
	}
	return videos, total, rows.Err()
}

func (s *Store) ListVideosByChannel(channelID int64, limit, offset int) ([]domain.Video, int, error) {
	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE channel_id = ?", channelID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM videos v
		JOIN channels c ON c.id = v.channel_id
		WHERE v.channel_id = ?
		ORDER BY v.archived_at DESC
		LIMIT ? OFFSET ?`, channelID, limit, offset)
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
		videos = append(videos, v)
	}
	return videos, total, rows.Err()
}

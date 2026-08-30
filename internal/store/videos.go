package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) UpsertVideo(ctx context.Context, v *domain.Video) (int64, error) {
	ctx, span := tracer.Start(ctx, "store.UpsertVideo")
	defer span.End()

	var id int64
	row := s.db.QueryRow("SELECT id FROM videos WHERE youtube_video_id = ?", v.YoutubeVideoID)
	err := row.Scan(&id)

	if err == sql.ErrNoRows {
		_, err = s.db.Exec(`
			INSERT INTO videos (youtube_video_id, channel_id, title, description, duration_seconds,
				published_at, webpage_url, video_rel_path, video_ext, thumbnail_rel_path,
				info_json_rel_path, file_size_bytes, width, height)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			v.YoutubeVideoID, v.ChannelID, v.Title, v.Description, v.DurationSeconds,
			v.PublishedAt, v.WebpageURL, v.VideoRelPath, v.VideoExt, v.ThumbnailRelPath,
			v.InfoJSONRelPath, v.FileSizeBytes, v.Width, v.Height,
		)
		if err != nil {
			return 0, err
		}
		row = s.db.QueryRow("SELECT id FROM videos WHERE youtube_video_id = ?", v.YoutubeVideoID)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	if err != nil {
		return 0, err
	}

	_, err = s.db.Exec(`
		UPDATE videos SET
			channel_id         = ?,
			title              = ?,
			description        = ?,
			duration_seconds   = ?,
			published_at       = ?,
			webpage_url        = ?,
			video_rel_path     = ?,
			video_ext          = ?,
			thumbnail_rel_path = ?,
			info_json_rel_path = ?,
			file_size_bytes    = ?,
			width              = ?,
			height             = ?
		WHERE id = ?`,
		v.ChannelID, v.Title, v.Description, v.DurationSeconds,
		v.PublishedAt, v.WebpageURL, v.VideoRelPath, v.VideoExt, v.ThumbnailRelPath,
		v.InfoJSONRelPath, v.FileSizeBytes, v.Width, v.Height,
		id,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) GetVideoByYoutubeID(ctx context.Context, ytID string) (*domain.Video, error) {
	ctx, span := tracer.Start(ctx, "store.GetVideoByYoutubeID")
	defer span.End()

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

func (s *Store) GetVideoByID(ctx context.Context, id int64) (*domain.Video, error) {
	ctx, span := tracer.Start(ctx, "store.GetVideoByID")
	defer span.End()

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

func (s *Store) getDearrowTitle(youtubeId string, fallback string) string {
	if !s.cfg.Dearrow.Enabled || youtubeId == "" {
		return fallback
	}

	type Response struct {
		Titles []struct {
			Title string `json:"title"`
		} `json:"titles"`
	}

	resp, err := http.Get(s.cfg.Dearrow.ApiURL + "/api/branding?videoID=" + youtubeId)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fallback
	}

	var data Response
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fallback
	}

	if len(data.Titles) == 0 {
		return fallback
	}

	return data.Titles[0].Title
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
	v.Title = s.getDearrowTitle(v.YoutubeVideoID, v.Title)
	return v, nil
}

func (s *Store) ListVideos(ctx context.Context, query string, sort string, limit, offset int) ([]domain.Video, int, error) {
	ctx, span := tracer.Start(ctx, "store.ListVideos")
	defer span.End()

	orderDir := "DESC"
	if sort == "asc" {
		orderDir = "ASC"
	}

	where := ""
	var args []any
	if query != "" {
		where = "WHERE v.title ILIKE ? OR c.name ILIKE ?"
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
		ORDER BY v.published_at %s
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
		v.Title = s.getDearrowTitle(v.YoutubeVideoID, v.Title)
		videos = append(videos, v)
	}
	return videos, total, rows.Err()
}

func (s *Store) DeleteVideo(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "store.DeleteVideo")
	defer span.End()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM video_chapters WHERE video_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM video_subtitles WHERE video_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM video_audio_tracks WHERE video_id = ?", id); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM videos WHERE id = ?", id); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) CountVideos(ctx context.Context) (int, error) {
	ctx, span := tracer.Start(ctx, "store.CountVideos")
	defer span.End()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&count)
	return count, err
}

func (s *Store) SumArchiveSize(ctx context.Context) (int64, error) {
	ctx, span := tracer.Start(ctx, "store.SumArchiveSize")
	defer span.End()

	var size int64
	err := s.db.QueryRow("SELECT COALESCE(SUM(file_size_bytes), 0) FROM videos").Scan(&size)
	return size, err
}

func (s *Store) CountChannels(ctx context.Context) (int, error) {
	ctx, span := tracer.Start(ctx, "store.CountChannels")
	defer span.End()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM channels").Scan(&count)
	return count, err
}

func (s *Store) ListVideosByChannel(ctx context.Context, channelID int64, limit, offset int) ([]domain.Video, int, error) {
	ctx, span := tracer.Start(ctx, "store.ListVideosByChannel")
	defer span.End()

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
		ORDER BY v.published_at DESC
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
		v.Title = s.getDearrowTitle(v.YoutubeVideoID, v.Title)
		videos = append(videos, v)
	}
	return videos, total, rows.Err()
}

package store

import (
	"context"
	"database/sql"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) UpsertChannel(ctx context.Context, ch *domain.Channel) (int64, error) {
	ctx, span := tracer.Start(ctx, "store.UpsertChannel")
	defer span.End()

	var id int64
	row := s.db.QueryRow("SELECT id FROM channels WHERE youtube_channel_id = ?", ch.YoutubeChannelID)
	err := row.Scan(&id)

	if err == sql.ErrNoRows {
		_, err = s.db.Exec(`
			INSERT INTO channels (youtube_channel_id, handle, name, url, description, thumbnail_path, banner_path, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, now())`,
			ch.YoutubeChannelID, ch.Handle, ch.Name, ch.URL, ch.Description, ch.ThumbnailPath, ch.BannerPath,
		)
		if err != nil {
			return 0, err
		}
		row = s.db.QueryRow("SELECT id FROM channels WHERE youtube_channel_id = ?", ch.YoutubeChannelID)
		if err := row.Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	if err != nil {
		return 0, err
	}

	// avoids DuckDB FK violation on ON CONFLICT
	_, err = s.db.Exec(`
		UPDATE channels SET
			handle         = ?,
			name           = ?,
			url            = ?,
			description    = ?,
			thumbnail_path = CASE WHEN ? != '' THEN ? ELSE thumbnail_path END,
			banner_path    = CASE WHEN ? != '' THEN ? ELSE banner_path END,
			updated_at     = now()
		WHERE id = ?`,
		ch.Handle, ch.Name, ch.URL, ch.Description,
		ch.ThumbnailPath, ch.ThumbnailPath,
		ch.BannerPath, ch.BannerPath,
		id,
	)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) GetChannelByYoutubeID(ctx context.Context, ytID string) (*domain.Channel, error) {
	ctx, span := tracer.Start(ctx, "store.GetChannelByYoutubeID")
	defer span.End()

	ch := &domain.Channel{}
	err := s.db.QueryRow(`
		SELECT id, youtube_channel_id, handle, name, url, description, thumbnail_path, banner_path, created_at, updated_at
		FROM channels WHERE youtube_channel_id = ?`, ytID,
	).Scan(&ch.ID, &ch.YoutubeChannelID, &ch.Handle, &ch.Name, &ch.URL, &ch.Description,
		&ch.ThumbnailPath, &ch.BannerPath, &ch.CreatedAt, &ch.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *Store) CountVideosByChannel(ctx context.Context, channelID int64) (int, error) {
	ctx, span := tracer.Start(ctx, "store.CountVideosByChannel")
	defer span.End()

	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM videos WHERE channel_id = ?", channelID).Scan(&count)
	return count, err
}

func (s *Store) DeleteChannel(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "store.DeleteChannel")
	defer span.End()

	_, err := s.db.Exec("DELETE FROM channels WHERE id = ?", id)
	return err
}

func (s *Store) ClearChannelImages(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "store.ClearChannelImages")
	defer span.End()

	_, err := s.db.Exec("UPDATE channels SET thumbnail_path = '', banner_path = '', updated_at = now() WHERE id = ?", id)
	return err
}

func (s *Store) ListChannels(ctx context.Context) ([]domain.Channel, error) {
	ctx, span := tracer.Start(ctx, "store.ListChannels")
	defer span.End()

	rows, err := s.db.Query(`
		SELECT id, youtube_channel_id, handle, name, url, description, thumbnail_path, banner_path, created_at, updated_at
		FROM channels ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(&ch.ID, &ch.YoutubeChannelID, &ch.Handle, &ch.Name, &ch.URL,
			&ch.Description, &ch.ThumbnailPath, &ch.BannerPath, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

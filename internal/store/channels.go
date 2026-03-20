package store

import (
	"database/sql"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) UpsertChannel(ch *domain.Channel) (int64, error) {
	_, err := s.db.Exec(`
		INSERT INTO channels (youtube_channel_id, handle, name, url, description, thumbnail_path, banner_path, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(youtube_channel_id) DO UPDATE SET
			handle         = excluded.handle,
			name           = excluded.name,
			url            = excluded.url,
			description    = excluded.description,
			thumbnail_path = excluded.thumbnail_path,
			banner_path    = excluded.banner_path,
			updated_at     = CURRENT_TIMESTAMP`,
		ch.YoutubeChannelID, ch.Handle, ch.Name, ch.URL, ch.Description, ch.ThumbnailPath, ch.BannerPath,
	)
	if err != nil {
		return 0, err
	}
	var id int64
	row := s.db.QueryRow("SELECT id FROM channels WHERE youtube_channel_id = ?", ch.YoutubeChannelID)
	if err := row.Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) GetChannelByYoutubeID(ytID string) (*domain.Channel, error) {
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

func (s *Store) ListChannels() ([]domain.Channel, error) {
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

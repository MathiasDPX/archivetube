package store

import "github.com/MathiasDPX/archivetube/internal/domain"

func (s *Store) ReplaceSubtitles(videoID int64, subs []domain.Subtitle) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM video_subtitles WHERE video_id = ?", videoID); err != nil {
		return err
	}

	for _, sub := range subs {
		isDefault := 0
		if sub.IsDefault {
			isDefault = 1
		}
		if _, err := tx.Exec(`
			INSERT INTO video_subtitles (video_id, language_code, language_name, ext, rel_path, is_default)
			VALUES (?, ?, ?, ?, ?, ?)`,
			videoID, sub.LanguageCode, sub.LanguageName, sub.Ext, sub.RelPath, isDefault,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetSubtitles(videoID int64) ([]domain.Subtitle, error) {
	rows, err := s.db.Query(`
		SELECT id, video_id, language_code, language_name, ext, rel_path, is_default
		FROM video_subtitles
		WHERE video_id = ?
		ORDER BY language_code`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []domain.Subtitle
	for rows.Next() {
		var sub domain.Subtitle
		var isDefault int
		if err := rows.Scan(&sub.ID, &sub.VideoID, &sub.LanguageCode, &sub.LanguageName,
			&sub.Ext, &sub.RelPath, &isDefault); err != nil {
			return nil, err
		}
		sub.IsDefault = isDefault != 0
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

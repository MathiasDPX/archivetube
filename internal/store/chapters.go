package store

import "github.com/MathiasDPX/archivetube/internal/domain"

func (s *Store) ReplaceChapters(videoID int64, chapters []domain.Chapter) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM video_chapters WHERE video_id = ?", videoID); err != nil {
		return err
	}

	for _, ch := range chapters {
		if _, err := tx.Exec(`
			INSERT INTO video_chapters (video_id, position, title, start_seconds, end_seconds)
			VALUES (?, ?, ?, ?, ?)`,
			videoID, ch.Position, ch.Title, ch.StartSeconds, ch.EndSeconds,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetChapters(videoID int64) ([]domain.Chapter, error) {
	rows, err := s.db.Query(`
		SELECT id, video_id, position, title, start_seconds, end_seconds
		FROM video_chapters
		WHERE video_id = ?
		ORDER BY position`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []domain.Chapter
	for rows.Next() {
		var ch domain.Chapter
		if err := rows.Scan(&ch.ID, &ch.VideoID, &ch.Position, &ch.Title, &ch.StartSeconds, &ch.EndSeconds); err != nil {
			return nil, err
		}
		chapters = append(chapters, ch)
	}
	return chapters, rows.Err()
}

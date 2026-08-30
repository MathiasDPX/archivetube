package store

import (
	"context"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) ReplaceAudioTracks(ctx context.Context, videoID int64, tracks []domain.AudioTrack) error {
	ctx, span := tracer.Start(ctx, "store.ReplaceAudioTracks")
	defer span.End()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM video_audio_tracks WHERE video_id = ?", videoID); err != nil {
		return err
	}

	for _, track := range tracks {
		isOriginal := 0
		if track.IsOriginal {
			isOriginal = 1
		}
		if _, err := tx.Exec(`
			INSERT INTO video_audio_tracks (video_id, language_code, language_name, ext, rel_path, is_original)
			VALUES (?, ?, ?, ?, ?, ?)`,
			videoID, track.LanguageCode, track.LanguageName, track.Ext, track.RelPath, isOriginal,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetAudioTracks(ctx context.Context, videoID int64) ([]domain.AudioTrack, error) {
	ctx, span := tracer.Start(ctx, "store.GetAudioTracks")
	defer span.End()

	rows, err := s.db.Query(`
		SELECT id, video_id, language_code, language_name, ext, rel_path, is_original
		FROM video_audio_tracks
		WHERE video_id = ?
		ORDER BY is_original DESC, language_code`, videoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []domain.AudioTrack
	for rows.Next() {
		var track domain.AudioTrack
		var isOriginal int
		if err := rows.Scan(&track.ID, &track.VideoID, &track.LanguageCode, &track.LanguageName,
			&track.Ext, &track.RelPath, &isOriginal); err != nil {
			return nil, err
		}
		track.IsOriginal = isOriginal != 0
		tracks = append(tracks, track)
	}
	return tracks, rows.Err()
}

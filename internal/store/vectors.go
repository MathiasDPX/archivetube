package store

import (
	"context"
	"fmt"

	"github.com/MathiasDPX/archivetube/internal/domain"
)

func (s *Store) UpsertVideoVectors(ctx context.Context, youtubeVideoID string, titleVec, descriptionVec []float32) error {
	ctx, span := tracer.Start(ctx, "store.UpsertVideoVectors")
	defer span.End()

	titleStr := float32SliceToSQL(titleVec)
	descStr := float32SliceToSQL(descriptionVec)

	_, err := s.db.Exec(fmt.Sprintf(`
		INSERT INTO videos_vectors (youtube_video_id, title_vec, description_vec)
		VALUES (?, %s::FLOAT[384], %s::FLOAT[384])
		ON CONFLICT(youtube_video_id) DO UPDATE SET
			title_vec = excluded.title_vec,
			description_vec = excluded.description_vec`,
		titleStr, descStr),
		youtubeVideoID,
	)
	return err
}

func (s *Store) SearchVideosSmart(ctx context.Context, queryVec []float32, limit int) ([]domain.Video, error) {
	ctx, span := tracer.Start(ctx, "store.SearchVideosSmart")
	defer span.End()

	vecStr := float32SliceToSQL(queryVec)

	sql := fmt.Sprintf(`
		WITH title_scores AS (
			SELECT youtube_video_id,
				array_cosine_distance(title_vec::FLOAT[384], %[1]s::FLOAT[384]) AS distance
			FROM videos_vectors
			ORDER BY distance
			LIMIT %[2]d
		),
		desc_scores AS (
			SELECT youtube_video_id,
				array_cosine_distance(description_vec::FLOAT[384], %[1]s::FLOAT[384]) AS distance
			FROM videos_vectors
			ORDER BY distance
			LIMIT %[2]d
		),
		combined AS (
			SELECT youtube_video_id,
				COALESCE(t.distance, 0) * 1.0 + COALESCE(d.distance, 0) * 0.5 AS score
			FROM title_scores t
			FULL OUTER JOIN desc_scores d USING (youtube_video_id)
		)
		SELECT v.id, v.youtube_video_id, v.channel_id, v.title, v.description,
			v.duration_seconds, v.published_at, v.archived_at, v.webpage_url,
			v.video_rel_path, v.video_ext, v.thumbnail_rel_path, v.info_json_rel_path,
			v.file_size_bytes, v.width, v.height,
			c.name, c.youtube_channel_id
		FROM combined cm
		JOIN videos v ON v.youtube_video_id = cm.youtube_video_id
		JOIN channels c ON c.id = v.channel_id
		ORDER BY cm.score ASC
		LIMIT %[2]d`, vecStr, limit)

	rows, err := s.db.Query(sql)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

func (s *Store) DeleteVideoVectors(ctx context.Context, youtubeVideoID string) error {
	ctx, span := tracer.Start(ctx, "store.DeleteVideoVectors")
	defer span.End()

	_, err := s.db.Exec("DELETE FROM videos_vectors WHERE youtube_video_id = ?", youtubeVideoID)
	return err
}

func float32SliceToSQL(v []float32) string {
	s := "["
	for i, f := range v {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%g", f)
	}
	s += "]"
	return s
}

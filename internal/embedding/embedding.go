package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MathiasDPX/archivetube/internal/config"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("github.com/MathiasDPX/archivetube/internal/embedding")

const (
	dims = 384
)

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func GetEmbedding(ctx context.Context, config *config.SmartSearchConfig, text string) ([]float32, error) {
	ctx, span := tracer.Start(ctx, "embedding.GetEmbedding")
	defer span.End()

	body, err := json.Marshal(map[string]any{
		"model":      config.Model,
		"input":      text,
		"dimensions": dims,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", config.Backend, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API returned status %d", resp.StatusCode)
	}

	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("embedding API returned no data")
	}
	return result.Data[0].Embedding, nil
}

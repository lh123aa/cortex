package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VoyageEmbedding Voyage AI embedding provider
type VoyageEmbedding struct {
	APIKey  string
	Model   string
	BaseURL string
	Dim     int
	client  *http.Client
}

// NewVoyageEmbedding 创建 Voyage AI Embedding Provider
func NewVoyageEmbedding(apiKey, model, baseURL string, dim int) *VoyageEmbedding {
	if baseURL == "" {
		baseURL = "https://api.voyageai.com/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &VoyageEmbedding{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Dim:     dim,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 4,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 60 * time.Second,
		},
	}
}

type voyageRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type voyageResponse struct {
	Data []voyageEmbeddingData `json:"data"`
}

type voyageEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (v *VoyageEmbedding) Embed(text string) ([]float32, error) {
	results, err := v.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (v *VoyageEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := voyageRequest{
		Model: v.Model,
		Input: texts,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", v.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.APIKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voyage request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("voyage API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result voyageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse voyage response: %w", err)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (v *VoyageEmbedding) Dimension() int { return v.Dim }

func (v *VoyageEmbedding) Name() string { return "voyage:" + v.Model }

func (v *VoyageEmbedding) Health() error {
	_, err := v.Embed("health check")
	return err
}

var _ EmbeddingProvider = (*VoyageEmbedding)(nil)

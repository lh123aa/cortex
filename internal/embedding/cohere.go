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

// CohereEmbedding Cohere API embedding provider
type CohereEmbedding struct {
	APIKey  string
	Model   string
	BaseURL string
	Dim     int
	client  *http.Client
}

// NewCohereEmbedding 创建 Cohere Embedding Provider
func NewCohereEmbedding(apiKey, model, baseURL string, dim int) *CohereEmbedding {
	if baseURL == "" {
		baseURL = "https://api.cohere.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &CohereEmbedding{
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

type cohereRequest struct {
	Model  string   `json:"model"`
	Texts  []string `json:"texts"`
	Type   string   `json:"input_type"`
}

type cohereResponse struct {
	Data []cohereEmbeddingData `json:"data"`
	Meta *struct {
		     BilledUnits *struct {
			     InputTokens int `json:"input_tokens"`
		     } `json:"billed_units"`
	     } `json:"meta,omitempty"`
}

type cohereEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (c *CohereEmbedding) Embed(text string) ([]float32, error) {
	results, err := c.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (c *CohereEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := cohereRequest{
		Model: c.Model,
		Texts: texts,
		Type:  "search_document",
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", c.BaseURL+"/v1/embed", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cohere request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cohere API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result cohereResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cohere response: %w", err)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (c *CohereEmbedding) Dimension() int { return c.Dim }

func (c *CohereEmbedding) Name() string { return "cohere:" + c.Model }

func (c *CohereEmbedding) Health() error {
	// Cohere 没有轻量的 health 端点，用 embed 一个简单文本代替
	_, err := c.Embed("health check")
	return err
}

var _ EmbeddingProvider = (*CohereEmbedding)(nil)

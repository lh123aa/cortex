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

// ZhipuEmbedding 智谱 GLM embedding provider
// API 文档: https://open.bigmodel.cn/dev/api/vector
type ZhipuEmbedding struct {
	APIKey  string
	Model   string
	BaseURL string
	Dim     int
	client  *http.Client
}

// NewZhipuEmbedding 创建智谱 Embedding Provider
func NewZhipuEmbedding(apiKey, model, baseURL string, dim int) *ZhipuEmbedding {
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &ZhipuEmbedding{
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

type zhipuRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type zhipuResponse struct {
	Data []zhipuEmbeddingData `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type zhipuEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (z *ZhipuEmbedding) Embed(text string) ([]float32, error) {
	results, err := z.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (z *ZhipuEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := zhipuRequest{
		Model: z.Model,
		Input: texts,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", z.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+z.APIKey)

	resp, err := z.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zhipu request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("zhipu API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result zhipuResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse zhipu response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("zhipu error: %s (code=%s)", result.Error.Message, result.Error.Code)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (z *ZhipuEmbedding) Dimension() int { return z.Dim }

func (z *ZhipuEmbedding) Name() string { return "zhipu:" + z.Model }

func (z *ZhipuEmbedding) Health() error {
	_, err := z.Embed("健康检查")
	return err
}

var _ EmbeddingProvider = (*ZhipuEmbedding)(nil)

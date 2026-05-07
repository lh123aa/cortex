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

// DashScopeEmbedding 阿里云 DashScope（通义千问）embedding provider
// API 文档: https://help.aliyun.com/zh/model-studio/developer-reference
type DashScopeEmbedding struct {
	APIKey  string
	Model   string
	BaseURL string
	Dim     int
	client  *http.Client
}

// NewDashScopeEmbedding 创建 DashScope Embedding Provider
func NewDashScopeEmbedding(apiKey, model, baseURL string, dim int) *DashScopeEmbedding {
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &DashScopeEmbedding{
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

type dashScopeRequest struct {
	Model string   `json:"model"`
	Input dashScopeInput `json:"input"`
}

type dashScopeInput struct {
	Texts []string `json:"texts"`
}

type dashScopeResponse struct {
	Output *dashScopeOutput `json:"output"`
	Code   string           `json:"code,omitempty"`
	Message string          `json:"message,omitempty"`
}

type dashScopeOutput struct {
	Embeddings []dashScopeEmbeddingItem `json:"embeddings"`
}

type dashScopeEmbeddingItem struct {
	Embedding []float32 `json:"embedding"`
	TextIndex int       `json:"text_index"`
}

func (d *DashScopeEmbedding) Embed(text string) ([]float32, error) {
	results, err := d.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (d *DashScopeEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := dashScopeRequest{
		Model: d.Model,
		Input: dashScopeInput{Texts: texts},
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", d.BaseURL+"/services/embeddings/text-embedding/text-embedding", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.APIKey)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dashscope request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("dashscope API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result dashScopeResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse dashscope response: %w", err)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("dashscope error: %s - %s", result.Code, result.Message)
	}
	if result.Output == nil {
		return nil, fmt.Errorf("dashscope: empty response")
	}

	embeddings := make([][]float32, len(texts))
	for _, item := range result.Output.Embeddings {
		if item.TextIndex < len(embeddings) {
			embeddings[item.TextIndex] = item.Embedding
		}
	}
	return embeddings, nil
}

func (d *DashScopeEmbedding) Dimension() int { return d.Dim }

func (d *DashScopeEmbedding) Name() string { return "dashscope:" + d.Model }

func (d *DashScopeEmbedding) Health() error {
	_, err := d.Embed("健康检查")
	return err
}

var _ EmbeddingProvider = (*DashScopeEmbedding)(nil)

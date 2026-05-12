package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// OpenAIEmbedding OpenAI / 兼容接口 embedding provider
// 兼容 OpenAI API、Azure OpenAI、以及任何 OpenAI 兼容格式的服务
type OpenAIEmbedding struct {
	APIKey  string
	Model   string
	BaseURL string
	Dim     int
	client  *http.Client
}

// NewOpenAIEmbedding 创建 OpenAI Embedding Provider
func NewOpenAIEmbedding(apiKey, model, baseURL string, dim int) *OpenAIEmbedding {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &OpenAIEmbedding{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
		Dim:     dim,
		client: &http.Client{
			Transport: getSharedTransport(),
			Timeout: 60 * time.Second,
		},
	}
}

type openAIRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
	Dim   int      `json:"dimensions,omitempty"`
}

type openAIResponse struct {
	Data []openAIEmbeddingData `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type openAIEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (o *OpenAIEmbedding) Embed(text string) ([]float32, error) {
	results, err := o.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (o *OpenAIEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	reqBody := openAIRequest{
		Model: o.Model,
		Input: texts,
	}
	if o.Dim > 0 {
		reqBody.Dim = o.Dim
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequest("POST", o.BaseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("openai API error: %s (type=%s)", result.Error.Message, result.Error.Type)
	}

	// 按输入顺序排列结果
	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (o *OpenAIEmbedding) Dimension() int { return o.Dim }

func (o *OpenAIEmbedding) Name() string { return "openai:" + o.Model }

func (o *OpenAIEmbedding) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", o.BaseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.APIKey)
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("openai health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// Ensure OpenAIEmbedding implements EmbeddingProvider
var _ EmbeddingProvider = (*OpenAIEmbedding)(nil)

// ── OpenAI 兼容的并发批量 embedding 辅助 ──

// EmbedBatchConcurrent 并发执行批量 embedding（适用于不支持批量接口的 Provider）
func EmbedBatchConcurrent(ctx context.Context, embedFn func(string) ([]float32, error), texts []string, concurrency int) ([][]float32, error) {
	results := make([][]float32, len(texts))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)
	errs := make([]string, 0)

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			emb, err := embedFn(txt)
			mu.Lock()
			if err != nil {
				errs = append(errs, fmt.Sprintf("text[%d]: %v", idx, err))
			} else {
				results[idx] = emb
			}
			mu.Unlock()
		}(i, text)
	}
	wg.Wait()

	if len(errs) > 0 {
		return results, fmt.Errorf("batch embedding failed (%d/%d): %s", len(errs), len(texts), strings.Join(errs, "; "))
	}
	return results, nil
}

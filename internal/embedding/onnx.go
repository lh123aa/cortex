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

// ONNXEmbedding 调用本地 ONNX 模型推理服务器
// 要求服务器实现 /v1/embeddings 接口（兼容 OpenAI API 格式）
type ONNXEmbedding struct {
	BaseURL string
	Model   string
	Dim     int
	Timeout time.Duration
	client  *http.Client
}

// NewONNXEmbedding 创建 ONNX Embedding Provider
// baseURL: ONNX 模型服务器的地址，例如 http://localhost:8080
func NewONNXEmbedding(baseURL, model string, dim int) *ONNXEmbedding {
	transport := &http.Transport{
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     90 * time.Second,
	}
	return &ONNXEmbedding{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Model:   model,
		Dim:     dim,
		Timeout: 60 * time.Second,
		client: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
	}
}

// Embed 调用远程 ONNX 模型获取单个向量
func (o *ONNXEmbedding) Embed(text string) ([]float32, error) {
	reqBody := map[string]any{
		"model": o.Model,
		"input": text,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", o.BaseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ONNX server error: status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 || len(result.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}

	return result.Data[0].Embedding, nil
}

// EmbedBatch 批量调用（并发 + 错误汇总）
func (o *ONNXEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	return o.EmbedBatchWithContext(context.Background(), texts, 0)
}

// EmbedBatchWithContext 支持 context 和重试的批量调用
func (o *ONNXEmbedding) EmbedBatchWithContext(ctx context.Context, texts []string, maxRetries int) ([][]float32, error) {
	results := make([][]float32, len(texts))
	errors := make([]error, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	sem := make(chan struct{}, 4)

	for i, text := range texts {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return results, ctx.Err()
		}

		wg.Add(1)
		go func(idx int, txt string) {
			defer func() { <-sem }()
			defer wg.Done()

			var emb []float32
			var err error
			for retry := 0; retry <= maxRetries; retry++ {
				emb, err = o.EmbedWithContext(ctx, txt)
				if err == nil {
					break
				}
				if ctx.Err() != nil {
					break
				}
			}

			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("text[%d] %q: %w", idx, truncateString(txt, 50), err))
			} else {
				results[idx] = emb
			}
			mu.Unlock()
		}(i, text)
	}

	// Wait all goroutines using WaitGroup (not semaphore drain)
	wg.Wait()

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, e := range errors {
			errMsgs[i] = e.Error()
		}
		return results, fmt.Errorf("ONNX EmbedBatch failed (%d/%d): [%s]", len(errors), len(texts), strings.Join(errMsgs, "; "))
	}
	return results, nil
}

// EmbedWithContext 支持 context 的单个 embedding
func (o *ONNXEmbedding) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	reqBody := map[string]any{
		"model": o.Model,
		"input": text,
	}
	body, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ONNX server error: status %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, io.ErrUnexpectedEOF
	}

	return result.Data[0].Embedding, nil
}

func (o *ONNXEmbedding) Dimension() int {
	return o.Dim
}

func (o *ONNXEmbedding) Name() string {
	return "onnx:" + o.Model
}

func (o *ONNXEmbedding) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", o.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ONNX health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// Ensure ONNXEmbedding implements EmbeddingProvider
var _ EmbeddingProvider = (*ONNXEmbedding)(nil)

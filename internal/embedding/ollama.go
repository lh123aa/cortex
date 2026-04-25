package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

// EmbedResult 单个 embedding 结果（含错误）
type EmbedResult struct {
	Embedding []float32
	Err       error
}

type OllamaEmbedding struct {
	BaseURL     string
	Model       string
	CacheDim    int
	Timeout     time.Duration
	MaxRetries  int           // 最大重试次数
	RetryDelay time.Duration // 基础重试延迟
	client      *http.Client  // HTTP 连接池复用
}

func NewOllamaEmbedding(baseURL, model string, dim int) *OllamaEmbedding {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     90 * time.Second,
	}
	return &OllamaEmbedding{
		BaseURL:     baseURL,
		Model:       model,
		CacheDim:    dim,
		Timeout:     30 * time.Second,
		MaxRetries:  3,           // 默认 3 次重试
		RetryDelay:  100 * time.Millisecond, // 默认 100ms
		client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (o *OllamaEmbedding) Embed(text string) ([]float32, error) {
	return o.EmbedWithRetry(context.Background(), text)
}

// EmbedWithRetry 支持指数退避的重试机制
func (o *OllamaEmbedding) EmbedWithRetry(ctx context.Context, text string) ([]float32, error) {
	var lastErr error
	delay := o.RetryDelay

	for attempt := 0; attempt <= o.MaxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避: delay = base * 2^(attempt-1)，最多 5 秒
			delay = time.Duration(float64(o.RetryDelay) * math.Pow(2, float64(attempt-1)))
			if delay > 5*time.Second {
				delay = 5 * time.Second
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// 继续重试
			}
		}

		emb, err := o.embedOnce(ctx, text)
		if err == nil {
			return emb, nil
		}
		lastErr = err

		// 如果 context 已取消，不再重试
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// 判断是否应该重试
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("max retries (%d) exceeded: last error: %w", o.MaxRetries, lastErr)
}

// embedOnce 执行单次 embedding 请求
func (o *OllamaEmbedding) embedOnce(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaRequest{
		Model:  o.Model,
		Prompt: text,
	}
	body, _ := json.Marshal(reqBody)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 确保 client 不为 nil
	client := o.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama response error: %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

// EmbedWithContext 支持 context 的单个 embedding
func (o *OllamaEmbedding) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	return o.EmbedWithRetry(ctx, text)
}

// isRetryableError 判断错误是否应该重试
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	// 网络错误、超时、服务器错误通常应该重试
	retryableKeywords := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network",
		"503",
		"502",
		"500",
		"429", // Too Many Requests
	}
	for _, keyword := range retryableKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}
	return false
}

// EmbedBatch 批量 embedding
func (o *OllamaEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	return o.EmbedBatchWithContext(context.Background(), texts)
}

// EmbedBatchWithContext 支持 context 的批量 embedding
func (o *OllamaEmbedding) EmbedBatchWithContext(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	errors := make([]error, 0)
	var wg sync.WaitGroup
	var mu sync.Mutex

	sem := make(chan struct{}, 4) // 限制并发为4

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			emb, err := o.EmbedWithRetry(ctx, txt)

			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Errorf("text[%d] %q: %w", idx, truncateString(txt, 50), err))
			} else {
				results[idx] = emb
			}
			mu.Unlock()
		}(i, text)
	}
	wg.Wait()

	if len(errors) > 0 {
		errMsgs := make([]string, len(errors))
		for i, e := range errors {
			errMsgs[i] = e.Error()
		}
		return results, fmt.Errorf("EmbedBatch failed (%d/%d): [%s]", len(errors), len(texts), strings.Join(errMsgs, "; "))
	}
	return results, nil
}

// truncateString 截断字符串用于错误消息
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (o *OllamaEmbedding) Dimension() int {
	return o.CacheDim
}

func (o *OllamaEmbedding) Name() string {
	return "ollama:" + o.Model
}

func (o *OllamaEmbedding) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", o.BaseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama health check failed: status %d", resp.StatusCode)
	}
	return nil
}

// Ensure OllamaEmbedding implements EmbeddingProvider
var _ EmbeddingProvider = (*OllamaEmbedding)(nil)

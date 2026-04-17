package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	BaseURL  string
	Model    string
	CacheDim int
	Timeout  time.Duration
	client   *http.Client // HTTP 连接池复用
}

func NewOllamaEmbedding(baseURL, model string, dim int) *OllamaEmbedding {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     90 * time.Second,
	}
	return &OllamaEmbedding{
		BaseURL:  baseURL,
		Model:    model,
		CacheDim: dim,
		Timeout:  30 * time.Second,
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
	return o.EmbedWithContext(context.Background(), text)
}

// EmbedWithContext 支持 context 的单个 embedding
func (o *OllamaEmbedding) EmbedWithContext(ctx context.Context, text string) ([]float32, error) {
	req := ollamaRequest{
		Model:  o.Model,
		Prompt: text,
	}
	body, _ := json.Marshal(req)

	req, err := http.NewRequestWithContext(ctx, "POST", o.BaseURL+"/api/embeddings", bytes.NewReader(body))
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
		return nil, fmt.Errorf("ollama response error: %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Embedding, nil
}

// EmbedBatch 批量 embedding，P0-1 修复：汇总所有错误而非静默吞掉
func (o *OllamaEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	return o.EmbedBatchWithContext(context.Background(), texts, 0)
}

// EmbedBatchWithContext 支持 context 和重试的批量 embedding
func (o *OllamaEmbedding) EmbedBatchWithContext(ctx context.Context, texts []string, maxRetries int) ([][]float32, error) {
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

			var emb []float32
			var err error
			for retry := 0; retry <= maxRetries; retry++ {
				emb, err = o.EmbedWithContext(ctx, txt)
				if err == nil {
					break
				}
				// 仅 context 取消时不重试
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
	wg.Wait()

	if len(errors) > 0 {
		// 返回首个错误 + 所有错误摘要
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

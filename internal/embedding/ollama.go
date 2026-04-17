package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type OllamaEmbedding struct {
	BaseURL   string
	Model     string
	CacheDim  int
	Timeout   time.Duration
}

func NewOllamaEmbedding(baseURL, model string, dim int) *OllamaEmbedding {
	return &OllamaEmbedding{
		BaseURL:  baseURL,
		Model:    model,
		CacheDim: dim,
		Timeout:  30 * time.Second,
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
	req := ollamaRequest{
		Model:  o.Model,
		Prompt: text,
	}
	body, _ := json.Marshal(req)

	client := &http.Client{Timeout: o.Timeout}
	resp, err := client.Post(o.BaseURL+"/api/embeddings", "application/json", bytes.NewReader(body))
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

// EmbedBatch 因 Ollama 没有原生批量 API，通过 Goroutine 并发处理
func (o *OllamaEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	var wg sync.WaitGroup
	var globalErr error
	var mu sync.Mutex

	sem := make(chan struct{}, 4) // 限制并发为4

	for i, text := range texts {
		wg.Add(1)
		go func(idx int, txt string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			emb, err := o.Embed(txt)
			if err != nil {
				mu.Lock()
				if globalErr == nil {
					globalErr = err
				}
				mu.Unlock()
			} else {
				results[idx] = emb
			}
		}(i, text)
	}
	wg.Wait()

	if globalErr != nil {
		return nil, globalErr
	}
	return results, nil
}

func (o *OllamaEmbedding) Dimension() int {
	return o.CacheDim
}

func (o *OllamaEmbedding) Name() string {
	return "ollama:" + o.Model
}

func (o *OllamaEmbedding) Health() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(o.BaseURL + "/api/tags")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ollama health check failed")
	}
	return nil
}

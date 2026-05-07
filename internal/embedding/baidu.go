package embedding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BaiduEmbedding 百度文心 ERNIE embedding provider
// API 文档: https://cloud.baidu.com/doc/WENXINWORKSHOP/s/klqmvz26g
// 百度需要先通过 API Key + Secret Key 获取 access_token
type BaiduEmbedding struct {
	APIKey    string
	SecretKey string
	Model     string
	BaseURL   string
	Dim       int
	client    *http.Client

	mu           sync.RWMutex
	accessToken  string
	tokenExpires time.Time
}

// NewBaiduEmbedding 创建百度 Embedding Provider
func NewBaiduEmbedding(apiKey, secretKey, model, baseURL string, dim int) *BaiduEmbedding {
	if baseURL == "" {
		baseURL = "https://aip.baidubce.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	return &BaiduEmbedding{
		APIKey:    apiKey,
		SecretKey: secretKey,
		Model:     model,
		BaseURL:   baseURL,
		Dim:       dim,
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

type baiduTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"` // 秒，一般 2592000（30天）
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

func (b *BaiduEmbedding) getAccessToken() (string, error) {
	b.mu.RLock()
	if b.accessToken != "" && time.Now().Before(b.tokenExpires) {
		token := b.accessToken
		b.mu.RUnlock()
		return token, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// 双重检查
	if b.accessToken != "" && time.Now().Before(b.tokenExpires) {
		return b.accessToken, nil
	}

	params := url.Values{}
	params.Set("grant_type", "client_credentials")
	params.Set("client_id", b.APIKey)
	params.Set("client_secret", b.SecretKey)

	req, err := http.NewRequest("POST", b.BaseURL+"/oauth/2.0/token?"+params.Encode(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("baidu token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tr baiduTokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", fmt.Errorf("failed to parse baidu token response: %w", err)
	}

	if tr.Error != "" {
		return "", fmt.Errorf("baidu auth error: %s - %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("baidu: empty access token")
	}

	b.accessToken = tr.AccessToken
	b.tokenExpires = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	return b.accessToken, nil
}

type baiduEmbedRequest struct {
	Input []string `json:"input"`
}

type baiduEmbedResponse struct {
	Data []baiduEmbeddingData `json:"data"`
	ErrorCode int    `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

type baiduEmbeddingData struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

func (b *BaiduEmbedding) Embed(text string) ([]float32, error) {
	results, err := b.EmbedBatch([]string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 || results[0] == nil {
		return nil, fmt.Errorf("empty embedding result")
	}
	return results[0], nil
}

func (b *BaiduEmbedding) EmbedBatch(texts []string) ([][]float32, error) {
	token, err := b.getAccessToken()
	if err != nil {
		return nil, err
	}

	reqBody := baiduEmbedRequest{Input: texts}
	body, _ := json.Marshal(reqBody)

	apiURL := fmt.Sprintf("%s/rpc/2.0/ai_custom/v1/wenxinworkshop/embeddings/%s?access_token=%s",
		b.BaseURL, b.Model, token)
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("baidu embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var result baiduEmbedResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse baidu response: %w", err)
	}

	if result.ErrorCode != 0 {
		return nil, fmt.Errorf("baidu API error (code %d): %s", result.ErrorCode, result.ErrorMsg)
	}

	embeddings := make([][]float32, len(texts))
	for _, d := range result.Data {
		if d.Index < len(embeddings) {
			embeddings[d.Index] = d.Embedding
		}
	}
	return embeddings, nil
}

func (b *BaiduEmbedding) Dimension() int { return b.Dim }

func (b *BaiduEmbedding) Name() string { return "baidu:" + b.Model }

func (b *BaiduEmbedding) Health() error {
	_, err := b.Embed("健康检查")
	return err
}

var _ EmbeddingProvider = (*BaiduEmbedding)(nil)

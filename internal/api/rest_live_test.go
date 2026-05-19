//go:build cgo

package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lh123aa/cortex/internal/search"
	"github.com/lh123aa/cortex/internal/storage"
	"go.uber.org/zap"
)

func TestRESTServer_HealthEndpoint(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	emb := &mockEmbedding{}
	se := search.NewHybridSearchEngine(st, emb)
	logger := zap.NewNop()

	srv := NewRESTServer(se, st, emb, logger)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("failed to GET /health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRESTServer_StartupTime(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	emb := &mockEmbedding{}
	se := search.NewHybridSearchEngine(st, emb)
	logger := zap.NewNop()

	start := time.Now()
	srv := NewRESTServer(se, st, emb, logger)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	if start.IsZero() {
		t.Fatal("start should be valid")
	}
}

func TestRESTServer_SearchEndpoint(t *testing.T) {
	dbPath := "file::memory:?cache=shared"
	st, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer st.Close()

	emb := &mockEmbedding{}
	se := search.NewHybridSearchEngine(st, emb)
	logger := zap.NewNop()

	srv := NewRESTServer(se, st, emb, logger)
	ts := httptest.NewServer(srv.Router())
	defer ts.Close()

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ts.URL + "/v1/search?q=test&k=3")
	if err != nil {
		t.Fatalf("failed to GET /v1/search: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

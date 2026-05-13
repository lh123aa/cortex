package models

import (
	"testing"
	"time"
)

func TestUserTier_StorageLimit(t *testing.T) {
	tests := []struct {
		tier UserTier
		want int64
	}{
		{TierFree, 1 * 1024 * 1024 * 1024},
		{TierPro, 100 * 1024 * 1024 * 1024},
		{TierEnterprise, -1},
		{UserTier("invalid"), 1 * 1024 * 1024 * 1024},
	}
	for _, tt := range tests {
		got := tt.tier.StorageLimit()
		if got != tt.want {
			t.Errorf("UserTier(%q).StorageLimit() = %d, want %d", tt.tier, got, tt.want)
		}
	}
}

func TestUserTier_IsValid(t *testing.T) {
	if !TierFree.IsValid() {
		t.Error("TierFree should be valid")
	}
	if !TierPro.IsValid() {
		t.Error("TierPro should be valid")
	}
	if !TierEnterprise.IsValid() {
		t.Error("TierEnterprise should be valid")
	}
	if UserTier("unknown").IsValid() {
		t.Error("unknown tier should be invalid")
	}
}

func TestSearchOptions_Defaults(t *testing.T) {
	opts := SearchOptions{}
	if opts.TopK != 0 {
		t.Errorf("default TopK = %d, want 0", opts.TopK)
	}
	if opts.Mode != "" {
		t.Errorf("default Mode = %q, want empty", opts.Mode)
	}
}

func TestDocument_Creation(t *testing.T) {
	now := time.Now()
	doc := &Document{
		ID:          "doc-1",
		UserID:      "user-1",
		Path:        "/test/doc.md",
		FileType:    "md",
		ContentHash: "abc123",
		FileSize:    1024,
		ChunkCount:  3,
		IndexedAt:   now,
		Status:      "indexed",
	}
	if doc.ID != "doc-1" {
		t.Errorf("doc.ID = %q, want doc-1", doc.ID)
	}
	if doc.FileSize != 1024 {
		t.Errorf("doc.FileSize = %d, want 1024", doc.FileSize)
	}
}

func TestChunk_Creation(t *testing.T) {
	chunk := &Chunk{
		ID:           "chunk-1",
		UserID:       "user-1",
		DocumentID:   "doc-1",
		HeadingPath:  "Section 1",
		HeadingLevel: 1,
		ContentRaw:   "test content",
		TokenCount:   3,
	}
	if chunk.HeadingPath != "Section 1" {
		t.Errorf("chunk.HeadingPath = %q, want Section 1", chunk.HeadingPath)
	}
	if chunk.TokenCount != 3 {
		t.Errorf("chunk.TokenCount = %d, want 3", chunk.TokenCount)
	}
}

func TestSearchResult_Ordering(t *testing.T) {
	results := []*SearchResult{
		{Score: 0.5, Chunk: &Chunk{ID: "c1"}},
		{Score: 0.9, Chunk: &Chunk{ID: "c2"}},
		{Score: 0.7, Chunk: &Chunk{ID: "c3"}},
	}
	if results[0].Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", results[0].Score)
	}
	if results[1].Score != 0.9 {
		t.Errorf("expected score 0.9, got %f", results[1].Score)
	}
}

func TestIndexProgress_Status(t *testing.T) {
	p := &IndexProgress{
		RootPath:     "/test",
		TotalFiles:   100,
		IndexedFiles: 45,
		Status:       "running",
		StartedAt:    time.Now(),
	}
	if p.Status != "running" {
		t.Errorf("status = %q, want running", p.Status)
	}
	if p.IndexedFiles != 45 {
		t.Errorf("indexed = %d, want 45", p.IndexedFiles)
	}
	// Complete it
	p.Status = "completed"
	p.CompletedAt = time.Now()
	if !p.CompletedAt.IsZero() {
		t.Log("completed_at set correctly")
	}
}

func TestMemory_Creation(t *testing.T) {
	m := &Memory{
		ID:      "mem-1",
		UserID:  "user-1",
		Content: "test memory content",
		Tags:    []string{"test", "memory"},
		Source:  "manual",
	}
	if len(m.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(m.Tags))
	}
	if m.Source != "manual" {
		t.Errorf("source = %q, want manual", m.Source)
	}
}

func TestIndexProgressEvent_Progress(t *testing.T) {
	evt := IndexProgressEvent{
		Total:       100,
		Indexed:     50,
		Skipped:     10,
		Failed:      2,
		CurrentFile: "/test/file.go",
		Speed:       15.5,
	}
	completed := evt.Indexed + evt.Skipped + evt.Failed
	if completed != 62 {
		t.Errorf("completed = %d, want 62", completed)
	}
	if !evt.Done && completed < evt.Total {
		t.Log("progress correctly reports not done")
	}
}

func TestLicense_Defaults(t *testing.T) {
	lic := &License{
		Key:      "test-key",
		Tier:     "pro",
		MaxUsers: 10,
		Active:   true,
	}
	if !lic.Active {
		t.Error("license should be active")
	}
	if lic.MaxUsers != 10 {
		t.Errorf("max_users = %d, want 10", lic.MaxUsers)
	}
}

func TestUser_Creation(t *testing.T) {
	u := &User{
		ID:       "user-1",
		Username: "testuser",
		Role:     RoleUser,
		Tier:     string(TierFree),
		IsActive: true,
	}
	if u.Role != RoleUser {
		t.Errorf("role = %q, want %q", u.Role, RoleUser)
	}
	if !u.IsActive {
		t.Error("user should be active")
	}
}

package chunker

import (
	"testing"
)

func TestMarkdownChunker_Chunk(t *testing.T) {
	config := ChunkConfig{
		MinChars:          10,
		MaxTokens:         500,
		IncludeBreadcrumb: true,
	}

	ch, err := NewMarkdownChunker(config)
	if err != nil {
		t.Fatalf("Failed to create chunker: %v", err)
	}

	content := `
# Header 1
This is a test document.

It has multiple paragraphs.

Short.

This paragraph is longer and should definitely be included in the chunks because it passes the minimum character limit which we set to 10.
`

	chunks, err := ch.Chunk(content, "test.md")
	if err != nil {
		t.Fatalf("Chunking failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("Expected non-empty chunks list")
	}

	// "Short." has 6 chars contextually if trimmed, it should have been dropped if MinChars is 10
	for _, c := range chunks {
		if len(c.ContentRaw) < 10 {
			t.Errorf("Chunk contains text shorter than limit: %q", c.ContentRaw)
		}
		if c.DocumentID == "" {
			t.Errorf("Document ID hash must be generated")
		}
	}
}

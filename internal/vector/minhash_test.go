package vector

import (
	"testing"
)

func TestNewMinHash_DifferentTexts(t *testing.T) {
	m1 := NewMinHash("The quick brown fox jumps over the lazy dog")
	m2 := NewMinHash("The quick brown fox jumps over the lazy dog")
	j := m1.Jaccard(m2)
	if j != 1.0 {
		t.Errorf("identical texts should have Jaccard=1.0, got %f", j)
	}
}

func TestNewMinHash_SimilarTexts(t *testing.T) {
	m1 := NewMinHash("Go语言并发编程使用goroutine和channel")
	m2 := NewMinHash("Go语言并发使用goroutine和channel通信")
	j := m1.Jaccard(m2)
	if j < 0.1 {
		t.Errorf("similar texts should have Jaccard > 0.1, got %f", j)
	}
}

func TestNewMinHash_DifferentTexts_Low(t *testing.T) {
	m1 := NewMinHash("Python is a programming language")
	m2 := NewMinHash("The stock market opened higher today")
	j := m1.Jaccard(m2)
	if j > 0.8 {
		t.Errorf("different texts should have Jaccard < 0.8, got %f", j)
	}
}

func TestMinHash_Serialization(t *testing.T) {
	m1 := NewMinHash("Test serialization roundtrip for MinHash signature storage")
	data := m1.Bytes()
	m2 := MinHashFromBytes(data)
	if m2 == nil {
		t.Fatal("deserialization returned nil")
	}
	j := m1.Jaccard(m2)
	if j != 1.0 {
		t.Errorf("roundtrip should preserve signature, got Jaccard=%f", j)
	}
}

func TestMinHash_EmptyText(t *testing.T) {
	m := NewMinHash("")
	if m == nil {
		t.Fatal("should not return nil for empty text")
	}
	sig := m.Signature()
	if len(sig) != MinHashSigSize {
		t.Errorf("signature size should be %d, got %d", MinHashSigSize, len(sig))
	}
}

func TestMinHash_ShortText(t *testing.T) {
	m := NewMinHash("Hi")
	if m == nil {
		t.Fatal("should not return nil for short text")
	}
	sig := m.Signature()
	if len(sig) != MinHashSigSize {
		t.Errorf("short text should still produce %d-length signature, got %d", MinHashSigSize, len(sig))
	}
}

func BenchmarkMinHash(b *testing.B) {
	text := "Go语言并发编程使用goroutine和channel进行通信，这是Go语言的核心特性之一。"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewMinHash(text)
	}
}

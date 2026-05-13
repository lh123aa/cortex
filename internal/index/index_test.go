package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExcludedDir(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"node_modules", true},
		{".git", true},
		{".opencode", true},
		{"vendor", true},
		{"dist", true},
		{"src", false},
		{"docs", false},
		{"my-project", false},
		{"__pycache__", true},
		{".next", true},
		{".idea", true},
		{".vscode", true},
	}
	for _, tt := range tests {
		got := isExcludedDir(tt.name)
		if got != tt.want {
			t.Errorf("isExcludedDir(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsExcludedDir_EdgeCases(t *testing.T) {
	// Empty string should not be excluded
	if isExcludedDir("") {
		t.Error("empty string should not be excluded")
	}
	// Regular directory names should not be excluded
	if isExcludedDir("mydata") {
		t.Error("mydata should not be excluded")
	}
}

func TestIndexResult_Defaults(t *testing.T) {
	r := &IndexResult{}
	if r.Total != 0 {
		t.Errorf("expected Total=0, got %d", r.Total)
	}
	if r.Duration != 0 {
		t.Errorf("expected Duration=0, got %d", r.Duration)
	}
}

func TestFileResult_Values(t *testing.T) {
	r := fileResult{indexed: true, skipped: false, err: nil}
	if !r.indexed {
		t.Error("expected indexed=true")
	}
	if r.skipped {
		t.Error("expected skipped=false")
	}
	if r.err != nil {
		t.Errorf("expected err=nil, got %v", r.err)
	}
}

func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

func TestExcludedDirSkipped(t *testing.T) {
	// Create temp directory with an excluded dir
	tmpDir, err := os.MkdirTemp("", "cortex-index-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file inside excluded dir
	nodeModules := filepath.Join(tmpDir, "node_modules")
	os.MkdirAll(nodeModules, 0755)
	createTempFile(t, nodeModules, "test.js", "console.log('test')")

	// Create file in normal dir
	createTempFile(t, tmpDir, "readme.md", "# Hello")

	// Walk should only find the non-excluded file
	var files []string
	filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if isExcludedDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})

	if len(files) != 1 {
		t.Errorf("expected 1 file (excluding node_modules), got %d: %v", len(files), files)
	}
}

func TestIsSkippableExt(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".md", false},
		{".go", false},
		{".py", false},
		{".js", false},
		{".txt", false},
		{".png", true},
		{".jpg", true},
		{".mp4", true},
		{".zip", true},
		{".exe", true},
		{".dll", true},
		{".so", true},
		{".dylib", true},
		{".ico", true},
		{".woff2", true},
		{"", false},
		{".go", false},
	}
	for _, tt := range tests {
		got := isSkippableExt(tt.ext)
		if got != tt.want {
			t.Errorf("isSkippableExt(%q) = %v, want %v", tt.ext, got, tt.want)
		}
	}
}

func TestIndexFileResult(t *testing.T) {
	r := fileResult{indexed: true, skipped: false}
	if !r.indexed {
		t.Error("expected indexed=true")
	}
	if r.skipped {
		t.Error("expected skipped=false")
	}

	r2 := fileResult{indexed: false, skipped: true}
	if r2.indexed {
		t.Error("expected indexed=false")
	}
	if !r2.skipped {
		t.Error("expected skipped=true")
	}

	r3 := fileResult{indexed: false, skipped: false, err: os.ErrNotExist}
	if r3.err == nil {
		t.Error("expected non-nil error")
	}
}

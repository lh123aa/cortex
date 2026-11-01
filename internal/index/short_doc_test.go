package index

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lh123aa/cortex/internal/storage"
)

// TestShortDocumentCountedAsSkipped 验证「过短文档」不会被静默算作已索引。
//
// 真实事故：一篇 57 字节的 Markdown 低于 chunker 的 min_chars 阈值，
// 分块器返回 0 个块，索引器却返回 indexed=true。结果
//   - 进度条显示 3/3 全部完成
//   - 汇总显示 "2 indexed, 0 skipped, 0 failed"
//   - 那篇文档既不在库里，也不知道为什么被跳过
//
// 用户看到的就是「文件消失了」。
func TestShortDocumentCountedAsSkipped(t *testing.T) {
	dir := t.TempDir()
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}

	// 正常长度的文档
	longContent := "# Real Doc\n\n" + strings.Repeat(
		"This paragraph has enough content to be chunked properly. ", 40)
	if err := os.WriteFile(filepath.Join(corpus, "long.md"), []byte(longContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 明显过短的文档（只有一行标题 + 一句话）
	shortContent := "# Tiny\n\nShort one line only.\n"
	if err := os.WriteFile(filepath.Join(corpus, "short.md"), []byte(shortContent), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := storage.NewSQLiteStorage(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open storage: %v", err)
	}
	defer st.Close()

	idx, err := NewIndexer(st, nil, 2)
	if err != nil {
		t.Fatalf("new indexer: %v", err)
	}

	res, err := idx.IndexDirectory(context.Background(), corpus, "")
	if err != nil {
		t.Fatalf("index directory: %v", err)
	}

	// 长文档必须被索引
	if res.Indexed != 1 {
		t.Errorf("Indexed = %d, want 1 (long.md)", res.Indexed)
	}
	// 短文档必须计入 skipped，而不是被当成成功
	if res.Skipped < 1 {
		t.Errorf("Skipped = %d, want >= 1 (short.md below min_chars must be counted as skipped, "+
			"not silently reported as indexed)", res.Skipped)
	}
	if res.Failed != 0 {
		t.Errorf("Failed = %d, want 0", res.Failed)
	}

	// 关键不变式：indexed + skipped + failed 必须等于实际扫描到的文件数，
	// 否则用户会看到「进度 3/3，结果只有 2」这种对不上的数字。
	if total := res.Indexed + res.Skipped + res.Failed; total != 2 {
		t.Errorf("indexed(%d)+skipped(%d)+failed(%d) = %d, want 2 (all scanned files accounted for)",
			res.Indexed, res.Skipped, res.Failed, total)
	}
}

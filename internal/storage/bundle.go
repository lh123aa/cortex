package storage

import (
	"time"

	"github.com/lh123aa/cortex/internal/models"
	"github.com/lh123aa/cortex/internal/symbols"
	"github.com/lh123aa/cortex/internal/vector"
)

// DocumentBundle 是一次原子写入的完整单元：文档 + 分块 + 符号。
//
// 为什么需要它：索引一个文件原本要 3–4 次独立提交
// （SaveDocument / SaveChunks / DeleteSymbols / SaveSymbols），
// 每次提交在 WAL+FULL 下都是一次 fsync。合并成一个事务后，
// 单文件写入从最多 4 次 fsync 降到 1 次。
type DocumentBundle struct {
	Doc     *models.Document
	Chunks  []*models.Chunk
	Symbols []symbols.Symbol
}

// BundleStore 是存储层可选实现：把文档的整批写入放进一个事务。
//
// 索引器通过类型断言使用它；不实现该接口的存储自动回退到逐项写入。
type BundleStore interface {
	SaveDocumentBundle(b *DocumentBundle) error
}

// SaveDocumentBundle 在单个事务内写入文档、分块、向量与代码符号。
func (s *SQLiteStorage) SaveDocumentBundle(b *DocumentBundle) error {
	if b == nil || b.Doc == nil {
		return nil
	}
	doc := b.Doc
	// 时间戳兜底（与 SaveDocument 保持一致）
	if doc.IndexedAt.IsZero() {
		doc.IndexedAt = time.Now()
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = doc.IndexedAt
	}
	if doc.Status == "" {
		doc.Status = "indexed"
	}
	doc.IndexedAt = doc.IndexedAt.UTC()
	doc.UpdatedAt = doc.UpdatedAt.UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`
		INSERT OR REPLACE INTO documents
		(id, user_id, path, title, file_type, content_hash, file_size, chunk_count, status, indexed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.UserID, doc.Path, doc.Title, doc.FileType, doc.ContentHash,
		doc.FileSize, doc.ChunkCount, doc.Status,
		doc.IndexedAt.Format(time.RFC3339Nano), doc.UpdatedAt.Format(time.RFC3339Nano),
	); err != nil {
		return err
	}

	if len(b.Chunks) > 0 {
		stmtChunk, err := tx.Prepare(`
			INSERT OR REPLACE INTO chunks
			(id, user_id, document_id, content_hash, minhash_sig, heading_path, heading_level, content, content_raw, line_start, line_end, char_start, char_end, token_count, embedding_model)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmtChunk.Close()

		stmtVec, err := tx.Prepare(`
			INSERT OR REPLACE INTO vectors (chunk_id, user_id, embedding, dimension, model) VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmtVec.Close()

		for _, c := range b.Chunks {
			var minhashData []byte
			if len(c.Embedding) > 0 {
				minhashData = vector.NewMinHash(c.ContentRaw).Bytes()
			}
			if _, err := stmtChunk.Exec(c.ID, c.UserID, c.DocumentID, c.ContentHash, minhashData,
				c.HeadingPath, c.HeadingLevel, c.Content, c.ContentRaw,
				c.LineStart, c.LineEnd, c.CharStart, c.CharEnd, c.TokenCount, c.EmbeddingModel); err != nil {
				return err
			}
			if len(c.Embedding) > 0 {
				if _, err := stmtVec.Exec(c.ID, c.UserID, Float32ArrayToBytes(c.Embedding), len(c.Embedding), c.EmbeddingModel); err != nil {
					return err
				}
				if s.useHNSW && s.hnsw != nil {
					s.hnsw.Add(c.ID, c.Embedding)
				}
			}
		}
	}

	if len(b.Symbols) > 0 {
		if _, err := tx.Exec(`DELETE FROM symbols WHERE document_id = ?`, doc.ID); err != nil {
			return err
		}
		// 列名必须与 symbolsSchema 一致：path / language（不是 file_path）。
		// 早期这里写成 file_path，导致「所有 .go 文件索引失败」：
		// SQL logic error: table symbols has no column named file_path
		stmtSym, err := tx.Prepare(`
			INSERT INTO symbols (document_id, path, name, kind, line, signature, language)
			VALUES (?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmtSym.Close()
		for _, sym := range b.Symbols {
			lang := sym.Language
			if lang == "" {
				lang = symbols.LanguageOf(doc.Path)
			}
			symPath := sym.Path
			if symPath == "" {
				symPath = doc.Path
			}
			if _, err := stmtSym.Exec(doc.ID, symPath, sym.Name, string(sym.Kind),
				sym.Line, sym.Signature, lang); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

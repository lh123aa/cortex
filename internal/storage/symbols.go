package storage

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lh123aa/cortex/internal/symbols"
)

// SymbolStore 是代码符号索引的可选能力。
//
// 之所以不并进主 Storage 接口：符号索引是 P1 新增能力，
// 并进接口会让所有既有实现（含测试 mock）立刻编译失败。
// 调用方用类型断言探测，未实现时自动降级为纯文本检索。
type SymbolStore interface {
	SaveSymbols(docID string, syms []symbols.Symbol) error
	DeleteSymbolsByDocument(docID string) error
	SearchSymbols(query string, language string, pathLike string, limit int) ([]symbols.Symbol, error)
	CountSymbols() (int, error)
}

const symbolsSchema = `
CREATE TABLE IF NOT EXISTS symbols (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	document_id TEXT NOT NULL,
	path        TEXT NOT NULL,
	name        TEXT NOT NULL,
	kind        TEXT NOT NULL,
	line        INTEGER NOT NULL DEFAULT 0,
	signature   TEXT NOT NULL DEFAULT '',
	language    TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_doc ON symbols(document_id);
CREATE INDEX IF NOT EXISTS idx_symbols_path ON symbols(path);
`

// InitSymbolsTable 创建符号表（幂等）。
func (s *SQLiteStorage) InitSymbolsTable() error {
	_, err := s.db.Exec(symbolsSchema)
	return err
}

// SaveSymbols 替换某个文档的全部符号。
func (s *SQLiteStorage) SaveSymbols(docID string, syms []symbols.Symbol) error {
	if err := s.DeleteSymbolsByDocument(docID); err != nil {
		return err
	}
	if len(syms) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`INSERT INTO symbols (document_id, path, name, kind, line, signature, language)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, sym := range syms {
		if _, err := stmt.Exec(docID, sym.Path, sym.Name, string(sym.Kind), sym.Line, sym.Signature, sym.Language); err != nil {
			return fmt.Errorf("insert symbol %s: %w", sym.Name, err)
		}
	}
	return tx.Commit()
}

// DeleteSymbolsByDocument 删除文档的符号。
func (s *SQLiteStorage) DeleteSymbolsByDocument(docID string) error {
	_, err := s.db.Exec(`DELETE FROM symbols WHERE document_id = ?`, docID)
	return err
}

// SearchSymbols 按名称查找符号：精确匹配优先，其次前缀、包含。
func (s *SQLiteStorage) SearchSymbols(query, language, pathLike string, limit int) ([]symbols.Symbol, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	// rank: 0=精确, 1=前缀, 2=包含
	rows, err := s.db.Query(`
		SELECT name, kind, line, signature, language, path,
		       CASE WHEN LOWER(name) = LOWER(?) THEN 0
		            WHEN LOWER(name) LIKE LOWER(?) THEN 1
		            ELSE 2 END AS rank
		FROM symbols
		WHERE (LOWER(name) LIKE LOWER(?) OR LOWER(signature) LIKE LOWER(?))
		  AND (? = '' OR LOWER(language) = LOWER(?))
		  AND (? = '' OR LOWER(path) LIKE LOWER(?))
		ORDER BY rank ASC, LENGTH(name) ASC, name ASC
		LIMIT ?`,
		q, q+"%", "%"+q+"%", "%"+q+"%",
		language, language,
		pathLike, "%"+pathLike+"%",
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []symbols.Symbol
	for rows.Next() {
		var sym symbols.Symbol
		var kind string
		var rank int
		if err := rows.Scan(&sym.Name, &kind, &sym.Line, &sym.Signature, &sym.Language, &sym.Path, &rank); err != nil {
			return nil, err
		}
		sym.Kind = symbols.Kind(kind)
		out = append(out, sym)
	}
	return out, rows.Err()
}

// CountSymbols 返回符号总数。
func (s *SQLiteStorage) CountSymbols() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM symbols`).Scan(&n)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return n, err
}

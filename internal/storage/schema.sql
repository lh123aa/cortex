-- 启用WAL模式与常规并发调优
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA cache_size=10000;
PRAGMA busy_timeout=5000;

-- 文档表
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT UNIQUE NOT NULL,
    title TEXT,
    file_type TEXT DEFAULT 'md',
    content_hash TEXT NOT NULL,
    file_size INTEGER,
    chunk_count INTEGER DEFAULT 0,
    indexed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'indexed'
);

CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);

-- 分块表 (Chunks)
CREATE TABLE IF NOT EXISTS chunks (
    id TEXT PRIMARY KEY,
    rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    document_id TEXT NOT NULL,
    heading_path TEXT,
    heading_level INTEGER,
    content TEXT NOT NULL,
    content_raw TEXT NOT NULL,
    line_start INTEGER,
    line_end INTEGER,
    char_start INTEGER,
    char_end INTEGER,
    token_count INTEGER,
    embedding BLOB,  /* 保存序列化后的 JSON 向量 float32 array */
    embedding_model TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_doc_id ON chunks(document_id);

-- 全文搜索 FTS5 虚拟表
CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    content_raw,
    heading_path,
    document_id,
    content='chunks',
    content_rowid='rowid',
    tokenize='unicode61' /* unicode61自带较为基础的分词支持 */
);

-- 维护 FTS 与 chunks 表一致性的 Triggers
CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
  INSERT INTO chunks_fts(rowid, content_raw, heading_path, document_id) 
  VALUES (new.rowid, new.content_raw, new.heading_path, new.document_id);
END;

CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
  INSERT INTO chunks_fts(chunks_fts, rowid, content_raw, heading_path, document_id) 
  VALUES ('delete', old.rowid, old.content_raw, old.heading_path, old.document_id);
END;

CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
  INSERT INTO chunks_fts(chunks_fts, rowid, content_raw, heading_path, document_id) 
  VALUES ('delete', old.rowid, old.content_raw, old.heading_path, old.document_id);
  INSERT INTO chunks_fts(rowid, content_raw, heading_path, document_id) 
  VALUES (new.rowid, new.content_raw, new.heading_path, new.document_id);
END;


-- 向量存储表（自建向量索引结构）
CREATE TABLE IF NOT EXISTS vectors (
    chunk_id TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    dimension INTEGER NOT NULL,
    model TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

-- 元数据表
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 索引进度追踪表（用于断点恢复）
CREATE TABLE IF NOT EXISTS index_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    root_path TEXT NOT NULL,
    last_file_path TEXT DEFAULT '',
    last_file_index INTEGER DEFAULT 0,
    total_files INTEGER DEFAULT 0,
    indexed_files INTEGER DEFAULT 0,
    indexed_chunks INTEGER DEFAULT 0,
    failed_files INTEGER DEFAULT 0,
    status TEXT DEFAULT 'running',
    started_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_progress_root ON index_progress(root_path);
CREATE INDEX IF NOT EXISTS idx_progress_status ON index_progress(status);

-- 搜索缓存表
CREATE TABLE IF NOT EXISTS search_cache (
    query_hash TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    mode TEXT NOT NULL,
    top_k INTEGER NOT NULL,
    results TEXT NOT NULL,
    hit_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_search_cache_expires ON search_cache(expires_at);

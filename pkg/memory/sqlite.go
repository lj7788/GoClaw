package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"strings"
)

func escapeFTSQuery(query string) string {
	specialChars := []string{"\"", "*", "(", ")", "-", "+", "~", "<", ">", "@", ":"}
	escaped := query
	for _, char := range specialChars {
		escaped = strings.ReplaceAll(escaped, char, " ")
	}
	
	words := strings.Fields(escaped)
	if len(words) == 0 {
		return query
	}
	
	return strings.Join(words, " ")
}

type SQLiteMemoryBackend struct {
	dbPath     string
	db         *sql.DB
	mu         sync.RWMutex
	vectorDB   bool
}

type EmbeddingCache struct {
	ContentHash string `json:"content_hash"`
	Embedding   []float32 `json:"embedding"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
}

func NewSQLiteMemoryBackend(dbPath string) (*SQLiteMemoryBackend, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".goclaw", "memory", "brain.db")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create memory directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &SQLiteMemoryBackend{
		dbPath: dbPath,
		db:     db,
	}

	if err := backend.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return backend, nil
}

func (m *SQLiteMemoryBackend) initSchema() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			key TEXT NOT NULL UNIQUE,
			content TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'core',
			session_id TEXT,
			embedding BLOB,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_key ON memories(key)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_session ON memories(session_id)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
			key, content, content=memories, content_rowid=rowid
		)`,
		`CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
			INSERT INTO memories_fts(rowid, key, content)
			VALUES (new.rowid, new.key, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, key, content)
			VALUES ('delete', old.rowid, old.key, old.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE OF key, content ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, key, content)
			VALUES ('delete', old.rowid, old.key, old.content);
			INSERT INTO memories_fts(rowid, key, content)
			VALUES (new.rowid, new.key, new.content);
		END`,
		`CREATE TABLE IF NOT EXISTS embedding_cache (
			content_hash TEXT PRIMARY KEY,
			embedding BLOB NOT NULL,
			created_at TEXT NOT NULL,
			accessed_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cache_accessed ON embedding_cache(accessed_at)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

func (m *SQLiteMemoryBackend) Store(ctx context.Context, key, content string, category *string, metadata map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	cat := "core"
	if category != nil {
		cat = *category
	}

	metadataJSON, _ := json.Marshal(metadata)

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO memories (id, key, content, category, session_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			content = excluded.content,
			updated_at = excluded.updated_at,
			session_id = excluded.session_id
	`, key, key, content, cat, metadataJSON, now, now)

	return err
}

func (m *SQLiteMemoryBackend) Recall(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries, err := m.searchFTS(ctx, query, limit, category)
	if err != nil {
		return nil, err
	}

	if len(entries) < limit && m.vectorDB {
		vectorEntries, err := m.searchVector(ctx, query, limit-len(entries), category)
		if err == nil {
			entries = append(entries, vectorEntries...)
		}
	}

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

func (m *SQLiteMemoryBackend) Search(ctx context.Context, query string, limit int) ([]MemoryEntry, error) {
	return m.Recall(ctx, query, limit, nil)
}

func (m *SQLiteMemoryBackend) Get(ctx context.Context, key string) (*MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var entry MemoryEntry
	err := m.db.QueryRowContext(ctx, `
		SELECT id, key, content, category, created_at, updated_at
		FROM memories
		WHERE key = ?
	`, key).Scan(&entry.ID, &entry.Key, &entry.Content, &entry.Category, &entry.CreatedAt, &entry.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (m *SQLiteMemoryBackend) Forget(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.ExecContext(ctx, `DELETE FROM memories WHERE key = ?`, key)
	return err
}

func (m *SQLiteMemoryBackend) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.ExecContext(ctx, `DELETE FROM memories`)
	return err
}

func (m *SQLiteMemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *SQLiteMemoryBackend) List(ctx context.Context, category *string) ([]MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `SELECT id, key, content, category, created_at, updated_at FROM memories`
	args := []interface{}{}

	if category != nil {
		query += ` WHERE category = ?`
		args = append(args, *category)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var entry MemoryEntry
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Content, &entry.Category, &entry.CreatedAt, &entry.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (m *SQLiteMemoryBackend) Count(ctx context.Context, category *string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `SELECT COUNT(*) FROM memories`
	args := []interface{}{}

	if category != nil {
		query += ` WHERE category = ?`
		args = append(args, *category)
	}

	var count int
	if err := m.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (m *SQLiteMemoryBackend) Compact(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	if err != nil {
		return fmt.Errorf("failed to checkpoint WAL: %w", err)
	}

	_, err = m.db.ExecContext(ctx, `VACUUM`)
	if err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

func (m *SQLiteMemoryBackend) Export(ctx context.Context, path string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	backupDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer backupDB.Close()

	if _, err := backupDB.ExecContext(ctx, `VACUUM INTO ?`, path); err != nil {
		return fmt.Errorf("failed to export database: %w", err)
	}

	return nil
}

func (m *SQLiteMemoryBackend) Import(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	importDB, err := sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("failed to open import database: %w", err)
	}
	defer importDB.Close()

	_, err = m.db.ExecContext(ctx, `ATTACH DATABASE ? AS import_db`, path)
	if err != nil {
		return fmt.Errorf("failed to attach import database: %w", err)
	}
	defer m.db.ExecContext(ctx, `DETACH DATABASE import_db`)

	_, err = m.db.ExecContext(ctx, `
		INSERT INTO memories (id, key, content, category, session_id, created_at, updated_at)
		SELECT id, key, content, category, session_id, created_at, updated_at
		FROM import_db.memories
		ON CONFLICT(key) DO UPDATE SET
			content = excluded.content,
			updated_at = excluded.updated_at
	`)

	return err
}

func (m *SQLiteMemoryBackend) searchFTS(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error) {
	entries := []MemoryEntry{}

	escapedQuery := escapeFTSQuery(query)

	sqlQuery := `
		SELECT m.id, m.key, m.content, m.category, m.created_at, m.updated_at,
			   bm25(memories_fts) as score
		FROM memories m
		JOIN memories_fts fts ON m.rowid = fts.rowid
		WHERE memories_fts MATCH ?
	`
	args := []interface{}{escapedQuery}

	if category != nil {
		sqlQuery += ` AND m.category = ?`
		args = append(args, *category)
	}

	sqlQuery += ` ORDER BY score LIMIT ?`
	args = append(args, limit)

	rows, err := m.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry MemoryEntry
		var score float64
		if err := rows.Scan(&entry.ID, &entry.Key, &entry.Content, &entry.Category, &entry.CreatedAt, &entry.UpdatedAt, &score); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (m *SQLiteMemoryBackend) searchVector(ctx context.Context, query string, limit int, category *string) ([]MemoryEntry, error) {
	return []MemoryEntry{}, nil
}

func (m *SQLiteMemoryBackend) cleanupCache(ctx context.Context, maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-maxAge).Format(time.RFC3339Nano)

	var err error
	_, err = m.db.ExecContext(ctx, `
		DELETE FROM embedding_cache WHERE accessed_at < ?
	`, cutoff)

	return err
}

func hashContent(content string) string {
	return fmt.Sprintf("%x", len(content))
}
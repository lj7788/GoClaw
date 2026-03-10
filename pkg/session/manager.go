package session

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
)

type Session struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	UserID       string    `json:"user_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type Message struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type Manager struct {
	dbPath string
	db     *sql.DB
	mu     sync.RWMutex
}

func NewManager(dbPath string) (*Manager, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".goclaw", "sessions.db")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create sessions directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_timeout=30000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	manager := &Manager{
		dbPath: dbPath,
		db:     db,
	}

	if err := manager.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return manager, nil
}

func (m *Manager) initSchema() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			user_id TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			message_count INTEGER DEFAULT 0,
			metadata TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			metadata TEXT,
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_updated ON sessions(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

func (m *Manager) CreateSession(ctx context.Context, userID string, title string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	sessionID := fmt.Sprintf("session_%d", now.UnixNano())

	if title == "" {
		title = "新对话"
	}

	metadataJSON, _ := json.Marshal(map[string]interface{}{})

	_, err := m.db.ExecContext(ctx, `
		INSERT INTO sessions (id, title, user_id, created_at, updated_at, message_count, metadata)
		VALUES (?, ?, ?, ?, ?, 0, ?)
	`, sessionID, title, userID, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &Session{
		ID:           sessionID,
		Title:        title,
		UserID:       userID,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
	}, nil
}

func (m *Manager) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var session Session
	var metadataJSON string
	var createdAtStr, updatedAtStr string

	err := m.db.QueryRowContext(ctx, `
		SELECT id, title, user_id, created_at, updated_at, message_count, metadata
		FROM sessions
		WHERE id = ?
	`, sessionID).Scan(&session.ID, &session.Title, &session.UserID, &createdAtStr, &updatedAtStr, &session.MessageCount, &metadataJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
	session.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &session.Metadata)
	}

	return &session, nil
}

func (m *Manager) ListSessions(ctx context.Context, userID string, limit int, offset int) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
		SELECT id, title, user_id, created_at, updated_at, message_count, metadata
		FROM sessions
	`
	args := []interface{}{}

	if userID != "" {
		query += ` WHERE user_id = ?`
		args = append(args, userID)
	}

	query += ` ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := m.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var metadataJSON string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(&session.ID, &session.Title, &session.UserID, &createdAtStr, &updatedAtStr, &session.MessageCount, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		session.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &session.Metadata)
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (m *Manager) UpdateSessionTitle(ctx context.Context, sessionID string, title string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()

	_, err := m.db.ExecContext(ctx, `
		UPDATE sessions
		SET title = ?, updated_at = ?
		WHERE id = ?
	`, title, now.Format(time.RFC3339Nano), sessionID)

	if err != nil {
		return fmt.Errorf("failed to update session title: %w", err)
	}

	return nil
}

func (m *Manager) DeleteSession(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (m *Manager) AddMessage(ctx context.Context, sessionID string, role string, content string, metadata map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	messageID := fmt.Sprintf("msg_%d", now.UnixNano())

	metadataJSON, _ := json.Marshal(metadata)

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO messages (id, session_id, role, content, created_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?)
	`, messageID, sessionID, role, content, now.Format(time.RFC3339Nano), metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE sessions
		SET message_count = message_count + 1, updated_at = ?
		WHERE id = ?
	`, now.Format(time.RFC3339Nano), sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session message count: %w", err)
	}

	// 如果是第一条消息，且标题还是默认值，则根据第一条用户消息自动生成标题
	var messageCount int
	err = tx.QueryRowContext(ctx, "SELECT message_count FROM sessions WHERE id = ?", sessionID).Scan(&messageCount)
	if err != nil {
		return fmt.Errorf("failed to check message count: %w", err)
	}

	if messageCount == 1 {
		// 检查当前标题是否还是默认值
		var currentTitle string
		err = tx.QueryRowContext(ctx, "SELECT title FROM sessions WHERE id = ?", sessionID).Scan(&currentTitle)
		if err != nil {
			return fmt.Errorf("failed to check session title: %w", err)
		}

		if currentTitle == "新对话" || currentTitle == "新会话" {
			// 提取第一条用户消息的前20个字符作为标题
			var firstMessage string
			err = tx.QueryRowContext(ctx, "SELECT content FROM messages WHERE session_id = ? AND role = 'user' ORDER BY created_at ASC LIMIT 1", sessionID).Scan(&firstMessage)
			if err == nil && firstMessage != "" {
				if len(firstMessage) > 20 {
					firstMessage = firstMessage[:20] + "..."
				}
				_, err = tx.ExecContext(ctx, "UPDATE sessions SET title = ? WHERE id = ?", firstMessage, sessionID)
				if err != nil {
					return fmt.Errorf("failed to update session title: %w", err)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (m *Manager) GetMessages(ctx context.Context, sessionID string, limit int, offset int) ([]*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
		SELECT id, session_id, role, content, created_at, metadata
		FROM messages
		WHERE session_id = ?
		ORDER BY created_at ASC
		LIMIT ? OFFSET ?
	`

	rows, err := m.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		var metadataJSON string
		var createdAtStr string

		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &createdAtStr, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		msg.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)

		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &msg.Metadata)
		}

		messages = append(messages, &msg)
	}

	return messages, nil
}

func (m *Manager) SearchSessions(ctx context.Context, userID string, query string, limit int) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sqlQuery := `
		SELECT DISTINCT s.id, s.title, s.user_id, s.created_at, s.updated_at, s.message_count, s.metadata
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE 1=1
	`
	args := []interface{}{}

	if userID != "" {
		sqlQuery += ` AND s.user_id = ?`
		args = append(args, userID)
	}

	if query != "" {
		sqlQuery += ` AND (s.title LIKE ? OR m.content LIKE ?)`
		searchPattern := "%" + query + "%"
		args = append(args, searchPattern, searchPattern)
	}

	sqlQuery += ` ORDER BY s.updated_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := m.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var session Session
		var metadataJSON string
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(&session.ID, &session.Title, &session.UserID, &createdAtStr, &updatedAtStr, &session.MessageCount, &metadataJSON); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAtStr)
		session.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAtStr)

		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &session.Metadata)
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

package auth

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type User struct {
	ID        int       `json:"id"`
	OpenID    string    `json:"openid"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Email     string    `json:"email"`
	Status    int       `json:"status"` // 0: 待审核, 1: 已通过, 2: 已拒绝
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Admin struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"password"` // 哈希存储
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserManager struct {
	dbPath string
	db     *sql.DB
}

func NewUserManager(dbPath string) (*UserManager, error) {
	if dbPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(homeDir, ".goclaw", "auth.db")
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create auth directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_synchronous=NORMAL&_timeout=30000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	manager := &UserManager{
		dbPath: dbPath,
		db:     db,
	}

	if err := manager.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	if err := manager.initAdmin(); err != nil {
		return nil, fmt.Errorf("failed to initialize admin: %w", err)
	}

	return manager, nil
}

func (m *UserManager) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			openid TEXT NOT NULL UNIQUE,
			nickname TEXT NOT NULL,
			avatar TEXT,
			email TEXT,
			status INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_openid ON users(openid)`,
		`CREATE INDEX IF NOT EXISTS idx_users_status ON users(status)`,
		`CREATE TABLE IF NOT EXISTS admins (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_admins_username ON admins(username)`,
	}

	for _, query := range queries {
		if _, err := m.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

func (m *UserManager) initAdmin() error {
	// 检查是否已有管理员
	var count int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check admin count: %w", err)
	}

	// 如果没有管理员，创建默认管理员
	if count == 0 {
		now := time.Now().UTC().Format(time.RFC3339Nano)
		// 默认密码：admin123（实际部署时应该修改）
		hashedPassword := "$2a$10$EixZaYVK1fsbw1ZfbX3OXePaWxn96p36WQoeG6Lruj3vjPGga31lW" // admin123
		_, err := m.db.Exec(`
			INSERT INTO admins (username, password, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, "admin", hashedPassword, now, now)
		if err != nil {
			return fmt.Errorf("failed to create default admin: %w", err)
		}
	}

	return nil
}

func (m *UserManager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// 用户相关方法
func (m *UserManager) CreateUser(openid, nickname, avatar, email string) (*User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	result, err := m.db.Exec(`
		INSERT INTO users (openid, nickname, avatar, email, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, openid, nickname, avatar, email, 0, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	user := &User{
		ID:        int(id),
		OpenID:    openid,
		Nickname:  nickname,
		Avatar:    avatar,
		Email:     email,
		Status:    0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return user, nil
}

func (m *UserManager) GetUserByOpenID(openid string) (*User, error) {
	var user User
	var createdAt, updatedAt string

	err := m.db.QueryRow(`
		SELECT id, openid, nickname, avatar, email, status, created_at, updated_at
		FROM users
		WHERE openid = ?
	`, openid).Scan(&user.ID, &user.OpenID, &user.Nickname, &user.Avatar, &user.Email, &user.Status, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &user, nil
}

func (m *UserManager) GetUserByID(id int) (*User, error) {
	var user User
	var createdAt, updatedAt string

	err := m.db.QueryRow(`
		SELECT id, openid, nickname, avatar, email, status, created_at, updated_at
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.OpenID, &user.Nickname, &user.Avatar, &user.Email, &user.Status, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &user, nil
}

func (m *UserManager) UpdateUserStatus(id int, status int) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := m.db.Exec(`
		UPDATE users
		SET status = ?, updated_at = ?
		WHERE id = ?
	`, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	return nil
}

func (m *UserManager) UpdateUserInfo(id int, nickname, avatar, email string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := m.db.Exec(`
		UPDATE users
		SET nickname = ?, avatar = ?, email = ?, updated_at = ?
		WHERE id = ?
	`, nickname, avatar, email, now, id)
	if err != nil {
		return fmt.Errorf("failed to update user info: %w", err)
	}

	return nil
}

func (m *UserManager) ListUsers(status *int) ([]*User, error) {
	query := `
		SELECT id, openid, nickname, avatar, email, status, created_at, updated_at
		FROM users
	`
	args := []interface{}{}

	if status != nil {
		query += ` WHERE status = ?`
		args = append(args, *status)
	}

	query += ` ORDER BY created_at DESC`

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		var createdAt, updatedAt string

		if err := rows.Scan(&user.ID, &user.OpenID, &user.Nickname, &user.Avatar, &user.Email, &user.Status, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		user.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		user.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

		users = append(users, &user)
	}

	return users, nil
}

// 管理员相关方法
func (m *UserManager) GetAdminByUsername(username string) (*Admin, error) {
	var admin Admin
	var createdAt, updatedAt string

	err := m.db.QueryRow(`
		SELECT id, username, password, created_at, updated_at
		FROM admins
		WHERE username = ?
	`, username).Scan(&admin.ID, &admin.Username, &admin.Password, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get admin: %w", err)
	}

	admin.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	admin.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

	return &admin, nil
}

func (m *UserManager) UpdateAdminPassword(id int, hashedPassword string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := m.db.Exec(`
		UPDATE admins
		SET password = ?, updated_at = ?
		WHERE id = ?
	`, hashedPassword, now, id)
	if err != nil {
		return fmt.Errorf("failed to update admin password: %w", err)
	}

	return nil
}

func (m *UserManager) ListAdmins() ([]*Admin, error) {
	rows, err := m.db.Query(`
		SELECT id, username, password, created_at, updated_at
		FROM admins
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query admins: %w", err)
	}
	defer rows.Close()

	var admins []*Admin
	for rows.Next() {
		var admin Admin
		var createdAt, updatedAt string

		if err := rows.Scan(&admin.ID, &admin.Username, &admin.Password, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan admin: %w", err)
		}

		admin.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		admin.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updatedAt)

		admins = append(admins, &admin)
	}

	return admins, nil
}
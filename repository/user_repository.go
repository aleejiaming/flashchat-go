package repository

import (
	"database/sql"
	"errors"
	"log/slog"
)

type User struct {
	ID           int
	Username     string
	PasswordHash string // 嚴禁明文，這裡只存放 Hash
}

// ==========================================
// 2. 定義倉儲介面 (Repository Interface)
// ==========================================
type UserRepository interface {
	CreateUser(username, passwordHash string) error
	GetUserByUsername(username string) (*User, error)
}

// ==========================================
// 3. PostgreSQL 實作結構體
// ==========================================
type pgUserRepository struct {
	db *sql.DB
}

// ==========================================
// 4. 工廠函式與自動建表 (Auto Migration)
// ==========================================

func NewPGUserRepository(db *sql.DB) UserRepository {
	// 定義 Data Definition Language (DDL)
	query := `CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	username VARCHAR(50) UNIQUE NOT NULL,
	password_hash VARCHAR(255) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := db.Exec(query); err != nil {
		slog.Error("建立 user 資料表失敗", "component", "database", "table", "users")
	} else {
		slog.Info("users 資料表初始化成功",
			"component", "database",
			"table", "users",
		)
	}
	return &pgUserRepository{db: db}
}

// ==========================================
// 5. 實作介面方法 (Data Access Object 模式)
// ==========================================

// CreateUser 負責將新會員寫入資料庫
func (r *pgUserRepository) CreateUser(username, passwordHash string) error {
	// 🛡️ 資安防禦：使用 Parameterized Query (參數化查詢) 防止 SQL Injection
	query := `INSERT INTO users (username , password_hash) VALUES ($1,$2)`

	_, err := r.db.Exec(query, username, passwordHash)
	if err != nil {
		// 這裡先單純記錄錯誤，後續 Handler 層會負責解析是否為 UNIQUE 衝突 (帳號重複)
		slog.Error("寫入使用者失敗",
			"component", "database",
			"action", "create_user",
			"username", username,
			"error", err.Error(),
		)
		return err
	}
	return nil
}

// GetUserByUsername 負責透過帳號查詢會員 (登入時會用到)
func (r *pgUserRepository) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id, username, password_hash FROM users WHERE username = $1`

	row := r.db.QueryRow(query, username)

	var user User
	err := row.Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// 找不到使用者的正常業務邏輯，不當作系統 Error 記錄，改為 Debug 或直接回傳
			slog.Debug("查無此使用者", "username", username)
			return nil, errors.New("user_not_found")
		}

		slog.Error("查詢使用者異常",
			"component", "database",
			"username", username,
			"error", err.Error(),
		)
		return nil, err
	}
	return &user, nil
}

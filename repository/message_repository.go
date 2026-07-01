package repository

import (
	"database/sql"
	"log"

	"flashchat-go/ws"
)

// ==========================================
// 1. 定義介面 (合約)
// ==========================================
type MessageRepository interface {
	SaveMessage(msg ws.Message) error
}

// ==========================================
// 2. 實體結構 (PostgreSQL 版本的倉管員)
// ==========================================
type pgMessageRepository struct {
	db *sql.DB
}

// ==========================================
// 3. 工廠函式 (聘請倉管員，並自動建表)
// ==========================================
func NewPGMessageRepository(db *sql.DB) MessageRepository {
	// 🌟 貼心設計：自動檢查並建立資料表，省去手動下 SQL 的麻煩
	query := `CREATE TABLE IF NOT EXISTS chat_messages (
		id SERIAL PRIMARY KEY,
		name VARCHAR(50),
		content TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("❌ 無法建立 chat_messages 資料表:", err)
	}

	return &pgMessageRepository{db: db}
}

// ==========================================
// 4. 實作合約 (真實寫入硬碟的邏輯)
// ==========================================
func (r *pgMessageRepository) SaveMessage(msg ws.Message) error {
	// 為了隱私，如果是私訊就不存入大眾資料庫
	if msg.IsPrivate {
		return nil
	}
	log.Printf("進入 SaveMessage，準備寫入 %s: %s", msg.Name, msg.Content)
	query := `INSERT INTO chat_messages (name, content) VALUES ($1, $2)`
	_, err := r.db.Exec(query, msg.Name, msg.Content)
	return err
}

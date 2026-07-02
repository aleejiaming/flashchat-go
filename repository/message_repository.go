package repository

import (
	"database/sql"
	"log/slog"

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
	// 修改 1：拿掉冒號，統一標題風格，並加入 table 名稱作為上下文
	_, err := db.Exec(query)
	if err != nil {
		slog.Error("建立資料表失敗",
			"component", "database",
			"table", "chat_messages",
			"error", err.Error(),
		)
	} else {
		// 成功時留下一筆 Info，方便確認連線與初始化正常
		slog.Info("資料表初始化成功",
			"component", "database",
			"table", "chat_messages",
		)
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
	// 修改 2：將傳統 log.Printf 換成 slog.Debug 或 slog.Info
	slog.Debug("準備寫入歷史訊息",
		"component", "database",
		"sender_name", msg.Name,
		"msg_length", len(msg.Content), // 💡 關鍵技巧：記錄長度而非明文
	)
	query := `INSERT INTO chat_messages (name, content) VALUES ($1, $2)`
	_, err := r.db.Exec(query, msg.Name, msg.Content)
	// 修改 3：在實際發生錯誤的地方，補上錯誤日誌
	if err != nil {
		slog.Error("PostgreSQL 寫入失敗",
			"component", "database",
			"sender_name", msg.Name,
			"error", err.Error(),
		)
	}
	return err
}

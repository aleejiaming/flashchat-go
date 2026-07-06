package database

import (
	"database/sql"
	"log/slog"

	_ "github.com/lib/pq" // 引入驅動 (前面加底線代表自動註冊)
)

// NewPostgresDB 是一個工廠函數，負責產出已驗證連線的 sql.DB 實體
func NewPostgresDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		slog.Error("無法建立 PostgreSQL 連線物件", "component", "database", "error", err.Error())
		return nil, err
	}

	// 🛡️ 及早失敗 (Fail-Fast) 策略：確認網路與帳密是真的能通
	if err := db.Ping(); err != nil {
		slog.Error("PostgreSQL 網路或驗證失敗", "component", "database", "error", err.Error())
		return nil, err
	}

	slog.Info("PostgreSQL 連線成功", "component", "database")
	return db, nil
}

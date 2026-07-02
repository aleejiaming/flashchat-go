package logger

import (
	"log/slog"
	"os"
)

// InitLogger 初始化全域的結構化日誌設定
func InitLogger() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug, // 開發階段設為 Debug，方便追蹤
	}

	// 建立 JSON 格式的 Handler
	handler := slog.NewJSONHandler(os.Stdout, opts)

	// 建立 Logger 並設為全域預設
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

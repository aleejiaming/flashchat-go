package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"flashchat-go/internal/bootstrap"
	"flashchat-go/internal/database"
	"flashchat-go/internal/logger"
	"flashchat-go/repository"
)

func main() {
	// 系統環境初始化
	logger.InitLogger()
	slog.Info("FlashChat 系統啟動程序開始...", "component", "server")

	// 載入 .env 檔案
	err := godotenv.Load()
	if err != nil {
		slog.Info("沒找到 .env 檔案，將使用預設系統環境變數", "component", "server")
	}
	// 用 os.Getenv 讀取這些變數了
	pgConnStr := os.Getenv("DATABASE_URL")
	redisAddr := os.Getenv("REDIS_ADDR")
	port := os.Getenv("PORT")

	// 基礎連線初始化
	db, _ := database.NewPostgresDB(pgConnStr) // 補上你的 DSN
	msgRepo := repository.NewPGMessageRepository(db)
	userRepo := repository.NewPGUserRepository(db)

	// 透過裝配線取得「路由器(mux)」
	mux, _, err := bootstrap.InitializeApp(pgConnStr, redisAddr, msgRepo, userRepo)
	if err != nil {
		slog.Error("系統裝配失敗", "error", err.Error())
		os.Exit(1)
	}

	// 將 mux 直接塞給伺服器，不需要再使用 http.HandleFunc
	srv := &http.Server{
		Addr:    port, //讀取環境變數 PORT = os.Getenv("PORT")
		Handler: mux,  // 這裡明確指定使用 bootstrap 回傳的路由器
	}

	// ==========================================
	// 🚀 伺服器啟動與優雅關機 (Lifecycle)
	// ==========================================

	go func() {
		slog.Info("HTTP/WS 伺服器已在背景啟動", "component", "server", "port", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("伺服器崩潰", "component", "server", "error", err.Error())
			os.Exit(1)
		}
	}()

	// 監聽 OS 終止訊號 (Ctrl+C 或 Docker Stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Warn("接收到中斷訊號，開始優雅關機 (Graceful Shutdown)...", "component", "server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("伺服器強制關閉異常", "component", "server", "error", err.Error())
	}
	slog.Info("伺服器資源已安全釋放，Bye!", "component", "server")
}

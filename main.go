package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"flashchat-go/handler"
	"flashchat-go/internal/logger"
	"flashchat-go/repository" // 引入倉管部門
	"flashchat-go/ws"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq" // 引入 PostgreSQL 驅動 (前面加底線代表自動註冊)
)

// ==========================================
// 👷 背景打工人邏輯：死死盯著排隊箱，把資料寫進 DB
// ==========================================
func dbWorker(id int, queue chan ws.Message, repo repository.MessageRepository) {
	slog.Info("打工人已上線", "component", "hub", "acation", "save_msg")
	// range queue 會變成一個無窮迴圈，只要箱子裡有東西，打工人就會拿出來做
	for msg := range queue {
		err := repo.SaveMessage(msg)
		if err != nil {
			slog.Error("執行PostgreSQL 失敗", "component", "database", "action", "save_msg", "error", err.Error())
		}
	}
}

func main() {
	//導入結構化日誌
	logger.InitLogger()
	slog.Info("FlashChat 服務啟動", "component", "server")

	// ==========================================
	// 準備好 Redis 的連線鑰匙 (Client)
	// ==========================================
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379", // 假設你的 Redis 跑在本機預設 Port
	})

	// ==========================================
	// 🌟 新增：準備 PostgreSQL 連線與啟動打工人
	// ==========================================
	// 1. 連線到 PostgreSQL (帳密對應你 docker-compose 的設定)

	connStr := "postgres://postgres:Ming741852@localhost:5432/flashchat?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		slog.Error("無法連線至 PostgreSQL",
			"component", "database",
			"host", "localhost",
			"db_name", "falshchat",
			"error", err.Error())
	}

	// 2. 聘請倉管員 (Repository)
	msgRepo := repository.NewPGMessageRepository(db)

	// 3. 買一個可以裝 5000 封信的排隊箱 (Buffered Channel)
	saveQueue := make(chan ws.Message, 5000)

	// 4. 聘請 3 位打工人 (Goroutine)，讓他們在背景開始盯著排隊箱
	for i := 1; i <= 3; i++ {
		go dbWorker(i, saveQueue, msgRepo)
	}

	// ==========================================
	// 🌟 核心組裝區：依賴注入 (Dependency Injection)
	// ==========================================

	// 1. 誕生一位經理 (Hub 廣播中心)
	// 【架構升級】：把 Redis 鑰匙跟排隊箱都配發給經理！
	hub := ws.NewHub(rdb, saveQueue)

	// 2. 讓經理去背景開始工作 (開始監聽上下線與廣播通道)
	go hub.Run()

	// 3. 聘請一位服務生，並把經理的聯絡方式 (hub) 配發給他
	wsHandler := handler.NewWSHandler(hub)

	// ==========================================
	// 📍 路由綁定與伺服器啟動
	// ==========================================

	// 設定路由 1：網頁靜態畫面櫃台
	http.Handle("/", http.FileServer(http.Dir("./public")))

	// 設定路由 2：WebSocket 專屬櫃台 (交給我們剛剛聘請的服務生)
	http.HandleFunc("/ws", wsHandler.HandleConnections)

	// 1. 將 Port 抽成變數，避免重複硬編碼
	port := "8080"
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: nil, // 使用預設的 ServeMux
	}

	go func() {
		slog.Info("復古終端機伺服器已啟動", "component", "server", "port", port, "service", "flashchat")
		// 如果錯誤不是因為「伺服器正常關閉」引起的，就報錯
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("伺服器啟動失敗", "component", "server", "addr", srv.Addr, "error", err.Error())
			os.Exit(1)
		}
	}()

	// ==========================================
	// 🛑 6. 實作優雅關機 (Graceful Shutdown)
	// ==========================================
	// 建立一個通道來接收作業系統的訊號

	// 建立一個通道來接收作業系統的訊號
	quit := make(chan os.Signal, 1)
	// 監聽 SIGINT (Ctrl+C) 與 SIGTERM (Docker/K8s 關閉容器的訊號)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// 主執行序會卡在這裡，直到收到訊號
	<-quit
	slog.Warn("收到終止訊號，準備進行優雅關機...", "component", "server")

	// 建立一個有 5 秒超時限制的 Context
	// 意思是：給系統 5 秒鐘處理尚未完成的請求，超時就強制關閉
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel() //確保資源在函數結束時會釋放

	// 呼叫 srv.Shutdown()，這會停止接收新連線，並等待舊連線處理完畢
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("伺服器強制關閉異常", "component", "server", "error", err.Error())
	}
	slog.Info("伺服器資源已釋放，安全結束程式", "component", "server")
}

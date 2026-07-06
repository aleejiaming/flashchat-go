package bootstrap

import (
	"flashchat-go/handler"
	"flashchat-go/internal/database"
	"flashchat-go/repository"
	"flashchat-go/ws"
	"log/slog"
)

// AppHandlers 打包了系統中所有的控制器 (對外服務生)
type AppHandlers struct {
	Auth *handler.AuthHandler
	WS   *handler.WSHandler
}

// 👷 背景打工人邏輯 (從 main 搬遷過來)
func dbWorker(id int, queue chan ws.Message, repo repository.MessageRepository) {
	slog.Info("背景打工人已上線", "component", "worker", "worker_id", id)
	for msg := range queue {
		err := repo.SaveMessage(msg)
		if err != nil {
			slog.Error("執行 PostgreSQL 寫入失敗", "component", "database", "error", err.Error())
		}
	}
}

// InitializeApp 是整個系統的「依賴注入容器 (DI Container)」
func InitializeApp(pgConnStr, redisAddr string) (*AppHandlers, error) {
	// 1. 初始化資料庫 (呼叫工廠)
	db, err := database.NewPostgresDB(pgConnStr)
	if err != nil {
		return nil, err // 資料庫連線失敗，回傳給 main 決定如何處置
	}

	rdb, err := database.NewRedisClient(redisAddr)
	if err != nil {
		return nil, err
	}

	// 2. 初始化倉管部門 (Repositories)
	msgRepo := repository.NewPGMessageRepository(db)
	userRepo := repository.NewPGUserRepository(db)

	// 3. 準備背景架構 (Channel, Workers, Hub)
	saveQueue := make(chan ws.Message, 5000)
	for i := 1; i <= 3; i++ {
		go dbWorker(i, saveQueue, msgRepo)
	}

	hub := ws.NewHub(rdb, saveQueue)
	go hub.Run() // 啟動廣播中心

	// 4. 將完成裝配的資源注入控制器 (Handlers)
	authHandler := handler.NewAuthHandler(userRepo)
	wsHandler := handler.NewWSHandler(hub)

	// 5. 將所有對外窗口打包回傳
	return &AppHandlers{
		Auth: authHandler,
		WS:   wsHandler,
	}, nil
}

package bootstrap

import (
	"flashchat-go/handler"
	"flashchat-go/internal/database"
	"flashchat-go/middleware"
	"flashchat-go/repository"
	"flashchat-go/ws"
	"log/slog"
	"net/http"
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
// func InitializeApp(pgConnStr, redisAddr string) (*http.ServeMux, *AppHandlers, error) {
func InitializeApp(pgConnStr, redisAddr string, msgRepo repository.MessageRepository, userRepo repository.UserRepository) (*http.ServeMux, *AppHandlers, error) {

	// 1. 初始化背景架構
	saveQueue := make(chan ws.Message, 5000)
	for i := 1; i <= 3; i++ {
		go dbWorker(i, saveQueue, msgRepo)
	}

	rdb, _ := database.NewRedisClient(redisAddr) // 簡化處理
	hub := ws.NewHub(rdb, saveQueue)
	go hub.Run() // 啟動廣播中心

	// 4. 將完成裝配的資源注入控制器 (Handlers)
	authHandler := handler.NewAuthHandler(userRepo)
	wsHandler := handler.NewWSHandler(hub)

	// 建立並設定專屬的路由器 (Mux)
	mux := http.NewServeMux()

	// ==========================================
	// 📍 路由綁定區 (Routing)
	// ==========================================
	mux.Handle("/", http.FileServer(http.Dir("./public")))
	// 這些是不需要 Token 的公開路由 (Public Routes)
	mux.HandleFunc("POST /register", authHandler.RegisterHandler)
	mux.HandleFunc("POST /login", authHandler.LoginHandler)
	mux.HandleFunc("POST /guest", authHandler.GuestLoginHandler)
	mux.HandleFunc("POST /refresh", authHandler.RefreshHandler)
	mux.HandleFunc("POST /logout", authHandler.LogoutHandler)

	// 🌟 這些是需要 Token 保護的私人路由 (Private Routes)
	// 使用 middleware.AuthMiddleware 把原本的 Handler 「包」起來
	// 需要驗證的 WebSocket 路由
	mux.HandleFunc("GET /ws", middleware.AuthMiddleware(wsHandler.HandleConnections))

	// 5. 將所有對外窗口打包回傳
	return mux, &AppHandlers{
		Auth: authHandler,
		WS:   wsHandler,
	}, nil
}

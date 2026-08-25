package bootstrap

import (
	"context"
	"flashchat-go/handler"
	"flashchat-go/internal/auth"
	"flashchat-go/internal/database"
	"flashchat-go/middleware"
	"flashchat-go/repository"
	"flashchat-go/ws"
	"log/slog"
	"net/http"

	// 🌟 匯入 http-swagger 套件
	"github.com/go-redis/redis/v8"
	httpSwagger "github.com/swaggo/http-swagger"

	// 🌟 必須匯入剛剛 swag init 幫你產生的 docs 資料夾！
	// 前面的底線 "_" 代表我們只需要執行它的 init()，不直接呼叫裡面的函式
	_ "flashchat-go/docs"
)

// AppHandlers 打包了系統中所有的控制器 (對外服務生)
type AppHandlers struct {
	Auth *handler.AuthHandler
	WS   *handler.WSHandler
	Km   *auth.KeyManager
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

	// 裝配KeyManager 1.1
	keyManager := auth.NewKeyManager(rdb)

	// 檢察系統有沒有主鑰匙 (primary_kid) 1.2
	ctx := context.Background()
	_, err := rdb.Get(ctx, "primary_kid").Result()

	// 將完成裝配的資源注入控制器 (Handlers)
	authHandler := handler.NewAuthHandler(userRepo, keyManager)
	wsHandler := handler.NewWSHandler(hub, keyManager)

	if err == redis.Nil { // redis.Nil 代表「找不到這個 key」
		slog.Info("系統偵測到尚無密鑰，準備進行首次初始化...")
		// 呼叫我們寫好的輪換功能，產生第一把鑰匙！
		if err := keyManager.RotateKey(ctx); err != nil {
			slog.Error("首次密鑰初始化失敗", "error", err)
			// 實務上如果連密鑰都生不出來，系統應該直接停止 (panic 或 return err)
		}
	} else if err != nil {
		slog.Error("檢查 primary_kid 發生異常", "error", err)
	}

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
	mux.HandleFunc("GET /ws", middleware.AuthMiddleware(keyManager, wsHandler.HandleConnections))

	// 這樣當你訪問 /swagger/ 時，就會吐出 Swagger UI 網頁
	mux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8081/swagger/doc.json"),
	))

	// 5. 將所有對外窗口打包回傳
	return mux, &AppHandlers{
		Auth: authHandler,
		WS:   wsHandler,
	}, nil
}

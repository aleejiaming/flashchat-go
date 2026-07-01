package main

import (
	"database/sql"
	"flashchat-go/handler"
	"flashchat-go/repository" // 引入倉管部門
	"flashchat-go/ws"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	// 🌟 新增的套件：

	_ "github.com/lib/pq" // 引入 PostgreSQL 驅動 (前面加底線代表自動註冊)
)

// ==========================================
// 👷 背景打工人邏輯：死死盯著排隊箱，把資料寫進 DB
// ==========================================
func dbWorker(id int, queue chan ws.Message, repo repository.MessageRepository) {
	log.Printf("👷 [打工人 %d] 已上線，隨時準備寫入 PostgreSQL...\n", id)
	// range queue 會變成一個無窮迴圈，只要箱子裡有東西，打工人就會拿出來做
	for msg := range queue {
		err := repo.SaveMessage(msg)
		if err != nil {
			log.Printf("❌ [打工人 %d] 寫入 PostgreSQL 失敗: %v\n", id, err)
		}
	}
}

func main() {
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
		log.Fatal("無法連線至 PostgreSQL:", err)
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

	log.Println("🚀 復古終端機伺服器已啟動於 http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("伺服器啟動失敗:", err)
	}
}

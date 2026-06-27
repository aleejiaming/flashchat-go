package main

import (
	"log"
	"net/http"

	// 🚨 請確認這裡的模組名稱是否與你的 go.mod 檔案裡寫的一致
	// 如果你當初建立專案是用 go mod init flashchat-go，那這裡就是 flashchat-go
	"flashchat-go/handler"
	"flashchat-go/ws"
)

func main() {
	// ==========================================
	// 🌟 核心組裝區：依賴注入 (Dependency Injection)
	// ==========================================
	
	// 1. 誕生一位經理 (Hub 廣播中心)
	hub := ws.NewHub()

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
package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ==========================================
// 2. WebSocket 基礎設定 (無鎖化大升級 🚀)
// ==========================================
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients    = make(map[*websocket.Conn]bool) // 點名冊
	// 🌟 刪除了煩人的 sync.Mutex 鎖！我們改用下面三個通道 (Channel) 來發號施令：
	broadcast  = make(chan Message)             // 廣播麥克風通道
	register   = make(chan *websocket.Conn)     // 客人「上線」專用通道
	unregister = make(chan *websocket.Conn)     // 客人「下線」專用通道
)

func main() {
	// 啟動廣播中心 (唯一的點名冊管理員)
	go handleMessages()

	// 設定路由
	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", handleConnections)

	log.Println("🚀 復古終端機伺服器已啟動於 http://localhost:8080 (無鎖化 Hub 模式)")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("伺服器啟動失敗:", err)
	}
}

// ==========================================
// 3. 處理單一客人的連線與收信
// ==========================================
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升級 WebSocket 失敗:", err)
		return
	}
	// 確保函式結束時，一定會發送下線紙條給管理員，並關閉連線
	defer func() {
		unregister <- ws // 🌟 魔法：不用解鎖了，直接丟下線紙條給通道
		ws.Close()
	}()

	// 🌟 魔法：客人上線了！不自己改點名冊，把客人的連線丟進「上線通道」給管理員處理
	register <- ws

	// 死死盯著這個客人，看他有沒有傳訊息來
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			// 發生錯誤 (客人關掉網頁)，跳出迴圈，觸發上面的 defer 下線機制
			break
		}
		// 把收到的訊息，丟進廣播麥克風
		broadcast <- msg
	}
}

// ==========================================
// 4. 廣播中心 (唯一的點名冊管理員)
// ==========================================
func handleMessages() {
	// 這個無窮迴圈，就是小房間裡「唯一的管理員」。
	// 因為只有他會操作 clients 這個 map，所以絕對不會有記憶體衝突，直接免除 Mutex！
	for {
		select {
		// 狀況 A：收到「上線」紙條
		case client := <-register:
			clients[client] = true
			log.Println("✅ 一位客人已上線")

		// 狀況 B：收到「下線」紙條
		case client := <-unregister:
			if _, ok := clients[client]; ok { // 檢查是否還在點名冊裡
				delete(clients, client)
				log.Println("👋 一位客人已下線")
			}

		// 狀況 C：收到「廣播」紙條
		case msg := <-broadcast:
			for client := range clients {
				err := client.WriteJSON(msg)
				if err != nil {
					// 傳送失敗，代表這條連線壞了，踢出點名冊
					client.Close()
					delete(clients, client)
				}
			}
		}
	}
}

// ==========================================
// 3. 處理單一客人的連線與收信
// ==========================================
func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("升級 WebSocket 失敗:", err)
		return
	}
	defer ws.Close()

	// 客人上線，登記到點名冊
	clientsMu.Lock()
	clients[ws] = true
	clientsMu.Unlock()

	// 死死盯著這個客人，看他有沒有傳訊息來
	for {
		var msg Message
		// 🌟 魔法：ReadJSON 會自動把前端傳來的 JSON 字串，拆解並塞進我們的 msg 變數裡！
		err := ws.ReadJSON(&msg)
		if err != nil {
			// 如果發生錯誤 (通常是客人關掉網頁)，就把他從點名冊刪除
			clientsMu.Lock()
			delete(clients, ws)
			clientsMu.Unlock()
			break
		}
		// 把收到的訊息，丟進廣播麥克風
		broadcast <- msg
	}
}

// ==========================================
// 4. 廣播中心 (把麥克風的聲音傳給所有人)
// ==========================================
func handleMessages() {
	for {
		// 從廣播麥克風拿出訊息 (如果麥克風沒聲音，打工人會在這裡睡覺等待)
		msg := <-broadcast

		clientsMu.Lock()
		for client := range clients {
			// 🌟 魔法：WriteJSON 會自動把 msg 變數打包成 JSON 字串，傳給客人的瀏覽器！
			err := client.WriteJSON(msg)
			if err != nil {
				client.Close()
				delete(clients, client)
			}
		}
		clientsMu.Unlock()
	}
}
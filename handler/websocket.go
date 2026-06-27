package handler

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"

	// 🚨 請確認這行，把 flashchat-go 換成你 go.mod 裡寫的模組名稱
	"flashchat-go/ws"
)

// 升級器放在這裡很正確，因為只有接客時會用到
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ==========================================
// 1. 定義 Handler 結構體 (服務生)
// ==========================================
type WSHandler struct {
	// 🌟 關鍵：服務生的口袋裡帶著經理 (Hub) 的聯絡方式！			
	//ws.Hub 指的就是 資料夾 ws , Hub 是在 ws 裡面定義的結構體	
	Hub *ws.Hub
}


// 2. 工廠：聘請一位新的服務生
//這個 被 NewWSHandler 創造出來的新服務生 h
// 服務生 h 他可以 取得 Hub 的結構資料 
func NewWSHandler(h *ws.Hub) *WSHandler {
	return &WSHandler{
		Hub: h,
	}
}

// ==========================================
// 3. 接客邏輯
// ==========================================
func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("更新失敗:", err)
		return
	}

	// 離開時，把客人丟給經理的下線通道
	defer func() {
		h.Hub.Unregister <- conn
		conn.Close()
	}()

	// 報到時，把客人丟給經理的上線通道
	h.Hub.Register <- conn

	for {
		var msg ws.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		// 收到訊息，直接丟給經理的廣播通道
		h.Hub.Broadcast <- msg
	}
}
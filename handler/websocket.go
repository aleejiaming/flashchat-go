package handler

import (
	"log/slog"
	"net/http"
	"time" // 🌟 記得引入時間套件，為了實作 10 分鐘踢人機制

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
	// ws.Hub 指的就是 資料夾 ws , Hub 是在 ws 裡面定義的結構體
	Hub *ws.Hub
}

// 2. 工廠：聘請一位新的服務生
// 這個被 NewWSHandler 創造出來的新服務生 h
// 服務生 h 他可以取得 Hub 的結構資料
func NewWSHandler(h *ws.Hub) *WSHandler {
	return &WSHandler{
		Hub: h,
	}
}

// ==========================================
// 3. 接客邏輯
// ==========================================
func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	// 這裡的升級是把客戶端的連線，變成不單單只是一次的請求類型，而是可以持續、連續的請求狀態
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebScoket 升級失敗",
			"client_ip", r.RemoteAddr,
			"error", err.Error(),
		)
		return
	}

	// 離開時，把客人丟給經理的下線通道
	defer func() {
		h.Hub.Unregister <- conn
		conn.Close()
	}()

	// 報到時，把客人丟給經理的上線通道
	h.Hub.Register <- conn

	// ⏱️ 踢人機制 1：客人一進門，立刻設定 10 分鐘的死亡倒數計時！

	conn.SetReadDeadline(time.Now().Add(10 * time.Minute)) // 如果 10 分鐘內沒收到訊息，conn.ReadJSON 就會直接報錯，進而中斷連線。

	for {
		var rawMsg ws.Message
		err := conn.ReadJSON(&rawMsg)
		if err != nil {
			slog.Error("WebSocket 升級失敗",
				"client_ip", r.RemoteAddr,
				"reason", err.Error(),
			)
			break
		}

		// ⏱️ 踢人機制 2：客人有講話了！馬上幫他重置另外 10 分鐘的倒數計時器
		conn.SetReadDeadline(time.Now().Add(10 * time.Minute))

		// 🌟 核心修改：服務生在這裡負責把「客人」跟「紙條」裝進透明信封袋！
		clientMsg := ws.ClientMessage{
			Client: conn,   // 把客人的連線 (座位號碼) 綁上去
			Msg:    rawMsg, // 客人實際輸入的文字
		}

		// 收到訊息，把【信封袋】丟給經理的廣播通道
		h.Hub.Broadcast <- clientMsg
	}
}

package handler

import (
	"flashchat-go/internal/auth"
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
	km  *auth.KeyManager // 🌟 新增：服務生的第二個口袋，裝著倉管員的聯絡方式
}

// 2. 工廠：聘請一位新的服務生
// 這個被 NewWSHandler 創造出來的新服務生 h
// 服務生 h 他可以取得 Hub 的結構資料
// 🌟 修改：老闆聘請他時，除了給經理 (h)，也要給他倉管員 (km)
func NewWSHandler(h *ws.Hub, km *auth.KeyManager) *WSHandler {
	return &WSHandler{
		Hub: h,
		km:  km, // 🌟 把老闆給的倉管員放進口袋
	}
}

// ==========================================
// 3. 接客邏輯
// ==========================================
func (h *WSHandler) HandleConnections(w http.ResponseWriter, r *http.Request) {
	// 1. 身分驗證 (Authentication)
	token := r.URL.Query().Get("token")

	// 使用 auth 模組的真實解析邏輯
	// 🌟 修改：直接傳遞 r.Context() 和口袋裡的 h.km。並且用 claims 來接收！
	claims, err := auth.ValidateToken(r.Context(), h.km, token)
	// 這裡的升級是把客戶端的連線，變成不單單只是一次的請求類型，而是可以持續、連續的請求狀態
	if err != nil {
		slog.Warn("拒絕未授權連線", "component", "ws_handler", "ip", r.RemoteAddr, "error", err.Error())
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 🌟 黑名單機制：檢查該 UID 是否已被全域封鎖！
	isBanned, err := h.km.IsUserBlacklisted(r.Context(), claims.UID)
	if err != nil {
		slog.Error("WebSocket 驗證黑名單失敗", "uid", claims.UID, "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if isBanned {
		slog.Warn("拒絕被封鎖用戶建立 WebSocket 連線",
			"uid", claims.UID,
			"username", claims.Name,
			"ip", r.RemoteAddr,
		)
		// 直接中斷連線，回傳 403 Forbidden
		http.Error(w, "Account is banned", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("WebSocket 讀取失敗",
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

	// ⏱️ 配合心跳機制 減少 zombie 連線，設定 60 秒的讀取時間限制，超過就自動斷線

	conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // 如果 1分鐘內沒收到訊息，conn.ReadJSON 就會直接報錯，進而中斷連線。

	for {
		var rawMsg ws.Message
		err := conn.ReadJSON(&rawMsg)
		if err != nil {
			slog.Error("WebSocket 讀取失敗",
				"client_ip", r.RemoteAddr,
				"reason", err.Error(),
			)
			break
		}

		// ⏱️ 踢人機制 2：客人有講話了！馬上幫他重置另外 1 分鐘的倒數計時器
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// ==========================================
		// 🌟 核心優化：網路層攔截心跳，不讓訊息進入 Hub
		// ==========================================
		if rawMsg.Content == "/ping" {
			// 直接由當下連線回傳，背景處理完畢
			conn.WriteJSON(ws.Message{
				Username: "SYSTEM",
				Content:  "pong",
			})
			continue // 終止本次迴圈，跳過廣播
		}

		// 把剛剛驗證拿到的 userName，寫進客人傳來的訊息裡！
		// 這樣大家才知道這句話是誰說的
		rawMsg.Username = claims.Name
		clientMsg := ws.ClientMessage{
			Client: conn,   // 把客人的連線 (座位號碼) 綁上去
			Msg:    rawMsg, // 已經貼上 userName 名字的紙條
		}

		// 收到訊息，把【信封袋】丟給經理的廣播通道
		h.Hub.Broadcast <- clientMsg
	}
}

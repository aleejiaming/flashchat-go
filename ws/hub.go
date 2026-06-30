package ws

import (
	"context"
	"encoding/json"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
)

type Message struct {
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsPrivate  bool   `json:"-"` // 加上 json:"-" 代表這只是後端內部用的，不會傳給前端
	TargetName string `json:"-"` // 🌟 新增：這封信的「指定收件人」是誰？
}

type Hub struct {
	// 🌟 關鍵升級：右邊的 value (原本是使用 bool)變成了 string，用來記憶客人的名字！
	Clients    map[*websocket.Conn]string
	Broadcast  chan ClientMessage
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	// 🌟 新增 1：經理現在口袋裡多了一把打開 Redis 倉庫的鑰匙
	RedisClient *redis.Client
}

type ClientMessage struct {
	Client *websocket.Conn
	Msg    Message
}

func NewHub(rdb *redis.Client) *Hub {

	return &Hub{
		Clients:     make(map[*websocket.Conn]string), // 對應改為 string
		Broadcast:   make(chan ClientMessage),
		Register:    make(chan *websocket.Conn),
		Unregister:  make(chan *websocket.Conn),
		RedisClient: rdb, // 把鑰匙收進口袋
	}
}

func (h *Hub) Run() {
	log.Println("📡 [Hub] 廣播中心管理員已就緒...")
	ctx := context.Background()

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = ""
			log.Println("✅ 客人上線")

			// 🌟 新增 3：客人一進門，經理立刻去 Redis 翻找歷史紀錄！
			// LRange 意思是「拿出 List 裡面的資料」，0 到 -1 代表「從第一筆拿到最後一筆」
			history, err := h.RedisClient.LRange(ctx, "chat_history", 0, -1).Result()
			if err == nil {
				// 把歷史紀錄一筆一筆發給這個剛進門的客人
				for _, jsonStr := range history {
					var oldMsg Message
					// 把 JSON 字串解開變回 Message 結構體
					json.Unmarshal([]byte(jsonStr), &oldMsg)
					// 偷偷塞給這個客人 (不廣播)
					client.WriteJSON(oldMsg)
				}
			}

		case client := <-h.Unregister:
			if name, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				log.Printf("👋 客人下線: %s\n", name)
			}

		case clientMsg := <-h.Broadcast:
			h.Clients[clientMsg.Client] = clientMsg.Msg.Name

			processor := GetProcessor(clientMsg.Msg.Content)
			finalMsg := processor.Process(clientMsg.Msg)

			if finalMsg.IsPrivate {
				if finalMsg.TargetName != "" {
					// 模式 A：點對點私訊
					targetFound := false
					for clientConn, clientName := range h.Clients {
						if clientName == finalMsg.TargetName {
							clientConn.WriteJSON(finalMsg)
							targetFound = true
						}
					}
					clientMsg.Client.WriteJSON(finalMsg)
					if !targetFound {
						errorMsg := Message{
							Name:    "🤖 系統機器人",
							Content: "找不到使用者「" + finalMsg.TargetName + "」，他可能離線了。",
						}
						clientMsg.Client.WriteJSON(errorMsg)
					}
				} else {
					// 模式 B：系統悄悄話 (/help, /weather)
					clientMsg.Client.WriteJSON(finalMsg)
				}
			} else {
				// 🌍 模式 C：全伺服器廣播 (一般的聊天訊息)

				// 🌟 新增 4：把準備廣播的訊息，打包成 JSON 字串
				msgBytes, _ := json.Marshal(finalMsg)

				// 🌟 新增 5：把 JSON 字串塞進 Redis 的 "chat_history" 列表的最右邊 (RPush)
				h.RedisClient.RPush(ctx, "chat_history", string(msgBytes))

				// 🌟 新增 6：修剪列表 (LTrim)，永遠只保留最新的 50 筆！(-50 到 -1 代表倒數 50 個)
				h.RedisClient.LTrim(ctx, "chat_history", -50, -1)

				// 最後，照常廣播給所有人
				for client := range h.Clients {
					err := client.WriteJSON(finalMsg)
					if err != nil {
						client.Close()
						delete(h.Clients, client)
					}
				}
			}
		}
	}
}

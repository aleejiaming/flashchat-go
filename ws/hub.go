package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

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
	SaveQueue   chan Message // 🌟 補上這個：歷史紀錄排隊箱
}

type ClientMessage struct {
	Client *websocket.Conn
	Msg    Message
}

func NewHub(rdb *redis.Client, saveQueue chan Message) *Hub {

	return &Hub{
		Clients:     make(map[*websocket.Conn]string), // 對應改為 string
		Broadcast:   make(chan ClientMessage),
		Register:    make(chan *websocket.Conn),
		Unregister:  make(chan *websocket.Conn),
		RedisClient: rdb,       // 把鑰匙收進口袋
		SaveQueue:   saveQueue, // 🌟 補上這個：把參數收進口袋

	}
}

func (h *Hub) Run() {
	slog.Info("📡 [Hub] 廣播中心管理員已就緒", "component", "hub")
	ctx := context.Background()

	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = ""
			// 組合拳：記錄部門 + 動作 + 當前人數
			slog.Info("客人建立連線",
				"component", "hub",
				"action", "client_connect",
				"active_clients", len(h.Clients),
			)

			// 🌟 ZRANGE 讀取：因為我們有定時修剪，所以 ZSET 裡剩下的保證都是 7 天內的
			// 0 到 -1 代表從第一筆拿到最後一筆 (按時間順序)
			history, err := h.RedisClient.ZRange(ctx, "chat_history", 0, -1).Result()
			// 筆記 因為 ZRange 是用來操作 Redis 的 Sorted Set（有序集合） 資料型態，
			// 所以你可以把 "chat_history" 想像成是這整條聊天紀錄集合的「總名稱」。
			// 所有使用者的聊天紀錄（或這房間的紀錄）都被塞進了這個叫 "chat_history" 的 Key 裡面。
			if err == nil {
				for _, jsonStr := range history {
					var oldMsg Message
					json.Unmarshal([]byte(jsonStr), &oldMsg)
					// 偷偷塞給這個剛進門的客人，不廣播給其他人
					client.WriteJSON(oldMsg)
				}
			}
		// ------------------------------------------
		// 🔴 客人下線
		// ------------------------------------------
		case client := <-h.Unregister:
			if name, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				slog.Info("👋 客人下線", "client_name", name, "active_clients", len(h.Clients))
			}
		// ------------------------------------------
		// 📢 處理廣播訊息 (寫入 Redis ZSET)
		// ------------------------------------------

		case clientMsg := <-h.Broadcast:
			// 記住客人的名字
			h.Clients[clientMsg.Client] = clientMsg.Msg.Name
			// 找主廚加工訊息
			processor := GetProcessor(clientMsg.Msg.Content)
			finalMsg := processor.Process(clientMsg.Msg)
			// 🔒 模式 A/B：私訊或系統悄悄話 (注意：私訊【不會】存入 Redis 歷史紀錄)
			if finalMsg.IsPrivate {
				if finalMsg.TargetName != "" {
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
					clientMsg.Client.WriteJSON(finalMsg)
				}
			} else {
				// 🌍 模式 C：全伺服器廣播 (一般的聊天訊息)

				// 🌟 1. 取得現在的時間戳
				now := time.Now().Unix() //取得現在的時間
				msgBytes, _ := json.Marshal(finalMsg)
				// 🌟 2. 存入 Redis ZSET (分數 = 時間戳)、以及對話紀錄
				h.RedisClient.ZAdd(ctx, "chat_history", &redis.Z{
					Score:  float64(now),
					Member: string(msgBytes),
				})
				// 🌟 3. 執行清理：移除 7 天前的舊訊息 (7天 = 604800 秒)
				sevenDaysAgo := now - 604800
				h.RedisClient.ZRemRangeByScore(ctx, "chat_history", "-inf", fmt.Sprintf("%d", sevenDaysAgo))

				// 🌟 補上這行！把訊息丟進排隊箱，讓 PostgreSQL 打工人去存檔
				h.SaveQueue <- finalMsg

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

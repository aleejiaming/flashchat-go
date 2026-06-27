package ws

import (
	"log"

	"github.com/gorilla/websocket"
)

// Message 定義了前後端溝通的格式
type Message struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Hub struct {
	Clients    map[*websocket.Conn]bool
	Broadcast  chan Message
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*websocket.Conn]bool),
		Broadcast:  make(chan Message),
		Register:   make(chan *websocket.Conn),
		Unregister: make(chan *websocket.Conn),
	}
}

func (h *Hub) Run() {
	log.Println("📡 [Hub] 廣播中心管理員已就緒...")
	for {
		select {
		case client := <-h.Register:
			h.Clients[client] = true
			log.Println("✅ 客人上線")

		case client := <-h.Unregister:
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				log.Println("👋 客人下線")
			}

		case msg := <-h.Broadcast:
			// 🌟 關鍵拼圖：我們讓大廳經理拿著訊息，去詢問工廠要怎麼處理！
			processor := GetProcessor(msg.Content)
			finalMsg := processor.Process(msg)

			// 廣播「處理過後的新訊息 (finalMsg)」給所有人
			for client := range h.Clients {
				client.WriteJSON(finalMsg)
			}
		}
	}
}
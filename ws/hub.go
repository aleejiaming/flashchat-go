package ws

import (
	"log"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// 🌟 關鍵升級：右邊的 value (原本是使用 bool)變成了 string，用來記憶客人的名字！
	Clients    map[*websocket.Conn]string
	Broadcast  chan Message
	Register   chan *websocket.Conn
	Unregister chan *websocket.Conn
	
}

type ClientMessage struct {
	Client *websocket.Conn
	Msg    Message
}

func NewHub() *Hub {
	
	return &Hub{
		Clients:    make(map[*websocket.Conn]string), // 對應改為 string
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
			h.Clients[client] = "" // 剛進門還沒講話，名字先留空
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
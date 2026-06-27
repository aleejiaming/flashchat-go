⚡ FlashChat 系統架構與設計模式說明書

本專案採用 模組化設計 (Modular Design) 與 工廠模式 (Factory Pattern)，並深度運用 Go 語言的 無鎖化併發 (Lock-free Concurrency) 特性。

這份文件旨在紀錄系統中各個資料夾與檔案的「職責邊界 (Responsibilities)」，確保未來在擴充功能時（例如新增 API 機器人、連接資料庫），不會破壞現有的架構整潔度。

📂 目錄結構與職責對照

flashchat-go/
├── public/                  # 🌐 靜態資源區：復古終端機 HTML/CSS/JS
├── handler/
│   └── websocket.go         # 🛎️ 接收層：負責網路協定 (HTTP 升級 WS)
├── ws/
│   ├── hub.go               # 📡 狀態層：負責管理連線池、無鎖化廣播
│   └── processor.go         # 🏭 邏輯層：負責解析指令 (機器人工廠)
└── main.go                  # 🚀 組裝線：依賴注入 (DI) 與啟動伺服器


🏛️ 核心架構解析 (用餐廳比喻來理解)

1. main.go ➔ 【總部組裝線】

職責：專案的唯一入口。它不做任何具體的工作，只負責「生出零件並組裝」。

設計模式：依賴注入 (Dependency Injection, DI)。

運作邏輯：在這裡我們建立 Hub，然後把它「注入（塞進）」WebSocketHandler 的口袋裡。這樣一來，外場服務生跟內場大廳經理就完美地串聯起來了，且模組之間維持低耦合。

2. handler/websocket.go ➔ 【外場服務生】

職責：處理所有與「網路通訊」有關的髒活。

守則：絕對不包含商業邏輯！

運作邏輯：

客人連線時，負責把 HTTP 升級成 WebSocket。

負責監聽客人的瀏覽器，一旦收到 JSON 訊息，連看都不看內容，直接往後台大喊（呼叫 hub.BroadcastMsg()）交給後端處理。

客人斷線時，通知後台將其從點名冊移除。

3. ws/hub.go ➔ 【大廳經理 / 廣播中心】

職責：管理全局狀態 (State Management)，維護「現在有誰在線上」的點名冊 (clients map)。

核心技術：無鎖化併發 (Lock-free / Multiplexing)。

運作邏輯：

摒棄傳統容易死鎖的 sync.Mutex。

透過一個獨立背景執行的 Run() 無窮迴圈搭配 select 語法，統一監聽 Register、Unregister、broadcast 三個 Channel。

因為永遠只有這個「唯一的管理員」在修改點名冊，所以徹底消成了記憶體衝突的風險。

4. ws/processor.go ➔ 【內場主廚 / 機器人工廠】

職責：處理真正的「商業邏輯 (Business Logic)」，例如判斷使用者是不是輸入了指令。

設計模式：工廠模式 (Factory Pattern) + 介面 (Interface)。

運作邏輯：

定義了 MessageProcessor 合約，確保所有機器人都有 Process 方法。

GetProcessor 工廠會檢查訊息開頭，如果是 /weather 就派出天氣機器人，如果是 /help 就派出教學機器人。

擴充性極高 (符合開閉原則 OCP)：未來若要新增 /stock (查股票) 功能，只需在這裡新增一個 StockProcessor 結構體，其他檔案 (main.go, hub.go, handler) 完全不需要修改！

🔄 訊息的生命週期 (Data Flow)

當客人在瀏覽器輸入 /weather 台北 並按下發送時，資料在系統中的流動圖解與步驟如下：

sequenceDiagram
    autonumber
    actor User as [前端] public/index.html
    participant Handler as [接收層] handler/websocket.go
    participant Hub as [狀態層] ws/hub.go
    participant Factory as [邏輯層] ws/processor.go

    User->>Handler: 發送 JSON 訊息 ("/weather 台北")
    Handler->>Hub: 解析後丟進 hub.broadcast 通道
    Hub->>Factory: 喚醒 select，將內容丟給工廠
    Note over Factory: 根據開頭判定<br/>回傳 WeatherProcessor
    Factory-->>Hub: 產出加工後的訊息 ("今天天氣晴朗...")
    Hub->>User: 迴圈 WriteJSON 廣播給所有在線客戶端
    Note over User: 瀏覽器收到 JSON<br/>觸發綠色彈幕與畫面更新

public/
├── index.html       # 僅保留純 HTML 結構與資源引入
├── css/
│   └── style.css    # 抽離所有 CSS 樣式
└── js/
    ├── main.js      # 總控制器 (相當於 main.go)
    ├── ui.js        # 畫面渲染模組 (負責 DOM 操作、彈幕、側邊欄)
    ├── socket.js    # 網路通訊模組 (負責 WebSocket 連線、心跳)
    └── auth.js      # 身分驗證模組 (負責處理登入與 Token)
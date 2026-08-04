# FlashChat API 參考文件 (API Reference)

## 1. 總覽 (Overview)

本文件定義了 FlashChat 前後端分離架構下的通訊介面，包含基於 HTTP 的身分驗證 (RESTful 风格)，以及基於 WebSocket 的即時通訊協議。

*   **Base URL (本地開發):** `http://localhost:8081`
*   **資料格式:** 所有 HTTP 請求與回應的 `Content-Type` 皆為 `application/json` (除非另有說明)。

---

## 2. 身分驗證模組 (Authentication API)

### 2.1 註冊新會員

*   **Endpoint:** `POST /register`
*   **Description:** 建立新的會員帳號與密碼。

**Request Body**
```json
{
  "username": "Mike",
  "password": "mySecurePassword"
}
```

**Responses**
*   **200 OK:** 註冊成功。
*   **400/500 Bad Request / Server Error:** 註冊失敗（如帳號已存在）。
```text
"使用者名稱已被註冊或系統異常"
```

---

### 2.2 會員登入

*   **Endpoint:** `POST /login`
*   **Description:** 驗證帳號密碼。登入成功後，前端會獲得短期的 JWT Token，而後端會自動在瀏覽器寫入帶有 HttpOnly 屬性的 Refresh Token Cookie。

**Request Body**
```json
{
  "username": "Mike",
  "password": "mySecurePassword"
}
```

**Responses**
*   **200 OK:** 登入成功。
```json
{
  "token": "eyJhbGciOiJIUzI1Ni...",
  "username": "Mike"
}
```
*   **401 Unauthorized:** 帳號或密碼錯誤。

---

### 2.3 遊客登入

*   **Endpoint:** `POST /guest`
*   **Description:** 免密碼登入，系統會核發訪客專用的 Token。

**Request Body**
```json
{
  "username": "Guest_1234"
}
```

**Responses**
*   **200 OK:** 遊客登入成功。
```json
{
  "token": "eyJhbGciOiJIUzI1Ni...",
  "username": "Guest_1234"
}
```

---

### 2.4 刷新憑證 / 自動恢復連線

*   **Endpoint:** `POST /refresh`
*   **Description:** 使用者重整網頁時，透過瀏覽器自動帶上的 HttpOnly Cookie 來換發新的 JWT Token，實現自動登入。

**Request Headers**
*   需附帶有效的 Cookie (由瀏覽器自動處理)。

**Responses**
*   **200 OK:** 換發成功。
```json
{
  "token": "eyJhbGciOiJIUzI1Ni...",
  "username": "Mike"
}
```
*   **401 Unauthorized:** 無效的登入狀態或 Cookie 已過期。

---

### 2.5 登出

*   **Endpoint:** `POST /logout`
*   **Description:** 登出系統，後端會清除瀏覽器上的 HttpOnly Cookie。

**Responses**
*   **200 OK:** 登出成功。

---

## 3. 即時通訊模組 (WebSocket API)

### 3.1 建立 WebSocket 連線

*   **Endpoint:** `WS /ws`
*   **Description:** 將 HTTP 連線升級為 WebSocket，需在 Query String 中帶上驗證用的 JWT Token。

**Request Parameter (Query String)**

| 參數名稱 | 型別 | 必填 | 說明 |
| :--- | :--- | :--- | :--- |
| `token` | `string` | 是 | 透過 `/login` 或 `/guest` 取得的 JWT Token |

**Connection URI 範例:**
`ws://localhost:8081/ws?token=eyJhbGciOi...`

---

### 3.2 WebSocket 訊息格式 (Message Payload)

連線建立後，前後端互傳的資料皆必須符合以下 JSON 格式：

**Client 發送訊息給 Server (Send)**
```json
{
  "username": "Mike",
  "content": "大家好，今天天氣真好！"
}
```
*(系統指令如 `/weather 台北` 或 `/help` 也是放在 content 欄位中直接發送)*

**Client 接收系統心跳 (Ping/Pong)**
Client 每 30 秒需發送一次 Ping 訊息維持連線：
```json
{
  "name": "Mike",
  "content": "/ping"
}
```

**Server 廣播給 Client (Receive)**
```json
{
  "username": "Mike",
  "content": "大家好，今天天氣真好！"
}
```
*(如果發送人為系統機器人，username 可能會是 "🤖 系統機器人" 或 "🌤️ 天氣機器人")*
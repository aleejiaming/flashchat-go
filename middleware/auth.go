package middleware

import (
	"context"
	"flashchat-go/internal/auth"
	"log/slog"
	"net/http"
	"strings"
)

// 定義私有型別 contextKey，專門用於此 Package 的 Context，避免其
type contextKey string

// 宣告常數作為 Key，確保全域唯一性
const UserContextKey contextKey = "username"

// AuthMiddleware 是一個中介軟體，負責攔截請求並驗證 JWT
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 宣告一個空字串準備接收 Token
		var tokenString string

		// 1. 從 HTTP 標頭 (Header) 中取出 Authorization 欄位
		// 格式通常為： "Bearer xxxxx.yyyyy.zzzzz"
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1] // 成功從 Header 拿到 Token
			}
		}
		// 如果 Header 裡面空空如也，那可能是 WebSocket 連線！
		// 我們改去 URL 的查詢字串 (Query String) 裡面撈撈看有沒有 "?token=..."
		if tokenString == "" {
			tokenString = r.URL.Query().Get("token")
		}
		// 如果兩邊都找不到，直接把這個請求踢回前端 (回傳 401)
		if tokenString == "" {
			slog.Warn("【除錯】未能在 URL Query 中撈到 Token")
			http.Error(w, "缺少驗證憑證", http.StatusUnauthorized)
			return
		}

		// 3. 呼叫 auth 模組來驗證 Token，實際的 JWT 解析邏輯：驗證簽章、過期時間，並提取 payload
		username, err := auth.ValidateToken(tokenString)
		if err != nil {
			slog.Error("【除錯】JWT 驗證失敗", "error_detail", err.Error())
			// 如果過期或被篡改，拒絕請求
			http.Error(w, "Token 無效或已過期", http.StatusUnauthorized)
			return
		}
		// ==========================================
		// 🌟 商業邏輯核心：優雅的 Context (上下文) 傳遞
		// ==========================================
		// 將解析出來的「使用者名稱」塞進這個 HTTP 請求專屬的背包 (Context) 裡。
		// 這樣後面的 Handler 只要伸手進背包，就能知道現在是誰在操作了！
		ctx := context.WithValue(r.Context(), UserContextKey, username)

		// 產生一個帶有新背包的 Request
		rWithContext := r.WithContext(ctx)
		// 4. 警衛放行！把帶有名字的請求交給下一個處理器 (也就是真正的 Handler)
		next(w, rWithContext)
	}
}

// 封裝一個公開的輔助函式 (Helper)，讓下游的 Handler 能夠安全地提取 Username
func GetUsername(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(UserContextKey).(string)
	return username, ok

}

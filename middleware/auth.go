package middleware

import (
	"context"
	"net/http"
	"strings"

	"flashchat-go/internal/auth"
)

// AuthMiddleware 是一個中介軟體，負責攔截請求並驗證 JWT
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 從 HTTP 標頭 (Header) 中取出 Authorization 欄位
		// 格式通常為： "Bearer xxxxx.yyyyy.zzzzz"
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "缺少 Authorization 標頭", http.StatusUnauthorized)
			return
		}
		// 2. 檢查字串是否以 "Bearer " 開頭，並把 Token 切割出來
		parts := strings.SplitN(authHeader, "", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Authhorization 格式錯誤 (必須是Bearer <token>)", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		// 3. 呼叫您之前寫好的 auth 模組來驗證 Token
		username, err := auth.ValidateToken(tokenString)
		if err != nil {
			// 如果過期或被篡改，拒絕請求
			http.Error(w, "Token 無效或已過期", http.StatusUnauthorized)
			return
		}
		// ==========================================
		// 🌟 商業邏輯核心：優雅的 Context (上下文) 傳遞
		// ==========================================
		// 將解析出來的「使用者名稱」塞進這個 HTTP 請求專屬的背包 (Context) 裡。
		// 這樣後面的 Handler 只要伸手進背包，就能知道現在是誰在操作了！
		ctx := context.WithValue(r.Context(), "username", username)

		// 產生一個帶有新背包的 Request
		rWithContext := r.WithContext(ctx)
		// 4. 警衛放行！把帶有名字的請求交給下一個處理器 (也就是真正的 Handler)
		next(w, rWithContext)
	}
}

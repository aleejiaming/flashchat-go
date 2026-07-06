package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"flashchat-go/internal/auth"
	"flashchat-go/repository"

	"golang.org/x/crypto/bcrypt"
)

// ==========================================
// 1. 定義控制器結構 (Controller / Handler)
// ==========================================

type AuthHandler struct {
	UserRepo repository.UserRepository
}

func NewAuthHandler(repo repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		UserRepo: repo,
	}
}

// 定義前端傳來的 JSON 請求格式
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ==========================================
// 2. 註冊邏輯 (Register)
// ==========================================
func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON Request", http.StatusBadRequest)
		return
	}
	// 基礎防呆：帳號密碼不得為空
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		http.Error(w, "帳號密碼不得為空", http.StatusBadRequest)
		return
	}

	// 🛡️ 資安防禦 1：Bcrypt 密碼雜湊加密 (加鹽)
	// DefaultCost 為 10，能在安全性與效能間取得平衡
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("密碼加密失敗", "error", err.Error())

		// http.StatusInternalServerError 這是一個極為標準且安全的錯誤處理模式：
		//對內（Server-side）：將詳細的錯誤原因（err.Error()）寫入伺服器內部的 Log 中，讓開發或維運人員可以進行除錯（Debugging）。
		//對外（Client-side）：絕對不向客戶端暴露底層的錯誤細節（例如「密碼超過 72 bytes」或「記憶體不足」）
		//。只回傳模糊且標準的 500 Internal Server Error。這能防止攻擊者透過錯誤訊息的差異來探測系統內部的實作細節
		// （防禦 CWE-209：生成錯誤訊息時暴露敏感資訊）。
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// 將帳號與加密後的 Hash 存入資料庫
	err = h.UserRepo.CreateUser(req.Username, string(hashedPassword))
	if err != nil {
		// 🛡️ 業務邏輯：判斷是否為重複註冊 (UNIQUE 限制衝突)
		// PostgreSQL 的重複鍵錯誤通常包含 "unique constraint" 或 "duplicate key"
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "duplicate key") {
			http.Error(w, "此帳號已被註冊", http.StatusConflict) // 409 Conflict
			return
		}
		http.Error(w, "資料庫寫入失敗", http.StatusInternalServerError)
		return
	}
	// 回傳 201 Created 代表資源成功建立
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"message": "註冊成功"}`))
}

// ==========================================
// 3. 登入邏輯 (Login) - 升級為真實資料庫驗證
// ==========================================
func (h *AuthHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	//採用守衛子句（Guard Clause）與早期返回（Early Return）模式
	//這是一種軟體工程中的防禦性程式設計（Defensive Programming)
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON Request", http.StatusBadRequest)
		return
	}
	// 1. 去資料庫查詢是否有這個帳號
	user, err := h.UserRepo.GetUserByUsername(req.Username)
	if err != nil {
		// 🌟 新增詳細錯誤日誌
		slog.Error("登入失敗：資料庫找不到此帳號或查詢異常",
			"username", req.Username,
			"error", err.Error(),
		)
		http.Error(w, "帳號或密碼錯誤", http.StatusUnauthorized)
		return
	}
	// 2. 🛡️ 資安防禦：比對前端傳來的明文密碼與資料庫中的 Hash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		// 🌟 新增詳細錯誤日誌
		slog.Error("登入失敗：密碼 Hash 比對錯誤",
			"username", req.Username,
			"error", err.Error(),
			"hash_in_db", user.PasswordHash, // 偷偷印出資料庫裡的 Hash 長度來檢查
		)
		http.Error(w, "帳號或密碼錯誤", http.StatusUnauthorized)
		return
	}

	// 2. 帳密驗證成功，呼叫 auth 模組簽發 JWT Token
	tokenString, err := auth.GenerateToken(user.Username)
	if err != nil {
		slog.Error("Token 簽發失敗", "component", "auth", "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. 回傳 JSON 格式的 Token 給前端
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":    tokenString,
		"username": user.Username,
	})
}

// ==========================================
// 4. 遊客登入邏輯 (Guest Login)
// ==========================================
func (h *AuthHandler) GuestLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON Request", http.StatusBadRequest)
		return
	}

	// 如果沒填暱稱，預設給 Anonymous
	nickname := strings.TrimSpace(req.Username)
	if nickname == "" {
		nickname = "Anonymous"
	}

	//🛡️ 商業邏輯：為了區分正式會員與遊客，強制加上 [遊客] 標籤
	finalName := "[遊客]" + nickname

	// 直接發放 JWT
	tokenString, err := auth.GenerateToken(finalName)
	if err != nil {
		http.Error(w, "Token 發放失敗", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":    tokenString,
		"username": finalName,
	})
}

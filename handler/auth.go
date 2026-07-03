package handler

import (
	"encoding/json"
	"flashchat-go/internal/auth"
	"log/slog"
	"net/http"
)

type LoginRequest struct {
	Name string `json:"name"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid Request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		req.Name = "Guest"
	}

	// 呼叫 auth 模組簽發 Token
	tokenString, err := auth.GenerateToken(req.Name)
	if err != nil {
		slog.Error("Token 簽發失敗", "component", "auth", "error", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 回傳 JSON 格式的 Token
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
}
